package httpmiddleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMiddleware is a simple middleware that adds a header to the response
type mockMiddleware struct {
	key, value string
}

func (m mockMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add(m.key, m.value)
		next.ServeHTTP(w, r)
	})
}

func TestChain(t *testing.T) {
	tests := []struct {
		name            string
		middlewares     []Middleware
		expectedStatus  int
		expectedHeaders map[string][]string
	}{
		{
			name:            "Empty chain",
			middlewares:     []Middleware{},
			expectedStatus:  http.StatusOK,
			expectedHeaders: map[string][]string{},
		},
		{
			name: "Chain with multiple middlewares",
			middlewares: []Middleware{
				mockMiddleware{"X-Test-1", "Value1"},
				mockMiddleware{"X-Test-2", "Value2"},
			},
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string][]string{
				"X-Test-1": {"Value1"},
				"X-Test-2": {"Value2"},
			},
		},
		{
			name: "Middleware order",
			middlewares: []Middleware{
				mockMiddleware{"X-Order", "First"},
				mockMiddleware{"X-Order", "Second"},
			},
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string][]string{
				"X-Order": {"First", "Second"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := NewChain(tt.middlewares...)
			handler := chain.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.expectedStatus)
			}))

			req := httptest.NewRequest("GET", "/", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			for key, values := range tt.expectedHeaders {
				assert.Equal(t, values, rec.Header()[key])
			}
		})
	}
}

func TestChainExtend(t *testing.T) {
	m1 := mockMiddleware{"X-Test-1", "Value1"}
	m2 := mockMiddleware{"X-Test-2", "Value2"}

	chain := NewChain(m1)
	extendedChain := chain.Extend(m2)

	handler := extendedChain.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Value1", rec.Header().Get("X-Test-1"))
	assert.Equal(t, "Value2", rec.Header().Get("X-Test-2"))
}

func TestChainImplementsMiddleware(t *testing.T) {
	var _ Middleware = (*Chain)(nil)
	require.Implements(t, (*Middleware)(nil), new(Chain))
}
