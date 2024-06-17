package dbtest

import (
	"cmp"
	"context"
	"slices"
	"testing"
)

func TestFetchAllTables(t *testing.T) {
	ctx := context.Background()
	db := Must(ctx, t)
	defer db.Close()
	t.Run("it works with empty schema", func(t *testing.T) {
		ctx := context.Background()

		tables, err := db.fetchAllTables(ctx)
		if err != nil {
			t.Fatal(err)
		}

		slices.Equal([]string{}, tables)
	})

	t.Run("it works with a schema", func(t *testing.T) {
		ctx := context.Background()
		content, err := readFile("fixtures/schema.sql")
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.pool.Exec(ctx, content)
		if err != nil {
			t.Fatal(err)
		}

		tables, err := db.fetchAllTables(ctx)
		if err != nil {
			t.Fatal(err)
		}

		expectedTables := []string{"tests"}
		sliceUnorderedEqual(expectedTables, tables)
	})
}

func TestInit(t *testing.T) {
	ctx := context.Background()
	db := Must(ctx, t)
	defer db.Close()

	if err := db.Init(ctx); err != nil {
		t.Fatal(err)
	}

	resultedTables, err := db.fetchAllTables(ctx)
	if err != nil {
		t.Fatal(err)
	}

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
		if err := db.Init(ctx); err != nil {
			t.Fatal(err)
		}

		if _, err := db.pool.Exec(ctx, "INSERT INTO tests (id, name) VALUES (1, 'foo')"); err != nil {
			t.Fatal(err)
		}

		c, err := count(ctx, db)
		if err != nil {
			t.Fatal(err)
		}

		if c != 1 {
			t.Fatalf("expected count to be 1, got %d", c)
		}

		if err := db.TruncateTable(ctx, "tests"); err != nil {
			t.Fatal(err)
		}

		c, err = count(ctx, db)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestReset(t *testing.T) {
	ctx := context.Background()
	db := Must(ctx, t)
	defer db.Close()

	if err := db.Reset(ctx); err != nil {
		t.Fatal(err)
	}

	resultedTables, err := db.fetchAllTables(ctx)
	if err != nil {
		t.Fatal(err)
	}

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
