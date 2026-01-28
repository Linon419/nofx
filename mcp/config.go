package mcp

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"nofx/logger"
)

// Config client configuration (centralized management of all configurations)
type Config struct {
	// Provider configuration
	Provider string
	APIKey   string
	BaseURL  string
	Model    string

	// Behavior configuration
	MaxTokens   int
	Temperature float64
	UseFullURL  bool

	// Retry configuration
	MaxRetries      int
	RetryWaitBase   time.Duration
	RetryableErrors []string

	// Timeout configuration
	Timeout time.Duration

	// Dependency injection
	Logger     Logger
	HTTPClient *http.Client
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	timeoutSeconds := getEnvInt("AI_TIMEOUT_SECONDS", int(DefaultTimeout.Seconds()))
	timeout := time.Duration(timeoutSeconds) * time.Second
	return &Config{
		// Default values
		MaxTokens:     getEnvInt("AI_MAX_TOKENS", 2000),
		Temperature:   MCPClientTemperature,
		MaxRetries:    MaxRetryTimes,
		RetryWaitBase: 2 * time.Second,
		Timeout:       timeout,
		RetryableErrors: []string{
			"EOF",
			"timeout",
			"Timeout",
			"context deadline exceeded",
			"connection reset",
			"connection refused",
			"temporary failure",
			"no such host",
			"stream error",
			"INTERNAL_ERROR",
		},

		// Default dependencies (use global logger)
		Logger:     logger.NewMCPLogger(),
		HTTPClient: &http.Client{Timeout: timeout},
	}
}

// getEnvInt reads integer from environment variable, returns default value if failed
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultValue
}

// getEnvString reads string from environment variable, returns default value if empty
func getEnvString(key string, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
