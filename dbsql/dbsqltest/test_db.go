package dbsqltest

import (
	"context"
	"fmt"
	"hash/fnv"
	"runtime"
	"strconv"
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

type TestPoolManager struct {
	conn         *pgx.Conn
	templateName string

	cleanupTimeout time.Duration
	poolConfig     *pgxpool.Config

	sema      *semaphore.Weighted // semaphore for limiting concurrent pool creation
	mu        sync.Mutex
	openPools map[string]*pgxpool.Pool
	closeOnce sync.Once
}

type TestPoolManagerConfig struct {
	MaxPools       int64
	CleanupTimeout time.Duration
}

func (p *TestPoolManagerConfig) defaults() {
	p.MaxPools = int64(runtime.GOMAXPROCS(0))
	p.CleanupTimeout = time.Second * 5
}

// NewPoolManager initializes a new template database
func NewPoolManager(ctx context.Context, connString string, up Up, config *TestPoolManagerConfig) (*TestPoolManager, error) {
	if config == nil {
		config = &TestPoolManagerConfig{}
	}
	config.defaults()

	poolConfig, err := pgxpool.ParseConfig(connString)
	if poolConfig.ConnConfig.Database != "" {
		poolConfig.ConnConfig.Database = ""
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	var (
		connConfig = poolConfig.ConnConfig.Copy()
		user       = connConfig.User
		password   = connConfig.Password
	)

	conn, err := pgx.ConnectConfig(ctx, connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	h := fnv.New64()
	h.Write([]byte(user))
	h.Write([]byte(password))
	h.Write([]byte(up.Hash()))

	templateName := "test_template_" + strconv.FormatUint(h.Sum64(), 10)

	releaseLock, err := takeLock(ctx, conn, templateName)
	if err != nil {
		return nil, fmt.Errorf("failed to take lock: %w", err)
	}
	defer releaseLock()

	if err := mkTemplate(ctx, conn, user, templateName); err != nil {
		return nil, fmt.Errorf("failed to create database template: %w", err)
	}

	return &TestPoolManager{
		conn:           conn,
		cleanupTimeout: config.CleanupTimeout,
		poolConfig:     poolConfig,

		templateName: templateName,
		sema:         semaphore.NewWeighted(config.MaxPools),
		openPools:    make(map[string]*pgxpool.Pool, config.MaxPools),
	}, nil
}

// Pool returns a new pgxpool.Pool initialized from the supplied configuration
// to the PoolManager.
//
// PoolManager waits on distributed mutex for the database to be available.
// Once the database is available, it returns a new pool.
func (pm *TestPoolManager) Pool(tb testing.TB) *pgxpool.Pool {
	tb.Helper()

	ctx := tb.Context()

	err := pm.sema.Acquire(ctx, 1)
	require.NoError(tb, err, "failed to acquire semaphore")

	pool, dbName, err := pm.allocate(ctx)
	pm.mu.Lock()
	pm.openPools[dbName] = pool
	pm.mu.Unlock()

	require.NoError(tb, err)

	// Note that the TB context must not be used in the cleanup function as it
	// is closed right before cleanup.
	tb.Cleanup(func() {
		pm.sema.Release(1)
		pool.Close()

		// Leave the database template intact if the test failed.
		if tb.Failed() {
			return
		}

		ctx, cancel := context.WithTimeout(ctx, pm.cleanupTimeout)
		defer cancel()
		_, _ = pm.conn.Exec(ctx, fmt.Sprintf("DROP DATABASE %s", dbName))

		pm.mu.Lock()
		delete(pm.openPools, dbName)
		pm.mu.Unlock()
	})

	return pool
}

func (pm *TestPoolManager) allocate(ctx context.Context) (*pgxpool.Pool, string, error) {
	dbNameID, err := typeid.Generate(namesgenerator.GetRandomName(0))
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate database name: %w", err)
	}

	var (
		dbName = dbNameID.String()
		config = pm.poolConfig.Copy()
	)

	config.ConnConfig.Database = dbName

	if err = copyTemplate(ctx, pm.conn, pm.templateName, dbName); err != nil {
		return nil, "", fmt.Errorf("failed to initialize database: %w", err)
	}

	p, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create a pool: %w", err)
	}

	if err := p.Ping(ctx); err != nil {
		p.Close()

		return nil, "", fmt.Errorf("failed to ping database: %w", err)
	}

	return p, dbName, nil
}

func (pm *TestPoolManager) Close(ctx context.Context) {
	pm.closeOnce.Do(func() {
		pm.close(ctx)
	})
}

func (pm *TestPoolManager) close(ctx context.Context) {
	pm.mu.Lock()

	for _, pool := range pm.openPools {
		pool.Close()
	}

	pm.mu.Unlock()

	_ = pm.conn.Close(ctx)
}

func takeLock(ctx context.Context, db DBTX, name string) (func() error, error) {
	h := fnv.New32()
	h.Write([]byte(name))
	lockNum := int64(h.Sum32())

	if _, err := db.Exec(ctx, "SELECT pg_advisory_lock($1::BIGINT)", lockNum); err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return func() error {
		if _, err := db.Exec(ctx, "SELECT pg_advisory_unlock($1::BIGINT)", lockNum); err != nil {
			return fmt.Errorf("failed to release lock: %w", err)
		}

		return nil
	}, nil
}

// mkTemplate creates a new database template.
//
// It repairs the database template if it was previously in invalid state.
func mkTemplate(ctx context.Context, db DBTX, user, dbName string) error {
	var doesExist bool

	if err := db.
		QueryRow(ctx, "SELECT exists(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).
		Scan(&doesExist); err != nil {
		return fmt.Errorf("failed to check if template exists: %w", err)
	}

	// Template has already been created, skipping setup.
	if doesExist {
		return nil
	}

	if _, err := db.Exec(ctx, "DROP DATABASE IF EXISTS $1", dbName); err != nil {
		return fmt.Errorf("failed to drop existing database template: %w", err)
	}

	if _, err := db.Exec(ctx, "CREATE DATABASE $1 OWNER $2", dbName, user); err != nil {
		return fmt.Errorf("failed to create database template: %w", err)
	}

	if _, err := db.Exec(
		ctx,
		"UPDATE pg_database SET datistemplate = true WHERE datname = $1", dbName,
	); err != nil {
		return fmt.Errorf("failed to finalize database template: %w", err)
	}

	return nil
}

func copyTemplate(ctx context.Context, dbtx DBTX, dst, src string) error {
	if _, err := dbtx.Exec(ctx, "CREATE DATABASE $1 TEMPLATE $2", dst, src); err != nil {
		return fmt.Errorf("failed to copy database template: %w", err)
	}

	return nil
}
