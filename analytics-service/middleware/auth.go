package middleware

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("central_auth")
		if err != nil {
			loginRedirect(c)
			return
		}

		token, err := jwt.ParseWithClaims(cookie, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})
		if err != nil || !token.Valid {
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie("central_auth", "", -1, "/", "", false, true)
			loginRedirect(c)
			return
		}

		cl := token.Claims.(*Claims)
		if cl.ExpiresAt != nil && cl.ExpiresAt.Before(time.Now()) {
			loginRedirect(c)
			return
		}

		c.Set("email", cl.Email)
		c.Set("user_id", cl.Subject)
		c.Next()
	}
}

func loginRedirect(c *gin.Context) {
	returnTo := "http://" + c.Request.Host + c.Request.RequestURI
	authURL := os.Getenv("AUTH_SERVICE_URL")
	c.Redirect(http.StatusFound, authURL+"/login?return_to="+url.QueryEscape(returnTo))
	c.Abort()
}
