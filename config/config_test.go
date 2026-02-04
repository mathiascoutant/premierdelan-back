package config

import (
	"os"
	"testing"
)

const testSecret = "test-secret"
const loadErrFmt = "Load() erreur = %v"

func TestLoad(t *testing.T) {
	origJWT := os.Getenv("JWT_SECRET")
	origPort := os.Getenv("PORT")
	defer restoreEnv(origJWT, origPort)

	t.Run("erreur sans JWT_SECRET", func(t *testing.T) {
		os.Unsetenv("JWT_SECRET")
		_, err := Load()
		assertLoadError(t, err, "JWT_SECRET est requis")
	})

	t.Run("succès avec JWT_SECRET", func(t *testing.T) {
		os.Setenv("JWT_SECRET", testSecret)
		os.Unsetenv("PORT")
		cfg, err := Load()
		assertNoError(t, err)
		assertJWTSecret(t, cfg, testSecret)
		assertPort(t, cfg, "8090")
	})

	t.Run("PORT depuis env", func(t *testing.T) {
		os.Setenv("JWT_SECRET", testSecret)
		os.Setenv("PORT", "9999")
		cfg, err := Load()
		assertNoError(t, err)
		assertPort(t, cfg, "9999")
	})

	t.Run("CORS parsing", func(t *testing.T) {
		os.Setenv("JWT_SECRET", testSecret)
		os.Setenv("CORS_ALLOWED_ORIGINS", "http://a.com, http://b.com , c.com")
		cfg, err := Load()
		assertNoError(t, err)
		assertCORSOrigins(t, cfg, 3)
	})
}

func restoreEnv(origJWT, origPort string) {
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
}

func assertLoadError(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil {
		t.Error("Load() devrait échouer sans JWT_SECRET")
		return
	}
	if err.Error() != expected {
		t.Errorf(loadErrFmt+", attendu %q", err, expected)
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf(loadErrFmt, err)
	}
}

func assertJWTSecret(t *testing.T, cfg *Config, expected string) {
	t.Helper()
	if cfg.JWTSecret != expected {
		t.Errorf("JWTSecret = %v, attendu %s", cfg.JWTSecret, expected)
	}
}

func assertPort(t *testing.T, cfg *Config, expected string) {
	t.Helper()
	if cfg.Port != expected {
		t.Errorf("Port = %v, attendu %s", cfg.Port, expected)
	}
}

func assertCORSOrigins(t *testing.T, cfg *Config, expected int) {
	t.Helper()
	if len(cfg.CORSOrigins) != expected {
		t.Errorf("CORSOrigins = %v, attendu %d éléments", cfg.CORSOrigins, expected)
	}
}
