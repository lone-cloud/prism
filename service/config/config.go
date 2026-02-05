package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port           int
	APIKey         string
	VerboseLogging bool
	RateLimit      int

	DeviceName          string
	PrismEndpointPrefix string
	EnableSignal        bool
	SignalSocket        string

	EnableProton       bool
	ProtonIMAPUsername string
	ProtonIMAPPassword string
	ProtonBridgeAddr   string

	EnableTelegram   bool
	TelegramBotToken string
	TelegramChatID   int64

	StoragePath string
}

func Load() (*Config, error) {
	cfg := &Config{
		APIKey:         os.Getenv("API_KEY"),
		Port:           getEnvInt("PORT", 8080),
		VerboseLogging: getEnvBool("VERBOSE_LOGGING", false),
		RateLimit:      getEnvInt("RATE_LIMIT", 100),
		DeviceName:     getEnvString("DEVICE_NAME", "Prism"),

		EnableSignal: getEnvBool("FEATURE_ENABLE_SIGNAL", false),
		SignalSocket: getEnvString("SIGNAL_SOCKET", "/run/signal-cli/socket"),

		EnableProton:       getEnvBool("FEATURE_ENABLE_PROTON", false),
		ProtonIMAPUsername: os.Getenv("PROTON_IMAP_USERNAME"),
		ProtonIMAPPassword: os.Getenv("PROTON_IMAP_PASSWORD"),
		ProtonBridgeAddr:   getEnvString("PROTON_BRIDGE_ADDR", "protonmail-bridge:143"),

		EnableTelegram:   getEnvBool("FEATURE_ENABLE_TELEGRAM", false),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:   getEnvInt64("TELEGRAM_CHAT_ID", 0),

		StoragePath: getEnvString("STORAGE_PATH", "./data/prism.db"),
	}

	cfg.PrismEndpointPrefix = fmt.Sprintf("[%s:", cfg.DeviceName)

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

func (c *Config) IsProtonEnabled() bool {
	return c.EnableProton && c.ProtonIMAPUsername != "" && c.ProtonIMAPPassword != "" && c.ProtonBridgeAddr != ""
}

func (c *Config) IsSignalEnabled() bool {
	return c.EnableSignal
}

func (c *Config) IsTelegramEnabled() bool {
	return c.EnableTelegram
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

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
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
