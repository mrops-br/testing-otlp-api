package config

import (
	"os"
)

type Config struct {
	Server ServerConfig
	OTLP   OTLPConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type OTLPConfig struct {
	Enabled     bool
	Endpoint    string
	ServiceName string
	Environment string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnv("SERVER_PORT", "8080"),
		},
		OTLP: OTLPConfig{
			Enabled:     getEnvBool("OTEL_ENABLED", true),
			Endpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
			ServiceName: getEnv("OTEL_SERVICE_NAME", "products-api"),
			Environment: getEnv("OTEL_ENVIRONMENT", "development"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}
