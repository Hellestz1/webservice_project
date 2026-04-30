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
	"github.com/gin-gonic/gin"
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

	authHandler := handler.NewAuthHandler(authService)
	router.POST("/auth/register", authHandler.Register())
	router.POST("/auth/login", authHandler.Login())
	router.POST("/auth/api-key", authHandler.IssueAPIKey())

	apiV1 := router.Group("/api/v1")
	apiV1.Use(apiKeyAuth.Require(), rateLimiter.Require(), monthlyQuota.Require())

	comics := apiV1.Group("/comics")
	comics.GET("", featureGate.Require("comic:list"), h.ListComics())
	comics.GET("/search", featureGate.Require("comic:search"), h.SearchComics())
	comics.GET("/:id", featureGate.Require("comic:detail"), h.GetComicDetail())
	comics.GET("/:id/chapters", featureGate.Require("chapter:list"), h.ListChapters())

	port := envOrDefault("APP_PORT", "8080")

	log.Printf("comic provider started on :%s", port)
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
