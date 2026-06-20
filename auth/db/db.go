package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

var DB *sql.DB

func Connect() error {
	var err error
	DB, err = sql.Open("postgres", os.Getenv("DB_DSN"))
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)
	DB.SetConnMaxIdleTime(2 * time.Minute)
	return DB.Ping()
}

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
		hash, err := bcrypt.GenerateFromPassword([]byte("demo123"), bcryptCost)
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
