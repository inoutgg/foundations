package csrf

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.inout.gg/common/must"
)

var checksumSecret = "really-long-and-super-protected-checksum-secret"

var safeMethods = []string{http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace}
var unsafeMethods = []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
var testHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	tok, err := FromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	http.SetCookie(w, tok.Cookie())
	w.Write([]byte("ok"))
})

func TestMethods(t *testing.T) {
	mux := http.NewServeMux()
	middleware := must.Must(Middleware(WithChecksumSecret(checksumSecret)))
	route := middleware(mux)

	mux.Handle("/", testHandler)

	for _, method := range safeMethods {
		rr := httptest.NewRecorder()
		route.ServeHTTP(rr, httptest.NewRequest(method, "/", nil))

		if rr.Code != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
		}

		response := rr.Result()
		cookies := response.Cookies()
		if len(cookies) != 1 {
			t.Fatalf("expected one cookie, got %d", len(cookies))
		}

		cookie := cookies[0]
		if !cookie.HttpOnly {
			t.Fatalf("expected cookie to be http only")
		}

		if cookie.Name != DefaultCookieName {
			t.Fatalf("expected cookie %s to be set", DefaultCookieName)
		}
	}

	for _, method := range unsafeMethods {
		rr := httptest.NewRecorder()
		route.ServeHTTP(rr, httptest.NewRequest(method, "/", nil))

		if rr.Code != http.StatusForbidden {
			t.Errorf(
				"handler returned wrong status code: got %v want %v",
				rr.Code,
				http.StatusForbidden,
			)
		}
	}
}

func TestSuccessCase(t *testing.T) {
	mux := http.NewServeMux()
	middleware := must.Must(Middleware(WithChecksumSecret(checksumSecret)))
	route := middleware(mux)
	var tok *Token
	var err error

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok, err = FromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))

	// Get CSRF token.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	route.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// Test if token is set.
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(DefaultHeaderName, tok.String())
	req.AddCookie(tok.Cookie())
	route.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}
}

func TestMismatchingTokensFailureCase(t *testing.T) {
	mux := http.NewServeMux()
	middleware := must.Must(Middleware(WithChecksumSecret(checksumSecret)))
	route := middleware(mux)
	var tok *Token
	var err error

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok, err = FromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))

	// Get CSRF token.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	route.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// Test if token is set.
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(DefaultHeaderName, "hello-world")
	req.AddCookie(tok.Cookie())
	route.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf(
			"handler returned wrong status code: got %v want %v",
			rr.Code,
			http.StatusForbidden,
		)
	}
}
