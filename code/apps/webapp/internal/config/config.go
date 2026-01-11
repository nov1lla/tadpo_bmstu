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
}

type OpenAIConfig struct {
	APIKey  string
	Model   string
	BaseURL string
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
			APIKey:  os.Getenv("OPENAI_API_KEY"),
			Model:   os.Getenv("OPENAI_MODEL"),
			BaseURL: os.Getenv("OPENAI_BASE_URL"),
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
