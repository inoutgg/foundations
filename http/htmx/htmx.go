// Package htmx provides a helper tooling to work with HTMX.
package htmx

import (
	"net/http"
	"strings"
)

var (
	HxHeaderKeyBoosted               = "HX-Boosted"
	HxHeaderKeyCurrentURL            = "HX-Current-URL"
	HxHeaderKeyHistoryRestoreRequest = "HX-History-Restore-Request"
	HxHeaderKeyPrompt                = "HX-Prompt"
	HxHeaderKeyRequest               = "HX-Request"
	HxHeaderKeyTarget                = "HX-Target"
	HxHeaderKeyTriggerName           = "HX-Trigger-Name"
	HxHeaderKeyTrigger               = "HX-Trigger"
)

// Proto represents the HTMX request headers.
//
// See: https://htmx.org/reference/#request_headers
type Proto struct {
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
func (p *Proto) Partial() bool {
	return (p.Request || p.Boosted) && !p.HistoryRestoreRequest
}

// FromRequest creates a new Proto from the given HTTP request.
func FromRequest(r *http.Request) *Proto {
	return &Proto{
		Boosted:               toBool(r.Header.Get(HxHeaderKeyBoosted)),
		CurrentURL:            r.Header.Get(HxHeaderKeyCurrentURL),
		HistoryRestoreRequest: toBool(r.Header.Get(HxHeaderKeyHistoryRestoreRequest)),
		Prompt:                r.Header.Get(HxHeaderKeyPrompt),
		Request:               toBool(r.Header.Get(HxHeaderKeyRequest)),
		Target:                r.Header.Get(HxHeaderKeyTarget),
		TriggerName:           r.Header.Get(HxHeaderKeyTriggerName),
		Trigger:               r.Header.Get(HxHeaderKeyTrigger),
	}
}

func toBool(s string) bool { return strings.EqualFold(s, "true") }
