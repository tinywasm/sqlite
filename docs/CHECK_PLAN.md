> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

# Plan: sqlite — Migrate compiler to sqlt.NewCompiler()

## Context

`tinywasm/sqlite` (module `github.com/tinywasm/sqlite`) is the native SQLite ORM adapter using
`modernc.org/sqlite`. Its SQL generation logic (`translate.go`, `compiler.go`) was moved to
`github.com/tinywasm/sqlt` to avoid duplication with `goflare/d1`.

**Este módulo está actualmente roto.** Los archivos `translate.go` y `compiler.go` fueron
eliminados del repositorio. El paquete no compila hasta que este plan se ejecute.

Tests run via `gotest` — no TinyGo installation required.

## What Was Removed

| Archivo eliminado | Contenido | Reemplazado por |
|---|---|---|
| `sqlite/translate.go` | `translateQuery`, `buildInsert`, `buildSelect`, `buildUpdate`, `buildDelete`, `buildCreateTable`, `buildDropTable`, `buildConditions`, `sqliteType` | `github.com/tinywasm/sqlt` (ya contiene estos) |
| `sqlite/compiler.go` | `type sqliteCompiler struct{}` + `Compile()` | `sqlt.NewCompiler()` |

## Current Broken State

Tres símbolos rotos tras la eliminación:

**`adapter.go:31`** — `sqliteCompiler{}` ya no existe:
```go
compiler := sqliteCompiler{}  // ROTO
```

**`export_test.go`** — expone `translateQuery` que ya no existe:
```go
func ExportTranslateQuery(q orm.Query, m fmt.Model) (string, []any, error) {
    return translateQuery(q, m)  // ROTO — translateQuery está en sqlt
}
```

**`sqlite_test.go`** — `TestCompilerErrors` y dos casos de IN en `TestSqliteAdapter` usan
`sqlite.ExportTranslateQuery(...)`. Esos tests fueron movidos a `sqlt_test.go` — deben
eliminarse de `sqlite_test.go` para evitar duplicación. El resto de tests del adaptador
permanece intacto.

**Prerequisite**: `github.com/tinywasm/sqlt` debe estar publicado con `NewCompiler()`.

## Goal

- Agregar `require github.com/tinywasm/sqlt` a `go.mod`.
- Reemplazar `sqliteCompiler{}` con `sqlt.NewCompiler()` en `adapter.go`.
- Agregar import `"github.com/tinywasm/sqlt"` en `adapter.go`.
- `gotest` pasa sin regresiones.

## TinyWasm Constraints (mandatory)

- No `import "errors"`, `"fmt"`, `"strings"` — use `github.com/tinywasm/fmt`.

## Changes

### `go.mod`

```
require github.com/tinywasm/sqlt <version-with-NewCompiler>
```

### `adapter.go`

Reemplazar `compiler := sqliteCompiler{}` → `compiler := sqlt.NewCompiler()` y agregar import `"github.com/tinywasm/sqlt"`.

### `export_test.go`

Eliminar el archivo completo — `ExportTranslateQuery` ya no tiene sentido porque `translateQuery`
no existe en sqlite. `sqlt.Translate` es la API pública directa.

### `sqlite_test.go`

Eliminar `TestCompilerErrors` completo y los dos casos de IN de `TestSqliteAdapter`:

```go
// eliminar estas líneas de TestSqliteAdapter:
_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionReadAll, Table: "t", Conditions: []orm.Condition{orm.In("id", 1)}}, nil)
if err == nil { t.Errorf("Expected compile error for non-slice IN value") }

_, _, err = sqlite.ExportTranslateQuery(orm.Query{Action: orm.ActionReadAll, Table: "t", Conditions: []orm.Condition{orm.In("id", []any{})}}, nil)
if err == nil { t.Errorf("Expected compile error for empty slice IN value") }
```

## Stages

| # | Archivo | Acción |
|---|---|---|
| 1 | `sqlite/go.mod` | Agregar `github.com/tinywasm/sqlt` |
| 2 | `sqlite/adapter.go` | Reemplazar `sqliteCompiler{}` → `sqlt.NewCompiler()` + agregar import |
| 3 | `sqlite/export_test.go` | **YA ELIMINADO** — archivo ya no existe. |
| 4 | `sqlite/sqlite_test.go` | **YA LIMPIO** — `TestCompilerErrors` y los 2 casos IN de `TestSqliteAdapter` ya fueron eliminados. |

## Verification

```bash
gotest
```

Sin regresiones. La suite existente cubre la integración completa con la base de datos.
