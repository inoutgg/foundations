---
name: postgresql-tests
description: Use when creating, editing, running, or documenting PostgreSQL/pgx tests in this repository; enforce the PGX_TEST_DATABASE connection variable.
---

# PostgreSQL tests

- Follow `.opencode/skills/go-tests/SKILL.md` for top-level test names, subtest descriptions, and `t.Parallel()` usage.
- Use `PGX_TEST_DATABASE` as the only environment variable for PostgreSQL test connection strings. Do not read, set, or document aliases or introduce another test-database URL variable.
- Obtain the connection string through `testutil.PgxConnString(t)` (or `testutil.PgxConnString(b)` in benchmarks). The helper requires a nonempty value. Never provide a fallback connection string, assert it separately in a caller, or skip a PostgreSQL test because the variable is unset.
- For database-backed test commands, pass `PGX_TEST_DATABASE` if the development shell has not already set it. The development shell provides its local test database through that variable.
- Keep test helpers and test instructions consistent with this convention, including dbsql, dbsql/sqlcomment, and test/dbsqltest.
