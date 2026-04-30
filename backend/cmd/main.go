// Package main is the entry point of the Comic Provider API service.
//
//	@title			Comic Provider API
//	@version		1.0
//	@description	Web service for comic book data — tiered API access via API key.
//	@description
//	@description	## Authentication
//	@description	All `/api/v1/*` endpoints require the `X-API-Key` header.
//	@description	Obtain an API key via `POST /auth/register` or `POST /auth/api-key`.
//	@description
//	@description	## Plans
//	@description	| Plan | Quota | Rate Limit | Features |
//	@description	|---|---|---|---|
//	@description	| free | 1,000 req/month | 10 req/min | list, detail, chapters |
//	@description	| standard | 100,000 req/month | 120 req/min | + search |
//	@description	| premium | unlimited | 1,000 req/min | + recommend, analytics |
//
//	@contact.name	Comic Provider Support
//
//	@host		localhost:8080
//	@BasePath	/
//
//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						X-API-Key
//	@description				API key for accessing protected /api/v1/* endpoints
package main

import (
	"context"
	"log"
	"os"
	"time"

	"backend/internal/handler"
	"backend/internal/middleware"
	"backend/internal/repository"
	"backend/internal/service"
	"backend/internal/storage"
	"backend/internal/usecase"
	_ "backend/docs"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	dsn := envOrDefault("DB_DSN", "postgres://postgres:postgres@localhost:5432/comic_provider?sslmode=disable")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := storage.OpenPostgres(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := repository.NewPostgresComicRepository(db)
	uc := usecase.NewComicUsecase(repo)
	h := handler.NewComicHandler(uc)
	analyticsRepo := repository.NewPostgresAnalyticsRepository(db)
	analyticsUsecase := usecase.NewAnalyticsUsecase(analyticsRepo)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsUsecase)

	accessService := service.NewAccessService(db)
	authService := service.NewAuthService(db)
	planPolicies, err := accessService.LoadPlanPolicies(ctx)
	if err != nil {
		log.Fatal(err)
	}

	apiKeyAuth := middleware.NewAPIKeyAuthMiddleware(accessService)
	rateLimiter := middleware.NewRateLimiterMiddleware(planPolicies)
	monthlyQuota := middleware.NewMonthlyQuotaMiddleware(db, planPolicies)
	featureGate := middleware.NewFeatureGateMiddleware(planPolicies)

	router := gin.Default()
	router.GET("/health", handler.Health)

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authHandler := handler.NewAuthHandler(authService)
	router.POST("/auth/register", authHandler.Register())
	router.POST("/auth/login", authHandler.Login())
	router.POST("/auth/api-key", authHandler.IssueAPIKey())
	router.POST("/auth/plan", authHandler.ChangePlan())

	apiV1 := router.Group("/api/v1")
	apiV1.Use(apiKeyAuth.Require(), rateLimiter.Require(), monthlyQuota.Require())
	apiV1.GET("/analytics/usage", featureGate.Require("analytics:usage"), analyticsHandler.Usage())

	comics := apiV1.Group("/comics")
	comics.GET("", featureGate.Require("comic:list"), h.ListComics())
	comics.GET("/search", featureGate.Require("comic:search"), h.SearchComics())
	comics.GET("/recommend", featureGate.Require("comic:recommend"), h.RecommendComics())
	comics.GET("/:id", featureGate.Require("comic:detail"), h.GetComicDetail())
	comics.GET("/:id/chapters", featureGate.Require("chapter:list"), h.ListChapters())

	port := envOrDefault("APP_PORT", "8080")

	log.Printf("comic provider started on :%s", port)
	log.Printf("swagger UI: http://localhost:%s/swagger/index.html", port)
	log.Printf("demo keys: free-demo-key | standard-demo-key | premium-demo-key")
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
