package dbtesting

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
	"go.inout.gg/common/env"
)

type Config struct {
	FilePath   []string `env:"DB_SCHEMA_PATH" envSeparator:","`
	SchemaName string   `env:"DB_SCHEMA_NAME"`
}

// LoadConfig loads the configuration from the environment.
func LoadConfig(paths ...string) (*Config, error) {
	return env.Load[Config](paths...)
}

// DB is a wrapper around pgxpool.Pool with useful utilities for DB management
// in tests.
type DB struct {
	pool   *pgxpool.Pool
	config *Config
}

// New creates a new DB.
func New(pool *pgxpool.Pool, config *Config) *DB {
	return &DB{
		pool,
		config,
	}
}

var (
	fetchAllTables = `
SELECT table_name
FROM information_schema.tables
WHERE table_schema=$1;
`
	truncateTable = `TRUNCATE TABLE $1;`
)

// Init initializes tables in the database by creating a schema provided
// by the config.
func (db *DB) Init(ctx context.Context) error {
	var sql []string
	for _, path := range db.config.FilePath {
		schemaContent, err := readFile(path)
		if err != nil {
			return err
		}

		sql = append(sql, parseSchema(schemaContent)...)
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db/testing: error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var errs []error

	for _, s := range sql {
		_, err := tx.Exec(ctx, s)
		if err != nil {
			errs = append(errs, fmt.Errorf("db/testing: error executing query %s: %w", s, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("db/testing: error committing transaction: %w", err)
	}

	return nil
}

func readFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("db/testing: error opening file %s: %w", path, err)
	}
	defer file.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, file)
	if err != nil {
		return "", fmt.Errorf("db/testing: error reading file %s: %w", path, err)
	}

	return buf.String(), nil
}

func parseSchema(schema string) []string {
	return lo.Filter(strings.Split(schema, ";"), func(s string, _ int) bool {
		return strings.TrimSpace(s) != ""
	})

}

// TruncateTable truncates the given table.
func (db *DB) TruncateTable(ctx context.Context, table string) error {
	if _, err := db.pool.Exec(ctx, truncateTable, pgtype.Text{String: table}); err != nil {
		return fmt.Errorf("db/testing: error truncating table %s: %w", table, err)
	}

	return nil
}

// TruncateTables truncates the given tables.
func (db *DB) TruncateTables(ctx context.Context, tables []string) error {
	var errs []error

	for _, name := range tables {
		if err := db.TruncateTable(ctx, name); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// TruncateAllTables truncates all tables in the database.
func (db *DB) TruncateAllTables(ctx context.Context) error {
	rows, err := db.pool.Query(ctx, fetchAllTables, pgtype.Text{String: db.config.SchemaName})
	if err != nil {
		return fmt.Errorf("db/testing: error fetching tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table pgtype.Text
		if err := rows.Scan(&table); err != nil {
			return fmt.Errorf("db/testing: error scanning table name: %w", err)
		}

		tables = append(tables, table.String)
	}

	return nil
}
