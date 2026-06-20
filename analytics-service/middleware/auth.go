package middleware

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"analytics-service/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// centralClaims parses the central_auth identity token issued by auth-service.
type centralClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// RequireAuth implements two-tier token validation:
//  1. Check for a valid analytics_token (service-specific, short-lived).
//  2. Fall back to central_auth (identity token) and exchange it for an analytics_token.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ── Tier 1: service token ────────────────────────────────────────────
		if raw, err := c.Cookie(service.AnalyticsTokenCookie); err == nil {
			if cl, err := service.ValidateAnalyticsToken(raw); err == nil {
				setContext(c, cl.Subject, cl.Email, cl.Permissions)
				c.Next()
				return
			}
		}

		// ── Tier 2: central identity token → issue service token ─────────────
		central, err := c.Cookie("central_auth")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		cc, err := parseCentralToken(central)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		svcToken, ttl, err := service.IssueAnalyticsToken(cc.Subject, cc.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not issue service token"})
			c.Abort()
			return
		}

		// Service token is host-only (no Domain) — only sent to this API host.
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(service.AnalyticsTokenCookie, svcToken, ttl, "/", "", false, true)

		setContext(c, cc.Subject, cc.Email, []string{"read:stats", "read:channels"})
		c.Next()
	}
}

func setContext(c *gin.Context, userID, email string, permissions []string) {
	c.Set("user_id", userID)
	c.Set("email", email)
	c.Set("permissions", permissions)
}

func parseCentralToken(raw string) (*centralClaims, error) {
	t, err := jwt.ParseWithClaims(raw, &centralClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !t.Valid {
		return nil, fmt.Errorf("invalid central token")
	}
	cl := t.Claims.(*centralClaims)
	if cl.ExpiresAt != nil && cl.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("central token expired")
	}
	return cl, nil
}
