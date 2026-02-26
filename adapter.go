package sqlite

import (
	"database/sql"
	"errors"
	"sync"

	"github.com/tinywasm/orm"

	tfmt "github.com/tinywasm/fmt"
	_ "modernc.org/sqlite" // SQLite driver
)

var (
	dbRegistry = make(map[*orm.DB]*SqliteAdapter)
	dbMu       sync.RWMutex
)

// SqliteAdapter implements orm.Adapter for SQLite.
type SqliteAdapter struct {
	db *sql.DB
}

// New creates a new SqliteAdapter and wraps it in an orm.DB.
func New(dsn string) (*orm.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, errors.New(tfmt.Sprintf("failed to open sqlite database: %v", err))
	}

	if err := db.Ping(); err != nil {
		return nil, errors.New(tfmt.Sprintf("failed to ping sqlite database: %v", err))
	}

	adapter := &SqliteAdapter{db: db}
	ormDB := orm.New(adapter)

	dbMu.Lock()
	dbRegistry[ormDB] = adapter
	dbMu.Unlock()

	return ormDB, nil
}

// Close closes the database connection associated with the orm.DB.
func Close(db *orm.DB) error {
	dbMu.Lock()
	adapter, ok := dbRegistry[db]
	if ok {
		delete(dbRegistry, db)
	}
	dbMu.Unlock()

	if !ok {
		return errors.New("database instance not found in sqlite registry")
	}

	return adapter.Close()
}

// ExecSQL executes raw SQL using the adapter associated with the orm.DB.
func ExecSQL(db *orm.DB, query string, args ...any) error {
	dbMu.RLock()
	adapter, ok := dbRegistry[db]
	dbMu.RUnlock()

	if !ok {
		return errors.New("database instance not found in sqlite registry")
	}

	return adapter.ExecSQL(query, args...)
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
