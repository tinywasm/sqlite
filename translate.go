package sqlite

import (
	"fmt"
	"strings"

	"github.com/tinywasm/orm"
)

// translateQuery converts an orm.Query into a SQLite SQL string and arguments.
func translateQuery(q orm.Query) (string, []any, error) {
	switch q.Action {
	case orm.ActionCreate:
		return buildInsert(q)
	case orm.ActionReadOne, orm.ActionReadAll:
		return buildSelect(q)
	case orm.ActionUpdate:
		return buildUpdate(q)
	case orm.ActionDelete:
		return buildDelete(q)
	default:
		return "", nil, fmt.Errorf("unknown query action: %v", q.Action)
	}
}

func buildInsert(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, fmt.Errorf("table name is required for insert")
	}
	if len(q.Columns) == 0 {
		return "", nil, fmt.Errorf("columns are required for insert")
	}

	cols := strings.Join(q.Columns, ", ")
	placeholders := make([]string, len(q.Columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	vals := strings.Join(placeholders, ", ")

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", q.Table, cols, vals)
	return sql, q.Values, nil
}

func buildSelect(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, fmt.Errorf("table name is required for select")
	}

	// Always select all columns for now, as we map to the model struct
	cols := "*"

	var whereClauses []string
	var args []any

	for i, c := range q.Conditions {
		clause := fmt.Sprintf("%s %s ?", c.Field(), c.Operator())
		if i > 0 {
			clause = fmt.Sprintf(" %s %s", c.Logic(), clause)
		}
		whereClauses = append(whereClauses, clause)
		args = append(args, c.Value())
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, "")
	}

	orderBySQL := ""
	if len(q.OrderBy) > 0 {
		var orders []string
		for _, o := range q.OrderBy {
			orders = append(orders, fmt.Sprintf("%s %s", o.Column(), o.Dir()))
		}
		orderBySQL = " ORDER BY " + strings.Join(orders, ", ")
	}

	limitSQL := ""
	if q.Limit > 0 {
		limitSQL = fmt.Sprintf(" LIMIT %d", q.Limit)
	}

	offsetSQL := ""
	if q.Offset > 0 {
		offsetSQL = fmt.Sprintf(" OFFSET %d", q.Offset)
	}

	sql := fmt.Sprintf("SELECT %s FROM %s%s%s%s%s", cols, q.Table, whereSQL, orderBySQL, limitSQL, offsetSQL)
	return sql, args, nil
}

func buildUpdate(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, fmt.Errorf("table name is required for update")
	}
	if len(q.Columns) == 0 {
		return "", nil, fmt.Errorf("columns are required for update")
	}

	var setClauses []string
	var args []any

	for i, col := range q.Columns {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		args = append(args, q.Values[i])
	}

	var whereClauses []string
	for i, c := range q.Conditions {
		clause := fmt.Sprintf("%s %s ?", c.Field(), c.Operator())
		if i > 0 {
			clause = fmt.Sprintf(" %s %s", c.Logic(), clause)
		}
		whereClauses = append(whereClauses, clause)
		args = append(args, c.Value())
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, "")
	}

	sql := fmt.Sprintf("UPDATE %s SET %s%s", q.Table, strings.Join(setClauses, ", "), whereSQL)
	return sql, args, nil
}

func buildDelete(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, fmt.Errorf("table name is required for delete")
	}

	var whereClauses []string
	var args []any

	for i, c := range q.Conditions {
		clause := fmt.Sprintf("%s %s ?", c.Field(), c.Operator())
		if i > 0 {
			clause = fmt.Sprintf(" %s %s", c.Logic(), clause)
		}
		whereClauses = append(whereClauses, clause)
		args = append(args, c.Value())
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, "")
	}

	sql := fmt.Sprintf("DELETE FROM %s%s", q.Table, whereSQL)
	return sql, args, nil
}
