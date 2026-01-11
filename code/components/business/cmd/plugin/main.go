//go:build plugin

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"ppo/business/internal/adapter/openai"
	"ppo/business/internal/service/idgen"
	"ppo/business/internal/usecase"
	"ppo/sdk/component"
	"ppo/sdk/domain"
	sdkport "ppo/sdk/port"
	sdkusecase "ppo/sdk/usecase"
)

type provider struct {
	data    component.DataProvider
	userUC  sdkusecase.UserUseCase
	gameUC  sdkusecase.GameUseCase
	moveUC  sdkusecase.MoveUseCase
	authUC  sdkusecase.AuthUseCase
	animUC  sdkusecase.MoveAnimationUseCase
	closers []func() error
}

func (p *provider) UserUseCase() sdkusecase.UserUseCase { return p.userUC }

func (p *provider) GameUseCase() sdkusecase.GameUseCase { return p.gameUC }

func (p *provider) MoveUseCase() sdkusecase.MoveUseCase { return p.moveUC }

func (p *provider) AuthUseCase() sdkusecase.AuthUseCase { return p.authUC }

func (p *provider) MoveAnimationUseCase() sdkusecase.MoveAnimationUseCase { return p.animUC }

func (p *provider) Close() error {
	var first error
	for _, closer := range p.closers {
		if err := closer(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func NewBusinessProvider(data component.DataProvider, cfg component.BusinessConfig) (component.BusinessProvider, error) {
	if data == nil {
		return nil, fmt.Errorf("data provider required")
	}
	idGenerator := idgen.NewUUIDGenerator()
	opponent := sdkport.OpponentMoveProvider(noopOpponent{})

	if cfg.OpenAIKey != "" {
		if cfg.OpenAIModel == "" {
			return nil, errors.New("openai model is required when key is provided")
		}
		if cfg.OpenAIBaseURL == "" {
			return nil, errors.New("openai base url is required when key is provided")
		}
		timeout := cfg.HTTPTimeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		httpClient := &http.Client{Timeout: timeout}
		client, err := openai.NewClient(cfg.OpenAIKey,
			openai.WithHTTPClient(httpClient),
			openai.WithModel(cfg.OpenAIModel),
			openai.WithBaseURL(cfg.OpenAIBaseURL),
		)
		if err != nil {
			return nil, fmt.Errorf("openai client: %w", err)
		}
		opponent = client
	}

	moveUC := usecase.NewMoveUseCase(data.MoveRepository(), data.GameRepository(), opponent, idGenerator)
	gameUC := usecase.NewGameUseCase(data.GameRepository(), data.MoveRepository(), idGenerator, moveUC)
	userUC := usecase.NewUserUseCase(data.UserRepository(), idGenerator)
	authUC := usecase.NewAuthUseCase(data.UserRepository(), data.UserCredentialsRepository(), idGenerator)
	animUC := usecase.NewMoveAnimationUseCase()

	prov := &provider{
		data:   data,
		userUC: userUC,
		gameUC: gameUC,
		moveUC: moveUC,
		authUC: authUC,
		animUC: animUC,
	}
	prov.closers = append(prov.closers, data.Close)
	return prov, nil
}

type noopOpponent struct{}

func (noopOpponent) SuggestMove(context.Context, sdkport.OpponentMoveRequest) (domain.Move, error) {
	return domain.Move{}, errors.New("opponent move provider not configured")
}
