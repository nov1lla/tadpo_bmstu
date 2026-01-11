package usecase

import (
	"context"
	"errors"
	"time"

	"ppo/sdk/domain"
	sdkport "ppo/sdk/port"
	sdkrepo "ppo/sdk/port/repo"
	sdkusecase "ppo/sdk/usecase"
)

var ErrGameAlreadyFinished = errors.New("game already finished")

type gameUseCase struct {
	gameRepo sdkrepo.GameRepository
	moveRepo sdkrepo.MoveRepository
	ids      sdkport.IDGenerator
	moves    sdkusecase.MoveUseCase
}

func NewGameUseCase(gameRepo sdkrepo.GameRepository, moveRepo sdkrepo.MoveRepository, ids sdkport.IDGenerator, moves ...sdkusecase.MoveUseCase) sdkusecase.GameUseCase {
	var moveSvc sdkusecase.MoveUseCase
	if len(moves) > 0 {
		moveSvc = moves[0]
	}
	return &gameUseCase{gameRepo: gameRepo, moveRepo: moveRepo, ids: ids, moves: moveSvc}
}

func (uc *gameUseCase) New(ctx context.Context, cmd sdkusecase.CreateGameCommand) (domain.Game, error) {
	if uc.ids == nil {
		return domain.Game{}, errors.New("id generator not configured")
	}
	if cmd.UserID == "" {
		return domain.Game{}, errors.New("user id is required")
	}
	start := cmd.StartAt
	if start == nil {
		now := domain.NewTimestamp(time.Now().UTC())
		start = &now
	}
	gameID := domain.GameID(uc.ids.NewID())
	game := domain.NewGame(gameID, cmd.UserID, cmd.PlayerColor, cmd.IsPlayerFirst, *start)
	if err := uc.gameRepo.Save(ctx, game); err != nil {
		return domain.Game{}, err
	}
	initialBoard := domain.NewCheckersInitialState()
	if err := uc.gameRepo.UpdateBoardState(ctx, game.ID, initialBoard); err != nil {
		return domain.Game{}, err
	}
	return game, nil
}

func (uc *gameUseCase) GenerateHistory(ctx context.Context, id domain.GameID) ([]domain.Move, error) {
	return uc.moveRepo.ListByGame(ctx, id)
}

func (uc *gameUseCase) ListByUser(ctx context.Context, userID domain.UserID) ([]domain.Game, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	return uc.gameRepo.ListByUser(ctx, userID)
}

func (uc *gameUseCase) Get(ctx context.Context, id domain.GameID) (domain.Game, error) {
	return uc.gameRepo.GetByID(ctx, id)
}

func (uc *gameUseCase) GetChessboard(ctx context.Context, id domain.GameID) (domain.BoardState, error) {
	return uc.gameRepo.BoardState(ctx, id)
}

func (uc *gameUseCase) ProcessMove(ctx context.Context, cmd sdkusecase.ProcessMoveCommand) (domain.Move, error) {
	if uc.moves == nil {
		return domain.Move{}, errors.New("move usecase not configured")
	}
	if cmd.GameID == "" {
		return domain.Move{}, errors.New("game id is required")
	}
	return uc.moves.AddUserMove(ctx, sdkusecase.AddUserMoveCommand{
		GameID:        cmd.GameID,
		StartPosition: cmd.StartPosition,
		Trajectory:    cmd.Trajectory,
		EndPosition:   cmd.EndPosition,
		PerformedAt:   cmd.PerformedAt,
	})
}

func (uc *gameUseCase) FinishGame(ctx context.Context, cmd sdkusecase.FinishGameCommand) (domain.Game, error) {
	game, err := uc.gameRepo.GetByID(ctx, cmd.GameID)
	if err != nil {
		return domain.Game{}, err
	}

	if game.Status == domain.GameStatusFinished {
		return domain.Game{}, ErrGameAlreadyFinished
	}

	finished, err := game.Finish(cmd.End)
	if err != nil {
		return domain.Game{}, err
	}

	if err := uc.gameRepo.Update(ctx, finished); err != nil {
		return domain.Game{}, err
	}

	return finished, nil
}

func (uc *gameUseCase) UpdateBoardState(ctx context.Context, id domain.GameID, state domain.BoardState) error {
	return uc.gameRepo.UpdateBoardState(ctx, id, state)
}
