package service

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	ReportTokenCookie = "report_token"
	reportIssuer      = "report"
	reportAudience    = "report"
	reportScope       = "reports"
	reportTTL         = 30 * time.Minute
)

var reportPermissions = []string{"read:reports", "create:reports"}

type ServiceClaims struct {
	Email       string   `json:"email"`
	Scope       string   `json:"scope"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

func IssueReportToken(userID, email string) (string, int, error) {
	now := time.Now()
	ttl := int(reportTTL.Seconds())
	cl := ServiceClaims{
		Email:       email,
		Scope:       reportScope,
		Permissions: reportPermissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    reportIssuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{reportAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(reportTTL)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, cl)
	token, err := t.SignedString([]byte(os.Getenv("JWT_SECRET")))
	return token, ttl, err
}

func ValidateReportToken(tokenStr string) (*ServiceClaims, error) {
	t, err := jwt.ParseWithClaims(
		tokenStr,
		&ServiceClaims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		},
		jwt.WithIssuer(reportIssuer),
		jwt.WithAudience(reportAudience),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !t.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	cl, ok := t.Claims.(*ServiceClaims)
	if !ok || cl.Scope != reportScope {
		return nil, fmt.Errorf("wrong scope")
	}
	return cl, nil
}
