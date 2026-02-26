package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/tinywasm/orm"
)

// Execute executes a query against the database.
func (s *SqliteAdapter) Execute(q orm.Query, m orm.Model, factory func() orm.Model, each func(orm.Model)) error {
	sqlStr, args, err := translateQuery(q)
	if err != nil {
		return err
	}

	switch q.Action {
	case orm.ActionCreate, orm.ActionUpdate, orm.ActionDelete:
		_, err := s.db.Exec(sqlStr, args...)
		if err != nil {
			return err
		}
		return nil

	case orm.ActionReadOne:
		row := s.db.QueryRow(sqlStr, args...)
		if err := row.Scan(m.Pointers()...); err != nil {
			if err == sql.ErrNoRows {
				return nil // Or return a specific "not found" error if the ORM expects it
			}
			return err
		}
		return nil

	case orm.ActionReadAll:
		rows, err := s.db.Query(sqlStr, args...)
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
		return fmt.Errorf("unsupported action: %v", q.Action)
	}
}
