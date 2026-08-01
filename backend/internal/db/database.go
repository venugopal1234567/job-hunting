package db

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// Connect opens a PostgreSQL connection with retry logic and returns the db handle
func Connect(dsn string) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			if pingErr := db.Ping(); pingErr == nil {
				break
			}
		}
		log.Printf("[DB] Connection attempt %d failed, retrying in 3s...", i+1)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("[DB] Connected to PostgreSQL successfully")
	return db, nil
}
