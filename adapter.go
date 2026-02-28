package sqlite

import (
	"database/sql"
	"sync"

	"github.com/tinywasm/orm"

	. "github.com/tinywasm/fmt"
	_ "modernc.org/sqlite" // SQLite driver
)

var (
	dbRegistry = make(map[*orm.DB]*sql.DB)
	dbMu       sync.RWMutex
)

// Open creates a new sqlite connection and wraps it in an orm.DB.
func Open(dsn string) (*orm.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, Errf("failed to open sqlite database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, Errf("failed to ping sqlite database: %v", err)
	}

	exec := &sqliteExecutor{db: db}
	compiler := sqliteCompiler{}
	ormDB := orm.New(exec, compiler)

	dbMu.Lock()
	dbRegistry[ormDB] = db
	dbMu.Unlock()

	return ormDB, nil
}

// Close closes the database connection associated with the orm.DB.
func Close(db *orm.DB) error {
	dbMu.Lock()
	sqlDB, ok := dbRegistry[db]
	if ok {
		delete(dbRegistry, db)
	}
	dbMu.Unlock()

	if !ok {
		return Err("database instance not found in sqlite registry")
	}

	return sqlDB.Close()
}

// ExecSQL executes raw SQL. Useful for testing or migrations.
func ExecSQL(db *orm.DB, query string, args ...any) error {
	dbMu.RLock()
	sqlDB, ok := dbRegistry[db]
	dbMu.RUnlock()

	if !ok {
		return Err("database instance not found in sqlite registry")
	}

	_, err := sqlDB.Exec(query, args...)
	return err
}
