# PLAN (EJECUTADO 2026-07-14, LOCAL) — recompilado contra `model` v0.0.14 / `orm` v0.9.28

> Ejecutado directamente por el mantenedor (LOCAL, sin codejob). Fase D (propagación) de la
> ola CRUD Harness: https://github.com/tinywasm/app/blob/main/docs/CRUD_HARNESS_MASTER_PLAN.md

## El problema

`model` v0.0.14 amplía `model.Model` a `Fielder + ModuleNaming + Encodable + Decodable`. Este
repo declara sus fixtures de test **a mano** (no generados por `ormc`) para ejercitar el driver
sqlite directamente contra una BD real en memoria — ninguno serializa por el wire, así que
todos dejaron de compilar y necesitaban solo los tres métodos no-op que faltan.

## Cambios ejecutados

| Archivo | Tipos afectados |
|---|---|
| `sqlite_test.go` | `User`, `Order`, `UserTotalModel` |
| `tests/sync_test.go` | `SyncUser`, `SyncNewUser` |
| `tests/jules_integration_test.go` | `SimpleUser`, `SimpleSession` |
| `go.mod` | `tinywasm/model` → v0.0.14, `tinywasm/orm` → v0.9.28 |

Cada tipo gana `IsNil() bool`, `EncodeFields(model.FieldWriter)` y `DecodeFields(model.FieldReader)`
como no-ops, con el comentario que aclara por qué es seguro (fixtures que solo ejercitan el
driver, nunca viajan por el wire).

`gotest ./...` verde (incluye race). Publicado con gopush como v0.2.6.
