package config

import (
	"os"
	"strconv"
	"time"
)

// Config 应用配置
type Config struct {
	// 数据库
	DatabasePath string

	// 服务器
	Port     int
	LogLevel string
	Version  string

	// Mock Provider
	MockProviderBaseURL string

	// 安全
	APIKeyPrefixLength  int
	APIKeyHashAlgorithm string
	APIKeySecretLength  int

	// 用量记录
	UsageFlushInterval  time.Duration
	UsageFlushBatchSize int

	// CORS
	CORSAllowedOrigins string
}

// Load 从环境变量加载配置
func Load() *Config {
	return &Config{
		DatabasePath: getEnv("DATABASE_PATH", "./data/gateway.db"),

		Port:     getEnvInt("PORT", 8080),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		Version:  getEnv("VERSION", "0.1.0"),

		MockProviderBaseURL: getEnv("MOCK_PROVIDER_BASE_URL", "http://localhost:9090"),

		APIKeyPrefixLength:  getEnvInt("API_KEY_PREFIX_LENGTH", 8),
		APIKeyHashAlgorithm: getEnv("API_KEY_HASH_ALGORITHM", "sha256"),
		APIKeySecretLength:  getEnvInt("API_KEY_SECRET_LENGTH", 32),

		UsageFlushInterval:  getEnvDuration("USAGE_FLUSH_INTERVAL", 5*time.Second),
		UsageFlushBatchSize: getEnvInt("USAGE_FLUSH_BATCH_SIZE", 100),

		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
