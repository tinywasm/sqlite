# SQLite Adapter API Reference

## Model Definition
```go
type User struct {
	ID   int
	Name string
}

func (u *User) TableName() string { return "users" }
func (u *User) Columns() []string { return []string{"name"} }
func (u *User) Values() []any     { return []any{u.Name} }
func (u *User) Pointers() []any   { return []any{&u.ID, &u.Name} }
```

## Initialization & Schema
```go
// Returns *orm.DB, error
db, err := sqlite.New(":memory:")
if err != nil { panic(err) }
defer sqlite.Close(db)

// Execute raw SQL (e.g., migrations)
err = sqlite.ExecSQL(db, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)`)
```

## CRUD Operations
```go
// Create
user := &User{Name: "Alice"}
err := db.Create(user)

// ReadOne
readUser := &User{}
err := db.Query(readUser).Where(orm.Eq("name", "Alice")).ReadOne()

// ReadAll
var users []*User
err := db.Query(&User{}).ReadAll(func() orm.Model {
    return &User{}
}, func(m orm.Model) {
    users = append(users, m.(*User))
})

// Update
err := db.Update(&User{Name: "Bob"}, orm.Eq("name", "Alice"))

// Delete
err := db.Delete(&User{}, orm.Eq("name", "Bob"))
```

## Transactions
```go
err := db.Tx(func(tx *orm.DB) error {
    if err := tx.Create(&User{Name: "Charlie"}); err != nil {
        return err // Rolls back
    }
    return nil // Commits
})
```
