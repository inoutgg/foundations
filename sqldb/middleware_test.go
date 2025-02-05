package sqldb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {
	mockPool := &pgxpool.Pool{}

	t.Run("should bind pool to context", func(t *testing.T) {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Assert
			pool, err := FromRequest(r)

			assert.NoError(t, err)
			assert.Equal(t, mockPool, pool)

			pool, err = FromContext(r.Context())
			assert.NoError(t, err)
			assert.Equal(t, mockPool, pool)
		})

		// Arrange
		middleware := Middleware(mockPool)

		// Act
		middleware(testHandler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	})

	t.Run("should return error when pool not in context", func(t *testing.T) {
		emptyCtx := context.Background()
		_, err := FromContext(emptyCtx)
		assert.Error(t, err)
		assert.Equal(t, ErrDBPoolNotFound, err)
	})
}
