package sqlite

import (
	twfmt "github.com/tinywasm/fmt"

	"github.com/tinywasm/orm"
)

// translateQuery converts an orm.Query into a SQLite SQL string and arguments.
func translateQuery(q orm.Query, m twfmt.Model) (string, []any, error) {
	switch q.Action {
	case orm.ActionCreate:
		return buildInsert(q)
	case orm.ActionReadOne, orm.ActionReadAll:
		return buildSelect(q)
	case orm.ActionUpdate:
		return buildUpdate(q)
	case orm.ActionDelete:
		return buildDelete(q)
	case orm.ActionCreateTable:
		return buildCreateTable(q, m)
	case orm.ActionDropTable:
		return buildDropTable(q)
	default:
		return "", nil, twfmt.Errf("unknown query action: %v", q.Action)
	}
}

func buildCreateTable(q orm.Query, m twfmt.Model) (string, []any, error) {
	if m == nil {
		return "", nil, twfmt.Err("model is required for create table")
	}
	if q.Table == "" {
		return "", nil, twfmt.Err("table name is required for create table")
	}

	var sb []string
	sb = append(sb, twfmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", q.Table))

	fields := m.Schema()

	// Count composite PK fields upfront to decide between inline and table-level PK.
	var pkCols []string
	for _, f := range fields {
		if f.PK {
			pkCols = append(pkCols, f.Name)
		}
	}
	compositePK := len(pkCols) > 1

	var cols []string
	for _, f := range fields {
		col := twfmt.Sprintf("%s %s", f.Name, sqliteType(f.Type))
		if f.PK {
			if compositePK {
				// Composite PK: columns must be NOT NULL; constraint emitted as table-level below.
				col += " NOT NULL"
			} else {
				col += " PRIMARY KEY"
				// AUTOINCREMENT is only allowed on INTEGER PRIMARY KEY in SQLite
				if f.AutoInc && f.Type == twfmt.FieldInt {
					col += " AUTOINCREMENT"
				}
			}
		}
		if f.NotNull {
			col += " NOT NULL"
		}
		if f.Unique {
			col += " UNIQUE"
		}
		cols = append(cols, col)
	}

	if compositePK {
		cols = append(cols, twfmt.Sprintf("PRIMARY KEY (%s)", twfmt.Convert(pkCols).Join(", ").String()))
	}

	if ext, ok := m.(interface{ SchemaExt() []orm.FieldExt }); ok {
		for _, f := range ext.SchemaExt() {
			if f.Ref != "" {
				refCol := f.RefColumn
				if refCol == "" {
					refCol = "id"
				}
				cols = append(cols, twfmt.Sprintf("CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(%s)", q.Table, f.Name, f.Name, f.Ref, refCol))
			}
		}
	}

	sb = append(sb, twfmt.Convert(cols).Join(", ").String())
	sb = append(sb, ")")

	return twfmt.Convert(sb).Join("").String(), nil, nil
}

func buildDropTable(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, twfmt.Err("table name is required for drop table")
	}
	return twfmt.Sprintf("DROP TABLE IF EXISTS %s", q.Table), nil, nil
}

func sqliteType(t twfmt.FieldType) string {
	switch t {
	case twfmt.FieldInt:
		return "INTEGER"
	case twfmt.FieldFloat:
		return "REAL"
	case twfmt.FieldBool:
		return "INTEGER" // 0 or 1
	case twfmt.FieldBlob:
		return "BLOB"
	default:
		return "TEXT"
	}
}

func buildInsert(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, twfmt.Err("table name is required for insert")
	}
	if len(q.Columns) == 0 {
		return "", nil, twfmt.Err("columns are required for insert")
	}

	cols := twfmt.Convert(q.Columns).Join(", ").String()
	placeholders := make([]string, len(q.Columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	vals := twfmt.Convert(placeholders).Join(", ").String()

	sql := twfmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", q.Table, cols, vals)
	return sql, q.Values, nil
}

func buildSelect(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, twfmt.Err("table name is required for select")
	}
	cols := "*"

	whereSQL, args, err := buildConditions(q.Conditions)
	if err != nil {
		return "", nil, err
	}

	groupBySQL := ""
	if len(q.GroupBy) > 0 {
		groupBySQL = " GROUP BY " + twfmt.Convert(q.GroupBy).Join(", ").String()
	}

	orderBySQL := ""
	if len(q.OrderBy) > 0 {
		var orders []string
		for _, o := range q.OrderBy {
			orders = append(orders, twfmt.Sprintf("%s %s", o.Column(), o.Dir()))
		}
		orderBySQL = " ORDER BY " + twfmt.Convert(orders).Join(", ").String()
	}

	limitSQL := ""
	if q.Limit > 0 {
		limitSQL = twfmt.Sprintf(" LIMIT %d", q.Limit)
	}

	offsetSQL := ""
	if q.Offset > 0 {
		offsetSQL = twfmt.Sprintf(" OFFSET %d", q.Offset)
	}

	sql := twfmt.Sprintf("SELECT %s FROM %s%s%s%s%s%s", cols, q.Table, whereSQL, groupBySQL, orderBySQL, limitSQL, offsetSQL)
	return sql, args, nil
}

func buildUpdate(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, twfmt.Err("table name is required for update")
	}
	if len(q.Columns) == 0 {
		return "", nil, twfmt.Err("columns are required for update")
	}

	var setClauses []string
	var args []any

	for i, col := range q.Columns {
		setClauses = append(setClauses, twfmt.Sprintf("%s = ?", col))
		args = append(args, q.Values[i])
	}

	whereSQL, condArgs, err := buildConditions(q.Conditions)
	if err != nil {
		return "", nil, err
	}
	args = append(args, condArgs...)

	sql := twfmt.Sprintf("UPDATE %s SET %s%s", q.Table, twfmt.Convert(setClauses).Join(", ").String(), whereSQL)
	return sql, args, nil
}

func buildDelete(q orm.Query) (string, []any, error) {
	if q.Table == "" {
		return "", nil, twfmt.Err("table name is required for delete")
	}

	whereSQL, args, err := buildConditions(q.Conditions)
	if err != nil {
		return "", nil, err
	}

	sql := twfmt.Sprintf("DELETE FROM %s%s", q.Table, whereSQL)
	return sql, args, nil
}

func buildConditions(conditions []orm.Condition) (string, []any, error) {
	if len(conditions) == 0 {
		return "", nil, nil
	}

	var whereClauses []string
	var args []any

	for i, c := range conditions {
		var clause string
		if c.Operator() == "IN" {
			slice, ok := c.Value().([]any)
			if !ok {
				return "", nil, twfmt.Errf("IN operator requires []any value, got %T", c.Value())
			}
			if len(slice) == 0 {
				return "", nil, twfmt.Err("IN operator slice cannot be empty")
			}
			placeholders := make([]string, len(slice))
			for j := range placeholders {
				placeholders[j] = "?"
			}
			inVals := twfmt.Convert(placeholders).Join(", ").String()
			clause = twfmt.Sprintf("%s IN (%s)", c.Field(), inVals)
			args = append(args, slice...)
		} else {
			clause = twfmt.Sprintf("%s %s ?", c.Field(), c.Operator())
			args = append(args, c.Value())
		}

		if i > 0 {
			clause = twfmt.Sprintf(" %s %s", c.Logic(), clause)
		}
		whereClauses = append(whereClauses, clause)
	}

	return " WHERE " + twfmt.Convert(whereClauses).Join("").String(), args, nil
}
