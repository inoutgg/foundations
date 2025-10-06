//go:build !production

package httpresponse

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	// 4 spaces.
	JSONIndentation = "    "
)

func JSON(w http.ResponseWriter, v any, status int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.SetIndent("", JSONIndentation)

	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("foundations/httpresponse: unable to encode JSON: %w", err)
	}

	return nil
}
