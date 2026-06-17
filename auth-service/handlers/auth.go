package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"auth-service/db"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func GetLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"return_to": c.Query("return_to"),
		"error":     c.Query("error"),
	})
}

func PostLogin(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")
	returnTo := c.PostForm("return_to")

	bounce := func(msg string) {
		q := url.Values{"error": {msg}}
		if returnTo != "" {
			q.Set("return_to", returnTo)
		}
		c.Redirect(http.StatusFound, "/login?"+q.Encode())
	}

	user, err := db.FindUserByEmail(email)
	if err != nil || user == nil {
		bounce("invalid credentials")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		bounce("invalid credentials")
		return
	}

	maxAge := cookieMaxAge()
	signed, err := mintJWT(user, maxAge)
	if err != nil {
		c.String(http.StatusInternalServerError, "token error")
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("central_auth", signed, maxAge, "/", "", false, true)

	if returnTo == "" {
		returnTo = "/"
	}
	c.Redirect(http.StatusFound, returnTo)
}

func GetLogout(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("central_auth", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}

func mintJWT(user *db.User, maxAge int) (string, error) {
	cl := claims{
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(maxAge) * time.Second)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, cl)
	return t.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func cookieMaxAge() int {
	v, _ := strconv.Atoi(os.Getenv("COOKIE_MAX_AGE"))
	if v == 0 {
		return 86400
	}
	return v
}
