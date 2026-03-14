package sqlite_test

import (
	"testing"

	twfmt "github.com/tinywasm/fmt"
	"github.com/tinywasm/orm"
	"github.com/tinywasm/sqlite"
)

type SimpleUser struct {
	ID    string
	Email string
}

func (s *SimpleUser) TableName() string { return "simple_users" }
func (s *SimpleUser) Schema() []twfmt.Field {
	return []twfmt.Field{
		{Name: "id", Type: twfmt.FieldText, PK: true},
		{Name: "email", Type: twfmt.FieldText, Unique: true},
	}
}
func (s *SimpleUser) Pointers() []any { return []any{&s.ID, &s.Email} }

type SimpleSession struct {
	ID     string
	UserID string
}

func (s *SimpleSession) TableName() string { return "simple_sessions" }
func (s *SimpleSession) Schema() []twfmt.Field {
	return []twfmt.Field{
		{Name: "id", Type: twfmt.FieldText, PK: true},
		{Name: "user_id", Type: twfmt.FieldText},
	}
}
func (s *SimpleSession) SchemaExt() []orm.FieldExt {
	return []orm.FieldExt{
		{Field: twfmt.Field{Name: "user_id", Type: twfmt.FieldText}, Ref: "simple_users", RefColumn: "id"},
	}
}
func (s *SimpleSession) Pointers() []any { return []any{&s.ID, &s.UserID} }

func TestJulesScenario(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer sqlite.Close(db)

	// Create SimpleUser table
	err = db.CreateTable(&SimpleUser{})
	if err != nil {
		t.Fatalf("failed to create SimpleUser table: %v", err)
	}

	// Calling CreateTable twice should return success (IF NOT EXISTS)
	err = db.CreateTable(&SimpleUser{})
	if err != nil {
		t.Fatalf("failed to create SimpleUser table (second time): %v", err)
	}

	// Insert into SimpleUser
	user := &SimpleUser{ID: "user_123", Email: "test@example.com"}
	err = db.Create(user)
	if err != nil {
		t.Fatalf("failed to create simple user record: %v", err)
	}

	// Read from SimpleUser
	var readUser SimpleUser
	q := db.Query(&readUser)
	q.Where("id").Eq("user_123")
	err = q.ReadOne()
	if err != nil {
		t.Fatalf("failed to read simple user record: %v", err)
	}
	if readUser.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", readUser.Email)
	}

	// Create SimpleSession table
	err = db.CreateTable(&SimpleSession{})
	if err != nil {
		t.Fatalf("failed to create SimpleSession table: %v", err)
	}

	// Insert into SimpleSession
	session := &SimpleSession{ID: "sess_abc", UserID: "user_123"}
	err = db.Create(session)
	if err != nil {
		t.Fatalf("failed to create simple session record: %v", err)
	}
}
