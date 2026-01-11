package config

import (
	"os"
	"strconv"
	"time"
)

const (
	DefaultPostgresDSN        = "postgres://admin:admin123@localhost:5432/ppo_cource?sslmode=disable"
	DefaultOpenAIModel        = "gpt-4o-mini"
	DefaultOpenAIBaseURL      = "https://api.openai.com/v1/chat/completions"
	DefaultHTTPTimeout        = 10 * time.Second
	DefaultUIOperationTimeout = 5 * time.Second
	DefaultAnimationDelay     = 500 * time.Millisecond
)

type Config struct {
	PostgresDSN string
	OpenAI      OpenAIConfig
	HTTPTimeout time.Duration
	UI          UIConfig
}

type OpenAIConfig struct {
	APIKey  string
	Model   string
	BaseURL string
}

type UIConfig struct {
	OperationTimeout time.Duration
	AnimationDelay   time.Duration
	ClearScreen      bool
}

func Load() Config {
	cfg := Config{
		PostgresDSN: getenv("POSTGRES_DSN", DefaultPostgresDSN),
		HTTPTimeout: parseDurationEnv("HTTP_TIMEOUT", DefaultHTTPTimeout),
		OpenAI: OpenAIConfig{
			APIKey:  os.Getenv("OPENAI_API_KEY"),
			Model:   getenv("OPENAI_MODEL", DefaultOpenAIModel),
			BaseURL: getenv("OPENAI_BASE_URL", DefaultOpenAIBaseURL),
		},
		UI: UIConfig{
			OperationTimeout: parseDurationEnv("UI_OPERATION_TIMEOUT", DefaultUIOperationTimeout),
			AnimationDelay:   parseDurationMsEnv("ANIMATION_DELAY_MS", DefaultAnimationDelay),
			ClearScreen:      parseBoolEnv("UI_CLEAR_SCREEN", false),
		},
	}
	return cfg
}

func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	if parsed, err := time.ParseDuration(raw); err == nil {
		return parsed
	}
	if seconds, err := strconv.Atoi(raw); err == nil {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func parseDurationMsEnv(key string, fallback time.Duration) time.Duration {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	if duration, err := time.ParseDuration(raw); err == nil {
		return duration
	}
	if milliseconds, err := strconv.Atoi(raw); err == nil {
		return time.Duration(milliseconds) * time.Millisecond
	}
	return fallback
}

func parseBoolEnv(key string, fallback bool) bool {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "TRUE", "True", "yes", "on", "ON", "Yes":
		return true
	case "0", "false", "FALSE", "False", "no", "off", "OFF", "No":
		return false
	default:
		return fallback
	}
}
