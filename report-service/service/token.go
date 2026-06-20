package service

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	ReportTokenCookie = "report_token"
	reportScope       = "reports"
	reportTTL         = 30 * time.Minute // 30 minutes — more sensitive data
)

var reportPermissions = []string{"read:reports", "create:reports"}

type ServiceClaims struct {
	Email       string   `json:"email"`
	Scope       string   `json:"scope"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

func IssueReportToken(userID, email string) (string, int, error) {
	ttl := int(reportTTL.Seconds())
	cl := ServiceClaims{
		Email:       email,
		Scope:       reportScope,
		Permissions: reportPermissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(reportTTL)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, cl)
	token, err := t.SignedString([]byte(os.Getenv("JWT_SECRET")))
	return token, ttl, err
}

func ValidateReportToken(tokenStr string) (*ServiceClaims, error) {
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
	if !ok || cl.Scope != reportScope {
		return nil, fmt.Errorf("wrong scope")
	}
	if cl.ExpiresAt != nil && cl.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}
	return cl, nil
}
