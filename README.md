# sqlite
<img src="docs/img/badges.svg">

This is the `tinywasm/sqlite` adapter for `github.com/tinywasm/orm`.

## Usage

You can now initialize your database with a single line of code:

```go
package main

import (
	"log"

	"github.com/tinywasm/sqlite"
)

func main() {
	// sqlite.Open returns a fully instantiated *orm.DB instance
	db, err := sqlite.Open("my_database.sqlite")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer sqlite.Close(db)

	// Ready to use db via github.com/tinywasm/orm fluent API
	// ...
}
```

## Update

`db.Update` always requires at least one `Condition`. This is enforced at
compile time by `tinywasm/orm`. There is no "update by PK implicitly" magic.

```go
// ✅ Correct
if err := db.Update(&user, orm.Eq("id", user.ID)); err != nil { ... }

// ❌ Compile error (caught by tinywasm/orm — will not reach the SQLite layer)
db.Update(&user)
```
