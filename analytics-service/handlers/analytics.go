package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func GetAnalytics(c *gin.Context) {
	c.HTML(http.StatusOK, "analytics.html", gin.H{
		"email":           c.GetString("email"),
		"report_url":      os.Getenv("REPORT_SERVICE_URL"),
		"transaction_url": os.Getenv("TRANSACTION_SERVICE_URL"),
		"auth_url":        os.Getenv("AUTH_SERVICE_URL"),
	})
}
