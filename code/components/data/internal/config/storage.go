package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type StorageBackend string

const (
	StorageBackendPostgres StorageBackend = "postgres"
	StorageBackendLocal    StorageBackend = "local"
)

type StorageConfig struct {
	Backend  StorageBackend         `json:"backend"`
	Postgres *PostgresStorageConfig `json:"postgres,omitempty"`
	Local    *LocalStorageConfig    `json:"local,omitempty"`
}

type PostgresStorageConfig struct {
	DSN string `json:"dsn"`
}

type LocalStorageConfig struct {
	Directory string `json:"directory"`
}

func LoadStorageConfig(path string) (StorageConfig, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return StorageConfig{}, fmt.Errorf("read storage config: %w", err)
	}

	var cfg StorageConfig
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return StorageConfig{}, fmt.Errorf("parse storage config: %w", err)
	}

	if cfg.Backend == "" {
		cfg.Backend = StorageBackendPostgres
	}

	switch cfg.Backend {
	case StorageBackendPostgres:
		if cfg.Postgres == nil {
			cfg.Postgres = &PostgresStorageConfig{}
		}
		if cfg.Postgres.DSN == "" {
			cfg.Postgres.DSN = getenv("POSTGRES_DSN", DefaultPostgresDSN)
		}
		if cfg.Postgres.DSN == "" {
			return StorageConfig{}, fmt.Errorf("postgres.dsn must be set in storage config or POSTGRES_DSN env")
		}
	case StorageBackendLocal:
		if cfg.Local == nil || cfg.Local.Directory == "" {
			return StorageConfig{}, fmt.Errorf("local.directory must be set for local backend")
		}
		if !filepath.IsAbs(cfg.Local.Directory) {
			base := filepath.Dir(path)
			cfg.Local.Directory = filepath.Clean(filepath.Join(base, cfg.Local.Directory))
		}
	default:
		return StorageConfig{}, fmt.Errorf("unsupported storage backend %q", cfg.Backend)
	}

	return cfg, nil
}
