package main

import (
	"log"
	"os"

	"analytics-service/handlers"
	"analytics-service/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, reading from environment")
	}

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	protected := r.Group("/", middleware.RequireAuth())
	{
		protected.GET("/", handlers.GetAnalytics)
		protected.GET("/analytics", handlers.GetAnalytics)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "4002"
	}
	log.Fatal(r.Run(":" + port))
}
