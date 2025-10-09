package dbsqltest

import (
	"cmp"
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.jetify.com/typeid/v2"
	"golang.org/x/sync/semaphore"
)

type TestPoolFactoryConfig struct {
	maxPoolSize    int64
	cleanupTimeout time.Duration
}

func (p *TestPoolFactoryConfig) defaults() {
	p.maxPoolSize = int64(runtime.GOMAXPROCS(0))
	p.cleanupTimeout = time.Second * 5
}

type TestPoolFactoryOptions func(*TestPoolFactoryConfig)

func WithCleanupTimeout(timeout time.Duration) func(*TestPoolFactoryConfig) {
	return func(config *TestPoolFactoryConfig) { config.cleanupTimeout = timeout }
}

func WithMaxPools(n int64) func(*TestPoolFactoryConfig) {
	return func(config *TestPoolFactoryConfig) { config.maxPoolSize = n }
}

type TestPoolFactory struct {
	mc             *pgx.Conn
	baseConfig     *pgxpool.Config
	sema           *semaphore.Weighted
	template       string
	cleanupTimeout time.Duration
	mu             sync.Mutex
}

// NewTestPoolFactory creates a new factory for managing test database pools.
func NewTestPoolFactory(
	ctx context.Context,
	connString string,
	migrator Migrator,
	opts ...TestPoolFactoryOptions,
) (*TestPoolFactory, error) {
	config := TestPoolFactoryConfig{}
	for _, opt := range opts {
		opt(&config)
	}

	config.defaults()

	baseConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("dbsqltest: failed to parse connection string: %w", err)
	}

	//nolint:exhaustruct
	f := TestPoolFactory{
		sema:           semaphore.NewWeighted(config.maxPoolSize),
		cleanupTimeout: config.cleanupTimeout,
		baseConfig:     baseConfig,
	}

	if err := f.init(ctx, migrator); err != nil {
		return nil, fmt.Errorf("dbsqltest: failed to initialize factory: %w", err)
	}

	return &f, nil
}

// Pool creates a new ephemeral database from the template and returns a pool connected to it.
// It waits on a semaphore for a database slot to be available, then creates a new database
// from the template and returns a pool connected to it.
func (f *TestPoolFactory) Pool(tb testing.TB) *pgxpool.Pool {
	tb.Helper()

	var (
		ctx = tb.Context()
		db  = typeid.MustGenerate(namesgenerator.GetRandomName(0)).String()
	)

	err := f.sema.Acquire(ctx, 1)
	require.NoError(tb, err, "failed to acquire semaphore")

	err = f.createEphemeralDB(ctx, db)
	require.NoError(tb, err, "failed to create ephemeral database")

	pool, err := f.pool(ctx, db)
	require.NoError(tb, err)

	tb.Logf("running test in database: %s", db)

	tb.Cleanup(func() {
		defer f.sema.Release(1)

		pool.Close()

		// Leave the database intact if the test has failed for debugging
		if tb.Failed() {
			return
		}

		_ = f.dropEphemeralDB(db)
	})

	return pool
}

// init initializes the factory by creating a template database with a generated
// template name and user.
func (f *TestPoolFactory) init(ctx context.Context, migrator Migrator) (err error) {
	var (
		user     = f.baseConfig.ConnConfig.User
		password = f.baseConfig.ConnConfig.Password
	)

	h := fnv.New64()
	h.Write([]byte(user))
	h.Write([]byte(password))
	h.Write([]byte(migrator.Hash()))

	template := "test_template_" + strconv.FormatUint(h.Sum64(), 10)
	f.template = template

	mc, err := f.mkMaintenanceConn(ctx)
	if err != nil {
		return fmt.Errorf("dbsqltest: failed to connect to database: %w", err)
	}

	// Linearize the creation of database template across multiple processes, since
	// it is a shared resource and can cause conflicts if multiple processes
	// try to create it simultaneously.
	releasePgLock, err := acquirePgLock(ctx, mc, template)
	if err != nil {
		return fmt.Errorf("dbsqltest: failed to take lock: %w", err)
	}

	defer func() { err = releasePgLock() }()

	if err := f.mkTemplate(ctx, migrator, user, template); err != nil {
		return fmt.Errorf("dbsqltest: failed to create database template: %w", err)
	}

	return err
}

// newConn creates a new connection to the specified database.
// If db is empty, connects to the default maintenance database.
func (f *TestPoolFactory) newConn(ctx context.Context, db string) (*pgx.Conn, error) {
	connConfig := f.baseConfig.ConnConfig.Copy()
	connConfig.Database = cmp.Or(db, connConfig.Database)

	conn, err := pgx.ConnectConfig(ctx, connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return conn, nil
}

// mkMaintenanceConn returns a maintenance connection, creating/refreshing it if needed.
// This method is thread-safe.
func (f *TestPoolFactory) mkMaintenanceConn(ctx context.Context) (*pgx.Conn, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.mc != nil && !f.mc.IsClosed() {
		// Check if the existing connection is still alive
		if err := f.mc.Ping(ctx); err == nil {
			return f.mc, nil
		}
		// Connection is dead, close it
		f.mc.Close(ctx)
		f.mc = nil
	}

	// Create a new maintenance connection
	newConn, err := f.newConn(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to acquire maintenance connection: %w", err)
	}

	f.mc = newConn

	return f.mc, nil
}

// pool creates a new pool connected to the specified database.
func (f *TestPoolFactory) pool(ctx context.Context, db string) (*pgxpool.Pool, error) {
	// Create pool configuration for the new database
	poolConfig := f.baseConfig.Copy()
	poolConfig.ConnConfig.Database = db

	p, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("dbsqltest: failed to create pool: %w", err)
	}

	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, fmt.Errorf("dbsqltest: failed to ping database: %w", err)
	}

	return p, nil
}

// mkTemplate creates a new database template with migrations applied.
// If the template exists, it will skip migration.
//
// Generally, mkTemplate is expected to be called only once at the factory
// initialization.
//
// mkTemplate is not thread-safe; attempting to run it concurrently will result in
// connection lock (pgx busy conn).
func (f *TestPoolFactory) mkTemplate(ctx context.Context, migrator Migrator, user, template string) error {
	mc, err := f.mkMaintenanceConn(ctx)
	if err != nil {
		return fmt.Errorf("dbsqltest: failed to get maintenance connection: %w", err)
	}

	// It is safe to check only if the database is marked as a template, since marking
	// it as a template is the last step in the process.
	var doesTemplateExists bool
	if err := mc.QueryRow(ctx, "SELECT exists(SELECT 1 FROM pg_database WHERE datname = $1)",
		template).Scan(&doesTemplateExists); err != nil {
		return fmt.Errorf("dbsqltest: failed to check if template exists: %w", err)
	}

	if doesTemplateExists {
		return nil // Template already exists
	}

	// If template doesn't exist, we could fail at marking it as a template, but
	// we could still succeed at creating it. Let's try to clean it up.
	if _, err := mc.Exec(ctx, strings.Join([]string{
		"DROP DATABASE IF EXISTS",
		pgx.Identifier{template}.Sanitize(),
	}, " ")); err != nil {
		return fmt.Errorf("dbsqltest: failed to drop existing database template: %w", err)
	}

	if _, err := mc.Exec(ctx, strings.Join([]string{
		"CREATE DATABASE",
		pgx.Identifier{template}.Sanitize(),
		"OWNER",
		pgx.Identifier{user}.Sanitize(),
	}, " ")); err != nil {
		return fmt.Errorf("dbsqltest: failed to create database template: %w", err)
	}

	// Connect to the template database to run migrations in the newly created database.
	tc, err := f.newConn(ctx, template)
	if err != nil {
		return fmt.Errorf("dbsqltest: failed to connect to template database: %w", err)
	}
	defer tc.Close(ctx)

	if err := migrator.Up(ctx, tc); err != nil {
		return fmt.Errorf("dbsqltest: failed to run migrations: %w", err)
	}

	if _, err := mc.Exec(ctx, "UPDATE pg_database SET datistemplate = true WHERE datname = $1", template); err != nil {
		return fmt.Errorf("dbsqltest: failed to finalize database template: %w", err)
	}

	return nil
}

// createEphemeralDB creates a new ephemeral database for testing.
func (f *TestPoolFactory) createEphemeralDB(ctx context.Context, db string) error {
	mc, err := f.mkMaintenanceConn(ctx)
	if err != nil {
		return fmt.Errorf("dbsqltest: failed to get maintenance connection: %w", err)
	}

	f.mu.Lock()
	_, err = mc.Exec(ctx, strings.Join([]string{
		"CREATE DATABASE",
		pgx.Identifier{db}.Sanitize(),
		"TEMPLATE",
		pgx.Identifier{f.template}.Sanitize(),
		"OWNER",
		pgx.Identifier{f.baseConfig.ConnConfig.User}.Sanitize(),
	}, " "))
	f.mu.Unlock()

	if err != nil {
		return fmt.Errorf("dbsqltest: failed to copy database template: %w", err)
	}

	return nil
}

// dropEphemeralDB drops the specified ephemeral database after testing.
func (f *TestPoolFactory) dropEphemeralDB(db string) error {
	ctx, cancel := context.WithTimeout(context.Background(), f.cleanupTimeout)
	defer cancel()

	mc, err := f.mkMaintenanceConn(ctx)
	if err != nil {
		return fmt.Errorf("dbsqltest: failed to connect to maintenance database: %w", err)
	}

	f.mu.Lock()

	_, err = mc.Exec(ctx, strings.Join([]string{"DROP DATABASE", pgx.Identifier{db}.Sanitize()}, " "))

	f.mu.Unlock()

	if err != nil {
		log.Printf("dbsqltest: failed to drop database %s: %v", db, err)
	}

	return nil
}

// acquirePgLock acquires a PostgreSQL advisory lock.
// Make sure to release the lock when the operation is completed.
func acquirePgLock(ctx context.Context, conn *pgx.Conn, name string) (func() error, error) {
	h := fnv.New32()
	h.Write([]byte(name))
	lockNum := int64(h.Sum32())

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1::BIGINT)", lockNum); err != nil {
		return nil, fmt.Errorf("dbsqltest: failed to acquire lock: %w", err)
	}

	return func() error {
		if _, err := conn.Exec(ctx, "SELECT pg_advisory_unlock($1::BIGINT)", lockNum); err != nil {
			return fmt.Errorf("dbsqltest: failed to release lock: %w", err)
		}

		return nil
	}, nil
}
