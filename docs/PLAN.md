# Plan: SQLite Adapter Enhancement

## Development Rules
- **Prerequisites:** External agents must install `gotest` first:
  ```bash
  go install github.com/tinywasm/devflow/cmd/gotest@latest
  ```
- **Standard Library Only:** NEVER use external assertion libraries (e.g., testify, gomega). Use only the standard testing, net/http/httptest, and reflect APIs.
- **Testing Runner (gotest):** For Go tests, ALWAYS use the globally installed `gotest` CLI command. DO NOT use `go test` directly. Simply type `gotest` (no arguments) for the full suite.
- **WASM Compatibility:** Use `tinywasm/fmt` instead of `fmt`/`strings`/`strconv`/`errors`.
- **Minimalist JS:** Use JavaScript only as a last resort. (N/A for this adapter)
- **Single Responsibility Principle:** Every file must have a single purpose.

## Goal
Improve `CreateTable` robustness and ensure compatibility with `ormc` generated models, specifically reproducing the "no such table" and connection issues when using `TEXT` Primary Keys.

## Proposed Changes

### [Component] Tests
#### [NEW] [jules_integration_test.go](tests/jules_integration_test.go)
- Implement a minimal test model to reproduce Jules's issue without replicating the full `user` schema:
  ```go
  type SimpleUser struct {
      ID    string `db:"pk"`
      Email string `db:"unique"`
  }
  ```
- Test `CreateTable` with this `TEXT` PK.
- Verify that `ActionCreateTable` returns success even if called twice (IF NOT EXISTS).
- Ensure that once `CreateTable` returns nil, the table is queryable.
- Test with Foreign Keys using a second minimal model:
  ```go
  type SimpleSession struct {
      ID     string `db:"pk"`
      UserID string `db:"ref=simple_users"`
  }
  ```

### [Component] SQL Generation
#### [MODIFY] [translate.go](translate.go)
- Review `buildCreateTable` for syntax errors that might cause "no such table" errors in edge cases.
- Ensure all constraints are correctly mapped to SQLite syntax.

## Verification Plan

### Automated Tests
- Run the full suite using `gotest`:
```bash
gotest
```
- Focus on the integration test:
```bash
gotest -run TestJulesScenario
```
