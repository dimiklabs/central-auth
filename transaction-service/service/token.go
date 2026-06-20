package service

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TransactionTokenCookie = "transaction_token"
	transactionScope       = "transactions"
	transactionTTL         = 15 * time.Minute // 15 minutes — most sensitive financial data
)

var transactionPermissions = []string{"read:transactions"}

type ServiceClaims struct {
	Email       string   `json:"email"`
	Scope       string   `json:"scope"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

func IssueTransactionToken(userID, email string) (string, int, error) {
	ttl := int(transactionTTL.Seconds())
	cl := ServiceClaims{
		Email:       email,
		Scope:       transactionScope,
		Permissions: transactionPermissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(transactionTTL)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, cl)
	token, err := t.SignedString([]byte(os.Getenv("JWT_SECRET")))
	return token, ttl, err
}

func ValidateTransactionToken(tokenStr string) (*ServiceClaims, error) {
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
	if !ok || cl.Scope != transactionScope {
		return nil, fmt.Errorf("wrong scope")
	}
	if cl.ExpiresAt != nil && cl.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}
	return cl, nil
}
