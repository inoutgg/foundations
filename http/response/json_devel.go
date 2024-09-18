//go:build !production

package response

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	// 4 spaces
	JSONIdentation = "    "
)

func JSON(w http.ResponseWriter, v any, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetIndent("", JSONIdentation)

	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("http/response: unable to encode JSON: %w", err)
	}

	return nil
}
