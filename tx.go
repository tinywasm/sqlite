package sqlite

import (
	"database/sql"

	"github.com/tinywasm/fmt"

	"github.com/tinywasm/orm"
)

// BeginTx starts a new transaction.
func (s *SqliteAdapter) BeginTx() (orm.TxBound, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	return &SqliteTxBound{tx: tx}, nil
}

// SqliteTxBound implements orm.TxBound.
type SqliteTxBound struct {
	tx *sql.Tx
}

// Commit commits the transaction.
func (s *SqliteTxBound) Commit() error {
	return s.tx.Commit()
}

// Rollback rolls back the transaction.
func (s *SqliteTxBound) Rollback() error {
	return s.tx.Rollback()
}

// Execute executes a query within the transaction.
func (s *SqliteTxBound) Execute(q orm.Query, m orm.Model, factory func() orm.Model, each func(orm.Model)) error {
	sqlStr, args, err := translateQuery(q)
	if err != nil {
		return err
	}

	switch q.Action {
	case orm.ActionCreate, orm.ActionUpdate, orm.ActionDelete:
		_, err := s.tx.Exec(sqlStr, args...)
		if err != nil {
			return err
		}
		return nil

	case orm.ActionReadOne:
		row := s.tx.QueryRow(sqlStr, args...)
		if err := row.Scan(m.Pointers()...); err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		return nil

	case orm.ActionReadAll:
		rows, err := s.tx.Query(sqlStr, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			newModel := factory()
			if err := rows.Scan(newModel.Pointers()...); err != nil {
				return err
			}
			each(newModel)
		}
		return rows.Err()

	default:
		return fmt.Errf("unsupported action: %v", q.Action)
	}
}
