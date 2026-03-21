// Package db provides structured access and database migrations for the PostgreSQL persistence layer.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps the PostgreSQL database connection
type DB struct {
	*sql.DB
}

// New creates a new database connection
func New(host string, port int, user, password, dbname, sslmode string) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		DB: db,
	}, nil
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	migrations := []string{
		// Service mappings
		`CREATE TABLE IF NOT EXISTS service_mappings (
			id SERIAL PRIMARY KEY,
			service_name TEXT UNIQUE NOT NULL,
			github_repo TEXT NOT NULL,
			prometheus_query TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		// Credentials
		`CREATE TABLE IF NOT EXISTS credentials (
			id SERIAL PRIMARY KEY,
			provider TEXT NOT NULL,
			key_name TEXT NOT NULL,
			key_value TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		// Incidents
		`CREATE TABLE IF NOT EXISTS incidents (
			id TEXT PRIMARY KEY,
			service_name TEXT NOT NULL,
			alert_name TEXT NOT NULL,
			severity TEXT NOT NULL,
			started_at TIMESTAMP NOT NULL,
			resolved_at TIMESTAMP,
			root_cause TEXT,
			ai_summary TEXT,
			status TEXT DEFAULT 'open',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		// Analysis results
		`CREATE TABLE IF NOT EXISTS analysis_results (
			id SERIAL PRIMARY KEY,
			incident_id TEXT NOT NULL,
			analysis_type TEXT NOT NULL,
			result_data TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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

// Incident represents an incident record
type Incident struct {
	ID          string
	ServiceName string
	AlertName   string
	Severity    string
	StartedAt   time.Time
	ResolvedAt  *time.Time
	RootCause   *string
	AISummary   *string
	Status      string
}

// CreateIncident inserts a new incident
func (db *DB) CreateIncident(incident *Incident) error {
	stmt, err := db.Prepare(`
		INSERT INTO incidents (id, service_name, alert_name, severity, started_at, status)
		VALUES ($1, $2, $3, $4, $5, 'open')
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(incident.ID, incident.ServiceName, incident.AlertName, incident.Severity, incident.StartedAt)
	if err != nil {
		return fmt.Errorf("failed to insert incident: %w", err)
	}
	return nil
}

// ResolveIncident marks an incident as resolved
func (db *DB) ResolveIncident(id, rootCause, aiSummary string) error {
	stmt, err := db.Prepare(`
		UPDATE incidents 
		SET status = 'resolved', resolved_at = NOW(), root_cause = $1, ai_summary = $2
		WHERE id = $3
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(rootCause, aiSummary, id)
	if err != nil {
		return fmt.Errorf("failed to resolve incident: %w", err)
	}
	return nil
}

// GetIncident retrieves an incident by ID
func (db *DB) GetIncident(id string) (*Incident, error) {
	stmt, err := db.Prepare(`
		SELECT id, service_name, alert_name, severity, started_at, resolved_at, root_cause, ai_summary, status
		FROM incidents WHERE id = $1
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	var i Incident
	err = stmt.QueryRow(id).Scan(
		&i.ID,
		&i.ServiceName,
		&i.AlertName,
		&i.Severity,
		&i.StartedAt,
		&i.ResolvedAt,
		&i.RootCause,
		&i.AISummary,
		&i.Status,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query incident: %w", err)
	}
	return &i, nil
}

// ListIncidents retrieves all incidents (optionally filtered by status)
func (db *DB) ListIncidents(status string) ([]Incident, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = `SELECT id, service_name, alert_name, severity, started_at, resolved_at, root_cause, ai_summary, status 
		        FROM incidents WHERE status = $1 ORDER BY started_at DESC LIMIT 100`
		args = []interface{}{status}
	} else {
		query = `SELECT id, service_name, alert_name, severity, started_at, resolved_at, root_cause, ai_summary, status 
		        FROM incidents ORDER BY started_at DESC LIMIT 100`
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query incidents: %w", err)
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		var i Incident
		err := rows.Scan(&i.ID, &i.ServiceName, &i.AlertName, &i.Severity, &i.StartedAt, &i.ResolvedAt, &i.RootCause, &i.AISummary, &i.Status)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident: %w", err)
		}
		incidents = append(incidents, i)
	}
	return incidents, nil
}

// GetEnv gets environment variable with fallback
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
