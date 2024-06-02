// Package htmx provides a helper tooling to work with HTMX.
package htmx

import (
	"net/http"
	"strings"
)

var (
	HxRequestHeaderKeyBoosted               = "HX-Boosted"
	HxRequestHeaderKeyCurrentURL            = "HX-Current-URL"
	HxRequestHeaderKeyHistoryRestoreRequest = "HX-History-Restore-Request"
	HxRequestHeaderKeyPrompt                = "HX-Prompt"
	HxRequestHeaderKeyRequest               = "HX-Request"
	HxRequestHeaderKeyTarget                = "HX-Target"
	HxRequestHeaderKeyTriggerName           = "HX-Trigger-Name"
	HxRequestHeaderKeyTrigger               = "HX-Trigger"
)

// IncomingMessage represents the HTMX request headers.
//
// See: https://htmx.org/reference/#request_headers
type IncomingMessage struct {
	Boosted               bool
	CurrentURL            string
	HistoryRestoreRequest bool
	Prompt                string
	Request               bool
	Target                string
	TriggerName           string
	Trigger               string
}

// Partial returns true if the request is a partial request.
func (p *IncomingMessage) Partial() bool {
	return (p.Request || p.Boosted) && !p.HistoryRestoreRequest
}

// FromRequest creates a new Proto from the given HTTP request.
func FromRequest(r *http.Request) *IncomingMessage {
	// TODO(roman@inout.gg): cache the result.
	return &IncomingMessage{
		Boosted:               toBool(r.Header.Get(HxRequestHeaderKeyBoosted)),
		CurrentURL:            r.Header.Get(HxRequestHeaderKeyCurrentURL),
		HistoryRestoreRequest: toBool(r.Header.Get(HxRequestHeaderKeyHistoryRestoreRequest)),
		Prompt:                r.Header.Get(HxRequestHeaderKeyPrompt),
		Request:               toBool(r.Header.Get(HxRequestHeaderKeyRequest)),
		Target:                r.Header.Get(HxRequestHeaderKeyTarget),
		TriggerName:           r.Header.Get(HxRequestHeaderKeyTriggerName),
		Trigger:               r.Header.Get(HxRequestHeaderKeyTrigger),
	}
}

func toBool(s string) bool { return strings.EqualFold(s, "true") }
