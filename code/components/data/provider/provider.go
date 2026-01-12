package provider

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"

	"ppo/data/internal/config"
	localrepo "ppo/data/internal/repository/localjson"
	postgresrepo "ppo/data/internal/repository/postgres"
	"ppo/sdk/component"
	sdkrepo "ppo/sdk/port/repo"
)

type dataProvider struct {
	userRepo  sdkrepo.UserRepository
	gameRepo  sdkrepo.GameRepository
	moveRepo  sdkrepo.MoveRepository
	credsRepo sdkrepo.UserCredentialsRepository
	closer    func() error
}

func (p *dataProvider) UserRepository() sdkrepo.UserRepository { return p.userRepo }

func (p *dataProvider) GameRepository() sdkrepo.GameRepository { return p.gameRepo }

func (p *dataProvider) MoveRepository() sdkrepo.MoveRepository { return p.moveRepo }

func (p *dataProvider) UserCredentialsRepository() sdkrepo.UserCredentialsRepository {
	return p.credsRepo
}

func (p *dataProvider) Close() error {
	if p.closer != nil {
		return p.closer()
	}
	return nil
}

func NewDataProvider(source string) (component.DataProvider, error) {
	cfg, fromFile, err := loadStorageConfig(source)
	if err != nil {
		return nil, err
	}
	if fromFile {
		switch cfg.Backend {
		case config.StorageBackendPostgres:
			return NewPostgresProvider(cfg.Postgres.DSN)
		case config.StorageBackendLocal:
			return NewLocalProvider(cfg.Local.Directory)
		default:
			return nil, fmt.Errorf("unsupported storage backend %q", cfg.Backend)
		}
	}

	dsn := source
	if dsn == "" {
		dsn = os.Getenv("POSTGRES_DSN")
	}
	if dsn == "" {
		dsn = config.DefaultPostgresDSN
	}
	return NewPostgresProvider(dsn)
}

func NewPostgresProvider(dsn string) (component.DataProvider, error) {
	if dsn == "" {
		return nil, fmt.Errorf("postgres dsn is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &dataProvider{
		userRepo:  postgresrepo.NewUserRepository(db),
		gameRepo:  postgresrepo.NewGameRepository(db),
		moveRepo:  postgresrepo.NewMoveRepository(db),
		credsRepo: postgresrepo.NewUserCredentialsRepository(db),
		closer:    db.Close,
	}, nil
}

func NewLocalProvider(dir string) (component.DataProvider, error) {
	storage, err := localrepo.NewStorage(dir)
	if err != nil {
		return nil, fmt.Errorf("init local storage: %w", err)
	}
	return &dataProvider{
		userRepo:  localrepo.NewUserRepository(storage),
		gameRepo:  localrepo.NewGameRepository(storage),
		moveRepo:  localrepo.NewMoveRepository(storage),
		credsRepo: localrepo.NewUserCredentialsRepository(storage),
	}, nil
}

func loadStorageConfig(source string) (config.StorageConfig, bool, error) {
	if source == "" {
		return config.StorageConfig{}, false, nil
	}
	if strings.HasSuffix(strings.ToLower(source), ".json") {
		cfg, err := config.LoadStorageConfig(source)
		if err != nil {
			return config.StorageConfig{}, false, err
		}
		return cfg, true, nil
	}
	if info, err := os.Stat(source); err == nil && !info.IsDir() {
		cfg, err := config.LoadStorageConfig(source)
		if err != nil {
			return config.StorageConfig{}, false, err
		}
		return cfg, true, nil
	}
	return config.StorageConfig{}, false, nil
}
