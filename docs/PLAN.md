# PLAN — tinywasm/sqlite: Document Update/Delete Condition Contract

## Context

A critical data-safety bug was found: `db.Update(&model)` without conditions
generates a full-table UPDATE with no WHERE clause.

**The fix lives entirely in `tinywasm/orm`** (see `tinywasm/orm/docs/PLAN.md`).
After that fix, `Update(m, cond, rest...)` requires at least one `Condition`
at compile time — it is impossible to call with zero conditions.

**This library requires no code changes.** `buildUpdate` in `translate.go`
already handles conditions correctly. Since `orm` now guarantees
`q.Conditions` is never empty for an Update query, the bug cannot reach this layer.

This plan exists only to **update the documentation** to reflect the new contract
and add a regression test that verifies the contract at the integration level.

---

## Development Rules

- Standard Library only — no external dependencies.
- Max 500 lines per file.
- Run tests with `gotest` (never `go test` directly).
- Publish with `gopush 'message'` after all tests pass.
- **Documentation must be updated before touching code.**
- Prerequisites in isolated environments:
  ```bash
  go install github.com/tinywasm/devflow/cmd/gotest@latest
  ```

---

## Step 1 — Update `README.md`

In any usage examples showing `db.Update(...)`, ensure at least one explicit
condition is always present. Update any snippet that showed zero-arg Update.

Example of what to add to the README:

```markdown
## Update

`db.Update` always requires at least one `Condition`. This is enforced at
compile time by `tinywasm/orm`. There is no "update by PK implicitly" magic.

```go
// ✅ Correct
if err := db.Update(&user, orm.Eq(User_.ID, user.ID)); err != nil { ... }

// ❌ Compile error (caught by tinywasm/orm — will not reach the SQLite layer)
db.Update(&user)
```
```

---

## Step 2 — Regression integration test

Add a test to `sqlite_test.go` that verifies the loop-over-N-rows pattern
(the exact scenario that triggered the bug in `appointment-booking`) works
correctly when explicit conditions are provided.

```go
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
	err = db.Query(&User{}).ReadAll(
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
		if err := db.Query(got).Where("id").Eq(u.ID).ReadOne(); err != nil {
			t.Fatalf("ReadOne user %d: %v", u.ID, err)
		}
		if got.Age != wantAges[u.ID] {
			t.Errorf("user %d: expected age %d, got %d", u.ID, wantAges[u.ID], got.Age)
		}
	}
}
```

> **Note:** `User` uses `int` PK with AUTOINCREMENT in the existing test suite.
> Adjust the type of `u.ID` and the `orm.Eq` field name to match the struct
> already defined in `sqlite_test.go`.

---

## Acceptance Criteria

- [ ] `README.md` updated — no zero-condition Update example
- [ ] `TestUpdate_ExplicitPK_MultiRow` added and passes
- [ ] All existing tests still pass (`gotest`)
- [ ] `gopush 'docs+test: document Update condition contract and add multi-row regression test'`
