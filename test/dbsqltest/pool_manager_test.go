package dbsqltest

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.segfaultmedaddy.com/pgxephemeraltest"
)

var _ pgxephemeraltest.Migrator = (*countingMigrator)(nil)

const testSchema = `CREATE TABLE IF NOT EXISTS kv (
  key TEXT PRIMARY KEY NOT NULL,
  value TEXT NOT NULL
);`

// countingMigrator is a Migrator that counts how many times Migrate is
// invoked, allowing tests to assert that the underlying factory is only
// initialized once.
type countingMigrator struct {
	schema string
	hash   string
	calls  atomic.Int64
}

func newCountingMigrator() *countingMigrator {
	return &countingMigrator{
		schema: testSchema,
		hash:   "dbsqltest" + strconv.FormatInt(rand.Int64(), 10), //nolint:gosec // test-only random hash
	}
}

func (m *countingMigrator) Hash() string { return m.hash }

func (m *countingMigrator) Migrate(ctx context.Context, conn *pgx.Conn) error {
	m.calls.Add(1)

	if _, err := conn.Exec(ctx, m.schema); err != nil {
		return fmt.Errorf("dbsqltest: failed to create table: %w", err)
	}

	return nil
}

// testConnString returns the connection string used to reach the test
// database. It honors TEST_DATABASE_URL and falls back to the default
// used by the devenv shell.
func testConnString(tb testing.TB) string {
	tb.Helper()

	if connString := os.Getenv("TEST_DATABASE_URL"); connString != "" {
		return connString
	}

	return "postgres://test:test@localhost:5432/test"
}

func testPoolConfig(tb testing.TB) *pgxpool.Config {
	tb.Helper()

	cfg, err := pgxpool.ParseConfig(testConnString(tb))
	require.NoError(tb, err)

	return cfg
}

func TestNewPoolFactoryConstructor_ReturnsConstructor(t *testing.T) {
	t.Parallel()

	ctor := NewPoolFactoryConstructor(newCountingMigrator(), testPoolConfig(t))
	assert.NotNil(t, ctor)
}

func TestPoolFactoryConstructor_CreatesWorkingPool(t *testing.T) {
	t.Parallel()

	// Arrange
	ctor := NewPoolFactoryConstructor(newCountingMigrator(), testPoolConfig(t))

	// Act
	pool := ctor(t)

	_, err := pool.Exec(
		t.Context(),
		"INSERT INTO kv (key, value) VALUES ($1, $2)",
		"foo",
		"bar",
	)
	require.NoError(t, err)

	// Assert
	var got string

	err = pool.QueryRow(t.Context(), "SELECT value FROM kv WHERE key = $1", "foo").Scan(&got)
	require.NoError(t, err)

	assert.Equal(t, "bar", got)
}

func TestPoolFactoryConstructor_InitializesFactoryOnce(t *testing.T) {
	t.Parallel()

	// Arrange
	migrator := newCountingMigrator()
	ctor := NewPoolFactoryConstructor(migrator, testPoolConfig(t))

	// Act: call the constructor several times. Each call must yield a
	// separate isolated pool, but the underlying factory (and thus the
	// migrator) must only be initialized once thanks to sync.Once.
	for i := range 3 {
		pool := ctor(t)

		_, err := pool.Exec(
			t.Context(),
			"INSERT INTO kv (key, value) VALUES ($1, $2)",
			strconv.Itoa(i),
			strconv.Itoa(i),
		)
		require.NoError(t, err)
	}

	// Assert
	assert.Equal(
		t,
		int64(1),
		migrator.calls.Load(),
		"Migrate must be called exactly once across multiple constructor invocations",
	)
}

func TestPoolFactoryConstructor_IsolatedPools(t *testing.T) {
	t.Parallel()

	// Arrange
	ctor := NewPoolFactoryConstructor(newCountingMigrator(), testPoolConfig(t))

	// Act
	pool1 := ctor(t)
	pool2 := ctor(t)

	_, err := pool1.Exec(
		t.Context(),
		"INSERT INTO kv (key, value) VALUES ($1, $2)",
		"k1",
		"v1",
	)
	require.NoError(t, err)

	_, err = pool2.Exec(
		t.Context(),
		"INSERT INTO kv (key, value) VALUES ($1, $2)",
		"k2",
		"v2",
	)
	require.NoError(t, err)

	// Assert: each pool only sees its own data.
	var v string

	require.NoError(
		t,
		pool1.QueryRow(t.Context(), "SELECT value FROM kv WHERE key = $1", "k1").Scan(&v),
	)
	assert.Equal(t, "v1", v)

	err = pool2.QueryRow(t.Context(), "SELECT value FROM kv WHERE key = $1", "k1").Scan(&v)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "pool2 must not see pool1's data")
}

func TestPoolFactoryConstructor_Parallel(t *testing.T) {
	t.Parallel()

	// Arrange
	ctor := NewPoolFactoryConstructor(newCountingMigrator(), testPoolConfig(t))

	var wg sync.WaitGroup

	// Act & Assert (per goroutine)
	for i := range 10 {
		wg.Go(func() {
			pool := ctor(t)
			expected := strconv.Itoa(i)

			_, err := pool.Exec(
				t.Context(),
				"INSERT INTO kv (key, value) VALUES ($1, $2)",
				"key",
				expected,
			)
			require.NoError(t, err)

			var v string

			err = pool.QueryRow(
				t.Context(),
				"SELECT value FROM kv WHERE key = $1",
				"key",
			).Scan(&v)
			require.NoError(t, err)

			assert.Equal(t, expected, v)
		})
	}

	wg.Wait()
}

// fatalSentinel is panicked by fatalTB to simulate testing.TB.Fatal
// without terminating the test goroutine.
type fatalSentinel struct{}

// fatalTB is a testing.TB that records Fatal/FailNow calls instead of
// terminating the goroutine. It panics with fatalSentinel so the caller
// can recover and assert on the failure.
//
// It embeds *testing.T to satisfy the testing.TB interface (including
// the unexported private method) while only overriding the methods
// relevant to the error path.
type fatalTB struct {
	*testing.T

	fatalMsg    string
	fatalCalled bool
}

func (t *fatalTB) Context() context.Context {
	return context.Background()
}

func (t *fatalTB) Helper() {}

func (t *fatalTB) Fatal(args ...any) {
	t.fatalCalled = true
	t.fatalMsg = fmt.Sprint(args...)

	panic(fatalSentinel{})
}

func (t *fatalTB) FailNow() {
	panic(fatalSentinel{})
}

func TestPoolFactoryConstructor_FailsOnUnreachableDB(t *testing.T) {
	t.Parallel()

	// Arrange: a config pointing at a closed port so that factory
	// initialization fails fast with a connection refused error.
	badCfg, err := pgxpool.ParseConfig("postgres://test:test@localhost:9/test")
	require.NoError(t, err)

	badCfg.ConnConfig.ConnectTimeout = 200 * time.Millisecond

	ctor := NewPoolFactoryConstructor(newCountingMigrator(), badCfg)

	// Act: invoking the constructor must call tb.Fatal because the
	// underlying factory could not be initialized. We use a fake TB
	// that records the fatal call via a recoverable panic.
	fakeTB := &fatalTB{}

	func() {
		defer func() { _ = recover() }()

		ctor(fakeTB)
	}()

	// Assert
	assert.True(
		t,
		fakeTB.fatalCalled,
		"constructor must call tb.Fatal when the factory cannot be initialized",
	)
	assert.Contains(t, fakeTB.fatalMsg, "failed to initialize factory")
}
