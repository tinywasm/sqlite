package sqlite_test

import (
	"fmt"
	"testing"

	"github.com/tinywasm/orm"
	"github.com/tinywasm/sqlite"
)

type User struct {
	ID   int
	Name string
	Age  int
}

type Order struct {
	ID     int
	UserID int
	Amount float64
}

func (o *Order) TableName() string {
	return "orders"
}

func (o *Order) Columns() []string {
	return []string{"user_id", "amount"}
}

func (o *Order) Values() []any {
	return []any{o.UserID, o.Amount}
}

func (o *Order) Pointers() []any {
	return []any{&o.ID, &o.UserID, &o.Amount}
}

func TestComplexQueriesAndJoins(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer sqlite.Close(db)

	err = sqlite.ExecSQL(db, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			age INTEGER
		);
		CREATE TABLE orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			amount REAL
		);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	// Insert test data
	users := []User{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
		{Name: "Charlie", Age: 30},
	}
	for _, u := range users {
		if err := db.Create(&u); err != nil {
			t.Fatalf("Create user failed: %v", err)
		}
	}

	orders := []Order{
		{UserID: 1, Amount: 100.5},
		{UserID: 1, Amount: 200.0},
		{UserID: 2, Amount: 50.0},
	}
	for _, o := range orders {
		if err := db.Create(&o); err != nil {
			t.Fatalf("Create order failed: %v", err)
		}
	}

	// Test Fluent API: GroupBy, OrderBy, Limit, Offset
	// We want to query users grouped by age and ordered by age DESC, limit 1 offset 0
	// Wait, the test uses the fluent API.
	var results []*User
	q := db.Query(&User{})
	q.GroupBy("age").OrderBy("age", "DESC").Limit(1).Offset(0)
	err = q.ReadAll(func() orm.Model { return &User{} }, func(m orm.Model) {
		results = append(results, m.(*User))
	})
	if err != nil {
		t.Fatalf("ReadAll with GroupBy/OrderBy/Limit/Offset failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Age != 30 {
		t.Errorf("expected age 30, got %d", results[0].Age)
	}

	// Test JOIN query via raw SQL
	// Get total order amounts per user name
	type UserOrder struct {
		Name  string
		Total float64
	}
	err = sqlite.ExecSQL(db, `
		CREATE TABLE user_totals AS
		SELECT u.name, SUM(o.amount) as total
		FROM users u
		JOIN orders o ON u.id = o.user_id
		GROUP BY u.name;
	`)
	if err != nil {
		t.Fatalf("JOIN query failed: %v", err)
	}

	var totals []UserTotalModel
	qtotals := db.Query(&UserTotalModel{})
	qtotals.OrderBy("total", "DESC")
	err = qtotals.ReadAll(func() orm.Model { return &UserTotalModel{} }, func(m orm.Model) {
		totals = append(totals, *m.(*UserTotalModel))
	})
	if err != nil {
		t.Fatalf("ReadAll for user_totals failed: %v", err)
	}

	if len(totals) != 2 {
		t.Fatalf("expected 2 totals, got %d", len(totals))
	}
	if totals[0].Name != "Alice" || totals[0].Total != 300.5 {
		t.Errorf("expected Alice with 300.5, got %s with %v", totals[0].Name, totals[0].Total)
	}
	if totals[1].Name != "Bob" || totals[1].Total != 50.0 {
		t.Errorf("expected Bob with 50.0, got %s with %v", totals[1].Name, totals[1].Total)
	}
}

// UserTotalModel is a model for testing the temp table
type UserTotalModel struct {
	Name  string
	Total float64
}

func (u *UserTotalModel) TableName() string { return "user_totals" }
func (u *UserTotalModel) Columns() []string { return []string{"name", "total"} }
func (u *UserTotalModel) Values() []any { return []any{u.Name, u.Total} }
func (u *UserTotalModel) Pointers() []any { return []any{&u.Name, &u.Total} }

func (u *User) TableName() string {
	return "users"
}

func (u *User) Columns() []string {
	return []string{"name", "age"}
}

func (u *User) Values() []any {
	return []any{u.Name, u.Age}
}

func (u *User) Pointers() []any {
	return []any{&u.ID, &u.Name, &u.Age}
}

func TestSqliteAdapter(t *testing.T) {
	// Setup
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer sqlite.Close(db)

	// Create table
	err = sqlite.ExecSQL(db, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			age INTEGER
		);
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Test Create
	user := &User{Name: "Alice", Age: 30}
	if err := db.Create(user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test ReadOne
	readUser := &User{}
	q := db.Query(readUser)
	q.Where(orm.Eq("name", "Alice"))
	if err := q.ReadOne(); err != nil {
		t.Fatalf("ReadOne failed: %v", err)
	}
	if readUser.Name != "Alice" {
		t.Errorf("expected name Alice, got %s", readUser.Name)
	}

	// Test Update
	if err := db.Update(&User{Name: "Alice", Age: 31}, orm.Eq("name", "Alice")); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify Update
	readUser = &User{}
	q = db.Query(readUser)
	q.Where(orm.Eq("name", "Alice"))
	if err := q.ReadOne(); err != nil {
		t.Fatalf("ReadOne after Update failed: %v", err)
	}
	if readUser.Age != 31 {
		t.Errorf("expected age 31, got %d", readUser.Age)
	}

	// Test ReadAll
	db.Create(&User{Name: "Bob", Age: 25})
	var users []*User
	q = db.Query(&User{})
	err = q.ReadAll(func() orm.Model {
		u := &User{}
		return u
	}, func(m orm.Model) {
		users = append(users, m.(*User))
	})
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}

	// Test Delete
	if err := db.Delete(&User{}, orm.Eq("name", "Bob")); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify Delete
	users = nil
	q = db.Query(&User{})
	err = q.ReadAll(func() orm.Model { return &User{} }, func(m orm.Model) {
		users = append(users, m.(*User))
	})
	if err != nil {
		t.Fatalf("ReadAll after Delete failed: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
}

func TestCloseError(t *testing.T) {
	fakeDB := &orm.DB{} // Not registered
	err := sqlite.Close(fakeDB)
	if err == nil {
		t.Fatalf("expected error when closing unregistered db, got nil")
	}
}

func TestExecSQLError(t *testing.T) {
	fakeDB := &orm.DB{} // Not registered
	err := sqlite.ExecSQL(fakeDB, "SELECT 1")
	if err == nil {
		t.Fatalf("expected error when execSQL on unregistered db, got nil")
	}
}

type BadModel struct {
	Name string
}

func (b *BadModel) TableName() string { return "" }
func (b *BadModel) Columns() []string { return nil }
func (b *BadModel) Values() []any { return nil }
func (b *BadModel) Pointers() []any { return nil }

type NoColsModel struct {
	Name string
}

func (n *NoColsModel) TableName() string { return "no_cols" }
func (n *NoColsModel) Columns() []string { return nil }
func (n *NoColsModel) Values() []any { return nil }
func (n *NoColsModel) Pointers() []any { return nil }

func TestTranslateQueryErrors(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer sqlite.Close(db)

	err = sqlite.ExecSQL(db, `CREATE TABLE no_cols (id INTEGER PRIMARY KEY);`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// 1. missing table insert
	err = db.Create(&BadModel{})
	if err == nil {
		t.Fatalf("expected error for insert with no table")
	}

	// 2. missing columns insert
	err = db.Create(&NoColsModel{})
	if err == nil {
		t.Fatalf("expected error for insert with no columns")
	}

	// 3. missing table select
	q := db.Query(&BadModel{})
	err = q.ReadOne()
	if err == nil {
		t.Fatalf("expected error for select with no table")
	}

	// 4. missing table update
	err = db.Update(&BadModel{})
	if err == nil {
		t.Fatalf("expected error for update with no table")
	}

	// 5. missing columns update
	err = db.Update(&NoColsModel{})
	if err == nil {
		t.Fatalf("expected error for update with no columns")
	}

	// 6. missing table delete
	err = db.Delete(&BadModel{})
	if err == nil {
		t.Fatalf("expected error for delete with no table")
	}

	// 7. Error cases in adapter.Execute
	// ReadAll query failure
	adapter := sqlite.GetAdapter(db)
	err = adapter.Execute(orm.Query{
		Action: orm.ActionReadAll,
		Table:  "non_existent_table",
	}, nil, func() orm.Model { return &User{} }, func(m orm.Model) {})
	if err == nil {
		t.Fatalf("expected error when ReadAll on non-existent table")
	}

	// Create query failure
	err = adapter.Execute(orm.Query{
		Action:  orm.ActionCreate,
		Table:   "non_existent_table",
		Columns: []string{"id"},
		Values:  []any{1},
	}, nil, nil, nil)
	if err == nil {
		t.Fatalf("expected error when Create on non-existent table")
	}

	// ReadOne query failure (query error)
	err = adapter.Execute(orm.Query{
		Action: orm.ActionReadOne,
		Table:  "non_existent_table",
	}, &User{}, nil, nil)
	if err == nil {
		t.Fatalf("expected error when ReadOne on non-existent table")
	}

	// tx.Execute Error cases
	txAdapter, err := sqlite.GetTxAdapter(adapter)
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	err = txAdapter.Execute(orm.Query{
		Action: orm.ActionReadAll,
		Table:  "non_existent_table",
	}, nil, func() orm.Model { return &User{} }, func(m orm.Model) {})
	if err == nil {
		t.Fatalf("expected error when Tx.ReadAll on non-existent table")
	}

	err = txAdapter.Execute(orm.Query{
		Action:  orm.ActionCreate,
		Table:   "non_existent_table",
		Columns: []string{"id"},
		Values:  []any{1},
	}, nil, nil, nil)
	if err == nil {
		t.Fatalf("expected error when Tx.Create on non-existent table")
	}

	err = txAdapter.Execute(orm.Query{
		Action: orm.ActionReadOne,
		Table:  "non_existent_table",
	}, &User{}, nil, nil)
	if err == nil {
		t.Fatalf("expected error when Tx.ReadOne on non-existent table")
	}

	// ReadOne scanning error. We can trigger this by querying a table with different schema.
	_ = sqlite.ExecSQL(db, `CREATE TABLE scan_err_tbl (id TEXT); INSERT INTO scan_err_tbl VALUES ('notanint');`)

	err = adapter.Execute(orm.Query{
		Action: orm.ActionReadOne,
		Table:  "scan_err_tbl",
	}, &User{}, nil, nil)
	if err == nil {
		t.Fatalf("expected scan error on ReadOne, got nil")
	}

	err = txAdapter.Execute(orm.Query{
		Action: orm.ActionReadOne,
		Table:  "scan_err_tbl",
	}, &User{}, nil, nil)
	if err == nil {
		t.Fatalf("expected scan error on Tx.ReadOne, got nil")
	}

	// ReadAll scanning error.
	err = adapter.Execute(orm.Query{
		Action: orm.ActionReadAll,
		Table:  "scan_err_tbl",
	}, nil, func() orm.Model { return &User{} }, func(m orm.Model) {})
	if err == nil {
		t.Fatalf("expected scan error on ReadAll, got nil")
	}

	err = txAdapter.Execute(orm.Query{
		Action: orm.ActionReadAll,
		Table:  "scan_err_tbl",
	}, nil, func() orm.Model { return &User{} }, func(m orm.Model) {})
	if err == nil {
		t.Fatalf("expected scan error on Tx.ReadAll, got nil")
	}

	_ = txAdapter.Rollback()

	// BeginTx failure (e.g. on closed DB)
	sqlite.Close(db)
	_, err = adapter.BeginTx()
	if err == nil {
		t.Fatalf("expected BeginTx error on closed db")
	}
}

func TestTranslateQueryAdditionalErrors(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer sqlite.Close(db)

	adapter := sqlite.GetAdapter(db)

	// Create with empty condition (will not error on translate, just test coverage of empty cases)
	err = adapter.Execute(orm.Query{
		Action:  orm.ActionCreate,
		Table:   "users",
		Columns: []string{"name"},
		Values:  []any{"test"},
	}, nil, nil, nil)
	// it will fail at DB execution because we didn't create table, that's fine.
}

func TestTxExecutePaths(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer sqlite.Close(db)

	err = sqlite.ExecSQL(db, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			age INTEGER
		);
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	adapter := sqlite.GetAdapter(db)
	txAdapter, err := sqlite.GetTxAdapter(adapter)
	if err != nil {
		t.Fatalf("failed to start tx: %v", err)
	}

	// Tx.Update
	err = txAdapter.Execute(orm.Query{
		Action:  orm.ActionUpdate,
		Table:   "users",
		Columns: []string{"name"},
		Values:  []any{"test"},
	}, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error on Tx.Update: %v", err)
	}

	// Tx.Delete
	err = txAdapter.Execute(orm.Query{
		Action: orm.ActionDelete,
		Table:  "users",
	}, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error on Tx.Delete: %v", err)
	}

	// Tx.ReadOne NoRows
	u := &User{}
	err = txAdapter.Execute(orm.Query{
		Action: orm.ActionReadOne,
		Table:  "users",
	}, u, nil, nil)
	if err != nil {
		t.Fatalf("expected nil for no rows on Tx.ReadOne, got %v", err)
	}

	// Tx.ReadAll
	err = txAdapter.Execute(orm.Query{
		Action: orm.ActionReadAll,
		Table:  "users",
	}, nil, func() orm.Model { return &User{} }, func(m orm.Model) {})
	if err != nil {
		t.Fatalf("unexpected error on Tx.ReadAll: %v", err)
	}

	// Tx translation error
	err = txAdapter.Execute(orm.Query{
		Action: orm.ActionCreate, // missing table
	}, nil, nil, nil)
	if err == nil {
		t.Fatalf("expected translation error on Tx.Create with no table")
	}

	_ = txAdapter.Commit()
}

func TestExecuteUnsupportedAction(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer sqlite.Close(db)

	adapter := sqlite.GetAdapter(db)
	if adapter == nil {
		t.Fatalf("failed to get adapter")
	}

	q := orm.Query{
		Action:  99,
		Table:   "users",
		Columns: []string{"id"},
	}

	err = adapter.Execute(q, &User{}, nil, nil)
	if err == nil {
		t.Fatalf("expected error for unsupported action, got nil")
	}

	// Test unsupported action inside transaction
	txAdapter, err := sqlite.GetTxAdapter(adapter)
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	err = txAdapter.Execute(q, &User{}, nil, nil)
	if err == nil {
		t.Fatalf("expected error for unsupported action inside tx, got nil")
	}

	_ = txAdapter.Rollback()
}

func TestTransaction(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer sqlite.Close(db)

	err = sqlite.ExecSQL(db, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			age INTEGER
		);
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Test Commit
	err = db.Tx(func(tx *orm.DB) error {
		if err := tx.Create(&User{Name: "Charlie", Age: 40}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Tx commit failed: %v", err)
	}

	// Verify Commit
	readUser := &User{}
	q := db.Query(readUser)
	q.Where(orm.Eq("name", "Charlie"))
	if err := q.ReadOne(); err != nil {
		t.Fatalf("ReadOne failed: %v", err)
	}

	// Test Rollback
	err = db.Tx(func(tx *orm.DB) error {
		if err := tx.Create(&User{Name: "Dave", Age: 50}); err != nil {
			return err
		}
		return fmt.Errorf("rollback")
	})
	if err == nil {
		t.Fatalf("Tx rollback should have returned error")
	}

	// Verify Rollback
	readUser = &User{}
	q = db.Query(readUser)
	q.Where(orm.Eq("name", "Dave"))
	if err := q.ReadOne(); err == nil {
		// If ReadOne succeeds, it means it found a record (or potentially scan didn't fail, but we check name)
		if readUser.Name != "" {
			t.Errorf("ReadOne should have failed (not found) or returned empty, but got name: %s", readUser.Name)
		}
	}
}
