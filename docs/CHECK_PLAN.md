# PLAN — Schema-sync support (SQLite executor adapter)

> `tinywasm/sqlite` is the **SQLite executor** module: [adapter.go](../adapter.go) wires a
> `modernc.org/sqlite` connection (`sqliteExecutor`) to the **`tinywasm/sqlt`** compiler via
> `orm.New(exec, compiler)`. The compiler's DDL translation lives in `tinywasm/sqlt` (its own plan).
> This module owns the **executor-side** schema-sync concerns:
> 1. **register** so `orm.Open("sqlite://…")` resolves it,
> 2. map `sql.ErrNoRows` → `orm.ErrNoRows`, and
> 3. (for full reconcile) implement `TableIntrospector`.
>
> **Self-contained, single-module plan** (`tinywasm/sqlite`). Prerequisite: `orm` published with the
> registry (`Open`/`Register`/`Factory`), `orm.ErrNoRows`, and `db.SyncSchema`; and `tinywasm/sqlt`
> translating the column actions. Bump both deps first.

---

## 1. Development Rules (constraints copied for execution context)

- **Executor owns connection + introspection.** Translation is `tinywasm/sqlt`'s job; this module
  executes SQL and inspects the live DB. Keep dialect SQL strings minimal (only the introspection
  query lives here).
- **No `database/sql` leakage into `orm`.** `orm.QB` compares the read error against `orm.ErrNoRows`
  (the agnostic sentinel exported by `orm`); the executor must translate the driver's `sql.ErrNoRows`
  to `orm.ErrNoRows` at the `Scanner` boundary, or `ReadOne` never detects "not found".
- **Single connection.** `Open` already pins `SetMaxOpenConns(1)` for SQLite (in-memory + no
  concurrent writers). Introspection must use the same `*sql.DB`.
- **Additive by default; rename/drop only with introspection.** `db.Sync` casts the executor to
  `TableIntrospector`. With it → reconcile (rename/drop); without it → additive-only. Both correct.
- **`gotest` (not `go test`).** Use a real in-memory SQLite (`sqlite::memory:` / `:memory:`).
- **Documentation first.**

---

## 2. Problem

1. `orm.Open("sqlite://…")` has nothing to resolve — the module never calls `orm.Register`.
2. `sqliteExecutor.QueryRow` ([executor.go:19](../executor.go#L19)) and `sqliteTxExecutor.QueryRow`
   ([executor.go:49](../executor.go#L49)) return `*sql.Row`, whose `Scan` returns `sql.ErrNoRows` —
   not the `orm.ErrNoRows` that `orm.QB.ReadOne` checks. Reads of a missing row leak the wrong error.
3. No `TableColumns` → `db.Sync` can only run additive; it never reconciles (rename/drop).

---

## 3. Decision

### 3.1 Register (Tier 1)

`Open` is `func(dsn string) (*orm.DB, error)` — matches `orm.Factory`. Register the **scheme**
`"sqlite"`:

```go
// adapter.go
func init() { orm.Register("sqlite", Open) }
```

> DSN note (**cross-adapter contract**): `orm.Open` uses the scheme only to **select** the factory
> and passes the **full original dsn** (scheme included) to it — this is required because
> `tinywasm/postgres`'s `New` feeds the whole `postgres://…` URL straight to `lib/pq`. So **`sqlite.Open`
> must strip its own `sqlite://` prefix** before `sql.Open("sqlite", …)` (which wants a bare path or
> `:memory:`). Add a small `normalizeDSN` that turns `sqlite://<path>` → `<path>` and
> `sqlite::memory:` → `:memory:`. Do not assume orm strips it for you.

### 3.2 Map no-rows error (Tier 1)

Wrap both executors' `QueryRow` so `Scan` translates the driver error:

```go
type errScanner struct{ s orm.Scanner }
func (e errScanner) Scan(dest ...any) error {
    if err := e.s.Scan(dest...); err != nil {
        if err == sql.ErrNoRows {
            return orm.ErrNoRows
        }
        return err
    }
    return nil
}

func (e *sqliteExecutor)   QueryRow(q string, a ...any) orm.Scanner { return errScanner{e.db.QueryRow(q, a...)} }
func (e *sqliteTxExecutor) QueryRow(q string, a ...any) orm.Scanner { return errScanner{e.tx.QueryRow(q, a...)} }
```

### 3.3 `TableIntrospector` (Tier 2 — enables rename/drop)

> **Critical — implement it on the TX-bound executor.** Per the orm contract, `db.Sync` runs its
> work in a transaction whenever the executor implements `orm.TxExecutor` (this module does, via
> `BeginTx`), and it performs the `db.exec.(orm.TableIntrospector)` cast **inside** that transaction.
> So the executor it inspects is `*sqliteTxExecutor`, **not** `*sqliteExecutor`. If `TableColumns`
> lives only on the base executor, the cast **fails inside the tx** and reconcile silently degrades to
> additive-only. Implement it on **both** executors (and it's also correct that the tx view sees the
> table `CreateTable` just made in the same tx).

Share one helper, expose it on both executor types:

```go
// introspect.go
func tableColumns(q interface{ Query(string, ...any) (orm.Rows, error) }, table string) ([]string, error) {
    rows, err := q.Query("PRAGMA table_info(" + table + ")")
    if err != nil { return nil, err }
    defer rows.Close()
    var cols []string
    for rows.Next() {
        var cid int
        var name, ctype string
        var notnull, pk int
        var dflt any
        if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
            return nil, err
        }
        cols = append(cols, name)
    }
    return cols, rows.Err()
}

func (e *sqliteExecutor)   TableColumns(table string) ([]string, error) { return tableColumns(e, table) }
func (e *sqliteTxExecutor) TableColumns(table string) ([]string, error) { return tableColumns(e, table) }
```
(`*sqliteExecutor` and `*sqliteTxExecutor` both already have `Query(string, ...any) (orm.Rows,
error)`.) `PRAGMA table_info` doesn't accept a bind parameter, so the table name is concatenated — it
comes from `ModelName()`/codegen (not user input), but keep an identifier guard if desired. With this
present, `db.Sync` reconciles (rename via `old_name`, safe-drop); without it, additive-only.

---

## 4. Implementation Steps

### Step 1 — Bump deps
`go get github.com/tinywasm/orm@vX` and `github.com/tinywasm/sqlt@vY` (column-action translation).

### Step 2 — Register + DSN normalize (Tier 1)
[adapter.go](../adapter.go): add `init()` (§3.1); ensure `Open` accepts the `sqlite://…` /
`sqlite::memory:` forms (normalize if needed).

### Step 3 — Error mapping (Tier 1)
[executor.go](../executor.go): wrap both `QueryRow`s with `errScanner` (§3.2).

### Step 4 — Introspector (Tier 2)
New `introspect.go`: add `TableColumns` on **both** `*sqliteExecutor` and `*sqliteTxExecutor` (§3.3)
— the tx-bound one is the one `db.Sync` casts inside its transaction.

### Step 5 — Documentation
README/architecture: note the module registers as `"sqlite"`, maps `ErrNoRows`, and supports full
reconcile via `TableIntrospector`; translation is `tinywasm/sqlt`.

---

## 5. Edge Cases

- **Read of a missing row** → `Scan` returns `orm.ErrNoRows` (both base and tx executors).
- **`sqlite::memory:` per-connection** → already handled by the single-connection pin in `Open`;
  introspection shares that connection.
- **No `TableIntrospector`** (if not added) → `db.Sync` additive-only; `ADD COLUMN` duplicates
  absorbed by log-and-continue. Still correct.
- **`DROP`/`RENAME COLUMN` on old SQLite** → requires SQLite 3.25+/3.35+; `modernc.org/sqlite` is
  current, so supported. Document the floor.

---

## 6. Test Strategy

`gotest` in `tinywasm/sqlite/tests/` with a real in-memory DB.

| # | Case | Assert |
|---|------|--------|
| S1 | `init()` registered | `orm.Open("sqlite::memory:")` returns a working `*orm.DB` |
| S2 | `ReadOne` on empty table | returns `orm.ErrNotFound` (via `orm.ErrNoRows` mapping), not `sql.ErrNoRows` |
| S3 | `TableColumns` after `CreateTable` | returns the created column names |
| S4 | `db.SyncSchema` add a field, re-run | column added once; second run no-op (introspector skips it) |
| S5 | `db.SyncSchema` with `old_name` rename | column renamed (reconcile path **runs inside the tx** — proves `*sqliteTxExecutor.TableColumns` is reached) |
| S6 | tx executor `QueryRow` no rows | `Scan` returns `orm.ErrNoRows` |
| S7 | `*sqliteTxExecutor` satisfies `orm.TableIntrospector` | compile-time assert (`var _ orm.TableIntrospector = (*sqliteTxExecutor)(nil)`) |

---

## 7. Out of Scope

- DDL/condition translation — `tinywasm/sqlt` plan (the compiler).
- `db.Sync` / `db.SyncSchema` algorithm and the registry contract — orm core plan.
- Destructive type-change / table rebuild — deferred (additive dev sync only).
