package main

import (
	"log"
	"os"

	"auth/db"
	"auth/handlers"
	"auth/middleware"
	"auth/repository"
	"auth/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, reading from environment")
	}

	service.ValidateConfig()

	if err := db.Connect(); err != nil {
		log.Fatalf("db connect: %v", err)
	}

	if err := db.SeedIfEmpty(); err != nil {
		log.Printf("seed warning: %v", err)
	}

	userRepo := repository.NewUserRepository(db.DB)
	authSvc := service.NewAuthService(userRepo)
	authHandler := handlers.NewAuthHandler(authSvc)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Trust only private RFC-1918 ranges (nginx Docker container).
	if err := r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}); err != nil {
		log.Fatalf("trusted proxies: %v", err)
	}

	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"service": "auth", "status": "ok"})
	})

	r.POST("/login", middleware.RateLimitLogin(), authHandler.PostLogin)
	r.GET("/logout", authHandler.GetLogout)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}
	log.Fatal(r.Run(":" + port))
}
