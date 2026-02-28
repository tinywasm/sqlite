package sqlite

import (
	. "github.com/tinywasm/fmt"

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
		return "", nil, Errf("unknown query action: %v", q.Action)
	}
}

func buildInsert(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, Err("table name is required for insert")
	}
	if len(q.Columns) == 0 {
		return "", nil, Err("columns are required for insert")
	}

	cols := Convert(q.Columns).Join(", ").String()
	placeholders := make([]string, len(q.Columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	vals := Convert(placeholders).Join(", ").String()

	sql := Sprintf("INSERT INTO %s (%s) VALUES (%s)", q.Table, cols, vals)
	return sql, q.Values, nil
}

func buildSelect(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, Err("table name is required for select")
	}

	// Always select all columns for now, as we map to the model struct
	cols := "*"

	var whereClauses []string
	var args []any

	for i, c := range q.Conditions {
		clause := Sprintf("%s %s ?", c.Field(), c.Operator())
		if i > 0 {
			clause = Sprintf(" %s %s", c.Logic(), clause)
		}
		whereClauses = append(whereClauses, clause)
		args = append(args, c.Value())
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + Convert(whereClauses).Join("").String()
	}

	groupBySQL := ""
	if len(q.GroupBy) > 0 {
		groupBySQL = " GROUP BY " + Convert(q.GroupBy).Join(", ").String()
	}

	orderBySQL := ""
	if len(q.OrderBy) > 0 {
		var orders []string
		for _, o := range q.OrderBy {
			orders = append(orders, Sprintf("%s %s", o.Column(), o.Dir()))
		}
		orderBySQL = " ORDER BY " + Convert(orders).Join(", ").String()
	}

	limitSQL := ""
	if q.Limit > 0 {
		limitSQL = Sprintf(" LIMIT %d", q.Limit)
	}

	offsetSQL := ""
	if q.Offset > 0 {
		offsetSQL = Sprintf(" OFFSET %d", q.Offset)
	}

	sql := Sprintf("SELECT %s FROM %s%s%s%s%s%s", cols, q.Table, whereSQL, groupBySQL, orderBySQL, limitSQL, offsetSQL)
	return sql, args, nil
}

func buildUpdate(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, Err("table name is required for update")
	}
	if len(q.Columns) == 0 {
		return "", nil, Err("columns are required for update")
	}

	var setClauses []string
	var args []any

	for i, col := range q.Columns {
		setClauses = append(setClauses, Sprintf("%s = ?", col))
		args = append(args, q.Values[i])
	}

	var whereClauses []string
	for i, c := range q.Conditions {
		clause := Sprintf("%s %s ?", c.Field(), c.Operator())
		if i > 0 {
			clause = Sprintf(" %s %s", c.Logic(), clause)
		}
		whereClauses = append(whereClauses, clause)
		args = append(args, c.Value())
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + Convert(whereClauses).Join("").String()
	}

	sql := Sprintf("UPDATE %s SET %s%s", q.Table, Convert(setClauses).Join(", ").String(), whereSQL)
	return sql, args, nil
}

func buildDelete(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, Err("table name is required for delete")
	}

	var whereClauses []string
	var args []any

	for i, c := range q.Conditions {
		clause := Sprintf("%s %s ?", c.Field(), c.Operator())
		if i > 0 {
			clause = Sprintf(" %s %s", c.Logic(), clause)
		}
		whereClauses = append(whereClauses, clause)
		args = append(args, c.Value())
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + Convert(whereClauses).Join("").String()
	}

	sql := Sprintf("DELETE FROM %s%s", q.Table, whereSQL)
	return sql, args, nil
}
