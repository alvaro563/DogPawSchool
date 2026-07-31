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
	activityuc "dogpaw/internal/usecase/activity"
	authuc "dogpaw/internal/usecase/auth"
	doguc "dogpaw/internal/usecase/dog"
	incompatuc "dogpaw/internal/usecase/incompatibility"
	passuc "dogpaw/internal/usecase/pass"
	reservationuc "dogpaw/internal/usecase/reservation"
	useruc "dogpaw/internal/usecase/user"
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

	// Use cases
	loginUC := authuc.NewLoginUseCase(userRepo, hasher, tokenGen)
	changePasswordUC := authuc.NewChangePasswordUseCase(userRepo, hasher, hasher)
	authH := NewAuthHandler(nil, loginUC, changePasswordUC)

	dogRepo := postgres.NewDogRepository(db)
	incompatRepo := postgres.NewIncompatibilityRepository(db)
	transactor := postgres.NewTransactor(db)
	activityRepo := postgres.NewActivityRepository(db)
	passRepo := postgres.NewPassRepository(db)
	reservationRepo := postgres.NewReservationRepository(db)

	dogUC := doguc.NewRegisterDogUseCase(dogRepo)
	getDogUC := doguc.NewGetDogUseCase(dogRepo)
	listAllDogUC := doguc.NewListAllDogsUseCase(dogRepo)
	listByOwnerDogUC := doguc.NewListByOwnerUseCase(dogRepo)
	listActiveDogUC := doguc.NewListActiveDogsUseCase(dogRepo)
	listByIsActiveDogUC := doguc.NewListByIsActiveUseCase(dogRepo)
	listByIncompatDogUC := doguc.NewListByIncompatibilityUseCase(dogRepo)
	listByBreedDogUC := doguc.NewListByBreedUseCase(dogRepo)
	listBySexDogUC := doguc.NewListBySexUseCase(dogRepo)
	listByNeuteredDogUC := doguc.NewListByNeuteredUseCase(dogRepo)
	listByHeatDogUC := doguc.NewListByHeatUseCase(dogRepo)
	listByAgeDogUC := doguc.NewListByAgeBracketUseCase(dogRepo)
	listBySizeDogUC := doguc.NewListBySizeBracketUseCase(dogRepo)
	modifyDogUC := doguc.NewModifyDogUseCase(transactor, dogRepo)
	addIncompatDogUC := doguc.NewAddDogIncompatibilityUseCase(transactor, dogRepo, incompatRepo)
	removeIncompatDogUC := doguc.NewRemoveDogIncompatibilityUseCase(transactor, dogRepo)
	deleteDogUC := doguc.NewDeleteDogUseCase(dogRepo)
	setNeuteredDogUC := doguc.NewSetDogNeuteredUseCase(transactor, dogRepo)
	setHeatDogUC := doguc.NewSetDogHeatUseCase(transactor, dogRepo)

	dogH := NewDogHandler(
		dogUC, getDogUC, listAllDogUC, listByOwnerDogUC, listActiveDogUC,
		listByIsActiveDogUC, listByIncompatDogUC, listByBreedDogUC, listBySexDogUC,
		listByNeuteredDogUC, listByHeatDogUC, listByAgeDogUC, listBySizeDogUC,
		modifyDogUC, addIncompatDogUC, removeIncompatDogUC, deleteDogUC,
		setNeuteredDogUC, setHeatDogUC,
	)

	getUserUC := useruc.NewGetUserUseCase(userRepo)
	listUsersUC := useruc.NewListUsersUseCase(userRepo)
	updateUserUC := useruc.NewUpdateUserUseCase(userRepo)
	deactivateUserUC := useruc.NewDeactivateUserUseCase(userRepo)
	listUserEmailsUC := useruc.NewListUserEmailsUseCase(userRepo)
	userH := NewUserHandler(getUserUC, listUsersUC, updateUserUC, deactivateUserUC, listUserEmailsUC)

	registerIncompatUC := incompatuc.NewRegisterIncompatibilityUseCase(incompatRepo)
	listIncompatUC := incompatuc.NewListIncompatibilitiesUseCase(incompatRepo)
	getIncompatUC := incompatuc.NewGetIncompatibilityUseCase(incompatRepo)
	modifyIncompatUC := incompatuc.NewModifyIncompatibilityUseCase(incompatRepo)
	deleteIncompatUC := incompatuc.NewDeleteIncompatibilityUseCase(incompatRepo)
	incompatH := NewIncompatibilityHandler(
		registerIncompatUC, listIncompatUC, getIncompatUC, modifyIncompatUC, deleteIncompatUC,
	)

	registerActivityUC := activityuc.NewRegisterActivityUseCase(activityRepo)
	getActivityUC := activityuc.NewGetActivityUseCase(activityRepo)
	modifyActivityUC := activityuc.NewModifyActivityUseCase(activityRepo)
	listAllActivityUC := activityuc.NewListAllActivitiesUseCase(activityRepo)
	listUpcomingActivityUC := activityuc.NewListUpcomingActivitiesUseCase(activityRepo)
	closeActivityUC := activityuc.NewCloseActivityUseCase(
		transactor, activityRepo, dogRepo, reservationRepo,
		reservationuc.NewMarkReservationNoShowUseCase(transactor, activityRepo, dogRepo, reservationRepo),
		reservationuc.NewCompleteReservationUseCase(transactor, activityRepo, dogRepo, reservationRepo),
	)
	activityH := NewActivityHandler(
		registerActivityUC, getActivityUC, modifyActivityUC,
		listAllActivityUC, listUpcomingActivityUC, closeActivityUC,
	)

	registerPassUC := passuc.NewRegisterPassUseCase(passRepo)
	modifyPassUC := passuc.NewModifyPassUseCase(passRepo)
	getPassUC := passuc.NewGetPassUseCase(passRepo)
	listAllPassUC := passuc.NewListAllPassesUseCase(passRepo)
	listByUserPassUC := passuc.NewListByUserPassesUseCase(passRepo)
	passH := NewPassHandler(registerPassUC, modifyPassUC, getPassUC, listAllPassUC, listByUserPassUC)

	registerReservationUC := reservationuc.NewRegisterReservationUseCase(
		transactor, activityRepo, dogRepo, passRepo, reservationRepo,
	)
	cancelReservationUC := reservationuc.NewCancelReservationUseCase(
		transactor, activityRepo, dogRepo, passRepo, reservationRepo,
	)
	markNoShowUC := reservationuc.NewMarkReservationNoShowUseCase(
		transactor, activityRepo, dogRepo, reservationRepo,
	)
	completeUC := reservationuc.NewCompleteReservationUseCase(
		transactor, activityRepo, dogRepo, reservationRepo,
	)
	getReservationUC := reservationuc.NewGetReservationUseCase(reservationRepo)
	listByUserReservationsUC := reservationuc.NewListByUserReservationsUseCase(reservationRepo)
	listUpcomingByUserReservationsUC := reservationuc.NewListUpcomingByUserUseCase(reservationRepo)
	listByDogReservationsUC := reservationuc.NewListByDogReservationsUseCase(reservationRepo)
	listByPassReservationsUC := reservationuc.NewListByPassReservationsUseCase(reservationRepo)
	listByActivityReservationsUC := reservationuc.NewListByActivityReservationsUseCase(reservationRepo)
	reservationH := NewReservationHandler(
		registerReservationUC, cancelReservationUC,
		getReservationUC, listByUserReservationsUC, listUpcomingByUserReservationsUC,
		listByDogReservationsUC, listByPassReservationsUC, listByActivityReservationsUC,
		markNoShowUC, completeUC,
	)

	v1 := r.Group("/api/v1")
	{
		// Public
		v1.POST("/auth/login", authH.Login)

		// Any authenticated user
		anyUser := v1.Group("")
		anyUser.Use(AuthRequired(jwtTestSecret))
		{
			anyUser.PATCH("/auth/password", authH.ChangePassword)
			anyUser.GET("/users/:user_id", userH.GetByID)
			anyUser.GET("/users/:user_id/passes", passH.ListByUser)
			anyUser.POST("/users/:user_id/reservations", reservationH.Register)
			anyUser.POST("/users/:user_id/reservations/:id/cancel", reservationH.Cancel)
			anyUser.GET("/users/:user_id/reservations", reservationH.ListByUser)
			anyUser.GET("/users/:user_id/reservations/upcoming", reservationH.ListUpcomingByUser)
			anyUser.GET("/users/:user_id/reservations/:id", reservationH.GetByID)
			anyUser.GET("/dogs/:id", dogH.GetByID)
			anyUser.GET("/dogs/owner/:owner_id", dogH.ListByOwner)
			anyUser.GET("/activities", activityH.List)
			anyUser.GET("/activities/upcoming", activityH.ListUpcoming)
			anyUser.GET("/activities/:id", activityH.GetByID)
		}

		// Admin only
		admin := v1.Group("")
		admin.Use(AuthRequired(jwtTestSecret))
		admin.Use(AdminRequired())
		{
			admin.GET("/users", userH.List)
			admin.PATCH("/users/:user_id", userH.Update)
			admin.POST("/users/:user_id/deactivate", userH.Deactivate)
			admin.GET("/users/emails", userH.ListEmails)
			admin.POST("/dogs", dogH.Register)
			admin.GET("/dogs", dogH.List)
			admin.GET("/dogs/active", dogH.ListActive)
			admin.GET("/dogs/is_active/:value", dogH.ListByIsActive)
			admin.GET("/dogs/incompatibility/:incompat_id", dogH.ListByIncompatibility)
			admin.GET("/dogs/breed/:breed", dogH.ListByBreed)
			admin.GET("/dogs/sex/:sex", dogH.ListBySex)
			admin.GET("/dogs/neutered/:value", dogH.ListByNeutered)
			admin.GET("/dogs/heat/:value", dogH.ListByHeat)
			admin.GET("/dogs/age/:bracket", dogH.ListByAgeBracket)
			admin.GET("/dogs/size/:bracket", dogH.ListBySizeBracket)
			admin.PATCH("/dogs/:id", dogH.Modify)
			admin.PATCH("/dogs/:id/neutered", dogH.SetNeutered)
			admin.PATCH("/dogs/:id/heat", dogH.SetHeat)
			admin.DELETE("/dogs/:id", dogH.Delete)
			admin.POST("/dogs/:id/incompatibilities", dogH.AddIncompatibility)
			admin.DELETE("/dogs/:id/incompatibilities/:incompatibility_id", dogH.RemoveIncompatibility)
			admin.POST("/incompatibilities", incompatH.Register)
			admin.GET("/incompatibilities", incompatH.List)
			admin.GET("/incompatibilities/:id", incompatH.GetByID)
			admin.PATCH("/incompatibilities/:id", incompatH.Modify)
			admin.DELETE("/incompatibilities/:id", incompatH.Delete)
			admin.POST("/activities", activityH.Register)
			admin.PATCH("/activities/:id", activityH.Modify)
			admin.POST("/activities/:id/close", activityH.Close)
			admin.POST("/users/:user_id/passes", passH.Register)
			admin.GET("/passes", passH.List)
			admin.GET("/passes/:id", passH.GetByID)
			admin.PATCH("/passes/:id", passH.Modify)
			admin.POST("/users/:user_id/reservations/:id/no-show", reservationH.MarkNoShow)
			admin.POST("/users/:user_id/reservations/:id/complete", reservationH.CompleteReservation)
			admin.GET("/dogs/:id/reservations", reservationH.ListByDog)
			admin.GET("/passes/:id/reservations", reservationH.ListByPass)
			admin.GET("/activities/:id/reservations", reservationH.ListByActivity)
		}
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

// --- Authz: admin endpoints ---

func seedAdminUser(t *testing.T, db *sql.DB, email, plainPassword string) *domain.User {
	t.Helper()
	hasher := crypto.NewDefaultBcryptHasher()
	hashed, err := hasher.Hash(plainPassword)
	require.NoError(t, err)
	user, err := domain.NewUser(0, "Admin User", email, hashed, domain.RoleAdmin)
	require.NoError(t, err)
	userRepo := postgres.NewUserRepository(db)
	userID, err := userRepo.Create(context.Background(), user)
	require.NoError(t, err)
	require.Positive(t, userID)
	user, err = domain.NewUser(userID, user.Name(), user.Email(), user.Password(), user.Role())
	require.NoError(t, err)
	return user
}

func loginAndGetToken(t *testing.T, router *gin.Engine, email, password string) string {
	t.Helper()
	w := loginHTTP(router, email, password)
	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	token, _ := body["token"].(string)
	require.NotEmpty(t, token)
	return token
}

func TestAuthz_AdminListUsers_Success(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	seedAdminUser(t, integrationDB, "admin-list@dogpaw.com", "admin-pw-123")
	seedTestUser(t, integrationDB, "user1@dogpaw.com", "pw1")
	seedTestUser(t, integrationDB, "user2@dogpaw.com", "pw2")

	token := loginAndGetToken(t, router, "admin-list@dogpaw.com", "admin-pw-123")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	users, ok := body["users"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(users), 3)
}

func TestAuthz_RegularUserBlockedFromListUsers(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	seedTestUser(t, integrationDB, "regular@dogpaw.com", "reg-pw")
	token := loginAndGetToken(t, router, "regular@dogpaw.com", "reg-pw")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "forbidden")
}

func TestAuthz_AdminCanListAllUserEmails(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	seedAdminUser(t, integrationDB, "admin-emails@dogpaw.com", "admin-pw-123")
	seedTestUser(t, integrationDB, "emails-user1@dogpaw.com", "pw1")
	seedTestUser(t, integrationDB, "emails-user2@dogpaw.com", "pw2")

	token := loginAndGetToken(t, router, "admin-emails@dogpaw.com", "admin-pw-123")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/emails", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Emails []string `json:"emails"`
		Count  int      `json:"count"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.GreaterOrEqual(t, len(body.Emails), 3)
	assert.Equal(t, len(body.Emails), body.Count)
	assert.NotContains(t, w.Body.String(), "password")
}

func TestAuthz_RegularUserBlockedFromListUserEmails(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	seedTestUser(t, integrationDB, "regular-emails@dogpaw.com", "reg-pw")
	token := loginAndGetToken(t, router, "regular-emails@dogpaw.com", "reg-pw")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/emails", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "forbidden")
}

func TestAuthz_RegularUserBlockedFromCreateDog(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	user := seedTestUser(t, integrationDB, "reg-create-dog@dogpaw.com", "reg-pw")
	token := loginAndGetToken(t, router, "reg-create-dog@dogpaw.com", "reg-pw")

	body := fmt.Sprintf(`{"name":"Luna","breed":"Labrador","age_in_months":24,"sex":"FEMALE","weight_kg":22.5,"passport":"ES-CREATE","user_id":%d}`, user.ID())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dogs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "forbidden")
}

func TestAuthz_AdminCanCreateDog(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	seedAdminUser(t, integrationDB, "admin-create@dogpaw.com", "admin-pw")
	user := seedTestUser(t, integrationDB, "owner-for-dog@dogpaw.com", "owner-pw")
	token := loginAndGetToken(t, router, "admin-create@dogpaw.com", "admin-pw")

	body := fmt.Sprintf(`{"name":"Luna","breed":"Labrador","age_in_months":24,"sex":"FEMALE","weight_kg":22.5,"passport":"ES-ADMIN-CREATE","user_id":%d}`, user.ID())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dogs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAuthz_AdminCanListAnyUserDogs(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	_ = seedAdminUser(t, integrationDB, "admin-list-dogs@dogpaw.com", "admin-pw")
	owner := seedTestUser(t, integrationDB, "owner-list-dogs@dogpaw.com", "owner-pw")

	adminToken := loginAndGetToken(t, router, "admin-list-dogs@dogpaw.com", "admin-pw")

	// Admin creates a dog for the owner
	dogBody := fmt.Sprintf(`{"name":"Luna","breed":"Labrador","age_in_months":24,"sex":"FEMALE","weight_kg":22.5,"passport":"ES-OWNER-DOG","user_id":%d}`, owner.ID())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dogs", strings.NewReader(dogBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Admin lists owner's dogs
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/dogs/owner/%d", owner.ID()), nil)
	req2.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestAuthz_OwnerCanSeeOwnDog(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	_ = seedAdminUser(t, integrationDB, "admin-ownerdog@dogpaw.com", "admin-pw")
	owner := seedTestUser(t, integrationDB, "owner-see-dog@dogpaw.com", "owner-pw")

	adminToken := loginAndGetToken(t, router, "admin-ownerdog@dogpaw.com", "admin-pw")

	// Admin creates dog for owner
	dogBody := fmt.Sprintf(`{"name":"Luna","breed":"Labrador","age_in_months":24,"sex":"FEMALE","weight_kg":22.5,"passport":"ES-SEE-DOG","user_id":%d}`, owner.ID())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dogs", strings.NewReader(dogBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Extract dog ID from Location header
	dogID := strings.TrimPrefix(w.Header().Get("Location"), "/api/v1/dogs/")

	// Owner logs in and sees their dog
	ownerToken := loginAndGetToken(t, router, "owner-see-dog@dogpaw.com", "owner-pw")
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/dogs/%s", dogID), nil)
	req2.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestAuthz_OtherUserBlockedFromDog(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	_ = seedAdminUser(t, integrationDB, "admin-otherdog@dogpaw.com", "admin-pw")
	owner := seedTestUser(t, integrationDB, "owner-other-dog@dogpaw.com", "owner-pw")
	_ = seedTestUser(t, integrationDB, "other-user@dogpaw.com", "other-pw")

	adminToken := loginAndGetToken(t, router, "admin-otherdog@dogpaw.com", "admin-pw")

	// Admin creates dog for owner
	dogBody := fmt.Sprintf(`{"name":"Luna","breed":"Labrador","age_in_months":24,"sex":"FEMALE","weight_kg":22.5,"passport":"ES-OTHER-DOG","user_id":%d}`, owner.ID())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/dogs", strings.NewReader(dogBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	dogID := strings.TrimPrefix(w.Header().Get("Location"), "/api/v1/dogs/")

	// Other user tries to see the dog
	otherToken := loginAndGetToken(t, router, "other-user@dogpaw.com", "other-pw")
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/dogs/%s", dogID), nil)
	req2.Header.Set("Authorization", "Bearer "+otherToken)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusForbidden, w2.Code)
}

func TestAuthz_RegularUserSeeOwnProfile(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	user := seedTestUser(t, integrationDB, "own-profile@dogpaw.com", "pw")
	token := loginAndGetToken(t, router, "own-profile@dogpaw.com", "pw")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%d", user.ID()), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthz_RegularUserBlockedFromOtherProfile(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	_ = seedTestUser(t, integrationDB, "other-profile@dogpaw.com", "pw")
	regular := seedTestUser(t, integrationDB, "regular-profile@dogpaw.com", "pw")

	token := loginAndGetToken(t, router, "regular-profile@dogpaw.com", "pw")

	// Try to access the other user's profile (ID=1)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%d", regular.ID()+1), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAuthz_AdminSeeAnyProfile(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	_ = seedAdminUser(t, integrationDB, "admin-anyprofile@dogpaw.com", "admin-pw")
	user := seedTestUser(t, integrationDB, "some-user@dogpaw.com", "pw")

	adminToken := loginAndGetToken(t, router, "admin-anyprofile@dogpaw.com", "admin-pw")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%d", user.ID()), nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
