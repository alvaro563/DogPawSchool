package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"dogpaw/internal/crypto"
	"dogpaw/internal/domain"
	"dogpaw/internal/repository/postgres"
	authuc "dogpaw/internal/usecase/auth"
	"dogpaw/migrations"
)

var integrationDB *sql.DB
var jwtTestSecret = "integration-test-secret"

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	c, connStr, err := startPostgresContainer(ctx)
	if err != nil {
		log.Fatalf("auth integration: start container: %v", err)
	}
	defer func() {
		if err := c.Terminate(ctx); err != nil {
			log.Printf("auth integration: terminate container: %v", err)
		}
	}()

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("auth integration: sql.Open: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("auth integration: db.Ping: %v", err)
	}

	if err := runMigrations(db); err != nil {
		log.Fatalf("auth integration: migrate: %v", err)
	}

	integrationDB = db
	os.Exit(m.Run())
}

func startPostgresContainer(ctx context.Context) (*tcpostgres.PostgresContainer, string, error) {
	c, err := tcpostgres.Run(ctx,
		"postgres:15-alpine",
		tcpostgres.WithDatabase("dogpaw_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategyAndDeadline(
			60*time.Second,
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		return nil, "", err
	}
	connStr, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, "", err
	}
	return c, connStr, nil
}

func runMigrations(db *sql.DB) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	defer src.Close()
	driver, err := migratepg.WithInstance(db, &migratepg.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func cleanIntegrationTables(t *testing.T, db *sql.DB) {
	t.Helper()
	tables := []string{
		"pass_movements", "reservations", "invitations",
		"dog_incompatibilities", "passes", "dogs",
		"activities", "incompatibilities", "users",
	}
	_, err := db.ExecContext(context.Background(),
		"TRUNCATE TABLE "+strings.Join(tables, ",")+" RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func buildAuthTestRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	userRepo := postgres.NewUserRepository(db)
	hasher := crypto.NewDefaultBcryptHasher()
	tokenGen := crypto.NewJWTTokenGenerator(jwtTestSecret, 1*time.Hour)

	loginUC := authuc.NewLoginUseCase(userRepo, hasher, tokenGen)
	changePasswordUC := authuc.NewChangePasswordUseCase(userRepo, hasher, hasher)
	authH := NewAuthHandler(nil, loginUC, changePasswordUC)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/login", authH.Login)
	}

	authProtected := v1.Group("/auth")
	authProtected.Use(AuthRequired(jwtTestSecret))
	{
		authProtected.PATCH("/password", authH.ChangePassword)
	}

	return r
}

func seedTestUser(t *testing.T, db *sql.DB, email, plainPassword string) *domain.User {
	t.Helper()
	hasher := crypto.NewDefaultBcryptHasher()
	hashed, err := hasher.Hash(plainPassword)
	require.NoError(t, err)
	user, err := domain.NewUser(0, "Integration User", email, hashed, domain.RoleRegular)
	require.NoError(t, err)
	userRepo := postgres.NewUserRepository(db)
	userID, err := userRepo.Create(context.Background(), user)
	require.NoError(t, err)
	require.Positive(t, userID)
	user, err = domain.NewUser(userID, user.Name(), user.Email(), user.Password(), user.Role())
	require.NoError(t, err)
	return user
}

func seedInactiveTestUser(t *testing.T, db *sql.DB, email, plainPassword string) *domain.User {
	t.Helper()
	user := seedTestUser(t, db, email, plainPassword)
	user.Deactivate()
	userRepo := postgres.NewUserRepository(db)
	err := userRepo.Update(context.Background(), user)
	require.NoError(t, err)
	return user
}

func loginHTTP(router *gin.Engine, email, password string) *httptest.ResponseRecorder {
	body := fmt.Sprintf(`{"email":"%s","password":"%s"}`, email, password)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

func changePasswordHTTP(router *gin.Engine, token, oldPassword, newPassword string) *httptest.ResponseRecorder {
	body := fmt.Sprintf(`{"old_password":"%s","new_password":"%s"}`, oldPassword, newPassword)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(w, req)
	return w
}

// --- Login HTTP tests ---

func TestLoginHTTPSuccess(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	seedTestUser(t, integrationDB, "http-login@dogpaw.com", "correct-password-123")

	w := loginHTTP(router, "http-login@dogpaw.com", "correct-password-123")
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	token, ok := body["token"].(string)
	require.True(t, ok, "response must contain a token")
	require.NotEmpty(t, token)

	claims, err := crypto.ParseToken(token, []byte(jwtTestSecret))
	require.NoError(t, err, "token must be parseable")
	assert.Positive(t, claims.UserID)
	assert.Equal(t, "REGULAR", claims.Role)

	userMap, ok := body["user"].(map[string]interface{})
	require.True(t, ok, "response must contain user object")
	assert.Equal(t, "http-login@dogpaw.com", userMap["email"])
}

func TestLoginHTTPWrongPassword(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	seedTestUser(t, integrationDB, "wrong-pw@dogpaw.com", "real-password")

	w := loginHTTP(router, "wrong-pw@dogpaw.com", "not-the-real-password")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginHTTPEmailNotFound(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	w := loginHTTP(router, "nobody@dogpaw.com", "any-password")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginHTTPInactiveUser(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	seedInactiveTestUser(t, integrationDB, "inactive@dogpaw.com", "some-password")

	w := loginHTTP(router, "inactive@dogpaw.com", "some-password")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Change Password HTTP tests ---

func TestChangePasswordHTTPSuccess(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	oldPw := "old-password-123"
	newPw := "new-secure-password-456"
	seedTestUser(t, integrationDB, "pw-change@dogpaw.com", oldPw)

	// Step 1: Login to get a token
	loginW := loginHTTP(router, "pw-change@dogpaw.com", oldPw)
	require.Equal(t, http.StatusOK, loginW.Code)
	var loginBody map[string]interface{}
	require.NoError(t, json.Unmarshal(loginW.Body.Bytes(), &loginBody))
	token := loginBody["token"].(string)

	// Step 2: Change password with valid token
	w := changePasswordHTTP(router, token, oldPw, newPw)
	assert.Equal(t, http.StatusOK, w.Code)
	var respBody map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &respBody))
	assert.Equal(t, "password_updated", respBody["message"])

	// Step 3: Old password no longer works
	w2 := loginHTTP(router, "pw-change@dogpaw.com", oldPw)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)

	// Step 4: New password works
	w3 := loginHTTP(router, "pw-change@dogpaw.com", newPw)
	require.Equal(t, http.StatusOK, w3.Code)
	var loginBody3 map[string]interface{}
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &loginBody3))
	token3 := loginBody3["token"].(string)
	require.NotEmpty(t, token3)

	claims, err := crypto.ParseToken(token3, []byte(jwtTestSecret))
	require.NoError(t, err)
	assert.Positive(t, claims.UserID)
}

func TestChangePasswordHTTPNoToken(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	w := changePasswordHTTP(router, "", "old-pw", "new-pw")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestChangePasswordHTTPInvalidToken(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	w := changePasswordHTTP(router, "invalid-token-that-is-not-a-valid-jwt", "old-pw", "new-pw")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestChangePasswordHTTPWrongOldPassword(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	seedTestUser(t, integrationDB, "wrong-old@dogpaw.com", "real-old-password")

	loginW := loginHTTP(router, "wrong-old@dogpaw.com", "real-old-password")
	require.Equal(t, http.StatusOK, loginW.Code)
	var loginBody map[string]interface{}
	require.NoError(t, json.Unmarshal(loginW.Body.Bytes(), &loginBody))
	token := loginBody["token"].(string)

	w := changePasswordHTTP(router, token, "not-the-real-old", "new-password")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestChangePasswordHTTPSamePassword(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	samePw := "my-password-123"
	seedTestUser(t, integrationDB, "same-pw@dogpaw.com", samePw)

	loginW := loginHTTP(router, "same-pw@dogpaw.com", samePw)
	require.Equal(t, http.StatusOK, loginW.Code)
	var loginBody map[string]interface{}
	require.NoError(t, json.Unmarshal(loginW.Body.Bytes(), &loginBody))
	token := loginBody["token"].(string)

	w := changePasswordHTTP(router, token, samePw, samePw)
	assert.Equal(t, http.StatusConflict, w.Code)
}
