package sqlite

import (
	"database/sql"
	"sync"

	"github.com/tinywasm/fmt"
	"github.com/tinywasm/orm"
	_ "modernc.org/sqlite" // SQLite driver
)

var (
	instances = make(map[*orm.DB]*SqliteAdapter)
	mu        sync.Mutex
)

// SqliteAdapter implements orm.Adapter for SQLite.
type SqliteAdapter struct {
	db *sql.DB
}

// New creates a new SqliteAdapter.
func New(dsn string) (*orm.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Err("failed to open sqlite database: %s", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Err("failed to ping sqlite database: %s", err)
	}

	adapter := &SqliteAdapter{db: db}
	ormDB := orm.New(adapter)

	mu.Lock()
	instances[ormDB] = adapter
	mu.Unlock()

	return ormDB, nil
}

// Close closes the database connection.
func Close(db *orm.DB) error {
	mu.Lock()
	adapter, ok := instances[db]
	if ok {
		delete(instances, db)
	}
	mu.Unlock()

	if !ok {
		return fmt.Err("database instance not found or already closed")
	}
	return adapter.Close()
}

// ExecSQL executes raw SQL. Useful for migrations.
func ExecSQL(db *orm.DB, query string, args ...any) error {
	mu.Lock()
	adapter, ok := instances[db]
	mu.Unlock()

	if !ok {
		return fmt.Err("database instance not found")
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
