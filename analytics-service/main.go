package main

import (
	"log"
	"os"

	"analytics-service/handlers"
	"analytics-service/middleware"
	"analytics-service/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, reading from environment")
	}

	analyticsSvc := service.NewAnalyticsService()
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsSvc)

	r := gin.Default()
	r.Use(middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"service": "analytics", "status": "ok"})
	})

	protected := r.Group("/", middleware.RequireAuth())
	{
		protected.GET("/analytics", analyticsHandler.GetAnalytics)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "4002"
	}
	log.Fatal(r.Run(":" + port))
}
