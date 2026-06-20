package handlers

import (
	"net/http"

	"transaction-service/service"

	"github.com/gin-gonic/gin"
)

type TransactionHandler struct {
	svc *service.TransactionService
}

func NewTransactionHandler(svc *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{svc: svc}
}

func (h *TransactionHandler) GetTransactions(c *gin.Context) {
	transactions := h.svc.GetTransactions()
	perms, _ := c.Get("permissions")
	c.JSON(http.StatusOK, gin.H{
		"email":        c.GetString("email"),
		"user_id":      c.GetString("user_id"),
		"scope":        "transactions",
		"permissions":  perms,
		"transactions": transactions,
	})
}
