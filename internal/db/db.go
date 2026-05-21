package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

type DBType string

const (
	Postgres DBType = "postgres"
	SQLite   DBType = "sqlite"
)

type DB struct {
	*sql.DB
	dbType DBType
}

func New(dbType DBType, host string, port int, user, password, dbname, sslmode, path string) (*DB, error) {
	var driverName, dsn string

	switch dbType {
	case SQLite:
		driverName = "sqlite"
		if path == "" {
			path = "helixops.db"
		}
		dsn = path
	case Postgres, "":
		dbType = Postgres
		driverName = "postgres"
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if dbType == Postgres {
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)
	} else {
		db.SetMaxOpenConns(1)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db, dbType: dbType}, nil
}

func (db *DB) p(n int) string {
	if db.dbType == SQLite {
		return "?"
	}
	return fmt.Sprintf("$%d", n)
}

func (db *DB) now() string {
	if db.dbType == SQLite {
		return "datetime('now')"
	}
	return "NOW()"
}

func (db *DB) Migrate() error {
	var migrations []string

	if db.dbType == SQLite {
		migrations = []string{
			`CREATE TABLE IF NOT EXISTS service_mappings (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				service_name TEXT UNIQUE NOT NULL,
				github_repo TEXT NOT NULL,
				prometheus_query TEXT,
				created_at TEXT DEFAULT (datetime('now')),
				updated_at TEXT DEFAULT (datetime('now'))
			)`,
			`CREATE TABLE IF NOT EXISTS credentials (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				provider TEXT NOT NULL,
				key_name TEXT NOT NULL,
				key_value TEXT NOT NULL,
				created_at TEXT DEFAULT (datetime('now'))
			)`,
			`CREATE TABLE IF NOT EXISTS incidents (
				id TEXT PRIMARY KEY,
				service_name TEXT NOT NULL,
				alert_name TEXT NOT NULL,
				severity TEXT NOT NULL,
				started_at TEXT NOT NULL,
				resolved_at TEXT,
				root_cause TEXT,
				ai_summary TEXT,
				status TEXT DEFAULT 'open',
				request_id TEXT,
				created_at TEXT DEFAULT (datetime('now'))
			)`,
			`CREATE TABLE IF NOT EXISTS analysis_results (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				incident_id TEXT NOT NULL,
				analysis_type TEXT NOT NULL,
				result_data TEXT NOT NULL,
				created_at TEXT DEFAULT (datetime('now')),
				FOREIGN KEY (incident_id) REFERENCES incidents(id)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_incidents_service ON incidents(service_name)`,
			`CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status)`,
			`CREATE INDEX IF NOT EXISTS idx_incidents_started ON incidents(started_at)`,
		}
	} else {
		migrations = []string{
			`CREATE TABLE IF NOT EXISTS service_mappings (
				id SERIAL PRIMARY KEY,
				service_name TEXT UNIQUE NOT NULL,
				github_repo TEXT NOT NULL,
				prometheus_query TEXT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE TABLE IF NOT EXISTS credentials (
				id SERIAL PRIMARY KEY,
				provider TEXT NOT NULL,
				key_name TEXT NOT NULL,
				key_value TEXT NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
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
				request_id TEXT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`CREATE TABLE IF NOT EXISTS analysis_results (
				id SERIAL PRIMARY KEY,
				incident_id TEXT NOT NULL,
				analysis_type TEXT NOT NULL,
				result_data TEXT NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (incident_id) REFERENCES incidents(id)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_incidents_service ON incidents(service_name)`,
			`CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status)`,
			`CREATE INDEX IF NOT EXISTS idx_incidents_started ON incidents(started_at)`,
		}

		if _, err := db.Exec(`ALTER TABLE incidents ADD COLUMN IF NOT EXISTS request_id TEXT`); err != nil {
			return fmt.Errorf("migration alter incidents failed: %w", err)
		}
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

func (db *DB) Ping(ctx context.Context) error {
	return db.DB.PingContext(ctx)
}

func (db *DB) Close() error {
	return db.DB.Close()
}

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
	RequestID   string
}

func (db *DB) CreateIncident(incident *Incident) error {
	query := fmt.Sprintf(`INSERT INTO incidents (id, service_name, alert_name, severity, started_at, status, request_id)
		VALUES (%s, %s, %s, %s, %s, 'open', %s)`,
		db.p(1), db.p(2), db.p(3), db.p(4), db.p(5), db.p(6))

	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(incident.ID, incident.ServiceName, incident.AlertName, incident.Severity, incident.StartedAt, incident.RequestID)
	if err != nil {
		return fmt.Errorf("failed to insert incident: %w", err)
	}
	return nil
}

func (db *DB) ResolveIncident(id, rootCause, aiSummary string) error {
	query := fmt.Sprintf(`UPDATE incidents
		SET status = 'resolved', resolved_at = %s, root_cause = %s, ai_summary = %s
		WHERE id = %s`,
		db.now(), db.p(1), db.p(2), db.p(3))

	stmt, err := db.Prepare(query)
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

func (db *DB) GetIncident(id string) (*Incident, error) {
	query := fmt.Sprintf(`SELECT id, service_name, alert_name, severity, started_at, resolved_at, root_cause, ai_summary, status, request_id
		FROM incidents WHERE id = %s`, db.p(1))

	stmt, err := db.Prepare(query)
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
		&i.RequestID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query incident: %w", err)
	}
	return &i, nil
}

func (db *DB) ListIncidents(status string) ([]Incident, error) {
	var query string
	var args []interface{}

	if status != "" {
		query = fmt.Sprintf(`SELECT id, service_name, alert_name, severity, started_at, resolved_at, root_cause, ai_summary, status, request_id
			FROM incidents WHERE status = %s ORDER BY started_at DESC LIMIT 100`, db.p(1))
		args = []interface{}{status}
	} else {
		query = `SELECT id, service_name, alert_name, severity, started_at, resolved_at, root_cause, ai_summary, status, request_id
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
		err := rows.Scan(&i.ID, &i.ServiceName, &i.AlertName, &i.Severity, &i.StartedAt, &i.ResolvedAt, &i.RootCause, &i.AISummary, &i.Status, &i.RequestID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident: %w", err)
		}
		incidents = append(incidents, i)
	}
	return incidents, nil
}

func (db *DB) CreateAnalysisResult(incidentID, analysisType, resultData string) error {
	query := fmt.Sprintf(`INSERT INTO analysis_results (incident_id, analysis_type, result_data)
		VALUES (%s, %s, %s)`, db.p(1), db.p(2), db.p(3))

	stmt, err := db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare analysis insert: %w", err)
	}
	defer stmt.Close()

	if _, err := stmt.Exec(incidentID, analysisType, resultData); err != nil {
		return fmt.Errorf("failed to insert analysis result: %w", err)
	}
	return nil
}


