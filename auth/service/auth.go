package service

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"auth/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	tokenIssuer  = "central-auth"
	BcryptCost   = 12
	MinSecretLen = 32
)

type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type AuthService struct {
	userRepo *repository.UserRepository
}

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

// ValidateConfig panics at startup if required environment variables are missing or too weak.
func ValidateConfig() {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < MinSecretLen {
		panic(fmt.Sprintf(
			"JWT_SECRET must be at least %d characters; got %d — generate one with: openssl rand -hex 32",
			MinSecretLen, len(secret),
		))
	}
	if os.Getenv("COOKIE_DOMAIN") == "" {
		panic("COOKIE_DOMAIN must be set (e.g. .centralauth.local)")
	}
}

func (s *AuthService) Login(email, password string) (string, int, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil || user == nil {
		return "", 0, fmt.Errorf("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", 0, fmt.Errorf("invalid credentials")
	}
	maxAge := CookieMaxAge()
	token, err := mintJWT(user, maxAge)
	return token, maxAge, err
}

func CookieMaxAge() int {
	v, _ := strconv.Atoi(os.Getenv("COOKIE_MAX_AGE"))
	if v == 0 {
		return 86400
	}
	return v
}

func mintJWT(user *repository.User, maxAge int) (string, error) {
	now := time.Now()
	cl := Claims{
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(maxAge) * time.Second)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, cl)
	return t.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
