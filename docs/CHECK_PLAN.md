# SQLite Adapter - Phase 2 (Refinements)

This master prompt continues from the previous integration round, refining the API according to the latest domain requirements.

## Development Rules
- Constraints remain active: `gotest`, SRP, standard DI, pure stdlib testing, 500 lines limit, and flat hierarchy.
- **WASM / TinyGo Compatibility (`gotest` requirement):** You MUST replace all usages of standard `fmt` and `strings` with `github.com/tinywasm/fmt`. This is strictly required for the `gotest` command to successfully pass its validations.

## Execution Steps

### 1. Module Path Correction
- Update `go.mod` to establish the definitive new ecosystem package path: `module github.com/tinywasm/sqlite` (replacing the old `github.com/cdvelop/sqlite`).

### 2. Direct ORM Injection (`adapter.go`)
- The user expressed that having to manually wrap `sqlite.New(dsn)` with `orm.New()` is tedious.
- Refactor the constructor `New(dsn string)` to **directly return an `*orm.DB`**.
- Example target signature: `func New(dsn string) (*orm.DB, error)`.
- The internal logic of `New` will continue creating the `*sql.DB` connection, instantiating the `SqliteAdapter`, but it must now wrap it by calling `orm.New(adapter)` and return the ready-to-use `*orm.DB` instance.
- The `SqliteAdapter` struct itself should remain available internally, preserving its exact implementation of `orm.Adapter` and `orm.TxAdapter`.
- `Close()` and `ExecSQL(query)` are still desirable behaviors on the internal connection. Ensure the architecture allows graceful tear down (e.g., exposing a wrapper method if needed).

### 3. Tests & Verification (`sqlite_test.go` or `tests/`)
- Adapt existing tests to operate directly with the newly returned `*orm.DB` instance instead of wrapping it manually within the tests.
- Re-validate logic by running `gotest`.
