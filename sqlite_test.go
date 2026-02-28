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
	q.GroupBy("age").OrderBy("age").Desc().Limit(1).Offset(0)
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
	qtotals.OrderBy("total").Desc()
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
func (u *UserTotalModel) Values() []any     { return []any{u.Name, u.Total} }
func (u *UserTotalModel) Pointers() []any   { return []any{&u.Name, &u.Total} }

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
	q.Where("name").Eq("Alice")
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
	q.Where("name").Eq("Alice")
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
func (b *BadModel) Values() []any     { return nil }
func (b *BadModel) Pointers() []any   { return nil }

type NoColsModel struct {
	Name string
}

func (n *NoColsModel) TableName() string { return "no_cols" }
func (n *NoColsModel) Columns() []string { return nil }
func (n *NoColsModel) Values() []any     { return nil }
func (n *NoColsModel) Pointers() []any   { return nil }

func TestCompilerErrors(t *testing.T) {
	// Test internal translateQuery default switch case
	_, _, err := sqlite.ExportTranslateQuery(orm.Query{Action: 99})
	if err == nil {
		t.Fatalf("expected error for unsupported action in translate")
	}

	// Test insert without table name
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionCreate, Columns: []string{"id"}})
	if err == nil {
		t.Fatalf("expected error for insert without table")
	}

	// Test insert without columns
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionCreate, Table: "t"})
	if err == nil {
		t.Fatalf("expected error for insert without columns")
	}

	// Test select without table
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionReadOne})
	if err == nil {
		t.Fatalf("expected error for select without table")
	}

	// Test update without table
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionUpdate, Columns: []string{"id"}})
	if err == nil {
		t.Fatalf("expected error for update without table")
	}

	// Test update without columns
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionUpdate, Table: "t"})
	if err == nil {
		t.Fatalf("expected error for update without columns")
	}

	// Test delete without table
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionDelete})
	if err == nil {
		t.Fatalf("expected error for delete without table")
	}
}

func TestExecutorErrors(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer sqlite.Close(db)

	exec := sqlite.GetExecutor(db)

	// Test Exec error
	err = exec.Exec("INVALID SQL")
	if err == nil {
		t.Fatalf("expected error on invalid sql format")
	}

	// Test QueryRow error
	row := exec.QueryRow("SELECT * FROM non_existent")
	err = row.Scan()
	if err == nil {
		t.Fatalf("expected error scanning from invalid table")
	}

	// Test Query error
	_, err = exec.Query("SELECT * FROM non_existent")
	if err == nil {
		t.Fatalf("expected error querying invalid table")
	}

	// BeginTx failure
	sqlDB := sqlite.GetSqlDB(db)
	sqlDB.Close() // Force BeginTx to fail

	txExec, ok := exec.(orm.TxExecutor)
	if !ok {
		t.Fatalf("executor does not implement TxExecutor")
	}
	_, err = txExec.BeginTx()
	if err == nil {
		t.Fatalf("expected BeginTx error on closed DB")
	}
}

func TestTxExecutorErrors(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer sqlite.Close(db)

	txExec, err := sqlite.GetTxExecutor(db)
	if err != nil {
		t.Fatalf("failed to open tx: %v", err)
	}

	// Test Exec
	err = txExec.Exec("INVALID SQL")
	if err == nil {
		t.Fatalf("expected error on invalid tx sql")
	}

	// Test QueryRow
	row := txExec.QueryRow("SELECT * FROM non_existent")
	err = row.Scan()
	if err == nil {
		t.Fatalf("expected error scanning tx invalid table")
	}

	// Test Query
	_, err = txExec.Query("SELECT * FROM non_existent")
	if err == nil {
		t.Fatalf("expected error querying tx invalid table")
	}

	// Rollback
	err = txExec.Rollback()
	if err != nil {
		t.Fatalf("failed rollback: %v", err)
	}

	// Commit
	txExec2, _ := sqlite.GetTxExecutor(db)
	err = txExec2.Commit()
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}
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
	q.Where("name").Eq("Charlie")
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
	q.Where("name").Eq("Dave")
	if err := q.ReadOne(); err == nil {
		// If ReadOne succeeds, it means it found a record (or potentially scan didn't fail, but we check name)
		if readUser.Name != "" {
			t.Errorf("ReadOne should have failed (not found) or returned empty, but got name: %s", readUser.Name)
		}
	}
}
