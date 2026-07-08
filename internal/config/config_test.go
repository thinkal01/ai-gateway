package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()
	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
	if cfg.DatabasePath != "./data/gateway.db" {
		t.Errorf("expected default database path './data/gateway.db', got '%s'", cfg.DatabasePath)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default log level 'info', got '%s'", cfg.LogLevel)
	}
	if cfg.Version != "0.1.0" {
		t.Errorf("expected default version '0.1.0', got '%s'", cfg.Version)
	}
	if cfg.UsageFlushInterval != 5*time.Second {
		t.Errorf("expected default flush interval 5s, got %v", cfg.UsageFlushInterval)
	}
	if cfg.UsageFlushBatchSize != 100 {
		t.Errorf("expected default batch size 100, got %d", cfg.UsageFlushBatchSize)
	}
}

func TestLoad_WithEnv(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("DATABASE_PATH", "/tmp/test.db")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("VERSION", "2.0.0")
	os.Setenv("MOCK_PROVIDER_BASE_URL", "http://mock:8080")
	os.Setenv("API_KEY_PREFIX_LENGTH", "12")
	os.Setenv("API_KEY_HASH_ALGORITHM", "sha512")
	os.Setenv("API_KEY_SECRET_LENGTH", "64")
	os.Setenv("USAGE_FLUSH_INTERVAL", "10s")
	os.Setenv("USAGE_FLUSH_BATCH_SIZE", "200")
	os.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("DATABASE_PATH")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("VERSION")
		os.Unsetenv("MOCK_PROVIDER_BASE_URL")
		os.Unsetenv("API_KEY_PREFIX_LENGTH")
		os.Unsetenv("API_KEY_HASH_ALGORITHM")
		os.Unsetenv("API_KEY_SECRET_LENGTH")
		os.Unsetenv("USAGE_FLUSH_INTERVAL")
		os.Unsetenv("USAGE_FLUSH_BATCH_SIZE")
		os.Unsetenv("CORS_ALLOWED_ORIGINS")
	}()

	cfg := Load()
	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
	if cfg.DatabasePath != "/tmp/test.db" {
		t.Errorf("expected database path '/tmp/test.db', got '%s'", cfg.DatabasePath)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected log level 'debug', got '%s'", cfg.LogLevel)
	}
	if cfg.Version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got '%s'", cfg.Version)
	}
	if cfg.MockProviderBaseURL != "http://mock:8080" {
		t.Errorf("expected mock provider URL 'http://mock:8080', got '%s'", cfg.MockProviderBaseURL)
	}
	if cfg.APIKeyPrefixLength != 12 {
		t.Errorf("expected prefix length 12, got %d", cfg.APIKeyPrefixLength)
	}
	if cfg.APIKeyHashAlgorithm != "sha512" {
		t.Errorf("expected hash algorithm 'sha512', got '%s'", cfg.APIKeyHashAlgorithm)
	}
	if cfg.APIKeySecretLength != 64 {
		t.Errorf("expected secret length 64, got %d", cfg.APIKeySecretLength)
	}
	if cfg.UsageFlushInterval != 10*time.Second {
		t.Errorf("expected flush interval 10s, got %v", cfg.UsageFlushInterval)
	}
	if cfg.UsageFlushBatchSize != 200 {
		t.Errorf("expected batch size 200, got %d", cfg.UsageFlushBatchSize)
	}
	if cfg.CORSAllowedOrigins != "https://example.com" {
		t.Errorf("expected CORS origins 'https://example.com', got '%s'", cfg.CORSAllowedOrigins)
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_KEY", "value")
	defer os.Unsetenv("TEST_KEY")

	if v := getEnv("TEST_KEY", "default"); v != "value" {
		t.Errorf("expected 'value', got '%s'", v)
	}
	if v := getEnv("NONEXISTENT", "default"); v != "default" {
		t.Errorf("expected 'default', got '%s'", v)
	}
}

func TestGetEnvInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	if v := getEnvInt("TEST_INT", 0); v != 42 {
		t.Errorf("expected 42, got %d", v)
	}
	if v := getEnvInt("NONEXISTENT", 99); v != 99 {
		t.Errorf("expected 99, got %d", v)
	}
	// invalid int should return default
	os.Setenv("TEST_INVALID", "notanumber")
	defer os.Unsetenv("TEST_INVALID")
	if v := getEnvInt("TEST_INVALID", 10); v != 10 {
		t.Errorf("expected 10 for invalid int, got %d", v)
	}
}

func TestGetEnvDuration(t *testing.T) {
	os.Setenv("TEST_DURATION", "30s")
	defer os.Unsetenv("TEST_DURATION")

	if v := getEnvDuration("TEST_DURATION", 5*time.Second); v != 30*time.Second {
		t.Errorf("expected 30s, got %v", v)
	}
	if v := getEnvDuration("NONEXISTENT", 5*time.Second); v != 5*time.Second {
		t.Errorf("expected 5s, got %v", v)
	}
	// invalid duration should return default
	os.Setenv("TEST_INVALID_DURATION", "notaduration")
	defer os.Unsetenv("TEST_INVALID_DURATION")
	if v := getEnvDuration("TEST_INVALID_DURATION", 10*time.Second); v != 10*time.Second {
		t.Errorf("expected 10s for invalid duration, got %v", v)
	}
}
