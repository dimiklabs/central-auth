package handlers

import (
	"net/http"
	"net/url"
	"os"

	"auth-service/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

func (h *AuthHandler) PostLogin(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")
	returnTo := c.PostForm("return_to")

	authFrontendURL := os.Getenv("AUTH_FRONTEND_URL")

	bounce := func(msg string) {
		q := url.Values{"error": {msg}}
		if returnTo != "" {
			q.Set("return_to", returnTo)
		}
		c.Redirect(http.StatusFound, authFrontendURL+"?"+q.Encode())
	}

	token, maxAge, err := h.authSvc.Login(email, password)
	if err != nil {
		bounce("invalid credentials")
		return
	}

	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("central_auth", token, maxAge, "/", cookieDomain, false, true)

	if returnTo == "" {
		returnTo = os.Getenv("DEFAULT_REDIRECT_URL")
		if returnTo == "" {
			returnTo = authFrontendURL
		}
	}
	c.Redirect(http.StatusFound, returnTo)
}

func (h *AuthHandler) GetLogout(c *gin.Context) {
	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("central_auth", "", -1, "/", cookieDomain, false, true)
	c.Redirect(http.StatusFound, os.Getenv("AUTH_FRONTEND_URL"))
}
