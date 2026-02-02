package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsOriginAllowed(t *testing.T) {
	origins := []string{"https://example.com", "https://app.example.com"}

	tests := []struct {
		origin string
		want   bool
	}{
		{"https://example.com", true},
		{"https://app.example.com", true},
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

func TestCORS_allowedOrigin(t *testing.T) {
	handler := CORS([]string{"https://app.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %v", got)
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Code = %v", rr.Code)
	}
}

func TestCORS_optionsPreflight(t *testing.T) {
	handler := CORS([]string{"https://example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Code = %v, attendu 204", rr.Code)
	}
}
