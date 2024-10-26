//go:build production

package response

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// JSON writes the provided value v as JSON to the response w.
// It sets the HTTP status code to the specified status and
// sets the Content-Type header to "application/json".
func JSON(w http.ResponseWriter, v any, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("foundations/httpresponse: unable to encode JSON: %w", err)
	}

	return nil
}
