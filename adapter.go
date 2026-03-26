package sqlite

import (
	"database/sql"

	"github.com/tinywasm/orm"

	"github.com/tinywasm/fmt"
	_ "modernc.org/sqlite" // SQLite driver
)

// Open creates a new sqlite connection and wraps it in an orm.DB.
func Open(dsn string) (*orm.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errf("failed to open sqlite database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errf("failed to ping sqlite database: %v", err)
	}

	// SQLite does not support concurrent writers. In-memory databases (:memory:)
	// are per-connection — each new connection sees an empty database. Limiting
	// to a single connection prevents both "database is locked" and
	// "no such table" errors when multiple goroutines share the same orm.DB.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	exec := &sqliteExecutor{db: db}
	compiler := sqliteCompiler{}
	return orm.New(exec, compiler), nil
}

// Close closes the database connection associated with the orm.DB.
func Close(db *orm.DB) error {
	if db == nil || db.RawExecutor() == nil {
		return fmt.Err("database instance or executor is nil")
	}
	return db.Close()
}

// ExecSQL executes raw SQL. Useful for testing or migrations.
func ExecSQL(db *orm.DB, query string, args ...any) error {
	if db == nil || db.RawExecutor() == nil {
		return fmt.Err("database instance or executor is nil")
	}
	return db.RawExecutor().Exec(query, args...)
}
