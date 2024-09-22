package httphandler

import "net/http"

var todoStr = []byte("todo")

// TODO returns an HTTP response with "todo" body and 200 status.
var TODO http.HandlerFunc = http.HandlerFunc(
	func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(todoStr)
	},
)
