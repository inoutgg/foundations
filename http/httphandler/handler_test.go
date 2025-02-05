package httphandler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTodo(t *testing.T) {
	// Create a new request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	// Call the handler
	TODO.ServeHTTP(rr, req)

	// Check status code
	assert.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	// Check response body
	body, err := io.ReadAll(rr.Body)
	assert.NoError(t, err)
	assert.Equal(t, "todo", string(body), "handler returned unexpected body")
}

func TestHealthCheck(t *testing.T) {
	// Create a new request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	// Call the handler
	HealthCheck(rr, req)

	// Check status code
	assert.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	// Check response body
	body, err := io.ReadAll(rr.Body)
	assert.NoError(t, err)
	assert.Equal(t, "ok", string(body), "handler returned unexpected body")
}
