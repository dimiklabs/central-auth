package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func GetReports(c *gin.Context) {
	c.HTML(http.StatusOK, "report.html", gin.H{
		"email":           c.GetString("email"),
		"analytics_url":   os.Getenv("ANALYTICS_SERVICE_URL"),
		"transaction_url": os.Getenv("TRANSACTION_SERVICE_URL"),
		"auth_url":        os.Getenv("AUTH_SERVICE_URL"),
	})
}
