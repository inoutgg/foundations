package dbtesting

import (
	"cmp"
	"context"
	"slices"
	"testing"
)

func TestReset(t *testing.T) {
	ctx := context.Background()
	db := Must(MustLoadConfig())
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
