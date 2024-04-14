package response

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func JSON[V any](w http.ResponseWriter, v V, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("http/response: unable to encode JSON: %w", err)
	}

	return nil
}
