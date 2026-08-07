package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"net/http"
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

func newRouter(db *sql.DB, cfg Config) *gin.Engine {
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

	authLimiter := newIPRateLimiter(rate.Limit(5.0/60.0), 5)

	r.GET("/health", healthHandler(db))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	repo := postgres.NewDogRepository(db)
	incompatRepo := postgres.NewIncompatibilityRepository(db)
	transactor := postgres.NewTransactor(db)
	registerUC := doguc.NewRegisterDogUseCase(repo)
	getDogUC := doguc.NewGetDogUseCase(repo)
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
	registerActivityUC := activityuc.NewRegisterActivityUseCase(activityRepo)
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
	passH := handler.NewPassHandler(registerPassUC, modifyPassUC, getPassUC, listAllPassUC, listByUserPassUC)

	reservationRepo := postgres.NewReservationRepository(db)
	dogRepo := postgres.NewDogRepository(db)
	registerReservationUC := reservationuc.NewRegisterReservationUseCase(
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
	reservationH := handler.NewReservationHandler(
		registerReservationUC, cancelReservationUC,
		getReservationUC, listByUserReservationsUC, listUpcomingByUserReservationsUC,
		listByDogReservationsUC, listByPassReservationsUC, listByActivityReservationsUC,
		markNoShowReservationUC, completeReservationUC,
		confirmPendingReservationUC, rejectPendingReservationUC,
	)

	closeActivityUC := activityuc.NewCloseActivityUseCase(
		transactor, activityRepo, dogRepo, reservationRepo,
		markNoShowReservationUC, completeReservationUC,
	)

	activityH := handler.NewActivityHandler(
		registerActivityUC, getActivityUC, modifyActivityUC,
		listAllActivityUC, listUpcomingActivityUC, closeActivityUC, reservationRepo,
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

	userRepo := postgres.NewUserRepository(db)
	getUserUC := useruc.NewGetUserUseCase(userRepo)
	listUsersUC := useruc.NewListUsersUseCase(userRepo)
	updateUserUC := useruc.NewUpdateUserUseCase(userRepo)
	deactivateUserUC := useruc.NewDeactivateUserUseCase(userRepo)
	listUserEmailsUC := useruc.NewListUserEmailsUseCase(userRepo)
	userH := handler.NewUserHandler(getUserUC, listUsersUC, updateUserUC, deactivateUserUC, listUserEmailsUC)

	invRepo := postgres.NewInvitationRepository(db)
	createInvUC := invitationuc.NewCreateInvitationUseCase(invRepo)
	registerAuthUC := authuc.NewRegisterWithInvitationUseCase(
		transactor, invRepo, userRepo, crypto.NewDefaultBcryptHasher(),
	)
	jwtSecret := cfg.JWTSecret
	loginAuthUC := authuc.NewLoginUseCase(
		userRepo,
		crypto.NewDefaultBcryptHasher(),
		crypto.NewJWTTokenGenerator(jwtSecret, 24*time.Hour),
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
		v1.POST("/auth/register", rateLimitMiddleware(authLimiter), authH.RegisterWithInvitation)
		v1.POST("/auth/login", rateLimitMiddleware(authLimiter), authH.Login)

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
			admin.GET("/users", userH.List)
			admin.PATCH("/users/:user_id", userH.Update)
			admin.POST("/users/:user_id/deactivate", userH.Deactivate)
			admin.GET("/users/emails", userH.ListEmails)

			admin.POST("/invitations", invH.Create)

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

			admin.POST("/users/:user_id/passes", passH.Register)
			admin.GET("/passes", passH.List)
			admin.GET("/passes/:id", passH.GetByID)
			admin.PATCH("/passes/:id", passH.Modify)

			admin.POST("/users/:user_id/reservations/:id/no-show", reservationH.MarkNoShow)
			admin.POST("/users/:user_id/reservations/:id/complete", reservationH.CompleteReservation)
			admin.POST("/users/:user_id/reservations/:id/confirm", reservationH.ConfirmPending)
			admin.POST("/users/:user_id/reservations/:id/reject", reservationH.RejectPending)
			admin.GET("/dogs/:id/reservations", reservationH.ListByDog)
			admin.GET("/passes/:id/reservations", reservationH.ListByPass)
			admin.GET("/activities/:id/reservations", reservationH.ListByActivity)
		}
	}

	return r
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
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastUsed time.Time
}

func newIPRateLimiter(r rate.Limit, burst int) *ipRateLimiter {
	return &ipRateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		rate:     r,
		burst:    burst,
	}
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

func rateLimitMiddleware(limiter *ipRateLimiter) gin.HandlerFunc {
	go func() {
		for range time.Tick(10 * time.Minute) {
			limiter.mu.Lock()
			cutoff := time.Now().Add(-10 * time.Minute)
			for ip, entry := range limiter.limiters {
				if entry.lastUsed.Before(cutoff) {
					delete(limiter.limiters, ip)
				}
			}
			limiter.mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
		if ip == "" {
			ip = c.ClientIP()
		}
		if !limiter.getLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate_limit_exceeded",
			})
			return
		}
		c.Next()
	}
}
