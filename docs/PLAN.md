# Implementation Plan: SQLite Adapter API Update (fmt.Model)

## Goal
Actualizar el adaptador `sqlite` para que sea perfectamente compatible con la reestructuración de `tinywasm/orm`, migrando sus dependencias de entidad a la nueva interfaz `fmt.Model`.

## Proposed Changes

### [Component] Core API Update
- **Target Files:** Archivos fuente principales del adaptador (`adapter.go`, compilador de `sqlite`, etc.).
- **Acciones:**
  - Modificar las firmas de interfaz y referencias donde antes se pasaba un `orm.Model` para que utilicen `fmt.Model` (importando `"github.com/tinywasm/fmt"`).
  - Sustituir cualquier llamada a `TableName()` dentro del generador de _queries_ SQLite para usar el nuevo método estandarizado `ModelName()`.

### [Component] Integrations & Tests
- **Target Files:** Suite de tests interna.
- **Acciones:**
  - Ajustar todos los wrappers, métodos emulados y comparaciones en los tests a la nueva firma.
  - Comprobar que los tipos subyacentes manejan correctamente el empaquetado y la nueva arquitectura de capas de `orm`.

## Verification Plan
- Ejecutar las pruebas estándar usando `gotest` en el directorio base del adaptador.
- El proyecto debe reaccionar de manera coherente sin generar compilaciones erróneas por "undefined orm.Model" o faltas de método en las interfaces de las estructuras falsas usadas para testing.
