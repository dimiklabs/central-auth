package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"report/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type centralClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// RequireAuth implements two-tier token validation:
//  1. Validate report_token (service-specific, short-lived, SameSite=Strict).
//  2. Fall back to central_auth (identity token from auth-service) and exchange it.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid, _ := c.Get("request_id")

		// Tier 1: service token — fast path.
		if raw, err := c.Cookie(service.ReportTokenCookie); err == nil {
			if cl, err := service.ValidateReportToken(raw); err == nil {
				setContext(c, cl.Subject, cl.Email, cl.Permissions)
				slog.Info("access_granted",
					slog.String("event", "data_access"),
					slog.String("service", "report"),
					slog.String("tier", "service_token"),
					slog.String("user_id", cl.Subject),
					slog.String("email", cl.Email),
					slog.String("ip", c.ClientIP()),
					slog.Any("request_id", rid),
				)
				c.Next()
				return
			}
		}

		// Tier 2: central identity token → exchange for report_token.
		central, err := c.Cookie("central_auth")
		if err != nil {
			slog.Warn("access_denied", slog.String("service", "report"), slog.String("reason", "no_token"), slog.String("ip", c.ClientIP()), slog.Any("request_id", rid))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		cc, err := parseCentralToken(central)
		if err != nil {
			slog.Warn("access_denied", slog.String("service", "report"), slog.String("reason", "invalid_central_token"), slog.String("ip", c.ClientIP()), slog.Any("request_id", rid))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		svcToken, ttl, err := service.IssueReportToken(cc.Subject, cc.Email)
		if err != nil {
			slog.Error("token_issue_failed", slog.String("service", "report"), slog.String("user_id", cc.Subject), slog.Any("request_id", rid))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not issue service token"})
			c.Abort()
			return
		}

		// SameSite=Strict: report_token is only sent by same-site fetch calls.
		// No Domain attribute → host-only: only sent to api.report.centralauth.local.
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie(service.ReportTokenCookie, svcToken, ttl, "/", "", false, true)

		setContext(c, cc.Subject, cc.Email, []string{"read:reports", "create:reports"})
		slog.Info("token_exchanged",
			slog.String("event", "token_exchange"),
			slog.String("service", "report"),
			slog.String("user_id", cc.Subject),
			slog.String("email", cc.Email),
			slog.String("ip", c.ClientIP()),
			slog.Any("request_id", rid),
		)
		c.Next()
	}
}

func setContext(c *gin.Context, userID, email string, permissions []string) {
	c.Set("user_id", userID)
	c.Set("email", email)
	c.Set("permissions", permissions)
}

// parseCentralToken validates the central_auth JWT issued by auth-service.
// Enforces: HMAC signing method, issuer == "central-auth", not expired.
func parseCentralToken(raw string) (*centralClaims, error) {
	t, err := jwt.ParseWithClaims(
		raw,
		&centralClaims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		},
		jwt.WithIssuer("central-auth"),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !t.Valid {
		return nil, fmt.Errorf("invalid central token")
	}
	return t.Claims.(*centralClaims), nil
}
