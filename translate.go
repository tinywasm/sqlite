package sqlite

import (
	"errors"
	tfmt "github.com/tinywasm/fmt"

	"github.com/tinywasm/orm"
)

func join(elems []string, sep string) string {
	if len(elems) == 0 {
		return ""
	}
	if len(elems) == 1 {
		return elems[0]
	}
	n := len(sep) * (len(elems) - 1)
	for i := 0; i < len(elems); i++ {
		n += len(elems[i])
	}

	b := make([]byte, n)
	bp := copy(b, elems[0])
	for _, s := range elems[1:] {
		bp += copy(b[bp:], sep)
		bp += copy(b[bp:], s)
	}
	return string(b)
}

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
		return "", nil, errors.New(tfmt.Sprintf("unknown query action: %v", q.Action))
	}
}

func buildInsert(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, errors.New("table name is required for insert")
	}
	if len(q.Columns) == 0 {
		return "", nil, errors.New("columns are required for insert")
	}

	cols := join(q.Columns, ", ")
	placeholders := make([]string, len(q.Columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	vals := join(placeholders, ", ")

	sql := tfmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", q.Table, cols, vals)
	return sql, q.Values, nil
}

func buildSelect(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, errors.New("table name is required for select")
	}

	// Always select all columns for now, as we map to the model struct
	cols := "*"

	var whereClauses []string
	var args []any

	for i, c := range q.Conditions {
		clause := tfmt.Sprintf("%s %s ?", c.Field(), c.Operator())
		if i > 0 {
			clause = tfmt.Sprintf(" %s %s", c.Logic(), clause)
		}
		whereClauses = append(whereClauses, clause)
		args = append(args, c.Value())
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + join(whereClauses, "")
	}

	orderBySQL := ""
	if len(q.OrderBy) > 0 {
		var orders []string
		for _, o := range q.OrderBy {
			orders = append(orders, tfmt.Sprintf("%s %s", o.Column(), o.Dir()))
		}
		orderBySQL = " ORDER BY " + join(orders, ", ")
	}

	limitSQL := ""
	if q.Limit > 0 {
		limitSQL = tfmt.Sprintf(" LIMIT %d", q.Limit)
	}

	offsetSQL := ""
	if q.Offset > 0 {
		offsetSQL = tfmt.Sprintf(" OFFSET %d", q.Offset)
	}

	sql := tfmt.Sprintf("SELECT %s FROM %s%s%s%s%s", cols, q.Table, whereSQL, orderBySQL, limitSQL, offsetSQL)
	return sql, args, nil
}

func buildUpdate(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, errors.New("table name is required for update")
	}
	if len(q.Columns) == 0 {
		return "", nil, errors.New("columns are required for update")
	}

	var setClauses []string
	var args []any

	for i, col := range q.Columns {
		setClauses = append(setClauses, tfmt.Sprintf("%s = ?", col))
		args = append(args, q.Values[i])
	}

	var whereClauses []string
	for i, c := range q.Conditions {
		clause := tfmt.Sprintf("%s %s ?", c.Field(), c.Operator())
		if i > 0 {
			clause = tfmt.Sprintf(" %s %s", c.Logic(), clause)
		}
		whereClauses = append(whereClauses, clause)
		args = append(args, c.Value())
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + join(whereClauses, "")
	}

	sql := tfmt.Sprintf("UPDATE %s SET %s%s", q.Table, join(setClauses, ", "), whereSQL)
	return sql, args, nil
}

func buildDelete(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, errors.New("table name is required for delete")
	}

	var whereClauses []string
	var args []any

	for i, c := range q.Conditions {
		clause := tfmt.Sprintf("%s %s ?", c.Field(), c.Operator())
		if i > 0 {
			clause = tfmt.Sprintf(" %s %s", c.Logic(), clause)
		}
		whereClauses = append(whereClauses, clause)
		args = append(args, c.Value())
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + join(whereClauses, "")
	}

	sql := tfmt.Sprintf("DELETE FROM %s%s", q.Table, whereSQL)
	return sql, args, nil
}
