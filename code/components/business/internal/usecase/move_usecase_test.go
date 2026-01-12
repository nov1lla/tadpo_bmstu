package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"ppo/sdk/domain"
	"ppo/sdk/port"
	"ppo/sdk/port/repo"
	sdkusecase "ppo/sdk/usecase"
)

type stubIDGen struct {
	values []string
	idx    int
}

func (g *stubIDGen) NewID() string {
	if g.idx >= len(g.values) {
		return "generated"
	}
	v := g.values[g.idx]
	g.idx++
	return v
}

func TestMoveUseCaseAddUserMove_SimpleStep(t *testing.T) {
	moveRepo := &repo.MoveRepositoryMock{}
	gameRepo := &repo.GameRepositoryMock{}
	opponent := &port.OpponentMoveProviderMock{}

	game := domain.NewGame("game-1", "user-1", domain.PlayerColorLight, true, domain.NewTimestamp(time.Now()))
	board := domain.NewBoardState(8)
	board.Pieces["piece-1"] = domain.BoardPiece{
		ID:       "piece-1",
		Position: domain.Position{Row: 5, Col: 2},
		Color:    domain.PlayerColorLight,
		Kind:     domain.PieceKindMan,
	}
	board.Pieces["opponent-1"] = domain.BoardPiece{
		ID:       "opponent-1",
		Position: domain.Position{Row: 2, Col: 1},
		Color:    domain.PlayerColorDark,
		Kind:     domain.PieceKindMan,
	}

	gameRepo.GetByIDFunc = func(ctx context.Context, id domain.GameID) (domain.Game, error) {
		return game, nil
	}
	gameRepo.BoardStateFunc = func(ctx context.Context, id domain.GameID) (domain.BoardState, error) {
		return board, nil
	}

	updatedCalled := false
	gameRepo.UpdateFunc = func(ctx context.Context, g domain.Game) error {
		if g.Status != domain.GameStatusInProgress {
			t.Fatalf("expected game to be marked in_progress, got %v", g.Status)
		}
		updatedCalled = true
		return nil
	}

	var updatedBoard domain.BoardState
	gameRepo.UpdateBoardStateFunc = func(ctx context.Context, id domain.GameID, state domain.BoardState) error {
		updatedBoard = state
		return nil
	}

	moveRepo.ListByGameFunc = func(ctx context.Context, id domain.GameID) ([]domain.Move, error) {
		return nil, nil
	}

	var persisted domain.Move
	moveRepo.AddFunc = func(ctx context.Context, move domain.Move) error {
		persisted = move
		return nil
	}

	uc := NewMoveUseCase(moveRepo, gameRepo, opponent, &stubIDGen{values: []string{"move-1"}})
	end := domain.Position{Row: 4, Col: 3}
	move, err := uc.AddUserMove(context.Background(), sdkusecase.AddUserMoveCommand{
		GameID:        game.ID,
		StartPosition: domain.Position{Row: 5, Col: 2},
		EndPosition:   &end,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if move.ID != "move-1" || persisted.ID != "move-1" {
		t.Fatalf("expected generated id to be used: %+v %+v", move, persisted)
	}
	if len(move.Trajectory) != 1 || move.Trajectory[0] != end {
		t.Fatalf("unexpected trajectory: %+v", move.Trajectory)
	}
	if !updatedCalled {
		t.Fatalf("expected game update to be invoked")
	}
	updatedPiece, ok := updatedBoard.Pieces["piece-1"]
	if !ok || updatedPiece.Position != end {
		t.Fatalf("expected piece to move on board, got %+v", updatedPiece)
	}
}

func TestMoveUseCaseAddUserMoveRequiresGameID(t *testing.T) {
	uc := NewMoveUseCase(&repo.MoveRepositoryMock{}, &repo.GameRepositoryMock{}, &port.OpponentMoveProviderMock{}, &stubIDGen{})

	_, err := uc.AddUserMove(context.Background(), sdkusecase.AddUserMoveCommand{})

	if err == nil {
		t.Fatalf("expected error when game id is missing")
	}
}

func TestMoveUseCaseGetOpponentMove(t *testing.T) {
	moveRepo := &repo.MoveRepositoryMock{}
	gameRepo := &repo.GameRepositoryMock{}
	opponent := &port.OpponentMoveProviderMock{}

	game := domain.NewGame("game-1", "user-1", domain.PlayerColorLight, true, domain.NewTimestamp(time.Now()))
	board := domain.NewBoardState(8)
	board.Pieces["dark-1"] = domain.BoardPiece{
		ID:       "dark-1",
		Position: domain.Position{Row: 2, Col: 3},
		Color:    domain.PlayerColorDark,
		Kind:     domain.PieceKindMan,
	}
	board.Pieces["light-1"] = domain.BoardPiece{
		ID:       "light-1",
		Position: domain.Position{Row: 3, Col: 4},
		Color:    domain.PlayerColorLight,
		Kind:     domain.PieceKindMan,
	}

	gameRepo.GetByIDFunc = func(ctx context.Context, id domain.GameID) (domain.Game, error) {
		return game, nil
	}
	gameRepo.BoardStateFunc = func(ctx context.Context, id domain.GameID) (domain.BoardState, error) {
		return board, nil
	}
	gameRepo.UpdateBoardStateFunc = func(ctx context.Context, id domain.GameID, state domain.BoardState) error {
		board = state
		return nil
	}

	moveRepo.ListByGameFunc = func(ctx context.Context, id domain.GameID) ([]domain.Move, error) {
		return []domain.Move{{Number: 1}}, nil
	}

	var persisted domain.Move
	moveRepo.AddFunc = func(ctx context.Context, move domain.Move) error {
		persisted = move
		return nil
	}

	opponent.SuggestMoveFunc = func(ctx context.Context, req port.OpponentMoveRequest) (domain.Move, error) {
		return domain.Move{
			PieceID:       "dark-1",
			StartPosition: domain.Position{Row: 2, Col: 3},
			Trajectory:    []domain.Position{{Row: 4, Col: 5}},
			EndPosition:   domain.Position{Row: 4, Col: 5},
		}, nil
	}

	uc := NewMoveUseCase(moveRepo, gameRepo, opponent, &stubIDGen{values: []string{"move-ai"}})
	move, err := uc.GetOpponentMove(context.Background(), sdkusecase.GetOpponentMoveCommand{
		GameID:     game.ID,
		Difficulty: domain.DifficultyMedium,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if move.ID != "move-ai" || persisted.ID != "move-ai" {
		t.Fatalf("expected generated id, got move=%+v persisted=%+v", move, persisted)
	}
	if len(move.Trajectory) != 1 || move.EndPosition != (domain.Position{Row: 4, Col: 5}) {
		t.Fatalf("unexpected move: %+v", move)
	}
}

func TestMoveUseCaseGetOpponentMove_CaptureFallback(t *testing.T) {
	moveRepo := &repo.MoveRepositoryMock{}
	gameRepo := &repo.GameRepositoryMock{}
	opponent := &port.OpponentMoveProviderMock{}

	game := domain.NewGame("game-2", "user-1", domain.PlayerColorLight, true, domain.NewTimestamp(time.Now()))
	board := domain.NewBoardState(8)
	board.Pieces["dark-capturer"] = domain.BoardPiece{
		ID:       "dark-capturer",
		Position: domain.Position{Row: 2, Col: 1},
		Color:    domain.PlayerColorDark,
		Kind:     domain.PieceKindMan,
	}
	board.Pieces["light-target"] = domain.BoardPiece{
		ID:       "light-target",
		Position: domain.Position{Row: 3, Col: 2},
		Color:    domain.PlayerColorLight,
		Kind:     domain.PieceKindMan,
	}

	gameRepo.GetByIDFunc = func(ctx context.Context, id domain.GameID) (domain.Game, error) {
		return game, nil
	}
	gameRepo.BoardStateFunc = func(ctx context.Context, id domain.GameID) (domain.BoardState, error) {
		return board, nil
	}
	var updatedBoard domain.BoardState
	gameRepo.UpdateBoardStateFunc = func(ctx context.Context, id domain.GameID, state domain.BoardState) error {
		updatedBoard = state
		return nil
	}

	moveRepo.ListByGameFunc = func(ctx context.Context, id domain.GameID) ([]domain.Move, error) {
		return []domain.Move{{Number: 1}}, nil
	}
	moveRepo.AddFunc = func(ctx context.Context, move domain.Move) error {
		return nil
	}

	opponent.SuggestMoveFunc = func(ctx context.Context, req port.OpponentMoveRequest) (domain.Move, error) {
		return domain.Move{
			PieceID:       "dark-capturer",
			StartPosition: domain.Position{Row: 2, Col: 1},
			Trajectory: []domain.Position{
				{Row: 3, Col: 2},
				{Row: 4, Col: 3},
			},
			EndPosition: domain.Position{Row: 4, Col: 3},
		}, nil
	}

	uc := NewMoveUseCase(moveRepo, gameRepo, opponent, &stubIDGen{values: []string{"move-capture"}})
	move, err := uc.GetOpponentMove(context.Background(), sdkusecase.GetOpponentMoveCommand{
		GameID:     game.ID,
		Difficulty: domain.DifficultyHard,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if move.EndPosition != (domain.Position{Row: 4, Col: 3}) {
		t.Fatalf("unexpected end position: %+v", move.EndPosition)
	}
	if _, exists := updatedBoard.Pieces["light-target"]; exists {
		t.Fatalf("expected captured piece to be removed")
	}
	landing, ok := updatedBoard.Pieces["dark-capturer"]
	if !ok || landing.Position != (domain.Position{Row: 4, Col: 3}) {
		t.Fatalf("piece not moved as expected: %+v", landing)
	}
}

func TestMoveUseCaseGetOpponentMovePropagatesErrors(t *testing.T) {
	moveRepo := &repo.MoveRepositoryMock{}
	gameRepo := &repo.GameRepositoryMock{}
	opponent := &port.OpponentMoveProviderMock{}

	expected := errors.New("boom")
	gameRepo.GetByIDFunc = func(ctx context.Context, id domain.GameID) (domain.Game, error) {
		return domain.Game{}, expected
	}

	uc := NewMoveUseCase(moveRepo, gameRepo, opponent, &stubIDGen{})
	_, err := uc.GetOpponentMove(context.Background(), sdkusecase.GetOpponentMoveCommand{GameID: "game"})
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}
