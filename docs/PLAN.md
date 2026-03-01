# Implementation Plan: SQLite Adapter — ORM v0.0.13 DDL Support

## Development Rules
- **Single Responsibility Principle (SRP):** Every file must have a single, well-defined purpose.
- **Mandatory Dependency Injection (DI):** No global state. Interfaces for external dependencies.
- **Testing Runner (`gotest`):** Install first: `go install github.com/tinywasm/devflow/cmd/gotest@latest`
- **Standard Library Only in Tests:** NEVER use external assertion libraries.
- **Explicit Execution:** Do not modify source code unless all steps below are followed in order.

## Goal
Update `tinywasm/sqlite` to support `tinywasm/orm@v0.0.13`. Breaking changes:
- `Model.Columns() []string` is replaced by `Model.Schema() []orm.Field`
- New DDL actions: `ActionCreateTable`, `ActionDropTable` must be handled in `translate.go`
- All test models must implement the new `Schema()` interface
- FK constraints must generate valid SQLite `REFERENCES` clauses

## References
- ORM Skill: `github.com/tinywasm/orm/docs/SKILL.md`
- ORM PLAN: `github.com/tinywasm/orm/docs/PLAN.md`

## Execution Steps

### Step 1 — Update dependency
```bash
go get github.com/tinywasm/orm@v0.1.0
go mod tidy
```

### Step 2 — Update `translate.go`
In `buildSelect`, `buildUpdate`, `buildDelete` and `buildConditions`:
- **No change needed** for DML (these use `q.Columns []string` which is still populated by `db.go` from `Schema().Name`).
- **Add DDL cases** in the main `translate` switch:

```go
case orm.ActionCreateTable:
    sb.Write("CREATE TABLE IF NOT EXISTS ")
    sb.Write(q.Table)
    sb.Write(" (")
    fields := m.Schema()
    for i, f := range fields {
        if i > 0 { sb.Write(", ") }
        sb.Write(f.Name)
        sb.Write(" ")
        sb.Write(sqliteType(f.Type))
        if f.Constraints&orm.ConstraintPK != 0 { sb.Write(" PRIMARY KEY") }
        if f.Constraints&orm.ConstraintAutoIncrement != 0 { sb.Write(" AUTOINCREMENT") }
        if f.Constraints&orm.ConstraintNotNull != 0 { sb.Write(" NOT NULL") }
        if f.Constraints&orm.ConstraintUnique != 0 { sb.Write(" UNIQUE") }
        if f.Ref != "" {
            refCol := f.RefColumn
            if refCol == "" { refCol = "id" } // convention fallback
            sb.Write(Sprintf(", CONSTRAINT fk_%s_%s FOREIGN KEY (%s) REFERENCES %s(%s)",
                q.Table, f.Name, f.Name, f.Ref, refCol))
        }
    }
    sb.Write(")")

case orm.ActionDropTable:
    sb.Write("DROP TABLE IF EXISTS ")
    sb.Write(q.Table)
```

Add helper function:
```go
func sqliteType(t orm.FieldType) string {
    switch t {
    case orm.TypeInt64:   return "INTEGER"
    case orm.TypeFloat64: return "REAL"
    case orm.TypeBool:    return "INTEGER" // 0 or 1
    case orm.TypeBlob:    return "BLOB"
    default:              return "TEXT"
    }
}
```

### Step 3 — Update `sqlite_test.go`
- Replace `Columns() []string` with `Schema() []orm.Field` on all test models (`User`, `Product`, etc.).
- Add test for `CreateTable`:
```go
func TestCreateTable(t *testing.T) {
    db, _ := sqlite.Open(":memory:")
    defer sqlite.Close(db)
    err := db.CreateTable(&User{})
    if err != nil { t.Fatalf("CreateTable failed: %v", err) }
}
```
- Add test for `DropTable`.
- Add test for FK constraint in generated SQL via `db.CreateTable`.
- Add negative test for unsupported Action in translate.

### Step 4 — Verify
```bash
gotest
```
Coverage must be ≥ 90%.

### Step 5 — Publish
```bash
gopush 'feat: support orm v0.0.13 DDL Schema API'
```
