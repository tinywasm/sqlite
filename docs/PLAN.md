---
PLAN: "refactor!: sqlite implementa db.Conn (contrato movido de orm a tinywasm/db), sin registro DSN"
TAG: v0.3.0
---

# PLAN — `tinywasm/sqlite`: adapter real, migrar de `*orm.DB` a `db.Conn`

Orquestado por
[`DB_PORT_MASTER_PLAN.md`](https://github.com/tinywasm/app/blob/main/docs/DB_PORT_MASTER_PLAN.md)
— **pieza #4b** (repo descubierto durante el rediseño, no listado en la tabla original — ver nota ahí).
Autocontenido, en español. **Solo tienes este repo** (`github.com/tinywasm/sqlite`).

> **Prerequisito:** `go install github.com/tinywasm/devflow/cmd/gotest@latest`.
> Tests con `gotest`. Publica con `gopush 'mensaje'`.
> Este plan **requiere `tinywasm/db@v0.0.1`, `tinywasm/ddl@v0.0.1` y `tinywasm/sqlt@v0.1.0` ya
> publicados** (`sqlt` es quien te da el `db.Compiler`+`ddl.Compiler` — ver su propio
> `docs/PLAN.md`). Si alguno no resuelve en `go get`, para y repórtalo.

## 0. Qué es este repo y qué cambia

`tinywasm/sqlite` es el **adapter real**: envuelve un `*sql.DB` de verdad (vía `modernc.org/sqlite`)
en el contrato de almacenamiento, y expone `Open(dsn) (...)` para que una app lo use. Es distinto de
`tinywasm/sqlt`, que **solo** traduce `db.Query`/`ddl.Stmt` a SQL — no abre conexiones ni ejecuta nada.
`sqlite` = `sqlt` (compilador) + `database/sql` (ejecución) + introspección PRAGMA, todo unido en un
`db.Conn`.

Hoy este repo depende de `orm` directamente: `Open(dsn) (*orm.DB, error)`, `sqliteExecutor` implementa
`orm.Executor`/`orm.TxExecutor`/`orm.TableIntrospector`/`orm.SchemaInspector`, y hay un
`init() { orm.Register("sqlite", Open) }`. Todo eso cambia:

- `Open(dsn) (db.Conn, error)` — devuelve el contrato crudo, no un `*orm.DB`. Un consumidor que quiera
  la capa ergonómica hace `d := orm.New(conn)` él mismo (ver `orm/docs/PLAN.md` §4.2).
- **Sin `init()`, sin `orm.Register`.** El registro DSN se elimina del ecosistema (ver
  `db/docs/PLAN.md` §2, `DB_PORT_PROPOSAL.md` §6.6) — un lookup por string que falla en runtime viola
  el harness. No lo repliques aquí tampoco.
- `sqliteExecutor` pasa a implementar `db.Conn` completo (Executor **y** Compiler — antes el
  `Compiler` lo aportaba `orm.New(exec, compiler)` como argumento separado; ahora `db.Conn` exige
  ambos en el mismo valor, así que `sqliteExecutor` necesita un método `Compile` que delegue al
  compilador de `sqlt`).
- `orm.ColumnInfo`/`orm.TableIntrospector`/`orm.SchemaInspector` ya no existen — son
  `ddl.ColumnInfo`/`ddl.TableIntrospector`/`ddl.SchemaInspector` (`tinywasm/ddl`, ver su
  `docs/PLAN.md` §2.2).

## 1. Estado verificado (código actual del repo, antes de este plan)

- `adapter.go:19` `Open(dsn string) (*orm.DB, error)`: abre `*sql.DB`, hace `Ping`, limita a 1 conexión
  (`:memory:` es por-conexión), construye `exec := &sqliteExecutor{db: db}` +
  `compiler := sqlt.NewCompiler()`, devuelve `orm.New(exec, compiler)`.
- `adapter.go:14` `init() { orm.Register("sqlite", Open) }` — **se borra entero**.
- `adapter.go:42-89`: `Close`/`ExecSQL`/`GetExecutor`/`GetSqlDB`/`GetTxExecutor` — helpers que operan
  sobre `*orm.DB` extrayendo su executor interno. **Ya no hacen falta la mayoría**: `Open` ahora
  devuelve el `db.Conn` directo, así que un consumidor ya lo tiene en la mano (no necesita "extraerlo"
  de un wrapper). Ver §3.3 qué se queda y qué se borra.
- `executor.go`: `sqliteExecutor`/`sqliteTxExecutor` implementan `orm.Executor`+`orm.TxExecutor` (vía
  `BeginTx`) sobre `*sql.DB`/`*sql.Tx`. `errScanner` mapea `sql.ErrNoRows`.
- `introspect.go`: `tableColumns`/`tables`/`columns` (funciones libres sobre una interfaz local
  `queryer`) + los métodos `TableColumns`/`Tables`/`Columns` en ambos executors, implementando
  `orm.TableIntrospector`+`orm.SchemaInspector`.
- `go.mod`: `ddlc@v0.0.5`, `orm@v0.9.28`, `sqlt@v0.0.7`, `modernc.org/sqlite@v1.53.0`.

## 2. Cambios

### 2.1 `go.mod`

```
go get github.com/tinywasm/db@v0.0.1
go get github.com/tinywasm/ddl@v0.0.1
go get github.com/tinywasm/sqlt@v0.1.0   # ya migrado a db.Compiler+ddl.Compiler, ver su PLAN.md
go mod tidy                               # esto debe QUITAR github.com/tinywasm/orm por completo
```

### 2.2 `executor.go` — un solo tipo `sqliteConn` que satisface `db.Conn` completo

```go
package sqlite

import (
	"database/sql"

	"github.com/tinywasm/db"
	"github.com/tinywasm/model"
)

// sqliteConn implements db.Conn (Executor+Compiler) plus db.TxExecutor, ddl.TableIntrospector,
// and ddl.SchemaInspector — everything a SQL backend can offer. Renamed from sqliteExecutor:
// it's no longer just an Executor, db.Conn requires the Compiler half too.
type sqliteConn struct {
	db       *sql.DB
	compiler db.Compiler // sqlt.NewCompiler() — also a ddl.Compiler, but this field's static
	                      // type only needs the DML half; CompileDDL is reached via a type
	                      // assertion where needed (§2.4/adapter.go's Open).
}

func (c *sqliteConn) Compile(q db.Query, m model.Model) (db.Plan, error) {
	return c.compiler.Compile(q, m)
}

func (c *sqliteConn) Exec(query string, args ...any) error {
	_, err := c.db.Exec(query, args...)
	return err
}

func (c *sqliteConn) QueryRow(query string, args ...any) db.Scanner {
	return &errScanner{s: c.db.QueryRow(query, args...)}
}

func (c *sqliteConn) Query(query string, args ...any) (db.Rows, error) {
	return c.db.Query(query, args...)
}

func (c *sqliteConn) Close() error {
	return c.db.Close()
}

func (c *sqliteConn) BeginTx() (db.TxBoundExecutor, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return nil, err
	}
	return &sqliteTxExecutor{tx: tx}, nil
}

// sqliteTxExecutor implements db.TxBoundExecutor. It does NOT implement db.Compiler on its
// own (compiling doesn't depend on being inside a transaction) — callers that need to compile
// while inside a Tx use the original sqliteConn's Compile, same pattern as
// orm.DB.Tx's boundConn (see orm/docs/PLAN.md §4.3) and ddl.DB.Sync's boundConn (see
// ddl/docs/PLAN.md §2.3). This repo doesn't need its own boundConn: nothing in sqlite itself
// opens a Tx and then needs to keep compiling through it — that composition, if ever needed,
// is orm's/ddl's job, not this adapter's.
type sqliteTxExecutor struct {
	tx *sql.Tx
}

func (e *sqliteTxExecutor) Exec(query string, args ...any) error {
	_, err := e.tx.Exec(query, args...)
	return err
}

func (e *sqliteTxExecutor) QueryRow(query string, args ...any) db.Scanner {
	return &errScanner{s: e.tx.QueryRow(query, args...)}
}

func (e *sqliteTxExecutor) Query(query string, args ...any) (db.Rows, error) {
	return e.tx.Query(query, args...)
}

func (e *sqliteTxExecutor) Commit() error   { return e.tx.Commit() }
func (e *sqliteTxExecutor) Rollback() error { return e.tx.Rollback() }
func (e *sqliteTxExecutor) Close() error    { return nil } // a *sql.Tx has no Close; Commit/Rollback end it

type errScanner struct{ s interface{ Scan(...any) error } }

func (s *errScanner) Scan(dest ...any) error {
	err := s.s.Scan(dest...)
	if err == sql.ErrNoRows {
		return db.ErrNoRows
	}
	return err
}

var (
	_ db.Conn            = (*sqliteConn)(nil)
	_ db.TxExecutor      = (*sqliteConn)(nil)
	_ db.TxBoundExecutor = (*sqliteTxExecutor)(nil)
)
```

> **Ajusta `errScanner`** a la firma real que ya tenías (probablemente tomaba `*sql.Row` directo, no
> una interfaz — usa lo que compile; el punto es que sigue mapeando `sql.ErrNoRows` → `db.ErrNoRows`,
> no `orm.ErrNoRows`). `sqliteTxExecutor.Close()` es un no-op nuevo — antes no hacía falta porque
> `orm.TxBoundExecutor` ya lo pedía igual (`Executor` incluye `Close()`); revisa que tu versión actual
> ya lo tuviera y solo le cambies el tipo de retorno de error si aplica.

### 2.3 `adapter.go` — `Open` sin registro, helpers recortados

```go
package sqlite

import (
	"database/sql"
	"strings"

	"github.com/tinywasm/db"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/sqlt"

	_ "modernc.org/sqlite" // SQLite driver
)

// Open creates a new sqlite connection and returns it as a db.Conn. No registry, no init() —
// construct explicitly: conn, err := sqlite.Open(dsn); d := orm.New(conn) (if you want the
// ergonomic layer) or use conn directly (e.g. from ddl.New(conn, ddlCompiler)).
func Open(dsn string) (db.Conn, error) {
	raw, err := sql.Open("sqlite", normalizeDSN(dsn))
	if err != nil {
		return nil, fmt.Errf("failed to open sqlite database: %v", err)
	}
	if err := raw.Ping(); err != nil {
		return nil, fmt.Errf("failed to ping sqlite database: %v", err)
	}

	// SQLite does not support concurrent writers. In-memory databases (:memory:)
	// are per-connection — each new connection sees an empty database. Limiting
	// to a single connection prevents both "database is locked" and
	// "no such table" errors when multiple goroutines share the same conn.
	raw.SetMaxOpenConns(1)
	raw.SetMaxIdleConns(1)

	return &sqliteConn{db: raw, compiler: sqlt.NewCompiler()}, nil
}

// DDLCompiler returns the ddl.Compiler half of the connection's compiler, for callers wiring
// up ddl.New(conn, sqlite.DDLCompiler(conn)). sqlt's compiler implements both db.Compiler and
// ddl.Compiler in the same concrete type; this accessor makes that reachable without exposing
// the unexported concrete type.
func DDLCompiler(conn db.Conn) (ddlCompiler interface {
	CompileDDL(s any, m any) (string, []any, error) // placeholder shape — use ddl.Compiler's real signature
}, ok bool) {
	// Implement via a type assertion: c, ok := conn.(ddl.Compiler); return c, ok
	// (shown as a placeholder above only to avoid importing ddl here just for this doc
	// snippet's signature — in the real file, import "github.com/tinywasm/ddl" and return
	// (ddl.Compiler, bool) directly.)
	panic("replace with: c, ok := conn.(ddl.Compiler); return c, ok")
}

// GetSqlDB returns the underlying *sql.DB, for callers that need to drop to raw SQL. Returns
// (nil, false) if conn isn't a *sqliteConn (e.g. it's db/mem or another backend).
func GetSqlDB(conn db.Conn) (*sql.DB, bool) {
	c, ok := conn.(*sqliteConn)
	if !ok {
		return nil, false
	}
	return c.db, true
}

func normalizeDSN(dsn string) string {
	if strings.HasPrefix(dsn, "sqlite://") {
		return strings.TrimPrefix(dsn, "sqlite://")
	}
	if strings.HasPrefix(dsn, "sqlite:") {
		return strings.TrimPrefix(dsn, "sqlite:")
	}
	return dsn
}
```

> **`DDLCompiler` está escrito como placeholder deliberado** — el bloque de arriba tiene una firma
> inválida a propósito (para que no lo copies literal sin pensarlo). Impleméntalo así, de verdad:
> ```go
> import "github.com/tinywasm/ddl"
>
> func DDLCompiler(conn db.Conn) (ddl.Compiler, bool) {
> 	c, ok := conn.(ddl.Compiler)
> 	return c, ok
> }
> ```
> Como `sqlt`'s `*compiler` implementa ambas interfaces (`db.Compiler` y `ddl.Compiler`) y
> `sqliteConn` guarda uno como su campo `compiler db.Compiler`, el type-assert `conn.(ddl.Compiler)`
> **no** funciona directo sobre `conn` (un `db.Conn`) — `sqliteConn` en sí no implementa
> `CompileDDL`. Dos opciones, elige una y documenta cuál en el código:
> 1. Añade `func (c *sqliteConn) CompileDDL(s ddl.Stmt, m model.Model) (string, []any, error) {
>    dc, ok := c.compiler.(ddl.Compiler); if !ok { return "", nil, fmt.Err("compiler does not
>    support DDL") }; return dc.CompileDDL(s, m) }` — así `*sqliteConn` mismo satisface
>    `ddl.Compiler` y `ddl.New(conn, conn)` funciona si `conn` ya es `*sqliteConn` (necesitas el tipo
>    concreto, no la interfaz `db.Conn`, para pasarlo dos veces).
> 2. Guarda el compilador dos veces tipado (`compiler db.Compiler` + `ddlCompiler ddl.Compiler`,
>    ambos apuntando al mismo `sqlt.NewCompiler()`) y expón un accessor separado.
> **Prefiere la opción 1** (menos estado duplicado); es la que deja `sqlite.Open` devolviendo un
> único valor que sirve para `orm.New(conn)` **y** `ddl.New(conn, conn)` a la vez — coherente con que
> `sqlt`'s compiler ya es ambas cosas.

### 2.4 `Close`/`ExecSQL`/`GetExecutor`/`GetTxExecutor` — se borran

Ya no hacen falta: antes existían para "sacar" el executor de dentro de un `*orm.DB` opaco. Ahora
`Open` devuelve el `db.Conn` directo — el consumidor ya lo tiene, llama `conn.Close()`/`conn.Exec(...)`
directo, o hace `conn.(db.TxExecutor)` si necesita transacciones. No repliques estos helpers.

### 2.5 `introspect.go` — `orm.` → `ddl.`

```go
package sqlite

import (
	"github.com/tinywasm/db"
	"github.com/tinywasm/ddl"
)

type queryer interface {
	Query(string, ...any) (db.Rows, error)
}

func tableColumns(q queryer, table string) ([]string, error) {
	// (cuerpo sin cambios — sigue siendo PRAGMA table_info)
}

func (c *sqliteConn) TableColumns(table string) ([]string, error) {
	return tableColumns(c, table)
}

func (e *sqliteTxExecutor) TableColumns(table string) ([]string, error) {
	return tableColumns(e, table)
}

func tables(q queryer) ([]string, error) {
	// (cuerpo sin cambios)
}

func columns(q queryer, table string) ([]ddl.ColumnInfo, error) {
	// (cuerpo sin cambios salvo el tipo de retorno: orm.ColumnInfo → ddl.ColumnInfo)
}

func (c *sqliteConn) Tables() ([]string, error)                { return tables(c) }
func (c *sqliteConn) Columns(table string) ([]ddl.ColumnInfo, error) { return columns(c, table) }
func (e *sqliteTxExecutor) Tables() ([]string, error)                { return tables(e) }
func (e *sqliteTxExecutor) Columns(table string) ([]ddl.ColumnInfo, error) { return columns(e, table) }

var (
	_ ddl.TableIntrospector = (*sqliteConn)(nil)
	_ ddl.TableIntrospector = (*sqliteTxExecutor)(nil)
	_ ddl.SchemaInspector   = (*sqliteConn)(nil)
	_ ddl.SchemaInspector   = (*sqliteTxExecutor)(nil)
)
```

> Solo cambian imports/tipos (`orm.Rows`→`db.Rows`, `orm.ColumnInfo`→`ddl.ColumnInfo`,
> `orm.TableIntrospector`→`ddl.TableIntrospector`, `orm.SchemaInspector`→`ddl.SchemaInspector`,
> `sqliteExecutor`→`sqliteConn`). El cuerpo SQL de `tableColumns`/`tables`/`columns` no cambia.

## 3. Tests

- `sqlite_test.go` (el que ya existe): adapta imports/tipos igual que arriba. Donde antes hacía
  `d, err := sqlite.Open(":memory:")` y luego `d.Create(...)` (API de `orm.DB`), ahora `Open` devuelve
  `db.Conn` — si el test quería la ergonomía, envuélvelo: `d := orm.New(conn)` (importa
  `github.com/tinywasm/orm`, que a su vez ya depende de `db`, sin ciclo). Si el test solo verificaba el
  contrato crudo, no envuelvas nada.
- Añade (si no existe) un test que corra `db/conformance` contra `sqlite.Open(":memory:")` — la prueba
  real de que este adapter, no solo `sqlt` en aislado, cumple el contrato de punta a punta:
  ```go
  func TestSqliteAdapter_DBConformance(t *testing.T) {
  	dbconf.Run(t, dbconf.Factory{
  		Name: "sqlite-adapter",
  		New: func(t *testing.T, models ...model.Model) db.Conn {
  			conn, err := sqlite.Open(":memory:")
  			if err != nil { t.Fatalf("Open: %v", err) }
  			// aplica el DDL de los models antes de devolver — mismo patrón que sqlt/docs/PLAN.md §3.5
  			return conn
  		},
  	})
  }
  ```

## 4. Criterios de aceptación

- `sqlite.Open(dsn) (db.Conn, error)` — sin `init()`, sin `orm.Register`. **Cero**
  `github.com/tinywasm/orm` en el repo (`grep -rn "tinywasm/orm" .` vacío).
- `sqliteConn` implementa `db.Conn`+`db.TxExecutor`+`ddl.TableIntrospector`+`ddl.SchemaInspector`
  (`var _` de los cuatro).
- `GetSqlDB` sigue disponible (escape hatch a `*sql.DB`); `Close`/`ExecSQL`/`GetExecutor`/
  `GetTxExecutor` eliminados (§2.4).
- Test de conformidad de punta a punta contra `sqlite.Open(":memory:")` verde (§3).
- `go.mod` en `db@v0.0.1`+, `ddl@v0.0.1`+, `sqlt@v0.1.0`+; `go mod tidy` limpio; publicado con
  `gopush`.

## 5. Etapas

| # | Etapa | Archivo(s) | Criterio |
|---|---|---|---|
| 1 | Bump deps, quitar orm | `go.mod` | `db`/`ddl`/`sqlt` nuevos; `orm` fuera |
| 2 | `sqliteConn` (Executor+Compiler+Tx) | `executor.go` | `var _ db.Conn`, `var _ db.TxExecutor` (§2.2) |
| 3 | `Open` sin registro | `adapter.go` | `Open(dsn) (db.Conn, error)`; `Close`/`ExecSQL`/etc. fuera (§2.3/§2.4) |
| 4 | Introspección | `introspect.go` | `ddl.TableIntrospector`/`ddl.SchemaInspector` (§2.5) |
| 5 | Tests | `sqlite_test.go` | adaptado + conformance de punta a punta (§3) |
| 6 | Publicar | — | `gotest` verde; `gopush 'refactor!: db.Conn, sin registro DSN'` |

## 6. Cierre

Tras `gopush`, **borra** `docs/PLAN.md`; el diseño duradero a `README.md`.
