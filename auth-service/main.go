package main

import (
	"log"
	"os"

	"auth-service/db"
	"auth-service/handlers"

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

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/login", handlers.GetLogin)
	r.POST("/login", handlers.PostLogin)
	r.GET("/logout", handlers.GetLogout)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}
	log.Fatal(r.Run(":" + port))
}
