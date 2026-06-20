package main

import (
	"log"
	"os"

	"transaction-service/handlers"
	"transaction-service/middleware"
	"transaction-service/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, reading from environment")
	}

	transactionSvc := service.NewTransactionService()
	transactionHandler := handlers.NewTransactionHandler(transactionSvc)

	r := gin.Default()
	r.Use(middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"service": "transaction", "status": "ok"})
	})

	protected := r.Group("/", middleware.RequireAuth())
	{
		protected.GET("/transactions", transactionHandler.GetTransactions)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "4003"
	}
	log.Fatal(r.Run(":" + port))
}
