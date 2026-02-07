package storage

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
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

// Migrate reads all .sql files from the given directory and executes them
// in alphabetical order. Migrations are tracked in a migrations table
// so they only run once.
func (db *DB) Migrate(migrationsDir string) error {
	// Migrations tracking table
	_, err := db.Conn.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)

	if err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}

	// Read migration files
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("error reading migrations directory: %w", err)
	}

	// Sort to ensure consistent execution order
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	for _, filename := range files {
		// Check if already applied
		var count int
		err := db.Conn.QueryRow("SELECT COUNT(*) FROM migrations WHERE filename = ?", filename).Scan(&count)

		if err != nil {
			return fmt.Errorf("checking migration %s: %w", filename, err)
		}

		if count > 0 {
			slog.Debug("migration already applied, skipping", "file", filename)
			continue
		}

		// Read and execute migration
		path := filepath.Join(migrationsDir, filename)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", filename, err)
		}

		if _, err := db.Conn.Exec(string(content)); err != nil {
			return fmt.Errorf("executing migration %s: %w", filename, err)
		}

		// Record migration
		if _, err := db.Conn.Exec("INSERT INTO migrations (filename) VALUES (?)", filename); err != nil {
			return fmt.Errorf("recording migration %s: %w", filename, err)
		}

		slog.Info("migration applied", "file", filename)
	}

	return nil
}
