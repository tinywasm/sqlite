package sqlite

import (
"testing"
"github.com/tinywasm/orm"
)

func TestInOperator(t *testing.T) {
q := orm.Query{
Action: orm.ActionReadAll,
Table: "users",
Conditions: []orm.Condition{
orm.In("id", []any{1, 2, 3}),
},
}
sql, args, err := translateQuery(q)
if err != nil {
t.Fatalf("translate error: %v", err)
}
if sql != "SELECT * FROM users WHERE id IN (?, ?, ?)" {
t.Errorf("wrong sql: %s", sql)
}
if len(args) != 3 {
t.Errorf("wrong args len: %d", len(args))
}
}
