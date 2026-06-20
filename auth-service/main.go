package main

import (
	"log"
	"os"

	"auth-service/db"
	"auth-service/handlers"
	"auth-service/middleware"
	"auth-service/repository"
	"auth-service/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, reading from environment")
	}

	if err := db.Connect(); err != nil {
		log.Fatalf("db connect: %v", err)
	}

	if err := db.SeedIfEmpty(); err != nil {
		log.Printf("seed warning: %v", err)
	}

	userRepo := repository.NewUserRepository(db.DB)
	authSvc := service.NewAuthService(userRepo)
	authHandler := handlers.NewAuthHandler(authSvc)

	r := gin.Default()
	r.Use(middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"service": "auth", "status": "ok"})
	})
	r.POST("/login", authHandler.PostLogin)
	r.GET("/logout", authHandler.GetLogout)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}
	log.Fatal(r.Run(":" + port))
}
