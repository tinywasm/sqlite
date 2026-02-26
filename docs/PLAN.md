# SQLite Adapter - API Documentation Plan

This master prompt outlines the specific steps to create an LLM-optimized API usage documentation for the `tinywasm/sqlite` adapter. 

## Development Rules
- **WASM / TinyGo Compatibility (`gotest` requirement):** You MUST replace all usages of standard `fmt` and `strings` with `github.com/tinywasm/fmt` in our codebase.
- Constraints remain active: `gotest`, SRP, standard DI, pure stdlib testing, 500 lines limit, and flat hierarchy.

## Execution Steps

### 1. Create LLM-Optimized API Documentation (`docs/SKILL.md`)
- **Goal**: Provide a highly condensed, minimum-token API reference so that any AI/LLM knows exactly how to instantiate and consume this `sqlite` adapter.
- **Action**: Create a new file `docs/SKILL.md`. 
- **Content Requirements**:
  - Show the standard way to define a model (struct with `TableName`, `Columns`, `Values`, `Pointers` methods).
  - Show exactly how to initialize the connection: `db, err := sqlite.New(":memory:")`. Note that this returns `*orm.DB` directly.
  - Show a dense, minimal sequence of CRUD operations: `db.Create(m)`, `db.Query(m).Where(orm.Eq("col", val)).ReadOne()`, `db.Update(...)`, `db.Delete(...)`, and `db.Tx(func(tx *orm.DB) error { ... })`.
  - Document how to properly close the connection `sqlite.Close(db)` and execute raw queries `sqlite.ExecSQL(db, "...")`.
  - **Crucial**: Omit verbose human-oriented explanations. Use only explicit, heavily commented code blocks with strict signatures. The focus MUST be on saving context window tokens for LLMs.

### 2. Update the README index
- **Goal**: Make the new documentation discoverable.
- **Action**: Add a link to the newly created `SKILL.md` in `README.md` under a "Documentation" section.
