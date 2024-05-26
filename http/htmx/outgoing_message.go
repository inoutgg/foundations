package htmx

import "net/http"

var (
	HxResponseHeaderKeyLocation    = "HX-Location"
	HxResponseHeaderKeyPushUrl     = "HX-Push-Url"
	HxResponseHeaderKeyRedirect    = "HX-Redirect"
	HxResponseHeaderKeyRefresh     = "HX-Refresh"
	HxResponseHeaderKeyReplaceUrl  = "HX-Replace-Url"
	HxResponseHeaderKeyReswap      = "HX-Reswap"
	HxResponseHeaderKeyRetarget    = "HX-Retarget"
	HxResponseHeaderKeyReselect    = "Hx-Reselect"
	HxResponseHeaderKeyTrigger     = "HX-Trigger"
	HxResponseHeaderKeyAfterSettle = "HX-Trigger-After-Settle"
	HxResponseHeaderKeyAfterSwap   = "HX-Trigger-After-Swap"
)

type OutgoingMessage struct {
	Redirect string
}

func (m *OutgoingMessage) Write(w http.ResponseWriter) {}

func Redirect(w http.ResponseWriter, r *http.Request, url string, code int) {
	incoming := FromRequest(r)

	if incoming.Request {
		outgoing := OutgoingMessage{
			Redirect: url,
		}

		outgoing.Write(w)
	} else {
		http.Redirect(w, r, url, code)
	}
}
