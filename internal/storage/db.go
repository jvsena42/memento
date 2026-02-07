package storage

import (
	"database/sql"
	"fmt"
)

type DB struct {
	Conn *sql.DB
}

func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)

	if err != nil {
		return nil, fmt.Errorf("opening database, %w", err)
	}

	// Verify the connection works
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	// Enable WAL mode for better read/write performance

	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enabilng foreign keys: %w", err)
	}

	return &DB{Conn: conn}, nil
}
