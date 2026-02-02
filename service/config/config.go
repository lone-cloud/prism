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
	SignalCLISocketPath string
	SignalCLIDataPath   string
	SignalCLIBinaryPath string

	ProtonIMAPUsername string
	ProtonIMAPPassword string
	ProtonBridgeHost   string
	ProtonBridgePort   int
	ProtonPrismTopic   string

	IMAPInbox                string
	IMAPSeenFlag             string
	IMAPReconnectBaseDelay   int
	IMAPMaxReconnectDelay    int
	IMAPMaxReconnectAttempts int

	EndpointPrefixProton string
	EndpointPrefixNtfy   string
	EndpointPrefixUP     string

	StoragePath string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:           getEnvInt("PORT", 8080),
		APIKey:         os.Getenv("API_KEY"),
		VerboseLogging: getEnvBool("VERBOSE_LOGGING", false),
		RateLimit:      getEnvInt("RATE_LIMIT", 100),

		DeviceName:          getEnvString("DEVICE_NAME", "Prism"),
		SignalCLISocketPath: getEnvString("SIGNAL_CLI_SOCKET", "./data/signal-cli.sock"),
		SignalCLIDataPath:   getEnvString("SIGNAL_CLI_DATA", "./data/prism"),
		SignalCLIBinaryPath: getEnvString("SIGNAL_CLI_BINARY", "./signal-cli/bin/signal-cli"),

		ProtonIMAPUsername: os.Getenv("PROTON_IMAP_USERNAME"),
		ProtonIMAPPassword: os.Getenv("PROTON_IMAP_PASSWORD"),
		ProtonBridgeHost:   getEnvString("PROTON_BRIDGE_HOST", "protonmail-bridge"),
		ProtonBridgePort:   getEnvInt("PROTON_BRIDGE_PORT", 143),
		ProtonPrismTopic:   getEnvString("PROTON_PRISM_TOPIC", "Proton Mail"),

		IMAPInbox:                "INBOX",
		IMAPSeenFlag:             "\\Seen",
		IMAPReconnectBaseDelay:   10000,
		IMAPMaxReconnectDelay:    300000,
		IMAPMaxReconnectAttempts: 50,

		EndpointPrefixProton: "proton-",
		EndpointPrefixNtfy:   "ntfy-",
		EndpointPrefixUP:     "up-",

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
	return c.ProtonIMAPUsername != "" && c.ProtonIMAPPassword != ""
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
