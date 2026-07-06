package sqlite_test

import "github.com/tinywasm/model"

import (
	"github.com/tinywasm/orm"
	"github.com/tinywasm/sqlite"
	"testing"
)

type SyncUser struct {
	ID   int
	Name string
	Age  int
}

func (u *SyncUser) ModelName() string { return "users" }
func (u *SyncUser) Schema() []model.Field {
	return []model.Field{
		{Name: "id", Type: model.FieldInt, DB: &model.FieldDB{PK: true, AutoInc: true}},
		{Name: "name", Type: model.FieldText},
		{Name: "age", Type: model.FieldInt},
	}
}
func (u *SyncUser) Pointers() []any { return []any{&u.ID, &u.Name, &u.Age} }

func TestRegistration(t *testing.T) {
	db, err := orm.Open("sqlite::memory:")
	if err != nil {
		t.Fatalf("failed to open via orm.Open: %v", err)
	}
	db.Close()
}

func TestErrNoRowsMapping(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open: %v", err)
	}
	defer db.Close()

	db.CreateTable(&SyncUser{})

	u := &SyncUser{}
	q := db.Query(u).Where("id").Eq(999)
	err = q.ReadOne()
	// orm.QB.ReadOne returns orm.ErrNotFound when it gets orm.ErrNoRows from Scan
	if err != orm.ErrNotFound {
		t.Errorf("expected orm.ErrNotFound, got %v", err)
	}
}

func TestTableColumnsIntrospection(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open: %v", err)
	}
	defer db.Close()

	db.CreateTable(&SyncUser{})

	exec := db.RawExecutor().(orm.TableIntrospector)
	cols, err := exec.TableColumns("users")
	if err != nil {
		t.Fatalf("TableColumns failed: %v", err)
	}

	expected := map[string]bool{"id": true, "name": true, "age": true}
	if len(cols) != len(expected) {
		t.Errorf("expected %d columns, got %d", len(expected), len(cols))
	}
	for _, c := range cols {
		if !expected[c] {
			t.Errorf("unexpected column: %s", c)
		}
	}
}

type SyncNewUser struct {
	ID   int
	Name string
	Age  int
	Bio  string
}

func (u *SyncNewUser) ModelName() string { return "users" }
func (u *SyncNewUser) Schema() []model.Field {
	return []model.Field{
		{Name: "id", Type: model.FieldInt, DB: &model.FieldDB{PK: true, AutoInc: true}},
		{Name: "name", Type: model.FieldText},
		{Name: "age", Type: model.FieldInt},
		{Name: "bio", Type: model.FieldText},
	}
}
func (u *SyncNewUser) Pointers() []any { return []any{&u.ID, &u.Name, &u.Age, &u.Bio} }

func TestSync(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open: %v", err)
	}
	defer db.Close()

	db.CreateTable(&SyncUser{})

	// Sync to SyncNewUser (adds 'bio')
	err = db.Sync(&SyncNewUser{})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	exec := db.RawExecutor().(orm.TableIntrospector)
	cols, _ := exec.TableColumns("users")
	foundBio := false
	for _, c := range cols {
		if c == "bio" {
			foundBio = true
			break
		}
	}
	if !foundBio {
		t.Errorf("expected 'bio' column after Sync, got %v", cols)
	}
}
