package dbsqltest

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/puddle/v2"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.inout.gg/foundations/debug"
	"go.inout.gg/foundations/internal/uuidv7"
	"go.inout.gg/foundations/must"
)

var DefaultPostgresImage = "postgres:17"

// ReleaseTimeoutDuration is the default timeout for releasing a database resource back to the pool.
var ReleaseTimeoutDuration = 10 * time.Second

func databaseName(
	baseDBName string,
	idx int32,
) string {
	return fmt.Sprintf("%s_%d", baseDBName, idx)
}

func makeContainer(
	ctx context.Context,
	maxDBNum int32,
) (string, func(context.Context) error, error) {
	user := namesgenerator.GetRandomName(0)
	dbname := namesgenerator.GetRandomName(0)
	pswd := uuidv7.Must()

	container, err := postgres.Run(
		ctx,
		DefaultPostgresImage,
		postgres.WithDatabase(dbname),
		postgres.WithUsername(user),
		postgres.WithPassword(pswd.String()),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		return "", nil, err
	}

	origConnString, err := container.ConnectionString(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("dbsqltest: failed to create container: %w", err)
	}

	connString, err := setupDB(ctx, origConnString, maxDBNum)
	if err != nil {
		return "", nil, err
	}

	close := func(ctx context.Context) error {
		if err := container.Terminate(ctx); err != nil {
			return fmt.Errorf("dbsqltest: failed to terminate container: %w", err)
		}

		return nil
	}

	return connString, close, nil
}

// setupDB creates maxDBNum databases and returns the connection string with
// the updated database to the base database.
//
// If there is no database in the connection string present a new random
// database name will be picked.
func setupDB(ctx context.Context, origConnString string, maxDBNum int32) (string, error) {
	config, err := pgx.ParseConfig(origConnString)
	if err != nil {
		return "", fmt.Errorf("dbsqltest: failed to parse connection string: %w", err)
	}

	dbName := config.Database
	if dbName == "" {
		dbName = namesgenerator.GetRandomName(0)
	}

	config.Database = dbName

	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		return "", fmt.Errorf("dbsqltest: failed to connect to database: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	for i := int32(0); i < maxDBNum; i++ {
		_, err := conn.Exec(ctx, "CREATE DATABASE "+databaseName(dbName, i))
		if err != nil {
			return "", fmt.Errorf("dbsqltest: failed to create database: %w", err)
		}
	}

	return config.ConnString(), nil
}

type DBPoolConfig struct {
	poolConfig *pgxpool.Config
	MaxDBNum   int32
	Up         Up
	Down       Down
}

func (c *DBPoolConfig) defaults() {
	c.MaxDBNum = cmp.Or(c.MaxDBNum, int32(runtime.NumCPU()))
}

func NewDBPoolConfig(opts ...func(*DBPoolConfig)) *DBPoolConfig {
	cfg := &DBPoolConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	cfg.defaults()
	debug.Assert(cfg.poolConfig != nil, "DBPoolConfig.PoolConfig is nil")

	return cfg
}

func WithConnectionString(connString string) func(*DBPoolConfig) {
	return func(c *DBPoolConfig) {
		config := must.Must(pgxpool.ParseConfig(connString))
		c.poolConfig = config
	}
}

func WithUp(up Up) func(*DBPoolConfig)       { return func(c *DBPoolConfig) { c.Up = up } }
func WithDown(down Down) func(*DBPoolConfig) { return func(c *DBPoolConfig) { c.Down = down } }

type DBPool struct {
	pool   *puddle.Pool[*DB]
	config *DBPoolConfig
	mu     sync.Mutex
	ctr    int
}

func New(config *DBPoolConfig) (*DBPool, error) {
	dbPool := &DBPool{}

	pool, err := puddle.NewPool(&puddle.Config[*DB]{
		Constructor: dbPool.allocate,
		Destructor:  dbPool.release,
		MaxSize:     config.MaxDBNum,
	})
	if err != nil {
		return nil, fmt.Errorf("sqldbtest: failed to create pool of databases: %w", err)
	}

	dbPool.pool = pool

	return dbPool, nil
}

func NewWithContainer(
	ctx context.Context,
	config *DBPoolConfig,
) (*DBPool, func(context.Context) error, error) {
	var err error

	connString, close, err := makeContainer(ctx, config.MaxDBNum)
	if err != nil {
		return nil, nil, fmt.Errorf("sqldbtest: failed to create container: %w", err)
	}
	defer func() {
		if err != nil {
			_ = close(ctx)
		}
	}()

	newPoolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, nil, fmt.Errorf("sqldbtest: failed to parse connection string: %w", err)
	}

	config.poolConfig = newPoolConfig

	dbPool, err := New(config)
	if err != nil {
		return nil, nil, fmt.Errorf("sqldbtest: failed to create database pool: %w", err)
	}

	return dbPool, close, err
}

func (p *DBPool) Acquire(ctx context.Context) (*dbResource, error) {
	res, err := p.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("sqldbtest: failed to acquire a database pool: %w", err)
	}

	return &dbResource{
		res: res,
	}, nil
}

func (p *DBPool) allocate(ctx context.Context) (*DB, error) {
	cfg := p.nextConfig()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("sqldbtest: failed to create database pool: %w", err)
	}

	return &DB{
		pool:       pool,
		poolConfig: cfg,
		up:         p.config.Up,
		down:       p.config.Down,
	}, nil
}

func (p *DBPool) release(db *DB) { db.Close() }

func (p *DBPool) nextConfig() *pgxpool.Config {
	p.mu.Lock()
	defer p.mu.Unlock()
	ctr := p.ctr
	p.ctr++

	databaseName := fmt.Sprintf("%s_%d", p.config.poolConfig.ConnConfig.Database, ctr)
	newCfg := p.config.poolConfig.Copy()
	newCfg.ConnConfig.Database = databaseName

	return newCfg
}

// dbResource represents a resource acquired from the database pool.
type dbResource struct {
	res         *puddle.Resource[*DB]
	releaseOnce sync.Once
}

// Close releases the database resource back to the pool.
func (w *dbResource) Close() {
	w.releaseOnce.Do(w.release)
}

// release releases the database resource back to the pool.
// It checks if the database pool is closed and recreates it if necessary.
func (w *dbResource) release() {
	ctx, cancel := context.WithTimeout(context.Background(), ReleaseTimeoutDuration)
	defer cancel()

	db := w.res.Value()
	if err := db.Pool().Ping(ctx); err != nil && errors.Is(err, puddle.ErrClosedPool) {
		if err := db.recreatePool(ctx); err != nil {
			w.res.Destroy()
			return
		}
	}

	w.res.Release()
}
