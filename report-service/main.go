package main

import (
	"log"
	"os"

	"report-service/handlers"
	"report-service/middleware"
	"report-service/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, reading from environment")
	}

	reportSvc := service.NewReportService()
	reportHandler := handlers.NewReportHandler(reportSvc)

	r := gin.Default()
	r.Use(middleware.CORS())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"service": "report", "status": "ok"})
	})

	protected := r.Group("/", middleware.RequireAuth())
	{
		protected.GET("/reports", reportHandler.GetReports)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "4001"
	}
	log.Fatal(r.Run(":" + port))
}
