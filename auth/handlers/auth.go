package handlers

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"auth/middleware"
	"auth/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

func (h *AuthHandler) PostLogin(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	returnTo := c.PostForm("return_to")

	authFrontendURL := os.Getenv("AUTH_FRONTEND_URL")
	rid, _ := c.Get("request_id")

	bounce := func(msg string) {
		q := url.Values{"error": {msg}}
		if returnTo != "" {
			q.Set("return_to", returnTo)
		}
		c.Redirect(http.StatusFound, authFrontendURL+"?"+q.Encode())
	}

	// Input validation — bcrypt silently truncates at 72 bytes.
	if len(email) == 0 || len(email) > 254 {
		slog.Warn("login_rejected", slog.String("reason", "invalid_email_length"), slog.String("ip", c.ClientIP()), slog.Any("request_id", rid))
		bounce("invalid credentials")
		return
	}
	if len(password) == 0 || len(password) > 72 {
		slog.Warn("login_rejected", slog.String("reason", "invalid_password_length"), slog.String("ip", c.ClientIP()), slog.Any("request_id", rid))
		bounce("invalid credentials")
		return
	}

	// Validate return_to against the allowed domain to prevent open redirect.
	if !isSafeRedirect(returnTo) {
		returnTo = ""
	}

	token, maxAge, err := h.authSvc.Login(email, password)
	if err != nil {
		middleware.RecordLoginFailure(c.ClientIP())
		slog.Warn("login_failed",
			slog.String("event", "login"),
			slog.String("result", "failure"),
			slog.String("email", email),
			slog.String("ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
			slog.Any("request_id", rid),
		)
		bounce("invalid credentials")
		return
	}

	middleware.RecordLoginSuccess(c.ClientIP())
	slog.Info("login_success",
		slog.String("event", "login"),
		slog.String("result", "success"),
		slog.String("email", email),
		slog.String("ip", c.ClientIP()),
		slog.String("user_agent", c.Request.UserAgent()),
		slog.Any("request_id", rid),
	)

	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	// SameSite=Lax allows the cookie to be sent on same-site top-level navigations
	// (e.g. address-bar visits to *.centralauth.local) while still blocking cross-site POSTs.
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
	rid, _ := c.Get("request_id")
	slog.Info("logout",
		slog.String("event", "logout"),
		slog.String("ip", c.ClientIP()),
		slog.String("user_agent", c.Request.UserAgent()),
		slog.Any("request_id", rid),
	)
	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("central_auth", "", -1, "/", cookieDomain, false, true)
	c.Redirect(http.StatusFound, os.Getenv("AUTH_FRONTEND_URL"))
}

// isSafeRedirect returns true only for URLs within the centralauth.local domain.
// Prevents open redirect: an attacker cannot redirect victims to external sites after login.
func isSafeRedirect(raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" {
		return false
	}
	host := u.Hostname()
	return host == "centralauth.local" || strings.HasSuffix(host, ".centralauth.local")
}
