package sqlite

import "github.com/tinywasm/orm"

// GetAdapter returns the adapter for a DB, for testing purposes.
func GetAdapter(db *orm.DB) *SqliteAdapter {
	dbMu.RLock()
	defer dbMu.RUnlock()
	return dbRegistry[db]
}

// GetTxAdapter gets a TxBound adapter for testing unsupported actions.
func GetTxAdapter(db *SqliteAdapter) (orm.TxBound, error) {
	return db.BeginTx()
}
