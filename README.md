# foundations

A modular library designed to build maintainable production-grade systems.

## SQL comments

`dbsql.NewPool` returns a `*pgxpool.Pool`.
[otelc](https://github.com/open-telemetry/opentelemetry-go-compile-instrumentation)
injects SQL comments into pgx when the application is built with
`go tool otelc go build`. Ordinary Go builds do not inject comments; pgx
tracing is configured by `dbsql.NewPool` independently of `otelc`.

Add application and request tags to the context passed to pgx:

```go
ctx = sqlcomment.WithTags(ctx, sqlcomment.Tags{
    sqlcomment.KeyApplication: "api",
    sqlcomment.KeyRoute: "/orders",
})
_, err := pool.Exec(ctx, "SELECT 1")
```

`WithTags` returns a new context; supplied tags override existing values.
Pass the tagged context to each operation, including queries within a transaction:
`Begin(ctx)` does not attach tags to later `tx.Exec` or `tx.Query` calls.

Applications using this library from another module must enable the rules in
their own `otel.instrumentation.go` alongside `go.mod`:

```go
//go:build tools

package tools

import _ "go.inout.gg/foundations/dbsql/pgxhook"
```

List any other desired otelc instrumentations in that file as well: a tool
file replaces otelc's default rule selection.

The hooks run in pgx's `Conn.Exec`, `Conn.Query`, and `Conn.SendBatch`, including
calls made through a pool or transaction. `dbsql.NewPool` also configures the
[otelpgx](https://github.com/exaring/otelpgx) tracer when no pool option sets a
different tracer:

```go
pool, err := dbsql.NewPool(ctx, databaseURL)
```

The pool's otelpgx tracer uses `pgxhook.SpanNameFromContext` to read the extractor
bound to the query context. By default, `pgxhook.DefaultQueryNameExtractor`
uses the SQL query (including its comments) unchanged as the span name. To use
the query name from a sqlc annotation such as `-- name: GetAuthor :one`, bind
an extractor to the context passed to each query or batch:

```go
ctx = pgxhook.WithQueryNameExtractor(ctx, pgxhook.SQLCQueryName)
```

`SQLCQueryName` also uses the full SQL as the name when no sqlc annotation is
present. `Begin(ctx)` does not carry the extractor to later transaction calls;
pass the bound context to each call. otelpgx names each batch query span using
the same extractor.
SQL comments support context tags
(`sqlcomment.WithTags` and `sqlcomment.WithTag`), `sqlcomment.Skip`, and
OpenTelemetry propagation fields.
The hooks do not cover explicitly prepared statements or `CopyFrom`.

## PostgreSQL server execution duration

Once a database administrator has arranged for the application connections to
receive `auto_explain` NOTICE-level text plans, receive PostgreSQL's measured
execution duration on the client while keeping normal query results:

```go
pool, err := dbsql.NewPool(ctx, databaseURL, dbsql.WithServerDuration())
if err != nil {
    return err
}
defer pool.Close()

ctx = dbsql.WithServerDurationCallback(ctx, func(d time.Duration) {
    // Optional: use the duration in application code.
    // Do not run another query on this connection from the callback.
    log.Printf("PostgreSQL execution: %s", d)
})
_, err = pool.Exec(ctx, "SELECT pg_sleep(0.01)")
```

PostgreSQL sends `auto_explain`'s NOTICE-level duration to pgx's `OnNotice`
handler. The measured value is recorded in the OTel histogram
`db.postgresql.server_execution.duration` (seconds), as well as on a
`postgresql.query` span (or a `postgresql.batch` span with an event per
statement). It composes with an existing pgx tracer and notice handler.
The active query context is kept under a private string key in pgx's
connection-local `CustomData()` and removed when the query finishes; no
connection-keyed global map or SQL tag lookup is needed.
No server logs need to be scraped; `Exec`, `Query` (after closing the rows),
`QueryRow` (after `Scan`), transactions, and batches retain their results.
`auto_explain` covers planned statements such as SELECT/INSERT/UPDATE/DELETE,
not utility commands such as COMMIT or DDL.
The callback runs once for each reported statement, including each batch
statement.

`WithServerDuration()` only installs a pgx tracer and NOTICE receiver. It never
sets server parameters or loads an extension and requires no elevated database
privileges itself. Without `auto_explain` loaded and configured for each
connection, it will not receive any duration measurements. For example, the
database administrator can arrange to preload `auto_explain` for application
sessions and configure these settings:

```ini
session_preload_libraries = 'auto_explain'
auto_explain.log_min_duration = 0
auto_explain.log_level = 'notice'
auto_explain.log_format = 'text'
auto_explain.log_analyze = off
client_min_messages = 'notice'
```

**Overhead:** `0` means PostgreSQL constructs a plan and sends a NOTICE for
every eligible query. This adds processing and response bytes; it can be
significant for short queries. Server-log volume also increases if the server
is configured to log NOTICE-level messages.
`log_analyze = off` avoids per-node instrumentation, but does not remove the
plan-formatting overhead. Benchmark against the application's query mix before
enabling it widely. A positive `auto_explain.log_min_duration` or a reduced
`auto_explain.sample_rate` cuts overhead but reports only a subset of queries.
The duration excludes client round-trip time, although time spent by the
server waiting to write rows to a slow client can still count toward execution.
It does not include planning time.
PostgreSQL prints this duration to three decimal places in milliseconds; very
short statements can therefore be reported as zero.

## License

Licensed under the MIT license.
