//go:build integration
// +build integration

package integration

import (
	"context"
	"errors"
	"testing"

	"ppo/business/internal/service/idgen"
	"ppo/business/internal/usecase"
	"ppo/data/provider"
	"ppo/data/testsupport"
	"ppo/sdk/domain"
	"ppo/sdk/port"
	sdkusecase "ppo/sdk/usecase"
)

func TestBusinessIntegrationUserMoveFlow(t *testing.T) {
	env := testsupport.SetupPostgres(t)
	defer env.Cleanup()

	dataProvider, err := provider.NewPostgresProvider(env.DSN)
	if err != nil {
		t.Fatalf("data provider: %v", err)
	}
	defer func() { _ = dataProvider.Close() }()

	ids := idgen.NewUUIDGenerator()
	opponent := &port.OpponentMoveProviderMock{
		SuggestMoveFunc: func(ctx context.Context, req port.OpponentMoveRequest) (domain.Move, error) {
			return domain.Move{}, errors.New("opponent not configured")
		},
	}

	moveUC := usecase.NewMoveUseCase(dataProvider.MoveRepository(), dataProvider.GameRepository(), opponent, ids)
	gameUC := usecase.NewGameUseCase(dataProvider.GameRepository(), dataProvider.MoveRepository(), ids, moveUC)
	userUC := usecase.NewUserUseCase(dataProvider.UserRepository(), ids)

	user, err := userUC.Create(context.Background(), sdkusecase.CreateUserCommand{Name: "Integration User"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	game, err := gameUC.New(context.Background(), sdkusecase.CreateGameCommand{
		UserID:        user.ID,
		PlayerColor:   domain.PlayerColorLight,
		IsPlayerFirst: true,
	})
	if err != nil {
		t.Fatalf("create game: %v", err)
	}

	end := domain.Position{Row: 4, Col: 1}
	_, err = gameUC.ProcessMove(context.Background(), sdkusecase.ProcessMoveCommand{
		GameID:        game.ID,
		StartPosition: domain.Position{Row: 5, Col: 0},
		Trajectory:    []domain.Position{end},
		EndPosition:   &end,
	})
	if err != nil {
		t.Fatalf("process move: %v", err)
	}

	history, err := gameUC.GenerateHistory(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 move, got %d", len(history))
	}
}
