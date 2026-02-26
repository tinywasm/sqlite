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
	db, err := sqlite.New(":memory:")
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

func TestTransaction(t *testing.T) {
	db, err := sqlite.New(":memory:")
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
