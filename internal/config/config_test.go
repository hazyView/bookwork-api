package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"PORT", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD",
		"DB_NAME", "JWT_SECRET", "JWT_ISSUER",
	}

	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
	}

	// Clean up after test
	defer func() {
		for _, env := range envVars {
			if val, exists := originalEnv[env]; exists {
				os.Setenv(env, val)
			} else {
				os.Unsetenv(env)
			}
		}
	}()

	// Set test environment variables
	os.Setenv("PORT", "8080")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-characters-long")
	os.Setenv("JWT_ISSUER", "test-issuer")

	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Server.Port != "8080" {
		t.Errorf("Expected port 8080, got %s", config.Server.Port)
	}

	if config.Database.Host != "localhost" {
		t.Errorf("Expected DB host localhost, got %s", config.Database.Host)
	}

	if config.Database.Port != "5432" {
		t.Errorf("Expected DB port 5432, got %s", config.Database.Port)
	}

	if config.JWT.SecretKey != "test-secret-that-is-at-least-32-characters-long" {
		t.Errorf("Expected JWT secret test-secret-that-is-at-least-32-characters-long, got %s", config.JWT.SecretKey)
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	// Clear all environment variables
	envVars := []string{
		"PORT", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD",
		"DB_NAME", "JWT_SECRET", "JWT_ISSUER",
	}

	originalEnv := make(map[string]string)
	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
		os.Unsetenv(env)
	}

	// Clean up after test
	defer func() {
		for _, env := range envVars {
			if val, exists := originalEnv[env]; exists {
				os.Setenv(env, val)
			}
		}
	}()

	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test default values
	if config.Server.Port != "8000" {
		t.Errorf("Expected default port 8000, got %s", config.Server.Port)
	}

	if config.Database.Host != "localhost" {
		t.Errorf("Expected default DB host localhost, got %s", config.Database.Host)
	}

	if config.Database.MaxOpenConns != 25 {
		t.Errorf("Expected default max open connections 25, got %d", config.Database.MaxOpenConns)
	}

	if config.Database.MaxIdleConns != 10 {
		t.Errorf("Expected default max idle connections 10, got %d", config.Database.MaxIdleConns)
	}

	if config.Database.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("Expected default connection max lifetime 5m, got %v", config.Database.ConnMaxLifetime)
	}
}

func TestJWTSecretValidation(t *testing.T) {
	originalSecret := os.Getenv("JWT_SECRET")
	originalEnv := os.Getenv("ENV")

	defer func() {
		if originalSecret != "" {
			os.Setenv("JWT_SECRET", originalSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
		if originalEnv != "" {
			os.Setenv("ENV", originalEnv)
		} else {
			os.Unsetenv("ENV")
		}
	}()

	// Test with valid secret
	os.Unsetenv("ENV") // Ensure not in production mode
	os.Setenv("JWT_SECRET", "this-is-a-valid-secret-key-that-is-long-enough")

	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.JWT.SecretKey != "this-is-a-valid-secret-key-that-is-long-enough" {
		t.Errorf("Expected JWT secret to match set value, got %s", config.JWT.SecretKey)
	}

	// Test with no secret (should use default in dev)
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("ENV")

	config, err = Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.JWT.SecretKey == "" {
		t.Error("Expected default JWT secret in development mode")
	}
}

func TestCORSConfig(t *testing.T) {
	originalOrigins := os.Getenv("ALLOWED_ORIGINS")
	defer func() {
		if originalOrigins != "" {
			os.Setenv("ALLOWED_ORIGINS", originalOrigins)
		} else {
			os.Unsetenv("ALLOWED_ORIGINS")
		}
	}()

	os.Setenv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")

	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	expectedOrigins := []string{"http://localhost:3000", "http://localhost:5173"}
	if len(config.CORS.AllowedOrigins) != len(expectedOrigins) {
		t.Errorf("Expected %d allowed origins, got %d", len(expectedOrigins), len(config.CORS.AllowedOrigins))
	}

	for i, origin := range expectedOrigins {
		if i >= len(config.CORS.AllowedOrigins) || config.CORS.AllowedOrigins[i] != origin {
			t.Errorf("Expected origin %s at index %d, got %s", origin, i, config.CORS.AllowedOrigins[i])
		}
	}
}
