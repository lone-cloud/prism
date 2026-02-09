package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port           int
	RateLimit      int
	APIKey         string
	StoragePath    string
	VerboseLogging bool
	EnableSignal   bool
	EnableTelegram bool
	EnableProton   bool
}

func Load() (*Config, error) {
	cfg := &Config{
		APIKey:         os.Getenv("API_KEY"),
		Port:           getEnvInt("PORT", 8080),
		VerboseLogging: getEnvBool("VERBOSE_LOGGING", false),
		RateLimit:      getEnvInt("RATE_LIMIT", 100),
		StoragePath:    getEnvString("STORAGE_PATH", "./data/prism.db"),
		EnableSignal:   getEnvBool("ENABLE_SIGNAL", false),
		EnableTelegram: getEnvBool("ENABLE_TELEGRAM", false),
		EnableProton:   getEnvBool("ENABLE_PROTON", false),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API_KEY environment variable is required")
	}
	return nil
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}
