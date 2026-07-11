# PLAN — Kind unification (phase B): fix test fixtures against phase-A model

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.
> Phase B of `tinywasm/docs/KIND_UNIFICATION_MASTER_PLAN.md` (Kind unification wave). Requires
> the published phase-A `tinywasm/model` (already pinned here: `go.mod` has
> `model v0.0.8`, `orm v0.9.27`, `sqlt v0.0.7`, `ddlc v0.0.4` — the module itself
> is on the right versions, only the tests weren't migrated). Runs parallel to
> orm/form/postgres/sqlt/mcp. See `sqlt/docs/PLAN.md` (already executed) for
the sibling repo that did the same migration.

## Context (zero-context summary)

Phase A changed `tinywasm/model`: `Field.Type` is no longer the `FieldType`
enum but the interface

```go
type Kind interface {
    Storage() FieldType   // the enum survives here — same values, same meaning
    Name() string
    Validate(value string) error
}
```

Non-test source files in this repo (`adapter.go`, `executor.go`,
`introspect.go`) already compile against phase-A model — no `.Type` enum
comparisons in production code. Only the **test fixtures** were never
migrated: they build `model.Field{..., Type: model.FieldText, ...}` literals
using the bare enum constants (`model.FieldText`, `model.FieldInt`,
`model.FieldFloat`), which no longer satisfy `model.Kind`. Additionally two
test files still reference `orm.FieldExt`, which moved to `ddlc.FieldExt`
during the orm/ddlc split (`ddlc` is already an indirect dependency here via
`sqlt`/`orm`).

`gotest` currently fails to compile in three places:

- `sqlite_test.go` — `Order.Schema()`, `UserTotalModel.Schema()`,
  `User.Schema()`, plus an `orm.FieldExt` literal for the `user_id` FK.
- `tests/sync_test.go` — `SyncUser.Schema()`, `SyncNewUser.Schema()`.
- `tests/jules_integration_test.go` — `SimpleUser.Schema()`,
  `SimpleSession.Schema()`, plus an `orm.FieldExt` literal for the `user_id`
  FK.

## Stage 1 — mechanical migration

- Every `Type: model.FieldX` literal in test fixtures becomes
  `Type: model.X()` (the phase-A base kind constructor): `model.FieldText` →
  `model.Text()`, `model.FieldInt` → `model.Int()`, `model.FieldFloat` →
  `model.Float()`. Grep for `model.Field(Text|Int|Float|Bool|Blob)\b` across
  `*_test.go` to find every site (`sqlite_test.go`, `tests/sync_test.go`,
  `tests/jules_integration_test.go`).
- Replace `orm.FieldExt` with `ddlc.FieldExt` in
  `sqlite_test.go:37-39` and `tests/jules_integration_test.go:38-40`; add the
  `github.com/tinywasm/ddlc` import (already resolvable — it's an indirect
  dependency of this module) and drop the now-unused `orm` import if nothing
  else in that file needs it.
- No production code changes expected (`adapter.go`, `executor.go`,
  `introspect.go` already compile clean).

## Stage 2 — tests

- `gotest ./...` green with no weakened assertions: behavior of the adapter
  (DDL/CRUD via `sqlt`) is unchanged — this is a call-site migration in test
  fixtures only, not a redesign.

## Harness checklist (mandatory)

- No behavior change: this is a fixture migration, not a redesign. If the
  `Kind` contract is insufficient here, **STOP and report** to the master
  plan.
- No unrelated refactors; `gotest` only.
- If `ddlc.FieldExt`'s shape differs from `orm.FieldExt` (field names,
  `RefColumn`), fix the fixture, don't reintroduce `orm.FieldExt`.

## Acceptance criteria

1. Module compiles against phase-A model; no bare `model.FieldX` enum
   literal remains as a `Field.Type` value; no `orm.FieldExt` reference
   remains (all `ddlc.FieldExt`).
2. `gotest ./...` green (vet + tests + race), matching the sibling `sqlt`
   module's clean state.

## Stages

| Stage | File(s) | Action |
|---|---|---|
| 1 | `sqlite_test.go`, `tests/sync_test.go`, `tests/jules_integration_test.go` | `model.FieldX` → `model.X()` constructor migration; `orm.FieldExt` → `ddlc.FieldExt` |
| 2 | all `*_test.go` | `gotest ./...` green, no assertion weakening |
