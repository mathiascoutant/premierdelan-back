package utils

import (
	"testing"
)

func TestGenerateToken(t *testing.T) {
	secret := "test-secret-key"
	userID := "user123"
	email := "test@example.com"

	token, err := GenerateToken(userID, email, secret)
	if err != nil {
		t.Fatalf("GenerateToken() erreur = %v", err)
	}
	if token == "" {
		t.Error("GenerateToken() ne doit pas retourner une chaîne vide")
	}
}

func TestValidateToken(t *testing.T) {
	secret := "test-secret-key"
	userID := "user456"
	email := "valid@example.com"

	token, err := GenerateToken(userID, email, secret)
	if err != nil {
		t.Fatalf("GenerateToken() erreur = %v", err)
	}

	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken() erreur = %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID = %v, attendu %v", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Errorf("Email = %v, attendu %v", claims.Email, email)
	}
}

func TestValidateTokenMauvaisSecret(t *testing.T) {
	token, _ := GenerateToken("u", "e@e.com", "secret1")
	_, err := ValidateToken(token, "secret2")
	if err == nil {
		t.Error("ValidateToken() devrait échouer avec un mauvais secret")
	}
}

func TestValidateTokenInvalide(t *testing.T) {
	_, err := ValidateToken("invalid-token", "secret")
	if err == nil {
		t.Error("ValidateToken() devrait échouer avec un token invalide")
	}
}
