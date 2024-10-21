// NOTE: the encoder is registered automatically for `dbtest` struct
// so no need to register it here.
package pgxuuid_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.inout.gg/foundations/sqldb/sqldbtest"
)

func TestCodecDecodeValue(t *testing.T) {
	ctx := context.Background()
	db := sqldbtest.Must(ctx, t)

	pool := db.Pool()
	original, err := uuid.NewV7()
	require.NoError(t, err)

	rows, err := pool.Query(ctx, `select $1::uuid`, original)
	require.NoError(t, err)

	for rows.Next() {
		values, err := rows.Values()
		require.NoError(t, err)

		require.Len(t, values, 1)
		v0, ok := values[0].(uuid.UUID)
		require.True(t, ok)
		require.Equal(t, original, v0)
	}

	require.NoError(t, rows.Err())

	rows, err = pool.Query(ctx, `select $1::uuid`, nil)
	require.NoError(t, err)

	for rows.Next() {
		values, err := rows.Values()
		require.NoError(t, err)

		require.Len(t, values, 1)
		require.Equal(t, nil, values[0])
	}

	require.NoError(t, rows.Err())
}

func TestArray(t *testing.T) {
	ctx := context.Background()
	db := sqldbtest.Must(ctx, t)
	p := db.Pool()

	inputSlice := []uuid.UUID{}

	for i := 0; i < 10; i++ {
		u, err := uuid.NewV7()
		require.NoError(t, err)
		inputSlice = append(inputSlice, u)
	}

	var outputSlice []uuid.UUID
	err := p.QueryRow(ctx, `select $1::uuid[]`, inputSlice).Scan(&outputSlice)
	require.NoError(t, err)
	require.Equal(t, inputSlice, outputSlice)
}
