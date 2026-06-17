package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func GetTransactions(c *gin.Context) {
	c.HTML(http.StatusOK, "transaction.html", gin.H{
		"email":         c.GetString("email"),
		"report_url":    os.Getenv("REPORT_SERVICE_URL"),
		"analytics_url": os.Getenv("ANALYTICS_SERVICE_URL"),
		"auth_url":      os.Getenv("AUTH_SERVICE_URL"),
	})
}
