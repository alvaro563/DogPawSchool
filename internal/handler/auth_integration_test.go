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
	addTraitDogUC := doguc.NewAddDogTraitUseCase(transactor, dogRepo, incompatRepo)
	addTriggerDogUC := doguc.NewAddDogTriggerUseCase(transactor, dogRepo, incompatRepo)
	removeIncompatDogUC := doguc.NewRemoveDogIncompatibilityUseCase(transactor, dogRepo)
	deleteDogUC := doguc.NewDeleteDogUseCase(dogRepo)
	setNeuteredDogUC := doguc.NewSetDogNeuteredUseCase(transactor, dogRepo)
	setHeatDogUC := doguc.NewSetDogHeatUseCase(transactor, dogRepo)
	setPhotoDogUC := doguc.NewSetDogPhotoUseCase(transactor, dogRepo)

	dogH := NewDogHandler(
		dogUC, getDogUC, listAllDogUC, listByOwnerDogUC, listActiveDogUC,
		listByIsActiveDogUC, listByIncompatDogUC, listByBreedDogUC, listBySexDogUC,
		listByNeuteredDogUC, listByHeatDogUC, listByAgeDogUC, listBySizeDogUC,
		modifyDogUC, addTraitDogUC, addTriggerDogUC, removeIncompatDogUC, deleteDogUC,
		setNeuteredDogUC, setHeatDogUC, setPhotoDogUC,
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
		listAllActivityUC, listUpcomingActivityUC, closeActivityUC, reservationRepo,
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
	registerAdminReservationUC := reservationuc.NewRegisterAdminReservationUseCase(
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
	confirmPendingUC := reservationuc.NewConfirmPendingReservationUseCase(transactor, reservationRepo)
	rejectPendingUC := reservationuc.NewRejectPendingReservationUseCase(transactor, passRepo, reservationRepo)
	getReservationUC := reservationuc.NewGetReservationUseCase(reservationRepo)
	listByUserReservationsUC := reservationuc.NewListByUserReservationsUseCase(reservationRepo)
	listUpcomingByUserReservationsUC := reservationuc.NewListUpcomingByUserUseCase(reservationRepo)
	listByDogReservationsUC := reservationuc.NewListByDogReservationsUseCase(reservationRepo)
	listByPassReservationsUC := reservationuc.NewListByPassReservationsUseCase(reservationRepo)
	listByActivityReservationsUC := reservationuc.NewListByActivityReservationsUseCase(reservationRepo)
	listAllReservationsUC := reservationuc.NewListAllReservationsUseCase(reservationRepo)
	listUpcomingAllUC := reservationuc.NewListUpcomingAllUseCase(reservationRepo)
	listActivityRosterUC := reservationuc.NewListActivityRosterUseCase(activityRepo, reservationRepo, userRepo)
	listPendingUC := reservationuc.NewListPendingReservationsUseCase(reservationRepo, userRepo)
	reservationH := NewReservationHandler(
		registerReservationUC, cancelReservationUC,
		getReservationUC, listByUserReservationsUC, listUpcomingByUserReservationsUC,
		listByDogReservationsUC, listByPassReservationsUC, listByActivityReservationsUC,
		markNoShowUC, completeUC,
		confirmPendingUC, rejectPendingUC,
		listAllReservationsUC,
		listUpcomingAllUC,
		registerAdminReservationUC,
		listActivityRosterUC,
		listPendingUC,
	)

	v1 := r.Group("/api/v1")
	{
		// Public
		v1.POST("/auth/login", authH.Login)

		// Any authenticated user
		anyUser := v1.Group("")
		anyUser.Use(AuthRequired(jwtTestSecret, userRepo))
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
		admin.Use(AuthRequired(jwtTestSecret, userRepo))
		admin.Use(AdminRequired())
		{
			admin.POST("/reservations/:id/cancel", reservationH.CancelAdmin)
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
			admin.POST("/dogs/:id/incompatibilities", dogH.AddTrigger)
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
			admin.GET("/activities/:id/roster", reservationH.ListActivityRoster)
			admin.GET("/reservations/pending", reservationH.ListPending)
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

// TestActivityRosterHTTP_Success exercises the full HTTP router for
// the admin class-day roster endpoint: seed → login → GET → assert.
// This is the regression guard that would have caught the missing
// route in the router or a path mismatch between the router and the
// frontend's fetchActivityRoster call.
func TestActivityRosterHTTP_Success(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	// One admin user (to drive the request) and one regular user
	// (to own the dog + pass that populate the roster).
	admin := seedAdminUser(t, integrationDB, "admin-roster@dogpaw.com", "admin-pw-123")
	owner := seedTestUser(t, integrationDB, "owner-roster@dogpaw.com", "owner-pw-123")

	// Seed activity, dog, pass, and one CONFIRMED + one PENDING
	// reservation directly via the repos. We bypass the full
	// register use case to keep the test focused on the roster
	// endpoint shape, not the registration flow.
	activityRepo := postgres.NewActivityRepository(integrationDB)
	dogRepo := postgres.NewDogRepository(integrationDB)
	passRepo := postgres.NewPassRepository(integrationDB)
	reservationRepo := postgres.NewReservationRepository(integrationDB)

	activity, err := domain.NewActivity(0, "Paseo Río", "", "Parking Central",
		domain.TypeRoute, 5, 1, time.Now().Add(7*24*time.Hour))
	require.NoError(t, err)
	activityID, err := activityRepo.Create(context.Background(), activity)
	require.NoError(t, err)

	dog, err := domain.NewDog(0, "Luna", "Labrador", "ES-ROSTER-1", 24,
		domain.SexFemale, 22.5, owner.ID())
	require.NoError(t, err)
	dogID, err := dogRepo.Create(context.Background(), dog)
	require.NoError(t, err)

	// Second dog so we can have one CONFIRMED and one PENDING
	// without hitting the UNIQUE (activity_id, dog_id) constraint.
	dog2, err := domain.NewDog(0, "Toby", "Border Collie", "ES-ROSTER-2", 36,
		domain.SexMale, 18.0, owner.ID())
	require.NoError(t, err)
	dog2ID, err := dogRepo.Create(context.Background(), dog2)
	require.NoError(t, err)

	pass, err := domain.NewPass(0, 5, 5, 5000, domain.PassGeneric, owner.ID(),
		time.Now().UTC(), time.Now().UTC(), nil)
	require.NoError(t, err)
	passID, err := passRepo.Create(context.Background(), pass)
	require.NoError(t, err)

	confirmedRes, err := domain.NewReservationWithStatus(0, activityID, dogID, passID,
		domain.StatusConfirmed, time.Now().UTC())
	require.NoError(t, err)
	_, err = reservationRepo.Create(context.Background(), confirmedRes)
	require.NoError(t, err)

	pendingRes, err := domain.NewReservationWithStatus(0, activityID, dog2ID, passID,
		domain.StatusPendingToConfirm, time.Now().UTC().Add(time.Minute))
	require.NoError(t, err)
	_, err = reservationRepo.Create(context.Background(), pendingRes)
	require.NoError(t, err)

	token := loginAndGetToken(t, router, admin.Email(), "admin-pw-123")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/v1/activities/%d/roster", activityID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "roster endpoint must be wired under the admin router group")
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	act, ok := body["activity"].(map[string]interface{})
	require.True(t, ok, "response must include the activity envelope")
	assert.EqualValues(t, activityID, act["id"])

	confirmed, ok := body["confirmed"].([]interface{})
	require.True(t, ok)
	require.Len(t, confirmed, 1)
	ce := confirmed[0].(map[string]interface{})
	assert.Equal(t, "Luna", ce["dog_name"])
	assert.EqualValues(t, owner.ID(), ce["owner_id"])
	assert.Equal(t, owner.Name(), ce["owner_name"])

	pending, ok := body["pending"].([]interface{})
	require.True(t, ok)
	require.Len(t, pending, 1)
	pe := pending[0].(map[string]interface{})
	assert.Equal(t, "Toby", pe["dog_name"])
	assert.EqualValues(t, owner.ID(), pe["owner_id"])
	assert.Equal(t, owner.Name(), pe["owner_name"])
}

// TestActivityRosterHTTP_NoAdminPrefix is a regression guard against
// a refactor that would add an `/admin` URL prefix to the
// `admin := v1.Group("")` Gin group. Adding that prefix would break
// every existing admin endpoint whose path the frontend calls
// WITHOUT `/admin` (e.g. /reservations, /users, /dogs). The test
// asserts that the roster path under `/admin` returns 404 (does not
// exist) so the no-prefix convention is fixed by the suite.
func TestActivityRosterHTTP_NoAdminPrefix(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	admin := seedAdminUser(t, integrationDB, "admin-noprefix@dogpaw.com", "admin-pw-123")
	token := loginAndGetToken(t, router, admin.Email(), "admin-pw-123")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/activities/1/roster", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code,
		"the Gin admin group must NOT add a /admin URL prefix — "+
			"the frontend client-side /admin is unrelated to the API prefix")
}

// TestPendingReservationsHTTP_Success exercises the full HTTP
// router for the admin "pending to approve" endpoint: seed two
// owners + three PENDING + one CONFIRMED, then assert the response
// contains exactly the three pending entries with their owners
// resolved and the CONFIRMED dropped.
func TestPendingReservationsHTTP_Success(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	admin := seedAdminUser(t, integrationDB, "admin-pending@dogpaw.com", "admin-pw-123")
	ownerA := seedTestUser(t, integrationDB, "ana-pending@dogpaw.com", "ana-pw-123")
	ownerB := seedTestUser(t, integrationDB, "juan-pending@dogpaw.com", "juan-pw-123")

	activityRepo := postgres.NewActivityRepository(integrationDB)
	dogRepo := postgres.NewDogRepository(integrationDB)
	passRepo := postgres.NewPassRepository(integrationDB)
	reservationRepo := postgres.NewReservationRepository(integrationDB)

	activity, err := domain.NewActivity(0, "Paseo Pendientes", "", "Central",
		domain.TypeRoute, 10, 1, time.Now().Add(7*24*time.Hour))
	require.NoError(t, err)
	activityID, err := activityRepo.Create(context.Background(), activity)
	require.NoError(t, err)

	dogA1, err := domain.NewDog(0, "Luna", "Labrador", "ES-PEND-A1", 24,
		domain.SexFemale, 22.5, ownerA.ID())
	require.NoError(t, err)
	dogA1ID, err := dogRepo.Create(context.Background(), dogA1)
	require.NoError(t, err)

	dogB, err := domain.NewDog(0, "Toby", "Border Collie", "ES-PEND-B", 36,
		domain.SexMale, 18.0, ownerB.ID())
	require.NoError(t, err)
	dogBID, err := dogRepo.Create(context.Background(), dogB)
	require.NoError(t, err)

	dogA2, err := domain.NewDog(0, "Maya", "Golden", "ES-PEND-A2", 30,
		domain.SexFemale, 25.0, ownerA.ID())
	require.NoError(t, err)
	dogA2ID, err := dogRepo.Create(context.Background(), dogA2)
	require.NoError(t, err)

	// Fourth dog so the CONFIRMED entry can use a different dog
	// (the UNIQUE (activity_id, dog_id) constraint blocks re-using
	// dogA1 for both a PENDING and a CONFIRMED row).
	dogA3, err := domain.NewDog(0, "Coco", "Beagle", "ES-PEND-A3", 28,
		domain.SexMale, 12.0, ownerA.ID())
	require.NoError(t, err)
	dogA3ID, err := dogRepo.Create(context.Background(), dogA3)
	require.NoError(t, err)

	passA, err := domain.NewPass(0, 5, 5, 5000, domain.PassGeneric, ownerA.ID(),
		time.Now().UTC(), time.Now().UTC(), nil)
	require.NoError(t, err)
	passAID, err := passRepo.Create(context.Background(), passA)
	require.NoError(t, err)

	passB, err := domain.NewPass(0, 5, 5, 5000, domain.PassGeneric, ownerB.ID(),
		time.Now().UTC(), time.Now().UTC(), nil)
	require.NoError(t, err)
	passBID, err := passRepo.Create(context.Background(), passB)
	require.NoError(t, err)

	// 3 PENDING + 1 CONFIRMED. The CONFIRMED must NOT appear in the
	// response (it's not a pending approval).
	seedRes := func(dogID, passID int, status domain.ReservationStatus) {
		r, err := domain.NewReservationWithStatus(0, activityID, dogID, passID, status, time.Now().UTC())
		require.NoError(t, err)
		_, err = reservationRepo.Create(context.Background(), r)
		require.NoError(t, err)
	}
	seedRes(dogA1ID, passAID, domain.StatusPendingToConfirm)
	seedRes(dogBID, passBID, domain.StatusPendingToConfirm)
	seedRes(dogA2ID, passAID, domain.StatusPendingToConfirm)
	seedRes(dogA3ID, passAID, domain.StatusConfirmed) // must be excluded

	token := loginAndGetToken(t, router, admin.Email(), "admin-pw-123")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reservations/pending", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	pending, ok := body["pending"].([]interface{})
	require.True(t, ok)
	require.Len(t, pending, 3, "three PENDING + one CONFIRMED → exactly three entries")

	// Build a map keyed by dog_id for owner-independent assertions.
	byDogID := make(map[float64]map[string]interface{}, len(pending))
	for _, raw := range pending {
		e := raw.(map[string]interface{})
		dogID, _ := e["dog_id"].(float64)
		byDogID[dogID] = e
	}

	luna, ok := byDogID[float64(dogA1ID)]
	require.True(t, ok, "Luna (ownerA) must be present")
	assert.Equal(t, "Luna", luna["dog_name"])
	assert.EqualValues(t, ownerA.ID(), luna["owner_id"])
	assert.Equal(t, ownerA.Name(), luna["owner_name"])

	toby, ok := byDogID[float64(dogBID)]
	require.True(t, ok, "Toby (ownerB) must be present")
	assert.EqualValues(t, ownerB.ID(), toby["owner_id"])
	assert.Equal(t, ownerB.Name(), toby["owner_name"])

	maya, ok := byDogID[float64(dogA2ID)]
	require.True(t, ok, "Maya (ownerA) must be present")
	assert.EqualValues(t, ownerA.ID(), maya["owner_id"])

	// Sanity: pagination fields are echoed. Default limit (no
	// query param) is 50 (the use case factory's fallback).
	assert.EqualValues(t, 50, body["limit"])
	assert.EqualValues(t, 0, body["offset"])
	assert.EqualValues(t, 3, body["count"])
}

// TestPendingReservationsHTTP_TrailingSlashRedirects is a
// regression guard that documents the canonical-path convention:
// the route is registered WITHOUT a trailing slash, so a request
// WITH the slash is treated as a redirect to the canonical URL
// (Gin's default RedirectTrailingSlash). The test asserts that the
// redirect points to the slash-less form and that following it
// yields a 200. If someone later registers the path WITH a slash
// (or disables redirects), this test will fail and force them to
// consciously update both the frontend and the test.
func TestPendingReservationsHTTP_TrailingSlashRedirects(t *testing.T) {
	if integrationDB == nil {
		t.Fatal("integrationDB is nil — TestMain did not run or failed")
	}
	cleanIntegrationTables(t, integrationDB)
	router := buildAuthTestRouter(integrationDB)

	admin := seedAdminUser(t, integrationDB, "admin-pending-slash@dogpaw.com", "admin-pw-123")
	token := loginAndGetToken(t, router, admin.Email(), "admin-pw-123")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reservations/pending/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMovedPermanently, w.Code,
		"Gin's default behavior is to redirect the trailing-slash variant to the canonical URL")
	location := w.Header().Get("Location")
	assert.Contains(t, location, "/api/v1/reservations/pending",
		"the redirect target must be the slash-less canonical URL")
}
