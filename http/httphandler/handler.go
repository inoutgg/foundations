package httphandler

import "net/http"

var (
	todoStr = []byte("todo")
	okStr   = []byte("ok")
)

// TODO returns an HTTP response with "todo" body and 200 status.
var TODO http.HandlerFunc = http.HandlerFunc(
	func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(todoStr)
	},
)

// HealthCheck returns an HTTP response with "ok" body and 200 status.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(okStr)
}
