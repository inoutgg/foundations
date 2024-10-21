// Package sqldbtest provides utilities for testing database-related code.
//
// It offers a set of utility functions to initialize a database pool,
// load DDL schema, clean it up, and perform other testing-related tasks.
//
// Note: This package is intended to be used only within test files.
package sqldbtest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.inout.gg/foundations/env"
	"go.inout.gg/foundations/must"
	"go.inout.gg/foundations/sqldb"
)

const (
	queryFetchAllTables = `
SELECT table_name
FROM information_schema.tables
WHERE table_schema=$1::text;
`
)

func queryTruncateTable(table string) string { return fmt.Sprintf("TRUNCATE %s;", table) }
func queryDropTable(table string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", table)
}

type Config struct {
	// Timeout is the timeout for establishing a connection to the database in milliseconds.
	Timeout int `env:"DATABASE_CONNECTION_TIMEOUT" envDefault:"5000"` // optional (default: 5000ms)

	// DatabaseURI is the connection string for the database.
	DatabaseURI string `env:"DATABASE_URI"`

	// Schema is the schema to use for the database.
	Schema string `env:"DB_SCHEMA" envDefault:"public"` // optional (default: "public")

	Up func(context.Context, *pgx.Conn) error
}

// WithUp sets the database initialization function.
func WithUp(up func(context.Context, *pgx.Conn) error) func(*Config) {
	return func(c *Config) { c.Up = up }
}

// MustConfig loads the configuration from the environment.
//
// If no paths are provided, it defaults to ".test.env" in the current
// working directory (which for tests is the directory in which they are located),
// and in the root of the project.
//
// It panics if there is an error loading the configuration.
func MustConfig(paths []string, opts ...func(*Config)) *Config {
	if len(paths) == 0 {
		currentModulePath := must.Must(os.Getwd())
		rootPath := findModuleRoot(currentModulePath)
		paths = []string{
			filepath.Join(rootPath, ".test.env"),
			filepath.Join(currentModulePath, ".test.env"),
		}
	}

	config := env.MustLoad[Config](paths...)
	for _, opt := range opts {
		opt(config)
	}

	return config
}

// DB is a wrapper around pgxpool.Pool with useful utilities for DB management
// in tests.
type DB struct {
	config *Config
	tb     testing.TB
	pool   *pgxpool.Pool

	closeOnce sync.Once
}

// Must loads the configuration from the environment and creates a new DB.
func Must(ctx context.Context, tb testing.TB, opts ...func(*Config)) *DB {
	tb.Helper()

	config := MustConfig([]string{}, opts...)
	return MustWithConfig(ctx, tb, config)
}

// MustWithConfig creates a new DB with the given config.
//
// It initializes a new pool with the given config.
//
// It panics if there is an error initializing a connection to the database.
func MustWithConfig(ctx context.Context, tb testing.TB, config *Config) *DB {
	tb.Helper()

	pool := sqldb.MustPool(ctx, config.DatabaseURI, sqldb.WithUUID())
	db := &DB{
		config,
		tb,
		pool,
		sync.Once{},
	}

	tb.Cleanup(db.Close)

	return db
}

// Pool returns the underlying connection pool.
func (db *DB) Pool() *pgxpool.Pool { return db.pool }

// Close closes the DB connection.
func (db *DB) Close() { db.closeOnce.Do(db.close) }
func (db *DB) close() { db.pool.Close() }

func (db *DB) Init(ctx context.Context) error {
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("foundations/sqdbtest: failed to acquire a connection: %w", err)
	}
	defer conn.Release()

	return db.up(ctx, conn.Conn())
}

func (db *DB) up(ctx context.Context, conn *pgx.Conn) error {
	db.tb.Helper()

	if db.config.Up != nil {
		return db.config.Up(ctx, conn)
	}

	return nil
}

// TruncateTable truncates the given table.
func (db *DB) TruncateTable(ctx context.Context, table string) error {
	db.tb.Helper()

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("foundations/sqdbtest: failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := db.truncateTable(ctx, table, tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("foundations/sqdbtest: failed to commit transaction: %w", err)
	}

	return nil
}

// truncateTable wipes the specified table within the given transaction.
func (db *DB) truncateTable(ctx context.Context, table string, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, queryTruncateTable(table)); err != nil {
		return fmt.Errorf("foundations/sqdbtest: failed to truncate table %s: %w", table, err)
	}

	return nil
}

// TruncateTables truncates the given tables.
func (db *DB) TruncateTables(ctx context.Context, tables []string) error {
	db.tb.Helper()

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("foundations/sqdbtest: failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var errs []error
	for _, name := range tables {
		if err := db.truncateTable(ctx, name, tx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("foundations/sqdbtest: failed to commit transaction: %w", err)
	}

	return nil
}

// TruncateAllTables truncates all tables in the database.
func (db *DB) TruncateAllTables(ctx context.Context) error {
	db.tb.Helper()

	tables, err := db.fetchAllTables(ctx)
	if err != nil {
		return err
	}

	return db.TruncateTables(ctx, tables)
}

// DropTables drops the specified tables from the schema.
func (db *DB) DropTables(ctx context.Context, tables []string) error {
	db.tb.Helper()

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("foundations/sqdbtest: failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, t := range tables {
		if _, err := tx.Exec(ctx, queryDropTable(t)); err != nil {
			return fmt.Errorf("foundations/sqdbtest: failed to drop table %s: %w", t, err)
		}
	}

	return nil
}

// dropTable drops a single table from the schema within the given transaction.
func (db *DB) dropTable(ctx context.Context, table string, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, queryDropTable(table)); err != nil {
		return fmt.Errorf("foundations/sqdbtest: failed to drop table %s: %w", table, err)
	}

	return nil
}

// DropAllTables drops all tables available in the schema.
func (db *DB) DropAllTables(ctx context.Context) error {
	db.tb.Helper()

	tables, err := db.fetchAllTables(ctx)
	if err != nil {
		return err
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("foundations/sqdbtest: failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var errs []error
	for _, name := range tables {
		if err := db.dropTable(ctx, name, tx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("foundations/sqdbtest: failed to commit transaction: %w", err)
	}

	return nil
}

// Reset resets the database by dropping all tables and re-creating them.
func (db *DB) Reset(ctx context.Context) error {
	db.tb.Helper()

	if err := db.DropAllTables(ctx); err != nil {
		return err
	}

	return db.Init(ctx)
}

// fetchAllTables returns a list of all tables available in the schema.
func (db *DB) fetchAllTables(ctx context.Context) ([]string, error) {
	var tables []string
	rows, err := db.pool.Query(
		ctx,
		queryFetchAllTables,
		pgtype.Text{String: db.config.Schema, Valid: true},
	)
	if err != nil {
		return tables, fmt.Errorf("foundations/sqdbtest: failed to fetch tables: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var table pgtype.Text
		if err := rows.Scan(&table); err != nil {
			return tables, fmt.Errorf("foundations/sqdbtest: failed to scan table name: %w", err)
		}

		tables = append(tables, table.String)
	}

	return tables, nil
}

// findModuleRoot returns the root path of the module.
// It was adapted from the Go source code. Attributed to the Go Authors.
//
// Source: https://github.com/golang/go/blob/377646589d5fb0224014683e0d1f1db35e60c3ac/src/cmd/go/internal/modload/init.go#L1565C1-L1583C2
func findModuleRoot(dir string) string {
	if dir == "" {
		panic("foundations/sqdbtest: dir is not set")
	}
	dir = filepath.Clean(dir)

	// Look for enclosing go.mod.
	for {
		if fi, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !fi.IsDir() {
			return dir
		}
		d := filepath.Dir(dir)
		if d == dir {
			break
		}
		dir = d
	}

	return ""
}
