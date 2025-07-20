package dbsqltest

import (
	"cmp"
	"context"
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

// databaseName generates a database name with the given base name and index.
func databaseName(baseDBName string, idx int32) string {
	return fmt.Sprintf("%s_%d", baseDBName, idx)
}

// makeContainer creates a PostgreSQL container and sets up the specified number of databases.
// Returns a connection string, cleanup function, and error.
func makeContainer(ctx context.Context, maxDBNum int32) (string, func(context.Context) error, error) {
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
		return "", nil, fmt.Errorf("foundations/dbsqltest: failed to create container: %w", err)
	}

	connString, err := setupDB(ctx, origConnString, maxDBNum)
	if err != nil {
		return "", nil, err
	}

	close := func(ctx context.Context) error {
		if err := container.Terminate(ctx); err != nil {
			return fmt.Errorf("foundations/dbsqltest: failed to terminate container: %w", err)
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
		return "", fmt.Errorf("foundations/dbsqltest: failed to parse connection string: %w", err)
	}

	dbName := config.Database
	if dbName == "" {
		dbName = namesgenerator.GetRandomName(0)
	}

	config.Database = dbName

	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		return "", fmt.Errorf("foundations/dbsqltest: failed to connect to database: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	for i := int32(0); i < maxDBNum; i++ {
		_, err := conn.Exec(ctx, "CREATE DATABASE "+databaseName(dbName, i))
		if err != nil {
			return "", fmt.Errorf("foundations/dbsqltest: failed to create database: %w", err)
		}
	}

	return config.ConnString(), nil
}

// DBPoolConfig holds configuration for creating a database pool.
type DBPoolConfig struct {
	poolConfig *pgxpool.Config
	MaxDBNum   int32
	Up         Up
	Down       Down
}

func (c *DBPoolConfig) defaults() {
	c.MaxDBNum = cmp.Or(c.MaxDBNum, int32(runtime.NumCPU()))
}

// NewDBPoolConfig creates a new DBPoolConfig with the given options.
func NewDBPoolConfig(opts ...func(*DBPoolConfig)) *DBPoolConfig {
	cfg := &DBPoolConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	cfg.defaults()
	debug.Assert(cfg.poolConfig != nil, "DBPoolConfig.PoolConfig is nil")

	return cfg
}

// WithConnectionString sets the connection string for the pool configuration.
func WithConnectionString(connString string) func(*DBPoolConfig) {
	return func(c *DBPoolConfig) {
		config := must.Must(pgxpool.ParseConfig(connString))
		c.poolConfig = config
	}
}

// WithUp sets the up migration function for the pool configuration.
func WithUp(up Up) func(*DBPoolConfig) { return func(c *DBPoolConfig) { c.Up = up } }

// WithDown sets the down migration function for the pool configuration.
func WithDown(down Down) func(*DBPoolConfig) { return func(c *DBPoolConfig) { c.Down = down } }

// DBPool manages a pool of database connections for testing.
type DBPool struct {
	pool   *puddle.Pool[*DB]
	config *DBPoolConfig
	mu     sync.Mutex
	ctr    int32
}

// New creates a new database pool with the given configuration.
func New(config *DBPoolConfig) (*DBPool, error) {
	debug.Assert(config != nil, "config must not be nil")

	dbPool := &DBPool{}

	pool, err := puddle.NewPool(&puddle.Config[*DB]{
		Constructor: dbPool.allocate,
		Destructor:  dbPool.release,
		MaxSize:     int32(config.MaxDBNum),
	})
	if err != nil {
		return nil, fmt.Errorf("foundations/dbsqltest: failed to create pool of databases: %w", err)
	}

	dbPool.pool = pool
	dbPool.config = config

	return dbPool, nil
}

// NewWithContainer creates a new database pool with a container-based setup.
// Returns the pool, cleanup function, and error.
func NewWithContainer(ctx context.Context, config *DBPoolConfig) (*DBPool, func(context.Context) error, error) {
	var err error

	connString, close, err := makeContainer(ctx, config.MaxDBNum)
	if err != nil {
		return nil, nil, fmt.Errorf("foundations/dbsqltest: failed to create container: %w", err)
	}
	defer func() {
		if err != nil {
			_ = close(ctx)
		}
	}()

	newPoolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, nil, fmt.Errorf("foundations/dbsqltest: failed to parse connection string: %w", err)
	}

	config.poolConfig = newPoolConfig

	dbPool, err := New(config)
	if err != nil {
		return nil, nil, fmt.Errorf("foundations/dbsqltest: failed to create database pool: %w", err)
	}

	return dbPool, close, err
}

// Acquire gets a database resource from the pool.
func (p *DBPool) Acquire(ctx context.Context) (*dbResource, error) {
	res, err := p.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("foundations/dbsqltest: failed to acquire a database pool: %w", err)
	}

	return &dbResource{
		res: res,
	}, nil
}

func (p *DBPool) allocate(ctx context.Context) (*DB, error) {
	cfg := p.nextConfig()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("foundations/dbsqltest: failed to create database pool: %w", err)
	}

	schema := cfg.ConnConfig.RuntimeParams["search_path"]

	db := &DB{
		pool:       pool,
		poolConfig: cfg,
		up:         p.config.Up,
		down:       p.config.Down,
		wrapper: &wrapper{
			schema:   schema,
			executor: pool,
		},
	}

	return db, nil
}

func (p *DBPool) release(db *DB) { db.Close() }

func (p *DBPool) nextConfig() *pgxpool.Config {
	p.mu.Lock()
	defer p.mu.Unlock()

	dbIndex := p.ctr % p.config.MaxDBNum
	p.ctr++

	dbName := databaseName(p.config.poolConfig.ConnConfig.Database, dbIndex)
	newCfg := p.config.poolConfig.Copy()
	newCfg.ConnConfig.Database = dbName

	return newCfg
}

// dbResource represents a database resource acquired from the pool.
// It ensures the resource is only released once.
type dbResource struct {
	res         *puddle.Resource[*DB]
	releaseOnce sync.Once
}

// DB returns the underlying DB instance.
func (r *dbResource) DB() *DB {
	return r.res.Value()
}

// Close releases the database resource back to the pool.
func (r *dbResource) Close() {
	r.releaseOnce.Do(r.release)
}

// release releases the database resource back to the pool.
func (r *dbResource) release() {
	ctx, cancel := context.WithTimeout(context.Background(), ReleaseTimeoutDuration)
	defer cancel()

	db := r.res.Value()
	if err := db.Pool().Ping(ctx); err != nil {
		r.res.Destroy()
		return
	}

	r.res.Release()
}
