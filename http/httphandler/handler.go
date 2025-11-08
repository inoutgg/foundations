package httphandler

import "net/http"

var (
	todoStr = []byte("todo") //nolint:gochecknoglobals
	okStr   = []byte("ok")   //nolint:gochecknoglobals
)

// TODO returns an HTTP response with "todo" body and 200 status.
//
//nolint:gochecknoglobals
var TODO = http.HandlerFunc(
	func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(todoStr)
	},
)

// HealthCheck returns an HTTP response with "ok" body and 200 status.
func HealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(okStr)
}
