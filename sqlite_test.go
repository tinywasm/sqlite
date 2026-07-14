package sqlite_test

import (
	"testing"

	"github.com/tinywasm/ddlc"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/model"
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

func (o *Order) ModelName() string {
	return "orders"
}

func (o *Order) Schema() []model.Field {
	return []model.Field{
		{Name: "id", Type: model.Int(), DB: &model.FieldDB{PK: true, AutoInc: true}},
		{Name: "user_id", Type: model.Int()},
		{Name: "amount", Type: model.Float()},
	}
}

func (o *Order) SchemaExt() []ddlc.FieldExt {
	return []ddlc.FieldExt{
		{Field: model.Field{Name: "user_id", Type: model.Int()}, Ref: "users", RefColumn: "id"},
	}
}

func (o *Order) Pointers() []any {
	return []any{&o.ID, &o.UserID, &o.Amount}
}

// IsNil, EncodeFields, DecodeFields satisfy model.Model. This fixture only exercises the
// sqlite driver (Create/Query/ReadAll); it never travels over the wire.
func (o *Order) IsNil() bool                      { return o == nil }
func (o *Order) EncodeFields(w model.FieldWriter) {}
func (o *Order) DecodeFields(r model.FieldReader) {}

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
	var results []*User
	q := db.Query(&User{})
	q.GroupBy("age").OrderBy("age").Desc().Limit(1).Offset(0)
	err = q.ReadAll(func() model.Model { return &User{} }, func(m model.Model) {
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
	err = qtotals.ReadAll(func() model.Model { return &UserTotalModel{} }, func(m model.Model) {
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

func (u *UserTotalModel) ModelName() string { return "user_totals" }
func (u *UserTotalModel) Schema() []model.Field {
	return []model.Field{
		{Name: "name", Type: model.Text()},
		{Name: "total", Type: model.Float()},
	}
}
func (u *UserTotalModel) Pointers() []any { return []any{&u.Name, &u.Total} }

// IsNil, EncodeFields, DecodeFields satisfy model.Model. This fixture only exercises the
// sqlite driver (Query/ReadAll against a temp table); it never travels over the wire.
func (u *UserTotalModel) IsNil() bool                      { return u == nil }
func (u *UserTotalModel) EncodeFields(w model.FieldWriter) {}
func (u *UserTotalModel) DecodeFields(r model.FieldReader) {}

func (u *User) ModelName() string {
	return "users"
}

func (u *User) Schema() []model.Field {
	return []model.Field{
		{Name: "id", Type: model.Int(), DB: &model.FieldDB{PK: true, AutoInc: true}},
		{Name: "name", Type: model.Text()},
		{Name: "age", Type: model.Int()},
	}
}

func (u *User) Pointers() []any {
	return []any{&u.ID, &u.Name, &u.Age}
}

// IsNil, EncodeFields, DecodeFields satisfy model.Model. This fixture only exercises the
// sqlite driver (Create/Query/ReadAll); it never travels over the wire.
func (u *User) IsNil() bool                      { return u == nil }
func (u *User) EncodeFields(w model.FieldWriter) {}
func (u *User) DecodeFields(r model.FieldReader) {}

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
	err = q.ReadAll(func() model.Model {
		u := &User{}
		return u
	}, func(m model.Model) {
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
	err = qIn.ReadAll(func() model.Model { return &User{} }, func(m model.Model) {
		inUsers = append(inUsers, m.(*User))
	})
	if err != nil {
		t.Fatalf("IN ReadAll failed: %v", err)
	}
	if len(inUsers) != 2 {
		t.Errorf("expected 2 users from IN, got %d", len(inUsers))
	}

	// Test Delete
	if err := db.Delete(&User{}, orm.Eq("name", "Bob")); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify Delete
	users = nil
	q = db.Query(&User{})
	err = q.ReadAll(func() model.Model { return &User{} }, func(m model.Model) {
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
	return fmt.Err("close error")
}

func (e *errorExecutor) Exec(query string, args ...any) error {
	return fmt.Err("exec error")
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

func (b *BadModel) ModelName() string     { return "" }
func (b *BadModel) Schema() []model.Field { return nil }
func (b *BadModel) Pointers() []any       { return nil }

type NoColsModel struct {
	Name string
}

func (n *NoColsModel) ModelName() string     { return "no_cols" }
func (n *NoColsModel) Schema() []model.Field { return nil }
func (n *NoColsModel) Pointers() []any       { return nil }

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
		return fmt.Err("rollback")
	})
	if err == nil {
		t.Fatalf("Tx rollback should have returned error")
	}

	// Verify Rollback
	readUser = &User{}
	q = db.Query(readUser)
	q.Where("name").Eq("Dave")
	if err := q.ReadOne(); err == nil {
		if readUser.Name != "" {
			t.Errorf("ReadOne should have failed (not found) or returned empty, but got name: %s", readUser.Name)
		}
	}
}

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
		func() model.Model { return &User{} },
		func(m model.Model) { users = append(users, m.(*User)) },
	)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}

	// Update each user's age in a loop using an explicit PK condition.
	err = db.Tx(func(tx *orm.DB) error {
		for _, u := range users {
			u.Age += 10
			if err := tx.Update(u, orm.Eq("id", u.ID)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
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

func testSchemaInspector(t *testing.T, inspector orm.SchemaInspector) {
	// Test Tables()
	tables, err := inspector.Tables()
	if err != nil {
		t.Fatalf("Tables() failed: %v", err)
	}

	// sqlite_master might contain other things, but our query filters sqlite_%
	expectedTables := map[string]bool{"users": true, "orders": true}
	foundCount := 0
	for _, table := range tables {
		if expectedTables[table] {
			foundCount++
		}
	}
	if foundCount != 2 {
		t.Errorf("expected 2 user tables, got %d from %v", foundCount, tables)
	}

	// Test Columns() for 'users'
	cols, err := inspector.Columns("users")
	if err != nil {
		t.Fatalf("Columns('users') failed: %v", err)
	}

	if len(cols) != 3 {
		t.Fatalf("expected 3 columns for 'users', got %d", len(cols))
	}

	colMap := make(map[string]orm.ColumnInfo)
	for _, col := range cols {
		colMap[col.Name] = col
	}

	idCol, ok := colMap["id"]
	if !ok {
		t.Errorf("column 'id' not found")
	} else {
		if !idCol.PK {
			t.Errorf("expected 'id' to be PK")
		}
		// SQLite type might be INTEGER or INT depending on how it was created
		if idCol.Type != "INTEGER" {
			t.Errorf("expected type INTEGER, got %s", idCol.Type)
		}
	}

	nameCol, ok := colMap["name"]
	if !ok {
		t.Errorf("column 'name' not found")
	} else {
		if !nameCol.NotNull {
			t.Errorf("expected 'name' to be NOT NULL")
		}
		if nameCol.Type != "TEXT" {
			t.Errorf("expected type TEXT, got %s", nameCol.Type)
		}
	}

	ageCol, ok := colMap["age"]
	if !ok {
		t.Errorf("column 'age' not found")
	} else {
		if ageCol.NotNull {
			t.Errorf("expected 'age' to be nullable")
		}
	}
}

func TestSchemaInspector(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer sqlite.Close(db)

	err = sqlite.ExecSQL(db, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
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

	t.Run("Executor", func(t *testing.T) {
		exec := sqlite.GetExecutor(db)
		inspector, ok := exec.(orm.SchemaInspector)
		if !ok {
			t.Fatalf("executor does not implement orm.SchemaInspector")
		}
		testSchemaInspector(t, inspector)
	})

	t.Run("TxExecutor", func(t *testing.T) {
		txExec, err := sqlite.GetTxExecutor(db)
		if err != nil {
			t.Fatalf("failed to get tx executor: %v", err)
		}
		defer txExec.Rollback()

		inspector, ok := txExec.(orm.SchemaInspector)
		if !ok {
			t.Fatalf("tx executor does not implement orm.SchemaInspector")
		}
		testSchemaInspector(t, inspector)
	})
}
