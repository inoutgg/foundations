package sqldbtest

import (
	"cmp"
	"context"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchAllTables(t *testing.T) {
	ctx := context.Background()
	db := Must(ctx, t)
	defer db.Close()

	t.Run("it works with empty schema", func(t *testing.T) {
		ctx := context.Background()

		tables, err := db.fetchAllTables(ctx)
		require.NoError(t, err)

		slices.Equal([]string{}, tables)
	})

	t.Run("it works with an Up config", func(t *testing.T) {
		ctx := context.Background()
		content, err := readFile("fixtures/schema.sql")
		require.NoError(t, err)

		_, err = db.pool.Exec(ctx, content)
		require.NoError(t, err)

		tables, err := db.fetchAllTables(ctx)
		require.NoError(t, err)

		expectedTables := []string{"tests"}
		sliceUnorderedEqual(expectedTables, tables)
	})
}

func TestInit(t *testing.T) {
	ctx := context.Background()
	db := Must(ctx, t)
	defer db.Close()

	err := db.Init(ctx)
	require.NoError(t, err)

	resultedTables, err := db.fetchAllTables(ctx)
	require.NoError(t, err)

	expectedTables := []string{"tests"}
	sliceUnorderedEqual(expectedTables, resultedTables)
}

func count(ctx context.Context, db *DB) (int, error) {
	row := db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM tests")
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func TestTruncateTable(t *testing.T) {
	ctx := context.Background()
	db := Must(ctx, t)
	defer db.Close()

	t.Run("it works", func(t *testing.T) {
		ctx := context.Background()
		err := db.Init(ctx)
		require.NoError(t, err)

		_, err = db.pool.Exec(ctx, "INSERT INTO tests (id, name) VALUES (1, 'foo')")
		require.NoError(t, err)

		c, err := count(ctx, db)
		require.NoError(t, err)

		if c != 1 {
			t.Fatalf("expected count to be 1, got %d", c)
		}

		err = db.TruncateTable(ctx, "tests")
		require.NoError(t, err)

		c, err = count(ctx, db)
		require.NoError(t, err)
	})
}

func TestReset(t *testing.T) {
	ctx := context.Background()
	db := Must(ctx, t)
	defer db.Close()

	err := db.Reset(ctx)
	require.NoError(t, err)

	resultedTables, err := db.fetchAllTables(ctx)
	require.NoError(t, err)

	expectedTables := []string{"tests"}
	sliceUnorderedEqual(expectedTables, resultedTables)
}

func sliceUnorderedEqual[T cmp.Ordered](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	slices.Sort(a)
	slices.Sort(b)

	return slices.Equal(a, b)
}

// readFile reads file from the given path.
func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
