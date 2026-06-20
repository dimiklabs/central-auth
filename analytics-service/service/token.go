package service

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AnalyticsTokenCookie = "analytics_token"
	analyticsScope       = "analytics"
	analyticsTTL         = time.Hour // 1 hour
)

var analyticsPermissions = []string{"read:stats", "read:channels"}

type ServiceClaims struct {
	Email       string   `json:"email"`
	Scope       string   `json:"scope"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

func IssueAnalyticsToken(userID, email string) (string, int, error) {
	ttl := int(analyticsTTL.Seconds())
	cl := ServiceClaims{
		Email:       email,
		Scope:       analyticsScope,
		Permissions: analyticsPermissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(analyticsTTL)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, cl)
	token, err := t.SignedString([]byte(os.Getenv("JWT_SECRET")))
	return token, ttl, err
}

func ValidateAnalyticsToken(tokenStr string) (*ServiceClaims, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &ServiceClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !t.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	cl, ok := t.Claims.(*ServiceClaims)
	if !ok || cl.Scope != analyticsScope {
		return nil, fmt.Errorf("wrong scope")
	}
	if cl.ExpiresAt != nil && cl.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}
	return cl, nil
}
