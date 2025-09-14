// Package dbsqltest provides utilities for testing database-related code.
//
// It offers a set of utility functions to initialize a database pool,
// load DDL schema, clean it up, and perform other testing-related tasks.
//
// Note: This package is intended to be used only within test files.
package dbsqltest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.inout.gg/foundations/debug"
)

const (
	queryFetchAllTables = `
SELECT table_name
FROM information_schema.tables
WHERE table_schema=$1::text;
`
)

type Executor interface {
	Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

func queryTruncateTable(table string) string { return fmt.Sprintf("TRUNCATE %s CASCADE;", table) }
func queryDropTable(table string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", table)
}

type (
	Up   func(context.Context, *DB) error
	Down func(context.Context, *DB) error
)

// DB is a wrapper around pgxpool.Pool with useful utilities for DB management
// in tests.
type DB struct {
	*wrapper

	poolConfig *pgxpool.Config
	pool       *pgxpool.Pool

	closeOnce sync.Once

	up   Up
	down Down
}

// NewDB creates a new DB instance with the given configuration.
func NewDB(ctx context.Context, poolConfig *pgxpool.Config) (*DB, error) {
	debug.Assert(poolConfig != nil, "poolConfig is required")

	//nolint:exhaustruct
	db := &DB{
		poolConfig: poolConfig,
		closeOnce:  sync.Once{},
	}

	if err := db.recreatePool(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

// NewDBWithContainer creates a new DB instance with the given configuration.
//
// The opts are optional configuration functions that modify the pool configuration.
func NewDBWithContainer(ctx context.Context, opts ...func(*pgxpool.Config)) (*DB, func(context.Context) error, error) {
	var err error

	connString, close, err := makeContainer(ctx, 1)
	if err != nil {
		return nil, nil, fmt.Errorf("foundations/dbsqltest: failed to create container: %w", err)
	}
	defer func() {
		if err != nil {
			_ = close(ctx)
		}
	}()

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, nil, fmt.Errorf("foundations/dbsqltest: failed to parse connection string: %w", err)
	}

	for _, opt := range opts {
		opt(poolConfig)
	}

	db, err := NewDB(ctx, poolConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("foundations/dbsqltest: failed to create database pool: %w", err)
	}

	return db, close, err
}

// Up sets the up migration function. Up is called on each db.Reset().
func (db *DB) Up(fn Up) { db.up = fn }

// Down sets the down migration function. Down is called on each db.Reset().
func (db *DB) Down(fn Down) { db.down = fn }

// Reset resets the database by calling the provided Down function
// followed by the Up function.
//
// It is typically used before running tests that require a clean database state.
func (db *DB) Reset(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	if db.down != nil {
		err := db.down(ctx, db)
		if err != nil {
			return fmt.Errorf("foundations/dbsqltest: failed to reset database: %w", err)
		}
	}

	if db.up != nil {
		err := db.up(ctx, db)
		if err != nil {
			return fmt.Errorf("foundations/dbsqltest: failed to reset database: %w", err)
		}
	}

	return nil
}

// Pool returns the underlying connection pool.
func (db *DB) Pool() *pgxpool.Pool { return db.pool }

// Close closes the DB connection.
//
// Any future calls to this method will be ignored.
func (db *DB) Close() { db.closeOnce.Do(db.close) }

// WithTx runs the provided function within a transaction.
//
// It is useful for running multiple tests in parallel.
func (db *DB) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("foundations/dbsqltest: failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("foundations/dbsqltest: failed to commit transaction: %w", err)
	}

	return nil
}

func (db *DB) close() { db.pool.Close() }

// wrapper provides database utilities for managing tables and schema operations.
// It wraps an Executor interface and includes schema information for table operations.
type wrapper struct {
	schema   string
	executor Executor
}

func (db *wrapper) Executor() Executor { return db.executor }

// TruncateTable truncates the specified table.
func (db *wrapper) TruncateTable(ctx context.Context, table string) error {
	tx, err := db.executor.Begin(ctx)
	if err != nil {
		return fmt.Errorf("foundations/dbsqltest: failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := db.truncateTable(ctx, table, tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("foundations/dbsqltest: failed to commit transaction: %w", err)
	}

	return nil
}

// truncateTable truncates the specified table within the given transaction.
func (db *wrapper) truncateTable(ctx context.Context, table string, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, queryTruncateTable(table)); err != nil {
		return fmt.Errorf("foundations/dbsqltest: failed to truncate table %s: %w", table, err)
	}

	return nil
}

// TruncateTables truncates the specified tables.
func (db *wrapper) TruncateTables(ctx context.Context, tables []string) error {
	tx, err := db.executor.Begin(ctx)
	if err != nil {
		return fmt.Errorf("foundations/dbsqltest: failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

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
		return fmt.Errorf("foundations/dbsqltest: failed to commit transaction: %w", err)
	}

	return nil
}

// TruncateAllTables truncates all tables in the database.
func (db *wrapper) TruncateAllTables(ctx context.Context) error {
	tables, err := db.fetchAllTables(ctx)
	if err != nil {
		return err
	}

	return db.TruncateTables(ctx, tables)
}

// DropTables drops the specified tables from the schema.
func (db *wrapper) DropTables(ctx context.Context, tables []string) error {
	tx, err := db.executor.Begin(ctx)
	if err != nil {
		return fmt.Errorf("foundations/dbsqltest: failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, t := range tables {
		if _, err := tx.Exec(ctx, queryDropTable(t)); err != nil {
			return fmt.Errorf("foundations/dbsqltest: failed to drop table %s: %w", t, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("foundations/dbsqltest: failed to commit transaction: %w", err)
	}

	return nil
}

// dropTable drops a single table from the schema within the given transaction.
func (db *wrapper) dropTable(ctx context.Context, table string, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, queryDropTable(table)); err != nil {
		return fmt.Errorf("foundations/dbsqltest: failed to drop table %s: %w", table, err)
	}

	return nil
}

// DropAllTables drops all tables in the schema.
func (db *wrapper) DropAllTables(ctx context.Context) error {
	tables, err := db.fetchAllTables(ctx)
	if err != nil {
		return err
	}

	tx, err := db.executor.Begin(ctx)
	if err != nil {
		return fmt.Errorf("foundations/dbsqltest: failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

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
		return fmt.Errorf("foundations/dbsqltest: failed to commit transaction: %w", err)
	}

	return nil
}

// fetchAllTables returns a list of all tables in the schema.
func (db *wrapper) fetchAllTables(ctx context.Context) ([]string, error) {
	schema := db.schema
	if schema == "" {
		// PostgreSQL uses "public" as the default schema.
		schema = "public"
	}

	var tables []string
	rows, err := db.executor.Query(
		ctx,
		queryFetchAllTables,
		pgtype.Text{String: schema, Valid: true},
	)
	if err != nil {
		return tables, fmt.Errorf("foundations/dbsqltest: failed to fetch tables: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var table pgtype.Text
		if err := rows.Scan(&table); err != nil {
			return tables, fmt.Errorf("foundations/dbsqltest: failed to scan table name: %w", err)
		}

		tables = append(tables, table.String)
	}

	return tables, nil
}

func (db *DB) recreatePool(ctx context.Context) error {
	if db.pool != nil {
		db.pool.Close()
	}

	pool, err := pgxpool.NewWithConfig(ctx, db.poolConfig)
	if err != nil {
		return fmt.Errorf("foundations/dbsqltest: failed to recreate a pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("foundations/dbsqltest: failed to ping the database: %w", err)
	}

	schema := db.poolConfig.ConnConfig.RuntimeParams["search_path"]

	db.pool = pool
	db.wrapper = &wrapper{
		schema:   schema,
		executor: db.pool,
	}

	return nil
}
