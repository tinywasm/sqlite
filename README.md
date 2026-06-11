# sqlite
<img src="docs/img/badges.svg">

This is the `tinywasm/sqlite` adapter for `github.com/tinywasm/orm`.

## Usage

The adapter registers itself under the `"sqlite"` scheme. You can open a database using `sqlite.Open` directly or via `orm.Open`:

```go
package main

import (
	"log"

	"github.com/tinywasm/orm"
	"github.com/tinywasm/sqlite"
)

func main() {
	// Directly:
	db, err := sqlite.Open("my_database.sqlite")

	// Or via the registry (uses normalizeDSN to handle sqlite:// and sqlite::memory:):
	db, err = orm.Open("sqlite://my_database.sqlite")

	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer sqlite.Close(db)

	// Ready to use db via github.com/tinywasm/orm fluent API
	// ...
}
```

## Features

- **Registry Support:** Registers as `"sqlite"` to `orm.Register`.
- **Error Mapping:** Automatically maps `sql.ErrNoRows` to `orm.ErrNoRows`.
- **Schema Introspection:** Implements `TableIntrospector` for `db.SyncSchema` support, enabling column renames and drops.
- **WASM Compatible:** Uses `modernc.org/sqlite` for pure Go/CGO-less execution.

## Update

`db.Update` always requires at least one `Condition`. This is enforced at
compile time by `tinywasm/orm`. There is no "update by PK implicitly" magic.

```go
// ✅ Correct
if err := db.Update(&user, orm.Eq("id", user.ID)); err != nil { ... }

// ❌ Compile error (caught by tinywasm/orm — will not reach the SQLite layer)
db.Update(&user)
```
