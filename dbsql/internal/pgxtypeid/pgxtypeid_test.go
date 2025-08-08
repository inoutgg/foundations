// The encoder is registered automatically for sqldbtest.DB via sqldb.WithUUID
// so no need to register it here.
package pgxtypeid

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.jetify.com/typeid/v2"
)

func TestValue(t *testing.T) {
	t.Parallel()

	pool := db.Pool()

	t.Run("value", func(t *testing.T) {
		t.Parallel()

		expected, err := typeid.Generate("test")
		require.NoError(t, err)

		rows, err := pool.Query(t.Context(), `select $1::text`, expected)
		require.NoError(t, err)

		for rows.Next() {
			var actual typeid.TypeID
			err := rows.Scan(&actual)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		}

		require.NoError(t, rows.Err())
	})

	// t.Run("nil", func(t *testing.T) {
	// 	t.Parallel()

	// 	rows, err := pool.Query(t.Context(), `select $1::text`, nil)
	// 	require.NoError(t, err)

	// 	for rows.Next() {
	// 		var actual *typeid.TypeID
	// 		err := rows.Scan(actual)

	// 		require.NoError(t, err)
	// 		require.Equal(t, nil, actual)
	// 	}

	// 	require.NoError(t, rows.Err())
	// })
}

func TestArray(t *testing.T) {
	t.Parallel()

	pool := db.Pool()

	var expected []typeid.TypeID
	for i := 0; i < 10; i++ {
		u, err := typeid.Generate("test")

		require.NoError(t, err)
		expected = append(expected, u)
	}

	var actual []typeid.TypeID
	err := pool.QueryRow(t.Context(), `select $1::text[]`, expected).Scan(&actual)

	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
