package main

import (
	"fmt"
	"log"
	"os"

	"transaction/handlers"
	"transaction/middleware"
	"transaction/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, reading from environment")
	}

	validateConfig()

	transactionSvc := service.NewTransactionService()
	transactionHandler := handlers.NewTransactionHandler(transactionSvc)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	if err := r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}); err != nil {
		log.Fatalf("trusted proxies: %v", err)
	}

	r.Use(middleware.SecurityHeaders())
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

func validateConfig() {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		panic(fmt.Sprintf("JWT_SECRET must be at least 32 characters; got %d", len(secret)))
	}
}
