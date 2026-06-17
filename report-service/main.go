package main

import (
	"log"
	"os"

	"report-service/handlers"
	"report-service/middleware"

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
		protected.GET("/", handlers.GetReports)
		protected.GET("/reports", handlers.GetReports)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "4001"
	}
	log.Fatal(r.Run(":" + port))
}
