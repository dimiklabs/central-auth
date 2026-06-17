package main

import (
	"log"
	"os"

	"transaction-service/handlers"
	"transaction-service/middleware"

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
		protected.GET("/", handlers.GetTransactions)
		protected.GET("/transactions", handlers.GetTransactions)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "4003"
	}
	log.Fatal(r.Run(":" + port))
}
