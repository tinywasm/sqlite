# Implementation Plan: Eliminate Global State from Sqlite Adapter.

## Development Rules
- **WASM Environment (`tinywasm`):** Frontend Go Compatibility requires standard library replacements (`tinywasm/fmt`).
- **Single Responsibility Principle (SRP):** Every file must have a single, well-defined purpose.
- **Mandatory Dependency Injection (DI):** No global state. Interfaces for external dependencies.
- **Testing Runner (`gotest`):** ALWAYS use the globally installed `gotest` CLI command.
- **Documentation First:** Update docs before coding.

## Goal
The `sqlite` adapter currently uses a global registry (`dbRegistry` and `dbMu`) to associate an `*orm.DB` instance with its underlying driver connection. This was necessary because `*orm.DB` previously encapsulated its `Executor` privately without exposing a `Close()` or `RawExecutor()` method. Now that `github.com/tinywasm/orm` (v0.0.10) exposes these methods, the global state can be completely eliminated.

## Execution Steps

### 1. Update `go.mod`
- Update the `github.com/tinywasm/orm` dependency to the latest version.
- Run `go get github.com/tinywasm/orm@v0.0.10` to ensure you inherit the `Close()` and `RawExecutor()` interface updates.

### 2. Remove Global State from `adapter.go`
- Delete `dbRegistry` and `dbMu` variables.
- Update `sqlite.Open` to no longer register the database connection into a map.

### 3. Update `Close` and `ExecSQL` inside `adapter.go`
- Refactor the `Close` function to directly call `db.Close()`.
- Refactor the `ExecSQL` function to retrieve the raw executor via `db.RawExecutor()` and call `Exec` on it.
- **Note**: Ensure `tinywasm/fmt` is used appropriately for any error messages.

### 4. Verify Tests
- Run `gotest`. Ensure 100% test passage.
- Ensure test coverage remains remarkably high (>90%).

### 5. Update Documentation
- Ensure `README.md` reflects any internal change updates if necessary.

## Verification Plan
### Automated Tests
- Run `gotest` in `tinywasm/sqlite` to verify no regressions in the adapter logic, specifically related to migrations/connection closures.
