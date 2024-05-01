package ip

import (
	"net"
	"net/http"
	"strings"
)

// IPAddress returns a real IP address of the HTTP request.
//
// If X-Forwarded-For header is presented it tries to parse the IP from it.
// The first valid IP address is returned.
func IPAddress(req *http.Request) string {
	ip := ipAddressFromForwardedForHeader(req)
	if ip != "" {
		return ip
	}

	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}

	return ip
}

func ipAddressFromForwardedForHeader(req *http.Request) string {
	forwardedFor := req.Header.Get("X-Forwarded-For")
	if forwardedFor == "" {
		return ""
	}

	possibleIPs := strings.Split(forwardedFor, ",")
	for _, possibleIP := range possibleIPs {
		if possibleIP == "" {
			continue
		}

		ip := net.ParseIP(possibleIP)
		if ip != nil {
			return ip.String()
		}
	}

	return ""
}
