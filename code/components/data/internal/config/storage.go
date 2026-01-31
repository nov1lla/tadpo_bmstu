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

	if err := applyStorageBackendDefaults(&cfg, path); err != nil {
		return StorageConfig{}, err
	}

	return cfg, nil
}

func applyStorageBackendDefaults(cfg *StorageConfig, configPath string) error {
	switch cfg.Backend {
	case StorageBackendPostgres:
		return applyPostgresStorageDefaults(cfg)
	case StorageBackendLocal:
		return applyLocalStorageDefaults(cfg, configPath)
	default:
		return fmt.Errorf("unsupported storage backend %q", cfg.Backend)
	}
}

func applyPostgresStorageDefaults(cfg *StorageConfig) error {
	if cfg.Postgres == nil {
		cfg.Postgres = &PostgresStorageConfig{}
	}
	if cfg.Postgres.DSN == "" {
		cfg.Postgres.DSN = getenv("POSTGRES_DSN", DefaultPostgresDSN)
	}
	if cfg.Postgres.DSN == "" {
		return fmt.Errorf("postgres.dsn must be set in storage config or POSTGRES_DSN env")
	}
	return nil
}

func applyLocalStorageDefaults(cfg *StorageConfig, configPath string) error {
	if cfg.Local == nil || cfg.Local.Directory == "" {
		return fmt.Errorf("local.directory must be set for local backend")
	}
	if filepath.IsAbs(cfg.Local.Directory) {
		return nil
	}

	base := filepath.Dir(configPath)
	cfg.Local.Directory = filepath.Clean(filepath.Join(base, cfg.Local.Directory))
	return nil
}
