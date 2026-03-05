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

func (o *Order) Schema() []orm.Field {
	return []orm.Field{
		{Name: "id", Type: orm.TypeInt64, Constraints: orm.ConstraintPK | orm.ConstraintAutoIncrement},
		{Name: "user_id", Type: orm.TypeInt64, Ref: "users", RefColumn: "id"},
		{Name: "amount", Type: orm.TypeFloat64},
	}
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
	users := []*User{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
		{Name: "Charlie", Age: 30},
	}
	for _, u := range users {
		if err := db.Create(u); err != nil {
			t.Fatalf("Create user failed: %v", err)
		}
	}

	orders := []*Order{
		{UserID: 1, Amount: 100.5},
		{UserID: 1, Amount: 200.0},
		{UserID: 2, Amount: 50.0},
	}
	for _, o := range orders {
		if err := db.Create(o); err != nil {
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
func (u *UserTotalModel) Schema() []orm.Field {
	return []orm.Field{
		{Name: "name", Type: orm.TypeText},
		{Name: "total", Type: orm.TypeFloat64},
	}
}
func (u *UserTotalModel) Values() []any     { return []any{u.Name, u.Total} }
func (u *UserTotalModel) Pointers() []any   { return []any{&u.Name, &u.Total} }

func (u *User) TableName() string {
	return "users"
}

func (u *User) Schema() []orm.Field {
	return []orm.Field{
		{Name: "id", Type: orm.TypeInt64, Constraints: orm.ConstraintPK | orm.ConstraintAutoIncrement},
		{Name: "name", Type: orm.TypeText},
		{Name: "age", Type: orm.TypeInt64},
	}
}

func (o *Order) Values() []any {
	if o.ID == 0 {
		return []any{nil, o.UserID, o.Amount}
	}
	return []any{o.ID, o.UserID, o.Amount}
}

func (u *User) Values() []any {
	if u.ID == 0 {
		return []any{nil, u.Name, u.Age}
	}
	return []any{u.ID, u.Name, u.Age}
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
	// Note: We provide ID because otherwise User.Values() returns nil for ID,
	// and SQLite strictly rejects updating INTEGER PRIMARY KEY to NULL.
	if err := db.Update(&User{ID: readUser.ID, Name: "Alice", Age: 31}, orm.Eq("name", "Alice")); err != nil {
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

	// Test IN operator
	var inUsers []*User
	qIn := db.Query(&User{})
	qIn.Where("name").In([]any{"Alice", "Bob"})
	err = qIn.ReadAll(func() orm.Model { return &User{} }, func(m orm.Model) {
		inUsers = append(inUsers, m.(*User))
	})
	if err != nil {
		t.Fatalf("IN ReadAll failed: %v", err)
	}
	if len(inUsers) != 2 {
		t.Errorf("expected 2 users from IN, got %d", len(inUsers))
	}

	// Test IN internal coverage format (slice of different types/missing)
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionReadAll, Table: "t", Conditions: []orm.Condition{orm.In("id", 1)}}, nil)
	if err == nil {
		t.Errorf("Expected compile error for non-slice IN value")
	}

	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionReadAll, Table: "t", Conditions: []orm.Condition{orm.In("id", []any{})}}, nil)
	if err == nil {
		t.Errorf("Expected compile error for empty slice IN value")
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

type errorExecutor struct {
	orm.Executor
}

func (e *errorExecutor) Close() error {
	return fmt.Errorf("close error")
}

func (e *errorExecutor) Exec(query string, args ...any) error {
	return fmt.Errorf("exec error")
}

func TestCloseError(t *testing.T) {
	fakeDB := orm.New(&errorExecutor{}, nil)
	err := sqlite.Close(fakeDB)
	if err == nil {
		t.Fatalf("expected error when closing db, got nil")
	}
}

func TestExecSQLError(t *testing.T) {
	fakeDB := orm.New(&errorExecutor{}, nil)
	err := sqlite.ExecSQL(fakeDB, "SELECT 1")
	if err == nil {
		t.Fatalf("expected error when execSQL fails, got nil")
	}
}

type BadModel struct {
	Name string
}

func (b *BadModel) TableName() string { return "" }
func (b *BadModel) Schema() []orm.Field { return nil }
func (b *BadModel) Values() []any     { return nil }
func (b *BadModel) Pointers() []any   { return nil }

type NoColsModel struct {
	Name string
}

func (n *NoColsModel) TableName() string { return "no_cols" }
func (n *NoColsModel) Schema() []orm.Field { return nil }
func (n *NoColsModel) Values() []any     { return nil }
func (n *NoColsModel) Pointers() []any   { return nil }

func TestCreateTable(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer sqlite.Close(db)

	err = db.CreateTable(&User{})
	if err != nil {
		t.Fatalf("CreateTable User failed: %v", err)
	}

	// Verify table exists by inserting into it
	user := &User{Name: "Alice", Age: 30}
	if err := db.Create(user); err != nil {
		t.Fatalf("Create User record failed after CreateTable: %v", err)
	}
}

func TestDropTable(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer sqlite.Close(db)

	err = db.CreateTable(&User{})
	if err != nil {
		t.Fatalf("CreateTable User failed: %v", err)
	}

	err = db.DropTable(&User{})
	if err != nil {
		t.Fatalf("DropTable User failed: %v", err)
	}

	// Verify table is gone by attempting to insert
	user := &User{Name: "Alice", Age: 30}
	err = db.Create(user)
	if err == nil {
		t.Fatalf("Expected error inserting into dropped table, got nil")
	}
}

func TestCreateTableWithFK(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer sqlite.Close(db)

	// First create referenced table
	err = db.CreateTable(&User{})
	if err != nil {
		t.Fatalf("CreateTable User failed: %v", err)
	}

	// Then create table with FK
	err = db.CreateTable(&Order{})
	if err != nil {
		t.Fatalf("CreateTable Order (with FK) failed: %v", err)
	}
}

func TestCompilerErrors(t *testing.T) {
	// Test internal translateQuery default switch case
	_, _, err := sqlite.ExportTranslateQuery(orm.Query{Action: 99}, nil)
	if err == nil {
		t.Fatalf("expected error for unsupported action in translate")
	}

	// Test create table without table
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionCreateTable}, &User{})
	if err == nil {
		t.Fatalf("expected error for create table without table")
	}

	// Test create table without model
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionCreateTable, Table: "t"}, nil)
	if err == nil {
		t.Fatalf("expected error for create table without model")
	}

	// Test drop table without table
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionDropTable}, nil)
	if err == nil {
		t.Fatalf("expected error for drop table without table")
	}

	// Test insert without table name
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionCreate, Columns: []string{"id"}}, nil)
	if err == nil {
		t.Fatalf("expected error for insert without table")
	}

	// Test insert without columns
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionCreate, Table: "t"}, nil)
	if err == nil {
		t.Fatalf("expected error for insert without columns")
	}

	// Test select without table
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionReadOne}, nil)
	if err == nil {
		t.Fatalf("expected error for select without table")
	}

	// Test update without table
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionUpdate, Columns: []string{"id"}}, nil)
	if err == nil {
		t.Fatalf("expected error for update without table")
	}

	// Test update without columns
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionUpdate, Table: "t"}, nil)
	if err == nil {
		t.Fatalf("expected error for update without columns")
	}

	// Test delete without table
	_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionDelete}, nil)
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

// TestUpdate_ExplicitPK_MultiRow is a regression test for the bug where
// db.Update(&model) without conditions caused full-table updates, triggering
// UNIQUE constraint failures in loops.
//
// The fix is in tinywasm/orm (mandatory first Condition). This test verifies
// the integration behaviour: N rows updated in sequence via explicit PK condition,
// each targeting exactly one row.
func TestUpdate_ExplicitPK_MultiRow(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlite.Close(db)

	if err := db.CreateTable(&User{}); err != nil {
		t.Fatalf("CreateTable: %v", err)
	}

	// Insert three users.
	seeds := []User{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
		{Name: "Charlie", Age: 20},
	}
	for i := range seeds {
		if err := db.Create(&seeds[i]); err != nil {
			t.Fatalf("Create %s: %v", seeds[i].Name, err)
		}
	}

	// Read them back to obtain DB-assigned IDs.
	var users []*User
	q := db.Query(&User{})
	err = q.ReadAll(
		func() orm.Model { return &User{} },
		func(m orm.Model) { users = append(users, m.(*User)) },
	)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}

	// Update each user's age in a loop using an explicit PK condition.
	// This is the pattern that previously triggered UNIQUE constraint failures.
	err = db.Tx(func(tx *orm.DB) error {
		for _, u := range users {
			u.Age += 10
			// Explicit condition required by tinywasm/orm (compile-time enforced).
			if err := tx.Update(u, orm.Eq("id", u.ID)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		// Regression: previously failed with "UNIQUE constraint failed: users.id"
		t.Fatalf("Tx multi-row Update: %v", err)
	}

	// Verify each row was updated independently.
	wantAges := map[int]int{users[0].ID: 40, users[1].ID: 35, users[2].ID: 30}
	for _, u := range users {
		got := &User{}
		q := db.Query(got)
		q.Where("id").Eq(u.ID)
		if err := q.ReadOne(); err != nil {
			t.Fatalf("ReadOne user %d: %v", u.ID, err)
		}
		if got.Age != wantAges[u.ID] {
			t.Errorf("user %d: expected age %d, got %d", u.ID, wantAges[u.ID], got.Age)
		}
	}
}
