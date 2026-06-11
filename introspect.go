package sqlite

import (
	"github.com/tinywasm/orm"
)

func tableColumns(q interface {
	Query(string, ...any) (orm.Rows, error)
}, table string) ([]string, error) {
	// PRAGMA table_info does not support parameter binding.
	// Table name comes from ModelName(), which is usually trusted,
	// but we could add basic validation if needed.
	rows, err := q.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

func (e *sqliteExecutor) TableColumns(table string) ([]string, error) {
	return tableColumns(e, table)
}

func (e *sqliteTxExecutor) TableColumns(table string) ([]string, error) {
	return tableColumns(e, table)
}

// Ensure both executors implement TableIntrospector
var _ orm.TableIntrospector = (*sqliteExecutor)(nil)
var _ orm.TableIntrospector = (*sqliteTxExecutor)(nil)
