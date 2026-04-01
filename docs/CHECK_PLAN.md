# Plan: FieldDB Compatibility

## Depends on

- github.com/tinywasm/fmt with FieldDB support

## Problem

translate.go accesses `f.PK`, `f.Unique`, `f.AutoInc` directly on `fmt.Field`. These fields moved to `fmt.FieldDB` struct behind `Field.DB *FieldDB` pointer.

## Changes

### 1. translate.go — use helper methods

| Line | Before | After |
|------|--------|-------|
| 45 | `if f.PK` | `if f.IsPK()` |
| 54 | `if f.PK` | `if f.IsPK()` |
| 61 | `if f.AutoInc && f.Type == fmt.FieldInt` | `if f.IsAutoInc() && f.Type == fmt.FieldInt` |
| 69 | `if f.Unique` | `if f.IsUnique()` |

### 2. Test schema literals

**sqlite_test.go**:
```go
// Before
{Name: "id", Type: fmt.FieldInt, PK: true, AutoInc: true}

// After
{Name: "id", Type: fmt.FieldInt, DB: &fmt.FieldDB{PK: true, AutoInc: true}}
```

**tests/jules_integration_test.go**:
```go
// Before
{Name: "id", Type: fmt.FieldText, PK: true}
{Name: "email", Type: fmt.FieldText, Unique: true}

// After
{Name: "id", Type: fmt.FieldText, DB: &fmt.FieldDB{PK: true}}
{Name: "email", Type: fmt.FieldText, DB: &fmt.FieldDB{Unique: true}}
```

### 3. Bump go.mod

Update `github.com/tinywasm/fmt` to version with FieldDB.

## Execution Order

1. Bump fmt dependency
2. Update translate.go (4 lines)
3. Update sqlite_test.go (2 schema literals)
4. Update tests/jules_integration_test.go (3 schema literals)
5. `go test ./...`
