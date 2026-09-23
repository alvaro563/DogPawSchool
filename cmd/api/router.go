package main

import (
	"context"
	"database/sql"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/time/rate"

	"dogpaw/internal/crypto"
	"dogpaw/internal/handler"
	"dogpaw/internal/repository/postgres"
	activityuc "dogpaw/internal/usecase/activity"
	authuc "dogpaw/internal/usecase/auth"
	doguc "dogpaw/internal/usecase/dog"
	incompatuc "dogpaw/internal/usecase/incompatibility"
	invitationuc "dogpaw/internal/usecase/invitation"
	passuc "dogpaw/internal/usecase/pass"
	reservationuc "dogpaw/internal/usecase/reservation"
	useruc "dogpaw/internal/usecase/user"

	_ "dogpaw/docs"
)

const version = "0.1.0"

// dogWithOwnerAdapter bridges *postgres.DogRepository (infra) to
// doguc.DogWithOwnerLister (use case) without creating an import
// cycle between the two packages. It maps each infra row to the use
// case's read model.
type dogWithOwnerAdapter struct {
	repo *postgres.DogRepository
}

func (a dogWithOwnerAdapter) ListActiveWithOwner(ctx context.Context, limit, offset int) ([]*doguc.DogWithOwner, error) {
	rows, err := a.repo.ListActiveWithOwnerRaw(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]*doguc.DogWithOwner, len(rows))
	for i, r := range rows {
		out[i] = &doguc.DogWithOwner{Dog: r.Dog, OwnerName: r.OwnerName}
	}
	return out, nil
}

func newRouter(ctx context.Context, db *sql.DB, cfg Config) (*gin.Engine, func(), error) {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery(), requestLogger())

	if len(cfg.TrustedProxies) > 0 {
		if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
			slog.Error("set trusted proxies", "err", err)
		}
	} else if cfg.Env == "production" {
		slog.Warn("no trusted proxies configured in production")
	}

	corsConfig := cors.DefaultConfig()
	if len(cfg.CORSOrigins) > 0 {
		corsConfig.AllowOrigins = cfg.CORSOrigins
	} else {
		corsConfig.AllowAllOrigins = true
	}
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "Authorization")
	corsConfig.AllowMethods = []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"}
	r.Use(cors.New(corsConfig))

	// IP rate limiters per endpoint. Behind a reverse proxy these
	// keys are the proxy's IP — that's intentional: per-real-client
	// limits belong upstream (nginx, Caddy, Cloudflare), this layer
	// is the volumetric fallback when the upstream is misconfigured.
	// Login and register have independent buckets so an attacker
	// hammering register cannot starve login.
	loginIPLimiter := newIPRateLimiter(
		rate.Limit(float64(cfg.LoginIPRateLimitPerMinute)/60.0),
		cfg.LoginIPBurst,
	)
	registerIPLimiter := newIPRateLimiter(
		rate.Limit(float64(cfg.RegisterIPRateLimitPerMinute)/60.0),
		cfg.RegisterIPBurst,
	)

	// Account lockout (Postgres-backed). Wired into LoginUseCase so
	// it runs AFTER body parsing — the email is needed to count
	// failures against a specific account.
	loginLockout := postgres.NewLoginAttemptRepository(
		db,
		cfg.LockoutEmailMaxFailures,
		cfg.LockoutEmailWindow,
		cfg.LockoutIPMaxFailures,
		cfg.LockoutIPWindow,
	)

	r.GET("/health", healthHandler(db))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	repo := postgres.NewDogRepository(db)
	incompatRepo := postgres.NewIncompatibilityRepository(db)
	transactor := postgres.NewTransactor(db)
	registerUC := doguc.NewRegisterDogUseCase(repo)
	getDogUC := doguc.NewGetDogUseCase(repo, postgres.NewUserRepository(db))
	listAllUC := doguc.NewListAllDogsUseCase(repo)
	listByOwnerUC := doguc.NewListByOwnerUseCase(repo)
	listActiveUC := doguc.NewListActiveDogsUseCase(repo)
	listByIsActiveUC := doguc.NewListByIsActiveUseCase(repo)
	listByIncompatibilityUC := doguc.NewListByIncompatibilityUseCase(repo)
	listByBreedUC := doguc.NewListByBreedUseCase(repo)
	listBySexUC := doguc.NewListBySexUseCase(repo)
	listByNeuteredUC := doguc.NewListByNeuteredUseCase(repo)
	listByHeatUC := doguc.NewListByHeatUseCase(repo)
	listByAgeBracketUC := doguc.NewListByAgeBracketUseCase(repo)
	listBySizeBracketUC := doguc.NewListBySizeBracketUseCase(repo)
	modifyUC := doguc.NewModifyDogUseCase(transactor, repo)
	addTraitUC := doguc.NewAddDogTraitUseCase(transactor, repo, incompatRepo)
	addTriggerUC := doguc.NewAddDogTriggerUseCase(transactor, repo, incompatRepo)
	removeIncompatUC := doguc.NewRemoveDogIncompatibilityUseCase(transactor, repo)
	deleteDogUC := doguc.NewDeleteDogUseCase(repo)
	setNeuteredUC := doguc.NewSetDogNeuteredUseCase(transactor, repo)
	setHeatUC := doguc.NewSetDogHeatUseCase(transactor, repo)
	setPhotoUC := doguc.NewSetDogPhotoUseCase(transactor, repo)

	registerIncompatUC := incompatuc.NewRegisterIncompatibilityUseCase(incompatRepo)
	listIncompatUC := incompatuc.NewListIncompatibilitiesUseCase(incompatRepo)
	getIncompatUC := incompatuc.NewGetIncompatibilityUseCase(incompatRepo)
	modifyIncompatUC := incompatuc.NewModifyIncompatibilityUseCase(incompatRepo)
	deleteIncompatUC := incompatuc.NewDeleteIncompatibilityUseCase(incompatRepo)
	incompatH := handler.NewIncompatibilityHandler(
		registerIncompatUC, listIncompatUC, getIncompatUC, modifyIncompatUC, deleteIncompatUC,
	)

	activityRepo := postgres.NewActivityRepository(db)
	getActivityUC := activityuc.NewGetActivityUseCase(activityRepo)
	modifyActivityUC := activityuc.NewModifyActivityUseCase(activityRepo)
	listAllActivityUC := activityuc.NewListAllActivitiesUseCase(activityRepo)
	listUpcomingActivityUC := activityuc.NewListUpcomingActivitiesUseCase(activityRepo)

	passRepo := postgres.NewPassRepository(db)
	registerPassUC := passuc.NewRegisterPassUseCase(passRepo)
	modifyPassUC := passuc.NewModifyPassUseCase(passRepo)
	getPassUC := passuc.NewGetPassUseCase(passRepo)
	listAllPassUC := passuc.NewListAllPassesUseCase(passRepo)
	listByUserPassUC := passuc.NewListByUserPassesUseCase(passRepo)
	listByPaidPassUC := passuc.NewListByPaidPassesUseCase(passRepo)
	setPaidPassUC := passuc.NewSetPassPaidUseCase(passRepo)
	passH := handler.NewPassHandler(registerPassUC, modifyPassUC, getPassUC, listAllPassUC, listByUserPassUC, listByPaidPassUC, setPaidPassUC)

	reservationRepo := postgres.NewReservationRepository(db)
	dogRepo := postgres.NewDogRepository(db)
	userRepo := postgres.NewUserRepository(db)
	registerActivityUC := activityuc.NewRegisterActivityUseCase(activityRepo, dogRepo)
	registerReservationUC := reservationuc.NewRegisterReservationUseCase(
		transactor, activityRepo, dogRepo, passRepo, reservationRepo,
	)
	registerAdminReservationUC := reservationuc.NewRegisterAdminReservationUseCase(
		transactor, activityRepo, dogRepo, passRepo, reservationRepo,
	)
	cancelReservationUC := reservationuc.NewCancelReservationUseCase(
		transactor, activityRepo, dogRepo, passRepo, reservationRepo,
	)
	markNoShowReservationUC := reservationuc.NewMarkReservationNoShowUseCase(
		transactor, activityRepo, dogRepo, reservationRepo,
	)
	completeReservationUC := reservationuc.NewCompleteReservationUseCase(
		transactor, activityRepo, dogRepo, reservationRepo,
	)
	confirmPendingReservationUC := reservationuc.NewConfirmPendingReservationUseCase(transactor, reservationRepo)
	rejectPendingReservationUC := reservationuc.NewRejectPendingReservationUseCase(transactor, passRepo, reservationRepo)
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
	listAttendanceReportUC := reservationuc.NewListAttendanceReportUseCase(reservationRepo)
	forgiveReservationUC := reservationuc.NewForgiveReservationUseCase(transactor, passRepo, reservationRepo)
	attendanceH := handler.NewAttendanceHandler(listAttendanceReportUC)
	reservationH := handler.NewReservationHandler(
		registerReservationUC, cancelReservationUC,
		getReservationUC, listByUserReservationsUC, listUpcomingByUserReservationsUC,
		listByDogReservationsUC, listByPassReservationsUC, listByActivityReservationsUC,
		markNoShowReservationUC, completeReservationUC,
		confirmPendingReservationUC, rejectPendingReservationUC,
		forgiveReservationUC,
		listAllReservationsUC,
		listUpcomingAllUC,
		registerAdminReservationUC,
		listActivityRosterUC,
		listPendingUC,
	)

	closeActivityUC := activityuc.NewCloseActivityUseCase(
		transactor, activityRepo, dogRepo, reservationRepo,
		markNoShowReservationUC, completeReservationUC,
	)
	bulkCompleteUC := activityuc.NewBulkCompleteReservationsUseCase(
		transactor, activityRepo, dogRepo, reservationRepo, completeReservationUC,
	)

	activityH := handler.NewActivityHandler(
		registerActivityUC, getActivityUC, modifyActivityUC,
		listAllActivityUC, listUpcomingActivityUC,
		closeActivityUC, bulkCompleteUC, reservationRepo,
	)

	dogH := handler.NewDogHandler(
		registerUC,
		getDogUC,
		listAllUC,
		listByOwnerUC,
		listActiveUC,
		listByIsActiveUC,
		listByIncompatibilityUC,
		listByBreedUC,
		listBySexUC,
		listByNeuteredUC,
		listByHeatUC,
		listByAgeBracketUC,
		listBySizeBracketUC,
		modifyUC,
		addTraitUC,
		addTriggerUC,
		removeIncompatUC,
		deleteDogUC,
		setNeuteredUC,
		setHeatUC,
		setPhotoUC,
	)

	// Enriched active-dogs listing: a JOIN-backed projection that
	// fills owner_name. Wired through the setter (Open/Closed) so
	// the constructor signature stays intact.
	listActiveWithOwnerUC := doguc.NewListActiveDogsWithOwnerUseCase(dogWithOwnerAdapter{repo: dogRepo})
	dogH.WithListActiveWithOwner(listActiveWithOwnerUC)

	getUserUC := useruc.NewGetUserUseCase(userRepo)
	listUsersUC := useruc.NewListUsersUseCase(userRepo)
	updateUserUC := useruc.NewUpdateUserUseCase(userRepo)
	deactivateUserUC := useruc.NewDeactivateUserUseCase(userRepo)
	activateUserUC := useruc.NewActivateUserUseCase(userRepo)
	listUserEmailsUC := useruc.NewListUserEmailsUseCase(userRepo)
	userH := handler.NewUserHandler(getUserUC, listUsersUC, updateUserUC, deactivateUserUC, activateUserUC, listUserEmailsUC)

	invRepo := postgres.NewInvitationRepository(db)
	createInvUC := invitationuc.NewCreateInvitationUseCase(invRepo)
	jwtSecret := cfg.JWTSecret
	jwtTokenGen := crypto.NewJWTTokenGenerator(jwtSecret, 24*time.Hour)
	registerAuthUC := authuc.NewRegisterWithInvitationUseCase(
		transactor, invRepo, userRepo, crypto.NewDefaultBcryptHasher(), jwtTokenGen,
	)
	loginAuthUC := authuc.NewLoginUseCase(
		userRepo,
		crypto.NewDefaultBcryptHasher(),
		jwtTokenGen,
		loginLockout,
	)
	changePasswordUC := authuc.NewChangePasswordUseCase(
		userRepo,
		crypto.NewDefaultBcryptHasher(),
		crypto.NewDefaultBcryptHasher(),
	)
	invH := handler.NewInvitationHandler(createInvUC)
	authH := handler.NewAuthHandler(registerAuthUC, loginAuthUC, changePasswordUC)

	v1 := r.Group("/api/v1")
	{
		// ── Public ──
		v1.POST("/auth/register", rateLimitMiddleware(registerIPLimiter), authH.RegisterWithInvitation)
		v1.POST("/auth/login", rateLimitMiddleware(loginIPLimiter), authH.Login)

		// ── Any authenticated user (ownership check inside handlers) ──
		anyUser := v1.Group("")
		anyUser.Use(handler.AuthRequired(jwtSecret, userRepo))
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

		// ── Admin only ──
		admin := v1.Group("")
		admin.Use(handler.AuthRequired(jwtSecret, userRepo))
		admin.Use(handler.AdminRequired())
		{
			admin.POST("/reservations", reservationH.RegisterAdmin)
			admin.POST("/reservations/:id/cancel", reservationH.CancelAdmin)
			admin.POST("/reservations/:id/forgive", reservationH.Forgive)
			admin.GET("/users", userH.List)
			admin.PATCH("/users/:user_id", userH.Update)
			admin.POST("/users/:user_id/deactivate", userH.Deactivate)
			admin.POST("/users/:user_id/activate", userH.Activate)
			admin.GET("/users/emails", userH.ListEmails)

			admin.POST("/invitations", invH.Create)

			admin.POST("/dogs", dogH.Register)
			admin.GET("/dogs", dogH.List)
			admin.GET("/dogs/active", dogH.ListActiveWithOwner)
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
			admin.PATCH("/dogs/:id/photo", dogH.SetPhoto)
			admin.DELETE("/dogs/:id", dogH.Delete)
			admin.POST("/dogs/:id/traits", dogH.AddTrait)
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
			admin.POST("/activities/:id/complete-all", activityH.BulkCompleteReservations)

			admin.POST("/users/:user_id/passes", passH.Register)
			admin.GET("/passes", passH.List)
			admin.GET("/passes/is_paid/:value", passH.ListByPaid)
			admin.GET("/passes/:id", passH.GetByID)
			admin.PATCH("/passes/:id", passH.Modify)
			admin.PATCH("/passes/:id/paid", passH.SetPaid)

			admin.POST("/users/:user_id/reservations/:id/no-show", reservationH.MarkNoShow)
			admin.POST("/users/:user_id/reservations/:id/complete", reservationH.CompleteReservation)
			admin.POST("/users/:user_id/reservations/:id/confirm", reservationH.ConfirmPending)
			admin.POST("/users/:user_id/reservations/:id/reject", reservationH.RejectPending)
			admin.GET("/dogs/:id/reservations", reservationH.ListByDog)
			admin.GET("/passes/:id/reservations", reservationH.ListByPass)
			admin.GET("/activities/:id/reservations", reservationH.ListByActivity)
			admin.GET("/activities/:id/roster", reservationH.ListActivityRoster)
			admin.GET("/reservations", reservationH.ListAll)
			admin.GET("/reservations/upcoming", reservationH.ListUpcomingAll)
			admin.GET("/reservations/pending", reservationH.ListPending)
			admin.GET("/reservations/attendance", attendanceH.List)
			admin.GET("/reservations/attendance.csv", attendanceH.DownloadCSV)
		}
	}

	return r, newRouterCleanup(loginIPLimiter, registerIPLimiter, loginLockout, cfg), nil
}

// newRouterCleanup wires the per-limiter Stop calls and the
// background login_attempts housekeeping goroutine. Returned
// closure is invoked from main on shutdown so the process does
// not leak goroutines or leave the cleanup channel open.
func newRouterCleanup(
	loginIPLimiter *ipRateLimiter,
	registerIPLimiter *ipRateLimiter,
	loginLockout *postgres.LoginAttemptRepository,
	cfg Config,
) func() {
	// Channel-based cancellation so the periodic cleanup goroutine
	// in loginLockout exits deterministically.
	loginCleanupDone := make(chan struct{})
	go func() {
		defer close(loginCleanupDone)
		// Use a child context detached from main so we are not
		// racing the signal.NotifyContext cancellation; the
		// explicit done channel is the signal.
		interval := cfg.LoginAttemptsCleanupInterval
		if interval <= 0 {
			interval = 15 * 24 * time.Hour // effectively disable
		}
		// Retain rows for max(LockoutEmailWindow, LockoutIPWindow) * 2
		// or 1h floor. Past that, no Check can see them.
		retention := cfg.LockoutEmailWindow
		if cfg.LockoutIPWindow > retention {
			retention = cfg.LockoutIPWindow
		}
		retention *= 2
		if retention < time.Hour {
			retention = time.Hour
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-loginCleanupDone:
				return
			case <-ticker.C:
				cutoff := time.Now().Add(-retention)
				// Use Background ctx — the goroutine outlives
				// the request-scoped ctx that initiated it.
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				if _, err := loginLockout.DeleteOlderThan(ctx, cutoff); err != nil {
					slog.Error("login_attempts cleanup", "err", err.Error())
				}
				cancel()
			}
		}
	}()

	var cleanupOnce sync.Once
	return func() {
		cleanupOnce.Do(func() {
			// Halt the housekeeping goroutine first so it does
			// not race with limiter.Stop().
			loginIPLimiter.Stop()
			registerIPLimiter.Stop()
			// Drain the housekeeping goroutine. Bounded wait
			// because the goroutine is either idle (returning
			// immediately) or inside a 10s ctx — we cap at
			// 12s to leave headroom inside SHUTDOWN_TIMEOUT.
			select {
			case <-loginCleanupDone:
			case <-time.After(12 * time.Second):
				slog.Warn("login_attempts cleanup goroutine did not exit in time")
			}
		})
	}
}

func healthHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		dbStatus := "ok"
		httpStatus := http.StatusOK
		if err := db.PingContext(ctx); err != nil {
			slog.Error("health db ping failed", "err", err.Error())
			dbStatus = "down"
			httpStatus = http.StatusServiceUnavailable
		}
		c.JSON(httpStatus, gin.H{
			"status":    "ok",
			"database":  dbStatus,
			"version":   version,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("http request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"size", c.Writer.Size(),
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}

type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimiterEntry
	rate     rate.Limit
	burst    int

	stopCh chan struct{}
	stopOnce sync.Once
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastUsed time.Time
}

func newIPRateLimiter(r rate.Limit, burst int) *ipRateLimiter {
	l := &ipRateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rate:     r,
		burst:    burst,
		stopCh:   make(chan struct{}),
	}
	go l.runCleanup()
	return l
}

// runCleanup evicts idle entries every 10 minutes. Idempotent
// after Stop has been called.
func (l *ipRateLimiter) runCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-l.stopCh:
			return
		case <-ticker.C:
			l.mu.Lock()
			cutoff := time.Now().Add(-10 * time.Minute)
			for ip, entry := range l.limiters {
				if entry.lastUsed.Before(cutoff) {
					delete(l.limiters, ip)
				}
			}
			l.mu.Unlock()
		}
	}
}

// Stop halts the background cleanup goroutine. Safe to call
// multiple times. Called from the server shutdown sequence so the
// process does not leak goroutines between test runs.
func (l *ipRateLimiter) Stop() {
	l.stopOnce.Do(func() {
		close(l.stopCh)
	})
}

func (l *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.limiters[ip]
	if !ok {
		entry = &rateLimiterEntry{limiter: rate.NewLimiter(l.rate, l.burst)}
		l.limiters[ip] = entry
	}
	entry.lastUsed = time.Now()
	return entry.limiter
}

// rateLimitMiddleware is the volumetric defense on auth endpoints.
// It deliberately keys by RemoteAddr (the TCP peer) and NEVER
// consults c.ClientIP() — Gin's ClientIP falls back to RemoteAddr
// when TRUSTED_PROXIES is unset, but trusts X-Forwarded-For when
// the proxy is in the trusted list. Either way, the upstream proxy
// (nginx, Caddy, Cloudflare) is the right place for per-real-client
// limits: it sees the X-Forwarded-For it inserted. This middleware
// is the LAST line of defense when the upstream is misconfigured or
// saturated.
//
// Responses carry the IETF draft-8 headers: Retry-After (seconds),
// RateLimit-Limit (burst capacity), RateLimit-Remaining (tokens
// left), RateLimit-Reset (seconds until a token frees up).
func rateLimitMiddleware(limiter *ipRateLimiter) gin.HandlerFunc {
	burstStr := strconv.Itoa(limiter.burst)
	return func(c *gin.Context) {
		ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil || ip == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid_remote_addr",
			})
			return
		}
		reservation := limiter.getLimiter(ip).Reserve()
		if !reservation.OK() {
			// Should not happen with sensible config. Refuse hard.
			c.Header("Retry-After", "60")
			c.Header("RateLimit-Limit", burstStr)
			c.Header("RateLimit-Remaining", "0")
			c.Header("RateLimit-Reset", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate_limit_exceeded",
			})
			return
		}
		delay := reservation.Delay()
		if delay > 0 {
			reservation.Cancel()
			secs := int(math.Ceil(delay.Seconds()))
			if secs < 1 {
				secs = 1
			}
			c.Header("Retry-After", strconv.Itoa(secs))
			c.Header("RateLimit-Limit", burstStr)
			c.Header("RateLimit-Remaining", "0")
			c.Header("RateLimit-Reset", strconv.Itoa(secs))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate_limit_exceeded",
			})
			return
		}
		c.Header("RateLimit-Limit", burstStr)
		c.Next()
	}
}
