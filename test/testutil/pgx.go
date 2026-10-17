package testutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// PgxConnString returns the required PostgreSQL test connection string.
func PgxConnString(tb testing.TB) string {
	tb.Helper()

	connString := os.Getenv("PGX_TEST_DATABASE")
	require.NotEmpty(tb, connString, "PGX_TEST_DATABASE must be set for PostgreSQL tests")

	return connString
}
