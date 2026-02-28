# Implementation Plan: Upgrade SQLite Adapter API

## Development Rules
- **WASM Environment (`tinywasm`):** Frontend Go Compatibility requires standard library replacements (`tinywasm/fmt`).
- **Single Responsibility Principle (SRP):** Every file must have a single, well-defined purpose.
- **Mandatory Dependency Injection (DI):** No global state. Interfaces for external dependencies.
- **Testing Runner (`gotest`):** ALWAYS use the globally installed `gotest` CLI command. (If missing, run: `go install github.com/tinywasm/devflow/cmd/gotest@latest`).
- **Standard Library Only in Tests:** NEVER use external assertion libraries.
- **Documentation First:** Update docs before coding.

## Goal
Refactor the `tinywasm/sqlite` adapter so that its initialization function directly returns a fully instantiated `*orm.DB` instance from `github.com/tinywasm/orm`. This eliminates the need for the user to write two lines of code to boot the database. Furthermore, add complex queries (including JOINs) to the test suite, ensure coverage is >90%, and update all documentation.

## Execution Steps

### 1. Update Public API
- Modify the adapter initialization signature in `adapter.go` (e.g., `sqlite.Open`).
- Internally instantiate the SQLite Executor and Compiler.
- Pass them to `orm.New()` and return the resulting `*orm.DB`.
- Ensure backwards compatibility is broken cleanly if necessary.

### 2. Complex Queries & JOINs Tests
- Add comprehensive tests in `sqlite_test.go` or `test_files/` utilizing the `tinywasm/orm` Fluent API.
- Create tests that explicitly execute complex `JOIN` queries to validate the SQL translation (`translate.go`).

### 3. Coverage > 90%
- Run `gotest`.
- Identify uncovered lines in `execute.go`, `translate.go`, `tx.go`, and `adapter.go`.
- Add mock or integration tests specifically targeting error paths and edge cases until the coverage is strictly greater than 90%.

### 4. Update Documentation
- **CRITICAL:** The `README.md` must be updated to show the new single-line `db := sqlite.Open(...)` initialization returning `*orm.DB`.
- Update architecture or skill docs if the change affects the public contract.
