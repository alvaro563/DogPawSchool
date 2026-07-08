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
	doguc "dogpaw/internal/usecase/dog"

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
	registerUC := doguc.NewRegisterDogUseCase(repo)
	listByOwnerUC := doguc.NewListByOwnerUseCase(repo)
	dogH := handler.NewDogHandler(registerUC, listByOwnerUC)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/dogs", dogH.Register)
		v1.GET("/dogs", dogH.List)
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
