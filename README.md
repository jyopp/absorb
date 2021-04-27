# absorb
Absorb is a go library that provides (a basic slice-to-object reflection binding). It focuses on quickly unmarshaling columnar datasets into usable data (structs, maps, etc.) with minimal callsite fuss. It allows data sources to specify custom struct tags for mapping columns to struct fields. 

This library was originally created to make reading database types more convenient and less error-prone, especially with sqlite. Iterating database rows and copying values manually into an array of structs is extremely common, and the copious boilerplate code repeated in every method that touches the database is a source of frustration and bugs. Now, with a lightweight interface around your database statements, code like `var rows []RowObjects; absorb.Absorb(&rows, stmt)` just works.

Absorb wrangles most known data types. You can absorb data into arrays, slices, pointers, and channels, as well as slices of pointers, channels of pointers, pointers to slices of pointers, etc. The resulting per-row objects can be structs or maps with string keys, or even scalar values when a single column is emitted.

Absorb is lean and opinionated:
- No module imports, and minimal language imports (see [go.mod](go.mod)). Only relies on `sync`, `reflect`, and `strings`.
- It assumes a well-formed schema; Impossible type conversions are considered programming errors, which produce panics.
- Internal types used to perform conversions are shared, threadsafe, and cached.
- It isn't recursive; Applying compound keypaths to hierarchies of nested values is a non-goal.

### Example

For full examples, see the [_example](_example/) directory. General, reusable [csv](_example/csv/main.go) and [sqlite](_example/sqlite/statementwrapper) data source types are provided in the example projects. 

```go
type MyStruct struct {
  Name  string `mysource:"name"`
  Count int    `mysource:"count"`
}

func readSource() {
  // Source is an Absorbable type with keys "name", "count" in tag namespace "mysource"
  // Rows are sent as []interface{string,int}
  source := MySource(...)

  // Read the source in the background and send MyStructs to a channel without copying
  ch := make(chan *MyStruct, 50)
  go func() {
    defer close(ch)
    absorb.Absorb(ch, source)
  }()

  for mystruct := range ch {
    fmt.Printf("Got %+v\n", mystruct)
  }
}
```
