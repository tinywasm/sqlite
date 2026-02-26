# SQLite Adapter Implementation

This document is the **Master Prompt (PLAN.md)** for refactoring the `sqlite` library to act as an Adapter for the `tinywasm/orm` ecosystem. Every execution agent must follow this plan sequentially.

---

## Development Rules

- **SRP (Single Responsibility Principle):** Every file must have a single, well-defined purpose.
- **Mandatory Dependency Injection (DI):** The library acts as a dependency to be injected. It must import `github.com/tinywasm/orm` to implement the `Adapter` and `TxAdapter` interfaces.
- **Flat Hierarchy:** No subdirectories in the library root. All source files live in the root directory.
- **Max 500 lines per file:** If any file exceeds 500 lines, split it by domain.
- **Test Organization:** All test files go in `tests/` directory if there are more than 5, but use the standard test structure dictated by the rules.
- **Testing Runner:** Always use `gotest`.
- **WASM Compatibility:** Backend-only adapters should use build tags (`//go:build !wasm`) if the ecosystem strictly enforces it, though pure backend libs might not strictly need it. Rely on `database/sql`. Use `modernc.org/sqlite` (pure Go implementation) instead of `github.com/mattn/go-sqlite3` (CGO).
- **Dependency Removal:** Completely remove the `github.com/cdvelop/objectdb` dependency. The only external dependencies should be `github.com/tinywasm/orm` and `modernc.org/sqlite` to simplify API usage and maintain a single source of truth for the ORM.
- **Mocking:** Use standard `testing`, no external assertion libraries.
- **Documentation First:** Update `README.md` and related docs alongside or before code changes.

---

## Architecture Overview

The goal is to replace the current custom CRUD methods with a single standard `Execute` method, strictly adhering to the `orm.Adapter` interface defined in `tinywasm/orm`:

```go
type SqliteAdapter struct {
    db *sql.DB // Underlying connection
}

func (s *SqliteAdapter) Execute(q orm.Query, m orm.Model, factory func() orm.Model, each func(orm.Model)) error
```

Furthermore, it must support atomic operations by implementing the `orm.TxAdapter` and `orm.TxBound` interfaces natively, wrapping `*sql.Tx`.

---

## Execution Phases

### Phase 1: Struct definition and Connection (`adapter.go`)
1. Define `SqliteAdapter` encapsulating the connection.
2. Refactor existing connection methods to return an instance of `SqliteAdapter`.
3. Update `go.mod`: 
   - Add `go get github.com/tinywasm/orm` and `go get modernc.org/sqlite`.
   - Remove `github.com/cdvelop/objectdb` and `github.com/mattn/go-sqlite3`.

### Phase 2: Translation Engine (`translate.go`)
1. Create logic to seamlessly convert `orm.Query` into a standard SQL string (SQLite dialect: `?` placeholders for parameterized queries) alongside an `args []any` slice.
2. Implement SQL statement builders for:
   - `ActionCreate` (INSERT INTO tables)
   - `ActionReadOne` / `ActionReadAll` (SELECT ... FROM ... WHERE ... ORDER BY ... LIMIT)
   - `ActionUpdate` (UPDATE ... SET ... WHERE)
   - `ActionDelete` (DELETE FROM ... WHERE)

### Phase 3: Core Implementation (`execute.go`)
1. Implement the `Execute` method natively on `SqliteAdapter`.
2. Based on `q.Action`, route commands to internal executor functions that fire the translated SQL against `s.db` (`Exec`, `QueryRow`, `Query`).
3. For `ActionReadAll`, continuously invoke the `factory` func to allocate instances, array scan fields directly into `m.Pointers()`, and trigger the `each(m)` callback to push records downstream with ZERO slice allocations.

### Phase 4: Transaction Support (`tx.go`)
1. Adopt the `orm.TxAdapter` interface natively on `SqliteAdapter`: `BeginTx() (orm.TxBound, error)`.
2. Define `SqliteTxBound` struct embedding standard `Execute` capabilities but leveraging an active `*sql.Tx`.
3. Provide standardized `Commit` and `Rollback` methods.

### Phase 5: Cleanup and Testing
1. Safely remove entirely all legacy `.go` source logic (`add.go`, `delete.go`, `functions.go`, etc.) preserving only what is strictly necessary.
2. Formulate comprehensive module integration tests located in `sqlite_test.go` or `tests/` to guarantee SQL translation correctness using an ephemeral in-memory SQLite database (`:memory:`).
3. Validate overall completion executing `gotest`.
