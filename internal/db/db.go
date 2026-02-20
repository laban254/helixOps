// Package db provides structured access and database migrations for the SQLite persistence layer.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database connection
type DB struct {
	*sql.DB
	path string
}

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		DB:  db,
		path: dbPath,
	}, nil
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	migrations := []string{
		// Service mappings
		`CREATE TABLE IF NOT EXISTS service_mappings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			service_name TEXT UNIQUE NOT NULL,
			github_repo TEXT NOT NULL,
			prometheus_query TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Credentials
		`CREATE TABLE IF NOT EXISTS credentials (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			key_name TEXT NOT NULL,
			key_value TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Incidents
		`CREATE TABLE IF NOT EXISTS incidents (
			id TEXT PRIMARY KEY,
			service_name TEXT NOT NULL,
			alert_name TEXT NOT NULL,
			severity TEXT NOT NULL,
			started_at DATETIME NOT NULL,
			resolved_at DATETIME,
			root_cause TEXT,
			ai_summary TEXT,
			status TEXT DEFAULT 'open',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Analysis results
		`CREATE TABLE IF NOT EXISTS analysis_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			incident_id TEXT NOT NULL,
			analysis_type TEXT NOT NULL,
			result_data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (incident_id) REFERENCES incidents(id)
		)`,
		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_incidents_service ON incidents(service_name)`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status)`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_started ON incidents(started_at)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
