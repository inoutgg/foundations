package migration

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown           = "down"
)

type Migrator interface {
	Migrate(context.Context, Direction) error
	MigrateTx(context.Context, Direction) error
}

type Config struct {
	Logger *slog.Logger
	FS     fs.FS
}

func New(pool *pgxpool.Pool, config *Config) Migrator {
	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		}))
	}

	return &migrator{
		pool:       pool,
		fs:         config.FS,
		logger:     logger,
		migrations: []migration{},
	}
}

type migrator struct {
	pool       *pgxpool.Pool
fs         fs.FS
	logger     *slog.Logger
	migrations []migration
}

// Migrate implements Migrator.
func (m *migrator) Migrate(ctx context.Context, direction Direction) error {
	panic("unimplemented")
}

// MigrateTx implements Migrator.
func (m *migrator) MigrateTx(ctx context.Context, direction Direction) error {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("migrations: unable to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	switch direction {
	case DirectionUp:
		return m.migrateUp(ctx, direction, tx)
	case DirectionDown:
		return m.migrateDown(ctx, direction, tx)
	}

	return errors.New("migrations: unknown direction, expected either up or down")
}

func (m *migrator) migrateUp(ctx context.Context, direction Direction, tx pgx.Tx) error   {}
func (m *migrator) migrateDown(ctx context.Context, direction Direction, tx pgx.Tx) error {}

type migration interface {
	Up(context.Context, pgx.Tx) error
	Down(context.Context, pgx.Tx) error
}

type MigrateFunc func(context.Context, pgx.Tx) error

var _ migration = (*goMigration)(nil)
var _ migration = (*sqlMigration)(nil)

type goMigration struct {
	version int

	up   MigrateFunc
	down MigrateFunc
}

func (m *goMigration) Up(ctx context.Context, tx pgx.Tx) error   { return nil }
func (m *goMigration) Down(ctx context.Context, tx pgx.Tx) error { return nil }

type sqlMigration struct {
	version int

	up   string
	down string
}

func (m *sqlMigration) Up(ctx context.Context, tx pgx.Tx) error   { return nil }
func (m *sqlMigration) Down(ctx context.Context, tx pgx.Tx) error { return nil }
