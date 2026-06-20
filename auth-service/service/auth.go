package service

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"auth-service/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
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
	cl := Claims{
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
