# sqlite — Implementar orm.SchemaInspector

Extender el executor de SQLite para implementar `orm.SchemaInspector`, la
interfaz que permite al subpaquete `orm/ormcp` registrar la tool `db_schema`.

**Prerequisito:** `tinywasm/orm` debe estar publicado con `orm.SchemaInspector`
y `orm.ColumnInfo` disponibles antes de aplicar este plan.

---

## Contexto

`introspect.go` ya implementa `orm.TableIntrospector` (`TableColumns` devuelve
solo nombres de columna para el sync del ORM). `orm.SchemaInspector` es una
interfaz separada más rica que devuelve también tipo, NOT NULL y PK — necesaria
para que el LLM entienda el esquema completo vía MCP.

Los datos ya están disponibles en `PRAGMA table_info`: la consulta actual los
lee pero descarta todo excepto el nombre.

---

## Cambio requerido — `sqlite/introspect.go`

Extender el archivo existente. No modificar `TableColumns` (sigue siendo usado
por `orm.TableIntrospector`).

```go
// Tables returns all user-defined table names in the database.
func (e *sqliteExecutor) Tables() ([]string, error) {
    rows, err := e.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var tables []string
    for rows.Next() {
        var name string
        if err := rows.Scan(&name); err != nil {
            return nil, err
        }
        tables = append(tables, name)
    }
    return tables, rows.Err()
}

// Columns returns full column metadata for the given table using PRAGMA table_info.
func (e *sqliteExecutor) Columns(table string) ([]orm.ColumnInfo, error) {
    rows, err := e.Query("PRAGMA table_info(" + table + ")")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var cols []orm.ColumnInfo
    for rows.Next() {
        var cid int
        var name, ctype string
        var notnull, pk int
        var dflt any
        if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
            return nil, err
        }
        cols = append(cols, orm.ColumnInfo{
            Name:    name,
            Type:    ctype,
            NotNull: notnull == 1,
            PK:      pk > 0,
        })
    }
    return cols, rows.Err()
}

// Ensure sqliteExecutor implements orm.SchemaInspector
var _ orm.SchemaInspector = (*sqliteExecutor)(nil)
```

> `sqliteTxExecutor` queda fuera por ahora — `SchemaInspector` solo se necesita
> en la conexión principal, no dentro de transacciones.

---

## Archivos afectados

| Archivo | Cambio |
|---------|--------|
| `sqlite/introspect.go` | Agregar `Tables()`, `Columns()`, y compile-check |

---

## Orden de ejecución

1. Verificar que `github.com/tinywasm/orm` en `go.mod` tenga la versión con `SchemaInspector`
2. Agregar `Tables()` y `Columns()` a `sqlite/introspect.go`
3. Agregar compile-check `var _ orm.SchemaInspector = (*sqliteExecutor)(nil)`
4. Publicar con `gopush`

---

## Verificación

```bash
gotest
```

El compile-check fallará si `orm.SchemaInspector` o `orm.ColumnInfo` no existen
en la versión importada de `tinywasm/orm` — señal de que el prerequisito no
está publicado.
