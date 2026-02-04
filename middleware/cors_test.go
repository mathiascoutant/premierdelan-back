package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	testOriginExample   = "https://example.com"
	testOriginApp       = "https://app.example.com"
)

func TestIsOriginAllowed(t *testing.T) {
	origins := []string{testOriginExample, testOriginApp}

	tests := []struct {
		origin string
		want   bool
	}{
		{testOriginExample, true},
		{testOriginApp, true},
		{"https://evil.com", false},
		{"https://example.com.evil.com", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isOriginAllowed(tt.origin, origins)
		if got != tt.want {
			t.Errorf("isOriginAllowed(%q) = %v, want %v", tt.origin, got, tt.want)
		}
	}
}

func TestCORSAllowedOrigin(t *testing.T) {
	handler := CORS([]string{testOriginApp})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", testOriginApp)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != testOriginApp {
		t.Errorf("Access-Control-Allow-Origin = %v", got)
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Code = %v", rr.Code)
	}
}

func TestCORSOptionsPreflight(t *testing.T) {
	handler := CORS([]string{testOriginExample})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", testOriginExample)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Code = %v, attendu 204", rr.Code)
	}
}
