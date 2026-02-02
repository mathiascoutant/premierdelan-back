package middleware

import (
	"net/http"
	"net/http/httptest"
	"premier-an-backend/utils"
	"testing"
)

func TestAuth_missingToken(t *testing.T) {
	handler := Auth("test-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Code = %v, attendu 401", rr.Code)
	}
}

func TestAuth_invalidFormat(t *testing.T) {
	handler := Auth("test-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Code = %v, attendu 401", rr.Code)
	}
}

func TestAuth_validToken(t *testing.T) {
	secret := "test-secret"
	token, err := utils.GenerateToken("user1", "test@example.com", secret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	handler := Auth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetUserFromContext(r.Context())
		if claims == nil {
			t.Error("GetUserFromContext retourne nil")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if claims.UserID != "user1" || claims.Email != "test@example.com" {
			t.Errorf("claims = %+v", claims)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Code = %v, attendu 200", rr.Code)
	}
}
