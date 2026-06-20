package handlers

import (
	"net/http"

	"analytics/service"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	svc *service.AnalyticsService
}

func NewAnalyticsHandler(svc *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

func (h *AnalyticsHandler) GetAnalytics(c *gin.Context) {
	data := h.svc.GetData()
	perms, _ := c.Get("permissions")
	c.JSON(http.StatusOK, gin.H{
		"email":       c.GetString("email"),
		"user_id":     c.GetString("user_id"),
		"scope":       "analytics",
		"permissions": perms,
		"stats":       data.Stats,
		"channels":    data.Channels,
	})
}
