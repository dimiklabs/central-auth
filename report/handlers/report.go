package handlers

import (
	"net/http"

	"report/service"

	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	svc *service.ReportService
}

func NewReportHandler(svc *service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

func (h *ReportHandler) GetReports(c *gin.Context) {
	reports := h.svc.GetReports()
	perms, _ := c.Get("permissions")
	c.JSON(http.StatusOK, gin.H{
		"email":       c.GetString("email"),
		"user_id":     c.GetString("user_id"),
		"scope":       "reports",
		"permissions": perms,
		"reports":     reports,
	})
}
