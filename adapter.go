package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // SQLite driver
)

// SqliteAdapter implements orm.Adapter for SQLite.
type SqliteAdapter struct {
	db *sql.DB
}

// New creates a new SqliteAdapter.
func New(dsn string) (*SqliteAdapter, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	return &SqliteAdapter{db: db}, nil
}

// Close closes the database connection.
func (s *SqliteAdapter) Close() error {
	return s.db.Close()
}

// ExecSQL executes raw SQL. Useful for migrations.
func (s *SqliteAdapter) ExecSQL(query string, args ...any) error {
	_, err := s.db.Exec(query, args...)
	return err
}
