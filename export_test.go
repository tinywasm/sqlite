package sqlite

import (
	"database/sql"

	"github.com/tinywasm/orm"
)

// ExportTranslateQuery exposes translation purely for testing unsupported action cases.
func ExportTranslateQuery(q orm.Query) (string, []any, error) {
	return translateQuery(q)
}

// GetExecutor exposes the executor for direct execution testing.
func GetExecutor(db *orm.DB) orm.Executor {
	dbMu.RLock()
	sqlDB := dbRegistry[db]
	dbMu.RUnlock()
	return &sqliteExecutor{db: sqlDB}
}

// GetTxExecutor exposes the tx executor for direct execution testing.
func GetTxExecutor(db *orm.DB) (orm.TxBoundExecutor, error) {
	dbMu.RLock()
	sqlDB := dbRegistry[db]
	dbMu.RUnlock()
	exec := &sqliteExecutor{db: sqlDB}
	return exec.BeginTx()
}

// GetSqlDB exposes the raw database instance to trigger closed-DB errors.
func GetSqlDB(db *orm.DB) *sql.DB {
	dbMu.RLock()
	defer dbMu.RUnlock()
	return dbRegistry[db]
}
