package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// Connect opens a PostgreSQL connection with retry logic and returns the db handle.
// It only returns a handle whose connection was verified with Ping; if every
// attempt fails it returns the last error, so callers never receive a *sql.DB
// that points at an unreachable database.
func Connect(dsn string) (*sql.DB, error) {
	var lastErr error

	for i := 0; i < 10; i++ {
		if i > 0 {
			time.Sleep(3 * time.Second)
		}

		db, err := sql.Open("postgres", dsn)
		if err != nil {
			lastErr = err
			log.Printf("[DB] Connection attempt %d failed: %v", i+1, err)
			continue
		}

		if pingErr := db.Ping(); pingErr != nil {
			// Ping opens a real connection; close it so a failed attempt
			// does not leak a pool for every retry.
			db.Close()
			lastErr = pingErr
			log.Printf("[DB] Connection attempt %d failed: %v", i+1, pingErr)
			continue
		}

		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)

		log.Println("[DB] Connected to PostgreSQL successfully")
		return db, nil
	}

	return nil, fmt.Errorf("database unreachable after 10 attempts: %w", lastErr)
}
