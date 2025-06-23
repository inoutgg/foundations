// Package sqldbtest provides utilities for testing database-related code.
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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
	Up   func(context.Context, *wrapper) error
	Down func(context.Context, *wrapper) error
)

type DBConfig struct {
	poolConfig *pgxpool.Config
	Up         Up
	Down       Down
}

// DB is a wrapper around pgxpool.Pool with useful utilities for DB management
// in tests.
type DB struct {
	*wrapper
	poolConfig *pgxpool.Config
	pool       *pgxpool.Pool

	up        Up
	down      Down
	closeOnce sync.Once
}

func NewDB(ctx context.Context, config *DBConfig) (*DB, error) {
	db := &DB{
		poolConfig: config.poolConfig,
		closeOnce:  sync.Once{},
		up:         config.Up,
		down:       config.Down,
	}
	if err := db.recreatePool(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) Reset(ctx context.Context) error {
	if db.down != nil {
		return db.down(ctx, db.wrapper)
	}

	if db.up != nil {
		return db.up(ctx, db.wrapper)
	}

	return nil
}

// Pool returns the underlying connection pool.
func (db *DB) Pool() *pgxpool.Pool { return db.pool }

// Close closes the DB connection.
func (db *DB) Close() { db.closeOnce.Do(db.close) }
func (db *DB) close() { db.pool.Close() }

type wrapper struct {
	schema   string
	executor Executor
}

func (db *wrapper) Executor() Executor { return db.executor }

// TruncateTable truncates the given table.
func (db *wrapper) TruncateTable(ctx context.Context, table string) error {
	tx, err := db.executor.Begin(ctx)
	if err != nil {
		return fmt.Errorf("foundations/sqldbtest: failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := db.truncateTable(ctx, table, tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("foundations/sqldbtest: failed to commit transaction: %w", err)
	}

	return nil
}

// truncateTable wipes the specified table within the given transaction.
func (db *wrapper) truncateTable(ctx context.Context, table string, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, queryTruncateTable(table)); err != nil {
		return fmt.Errorf("foundations/sqldbtest: failed to truncate table %s: %w", table, err)
	}

	return nil
}

// TruncateTables truncates the given tables.
func (db *wrapper) TruncateTables(ctx context.Context, tables []string) error {
	tx, err := db.executor.Begin(ctx)
	if err != nil {
		return fmt.Errorf("foundations/sqldbtest: failed to start transaction: %w", err)
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
		return fmt.Errorf("foundations/sqldbtest: failed to commit transaction: %w", err)
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
		return fmt.Errorf("foundations/sqldbtest: failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, t := range tables {
		if _, err := tx.Exec(ctx, queryDropTable(t)); err != nil {
			return fmt.Errorf("foundations/sqldbtest: failed to drop table %s: %w", t, err)
		}
	}

	return nil
}

// dropTable drops a single table from the schema within the given transaction.
func (db *wrapper) dropTable(ctx context.Context, table string, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, queryDropTable(table)); err != nil {
		return fmt.Errorf("foundations/sqldbtest: failed to drop table %s: %w", table, err)
	}

	return nil
}

// DropAllTables drops all tables available in the schema.
func (db *wrapper) DropAllTables(ctx context.Context) error {
	tables, err := db.fetchAllTables(ctx)
	if err != nil {
		return err
	}

	tx, err := db.executor.Begin(ctx)
	if err != nil {
		return fmt.Errorf("foundations/sqldbtest: failed to start transaction: %w", err)
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
		return fmt.Errorf("foundations/sqldbtest: failed to commit transaction: %w", err)
	}

	return nil
}

// fetchAllTables returns a list of all tables available in the schema.
func (db *wrapper) fetchAllTables(ctx context.Context) ([]string, error) {
	schema := db.schema
	if schema == "" {
		// pg uses "public" as default schema.
		schema = "public"
	}

	var tables []string
	rows, err := db.executor.Query(
		ctx,
		queryFetchAllTables,
		pgtype.Text{String: schema, Valid: true},
	)
	if err != nil {
		return tables, fmt.Errorf("foundations/sqldbtest: failed to fetch tables: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var table pgtype.Text
		if err := rows.Scan(&table); err != nil {
			return tables, fmt.Errorf("foundations/sqldbtest: failed to scan table name: %w", err)
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
		return fmt.Errorf("foundations/sqldbtest: failed to recreate a pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("foundations/sqldbtest: failed to ping the database: %w", err)
	}

	schema := db.poolConfig.ConnConfig.RuntimeParams["search_path"]

	db.pool = pool
	db.wrapper = &wrapper{
		schema:   schema,
		executor: db.pool,
	}

	return nil
}
