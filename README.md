# foundations

A modular library designed to build maintainable production-grade systems.

## SQL comments

`dbsql.NewPool` returns a `*pgxpool.Pool`. For an ordinary Go build, wrap it
explicitly to add SQL comments:

```go
db := sqlcomment.NewWithDefaults(pool, sqlcomment.WithDriverVerTag())
```

Alternatively, [otelc](https://github.com/open-telemetry/opentelemetry-go-compile-instrumentation)
injects comments directly into pgx when the application is built with
`go tool otelc go build`. An application can also use the manual wrapper in an
instrumented build; the wrapper prevents the hook from adding a second comment.
The instrumented integration tests run with `go tool otelc go test -tags=otelc ./dbsql`.

Applications using this library from another module must enable the rules in
their own `otel.instrumentation.go` alongside `go.mod`:

```go
//go:build tools

package tools

import _ "go.inout.gg/foundations/dbsql/sqlcomment/pgxhook"
```

List any other desired otelc instrumentations in that file as well: a tool
file replaces otelc's default rule selection.

The hooks run in pgx's `Conn.Exec`, `Conn.Query`, and `Conn.SendBatch`, including
calls made through a pool or transaction. Both modes support context tags
(`sqlcomment.WithTag`), `sqlcomment.Skip`, and OpenTelemetry propagation fields.
The hooks do not cover explicitly prepared statements or `CopyFrom`.

## License

Licensed under the MIT license.
