package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"dogpaw/internal/handler"
	"dogpaw/internal/repository/postgres"
	activityuc "dogpaw/internal/usecase/activity"
	doguc "dogpaw/internal/usecase/dog"
	incompatuc "dogpaw/internal/usecase/incompatibility"

	_ "dogpaw/docs"
)

const version = "0.1.0"

func newRouter(db *sql.DB, env string) *gin.Engine {
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery(), requestLogger())

	r.GET("/health", healthHandler(db))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	repo := postgres.NewDogRepository(db)
	incompatRepo := postgres.NewIncompatibilityRepository(db)
	registerUC := doguc.NewRegisterDogUseCase(repo)
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
	modifyUC := doguc.NewModifyDogUseCase(repo)
	addIncompatUC := doguc.NewAddDogIncompatibilityUseCase(repo, incompatRepo)
	removeIncompatUC := doguc.NewRemoveDogIncompatibilityUseCase(repo)
	deleteDogUC := doguc.NewDeleteDogUseCase(repo)
	setNeuteredUC := doguc.NewSetDogNeuteredUseCase(repo)
	setHeatUC := doguc.NewSetDogHeatUseCase(repo)

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
	activityH := handler.NewActivityHandler(
		registerActivityUC, getActivityUC, modifyActivityUC,
		listAllActivityUC, listUpcomingActivityUC,
	)

	dogH := handler.NewDogHandler(
		registerUC,
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
		addIncompatUC,
		removeIncompatUC,
		deleteDogUC,
		setNeuteredUC,
		setHeatUC,
	)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/dogs", dogH.Register)
		v1.GET("/dogs", dogH.List)
		v1.GET("/dogs/active", dogH.ListActive)
		v1.GET("/dogs/is_active/:value", dogH.ListByIsActive)
		v1.GET("/dogs/incompatibility/:incompat_id", dogH.ListByIncompatibility)
		v1.GET("/dogs/breed/:breed", dogH.ListByBreed)
		v1.GET("/dogs/sex/:sex", dogH.ListBySex)
		v1.GET("/dogs/neutered/:value", dogH.ListByNeutered)
		v1.GET("/dogs/heat/:value", dogH.ListByHeat)
		v1.GET("/dogs/age/:bracket", dogH.ListByAgeBracket)
		v1.GET("/dogs/size/:bracket", dogH.ListBySizeBracket)
		v1.GET("/dogs/owner/:owner_id", dogH.ListByOwner)
		v1.PATCH("/dogs/:id", dogH.Modify)
		v1.PATCH("/dogs/:id/neutered", dogH.SetNeutered)
		v1.PATCH("/dogs/:id/heat", dogH.SetHeat)
		v1.DELETE("/dogs/:id", dogH.Delete)
		v1.POST("/dogs/:id/incompatibilities", dogH.AddIncompatibility)
		v1.DELETE("/dogs/:id/incompatibilities/:incompatibility_id", dogH.RemoveIncompatibility)

		v1.POST("/incompatibilities", incompatH.Register)
		v1.GET("/incompatibilities", incompatH.List)
		v1.GET("/incompatibilities/:id", incompatH.GetByID)
		v1.PATCH("/incompatibilities/:id", incompatH.Modify)
		v1.DELETE("/incompatibilities/:id", incompatH.Delete)

		v1.POST("/activities", activityH.Register)
		v1.GET("/activities", activityH.List)
		v1.GET("/activities/upcoming", activityH.ListUpcoming)
		v1.GET("/activities/:id", activityH.GetByID)
		v1.PATCH("/activities/:id", activityH.Modify)
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
			dbStatus = "down: " + err.Error()
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
