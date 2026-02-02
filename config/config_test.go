package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Sauvegarder et restaurer les variables d'environnement
	origJWT := os.Getenv("JWT_SECRET")
	origPort := os.Getenv("PORT")
	defer func() {
		if origJWT != "" {
			os.Setenv("JWT_SECRET", origJWT)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
		if origPort != "" {
			os.Setenv("PORT", origPort)
		} else {
			os.Unsetenv("PORT")
		}
	}()

	t.Run("erreur sans JWT_SECRET", func(t *testing.T) {
		os.Unsetenv("JWT_SECRET")
		_, err := Load()
		if err == nil {
			t.Error("Load() devrait échouer sans JWT_SECRET")
		}
		if err != nil && err.Error() != "JWT_SECRET est requis" {
			t.Errorf("Load() erreur = %v, attendu 'JWT_SECRET est requis'", err)
		}
	})

	t.Run("succès avec JWT_SECRET", func(t *testing.T) {
		os.Setenv("JWT_SECRET", "test-secret")
		os.Unsetenv("PORT")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() erreur = %v", err)
		}
		if cfg.JWTSecret != "test-secret" {
			t.Errorf("JWTSecret = %v, attendu test-secret", cfg.JWTSecret)
		}
		if cfg.Port != "8090" {
			t.Errorf("Port = %v, attendu 8090 (défaut)", cfg.Port)
		}
	})

	t.Run("PORT depuis env", func(t *testing.T) {
		os.Setenv("JWT_SECRET", "test-secret")
		os.Setenv("PORT", "9999")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() erreur = %v", err)
		}
		if cfg.Port != "9999" {
			t.Errorf("Port = %v, attendu 9999", cfg.Port)
		}
	})

	t.Run("CORS parsing", func(t *testing.T) {
		os.Setenv("JWT_SECRET", "test-secret")
		os.Setenv("CORS_ALLOWED_ORIGINS", "http://a.com, http://b.com , c.com")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() erreur = %v", err)
		}
		if len(cfg.CORSOrigins) != 3 {
			t.Errorf("CORSOrigins = %v, attendu 3 éléments", cfg.CORSOrigins)
		}
	})
}
