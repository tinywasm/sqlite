package sqlite

import (
	"database/sql"

	"github.com/tinywasm/orm"
)

// ExportTranslateQuery exposes translation purely for testing unsupported action cases.
func ExportTranslateQuery(q orm.Query, m orm.Model) (string, []any, error) {
	return translateQuery(q, m)
}

// GetExecutor exposes the executor for direct execution testing.
func GetExecutor(db *orm.DB) orm.Executor {
	return db.RawExecutor()
}

// GetTxExecutor exposes the tx executor for direct execution testing.
func GetTxExecutor(db *orm.DB) (orm.TxBoundExecutor, error) {
	exec := db.RawExecutor().(*sqliteExecutor)
	return exec.BeginTx()
}

// GetSqlDB exposes the raw database instance to trigger closed-DB errors.
func GetSqlDB(db *orm.DB) *sql.DB {
	return db.RawExecutor().(*sqliteExecutor).db
}
