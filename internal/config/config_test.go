package config

import (
	"os"
	"testing"
)

func TestLoadWithoutConfigFile(t *testing.T) {
	// Set required env vars
	os.Setenv("DATAMAP_AUTH_JWT_SECRET", "test-secret-key-32-characters-long")
	defer os.Unsetenv("DATAMAP_AUTH_JWT_SECRET")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify defaults
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Server.Port)
	}

	if cfg.Database.Type != "sqlite" {
		t.Errorf("Expected database type sqlite, got %s", cfg.Database.Type)
	}

	if cfg.Database.SQLitePath != "./data/datamap.db" {
		t.Errorf("Expected SQLite path ./data/datamap.db, got %s", cfg.Database.SQLitePath)
	}

	if cfg.Auth.JWTSecret != "test-secret-key-32-characters-long" {
		t.Errorf("Expected JWT secret from env var")
	}
}

func TestLoadWithEnvOverride(t *testing.T) {
	os.Setenv("DATAMAP_AUTH_JWT_SECRET", "test-secret")
	os.Setenv("DATAMAP_SERVER_PORT", "9000")
	os.Setenv("DATAMAP_DATABASE_SQLITE_PATH", "/tmp/test.db")
	defer func() {
		os.Unsetenv("DATAMAP_AUTH_JWT_SECRET")
		os.Unsetenv("DATAMAP_SERVER_PORT")
		os.Unsetenv("DATAMAP_DATABASE_SQLITE_PATH")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", cfg.Server.Port)
	}

	if cfg.Database.SQLitePath != "/tmp/test.db" {
		t.Errorf("Expected SQLite path /tmp/test.db, got %s", cfg.Database.SQLitePath)
	}
}

func TestValidateAllowsMissingJWTSecretWhenAuthDisabled(t *testing.T) {
	cfg := &Config{
		Server:   ServerConfig{Port: 8080},
		Database: DatabaseConfig{Type: "sqlite"},
		Log:      LogConfig{Level: "info"},
		Auth:     AuthConfig{Enabled: false, JWTSecret: ""},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected validation to pass when auth is disabled, got %v", err)
	}
}

func TestValidateRequiresJWTSecretWhenAuthEnabled(t *testing.T) {
	cfg := &Config{
		Server:   ServerConfig{Port: 8080},
		Database: DatabaseConfig{Type: "sqlite"},
		Log:      LogConfig{Level: "info"},
		Auth:     AuthConfig{Enabled: true, JWTSecret: ""},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected validation error for missing JWT secret")
	}
}
