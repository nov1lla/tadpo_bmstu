package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address            string
	DataPluginPath     string
	BusinessPluginPath string
	StaticDir          string
	DataSource         string
	HTTPTimeout        time.Duration
	OpenAI             OpenAIConfig
	Observability      ObservabilityConfig
	Logging            LoggingConfig
}

type OpenAIConfig struct {
	APIKey      string
	Model       string
	BaseURL     string
	Temperature float64
}

type ObservabilityConfig struct {
	Enabled      bool
	ServiceName  string
	OTLPEndpoint string
	SampleRatio  float64
}

type LoggingConfig struct {
	Level        string
	LogHTTPBody  bool
	MaxBodyBytes int
}

const (
	defaultAddress        = ":8080"
	defaultDataPlugin     = "./build/data.so"
	defaultBusinessPlugin = "./build/business.so"
	defaultStaticDir      = "../../components/webui/dist"
	defaultHTTPTimeout    = 10 * time.Second
)

func Load() Config {
	cfg := Config{
		Address:            getenv("WEBAPP_ADDR", defaultAddress),
		DataPluginPath:     getenv("DATA_PLUGIN_PATH", defaultDataPlugin),
		BusinessPluginPath: getenv("BUSINESS_PLUGIN_PATH", defaultBusinessPlugin),
		StaticDir:          expandStaticDir(getenv("WEBAPP_STATIC_DIR", defaultStaticDir)),
		DataSource:         expandDataSource(os.Getenv("DATA_SOURCE")),
		HTTPTimeout:        parseDurationEnv("HTTP_TIMEOUT", defaultHTTPTimeout),
		OpenAI: OpenAIConfig{
			APIKey:      os.Getenv("OPENAI_API_KEY"),
			Model:       os.Getenv("OPENAI_MODEL"),
			BaseURL:     os.Getenv("OPENAI_BASE_URL"),
			Temperature: parseFloatEnv("OPENAI_TEMPERATURE", 0),
		},
		Observability: ObservabilityConfig{
			Enabled:      parseBoolEnv("OTEL_ENABLED", false),
			ServiceName:  getenv("OTEL_SERVICE_NAME", "webapp"),
			OTLPEndpoint: getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"),
			SampleRatio:  parseFloatEnv("OTEL_SAMPLE_RATIO", 1.0),
		},
		Logging: LoggingConfig{
			Level:        strings.ToLower(getenv("LOG_LEVEL", "info")),
			LogHTTPBody:  parseBoolEnv("LOG_HTTP_BODY", false),
			MaxBodyBytes: parseIntEnv("LOG_MAX_BODY_BYTES", 64*1024),
		},
	}
	if cfg.DataSource == "" {
		cfg.DataSource = expandDataSource(os.Getenv("POSTGRES_DSN"))
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

func parseFloatEnv(key string, fallback float64) float64 {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	if parsed, err := strconv.ParseFloat(raw, 64); err == nil {
		return parsed
	}
	return fallback
}

func parseIntEnv(key string, fallback int) int {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	if parsed, err := strconv.Atoi(raw); err == nil {
		return parsed
	}
	return fallback
}

func parseBoolEnv(key string, fallback bool) bool {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func expandStaticDir(path string) string {
	if path == "" {
		return defaultStaticDir
	}
	if filepath.IsAbs(path) {
		return path
	}
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	return filepath.Clean(filepath.Join(cwd, path))
}

func expandDataSource(source string) string {
	if source == "" {
		return ""
	}
	if strings.Contains(source, "://") {
		return source
	}
	if filepath.IsAbs(source) {
		return source
	}
	cwd, err := os.Getwd()
	if err != nil {
		return source
	}
	return filepath.Clean(filepath.Join(cwd, source))
}
