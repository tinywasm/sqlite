package sqlite

import (
	"database/sql"

	"github.com/tinywasm/orm"
)

// sqliteExecutor implements orm.Executor and orm.TxExecutor.
type sqliteExecutor struct {
	db *sql.DB
}

func (e *sqliteExecutor) Exec(query string, args ...any) error {
	_, err := e.db.Exec(query, args...)
	return err
}

func (e *sqliteExecutor) QueryRow(query string, args ...any) orm.Scanner {
	return e.db.QueryRow(query, args...)
}

func (e *sqliteExecutor) Query(query string, args ...any) (orm.Rows, error) {
	return e.db.Query(query, args...)
}

func (e *sqliteExecutor) Close() error {
	return e.db.Close()
}

func (e *sqliteExecutor) BeginTx() (orm.TxBoundExecutor, error) {
	tx, err := e.db.Begin()
	if err != nil {
		return nil, err
	}
	return &sqliteTxExecutor{tx: tx}, nil
}

// sqliteTxExecutor implements orm.TxBoundExecutor.
type sqliteTxExecutor struct {
	tx *sql.Tx
}

func (e *sqliteTxExecutor) Exec(query string, args ...any) error {
	_, err := e.tx.Exec(query, args...)
	return err
}

func (e *sqliteTxExecutor) QueryRow(query string, args ...any) orm.Scanner {
	return e.tx.QueryRow(query, args...)
}

func (e *sqliteTxExecutor) Query(query string, args ...any) (orm.Rows, error) {
	return e.tx.Query(query, args...)
}

func (e *sqliteTxExecutor) Commit() error {
	return e.tx.Commit()
}

func (e *sqliteTxExecutor) Rollback() error {
	return e.tx.Rollback()
}

func (e *sqliteTxExecutor) Close() error {
	return nil // sql.Tx doesn't have an explicit close outside of Commit/Rollback, but we must implement the interface
}
