package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

type User struct {
	ID           int
	Email        string
	PasswordHash string
}

func Connect() error {
	var err error
	DB, err = sql.Open("postgres", os.Getenv("DB_DSN"))
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	return DB.Ping()
}

func FindUserByEmail(email string) (*User, error) {
	u := &User{}
	err := DB.QueryRow(
		`SELECT id, email, password_hash FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// SeedIfEmpty inserts three demo users (password: demo123) when the table is empty.
func SeedIfEmpty() error {
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	demo := []string{"alice@example.com", "bob@example.com", "carol@example.com"}
	for _, email := range demo {
		hash, err := bcrypt.GenerateFromPassword([]byte("demo123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if _, err = DB.Exec(
			`INSERT INTO users (email, password_hash) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			email, string(hash),
		); err != nil {
			return err
		}
	}
	return nil
}
