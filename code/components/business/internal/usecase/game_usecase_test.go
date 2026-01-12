package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"ppo/sdk/domain"
	"ppo/sdk/port/repo"
	sdkusecase "ppo/sdk/usecase"
)

type sequenceIDGen struct {
	values []string
	idx    int
}

func (g *sequenceIDGen) NewID() string {
	if g.idx >= len(g.values) {
		return "generated-id"
	}
	v := g.values[g.idx]
	g.idx++
	return v
}

type moveUseCaseStub struct {
	received sdkusecase.AddUserMoveCommand
	move     domain.Move
	err      error
}

func (m *moveUseCaseStub) AddUserMove(ctx context.Context, cmd sdkusecase.AddUserMoveCommand) (domain.Move, error) {
	m.received = cmd
	return m.move, m.err
}

func (m *moveUseCaseStub) GetOpponentMove(ctx context.Context, cmd sdkusecase.GetOpponentMoveCommand) (domain.Move, error) {
	return domain.Move{}, errors.New("not implemented")
}

func TestGameUseCaseNewInitialisesBoard(t *testing.T) {
	gameRepo := &repo.GameRepositoryMock{}
	moveRepo := &repo.MoveRepositoryMock{}

	ids := &sequenceIDGen{values: []string{"game-1"}}

	var saved domain.Game
	gameRepo.SaveFunc = func(ctx context.Context, game domain.Game) error {
		saved = game
		return nil
	}

	var boardSaved domain.BoardState
	gameRepo.UpdateBoardStateFunc = func(ctx context.Context, id domain.GameID, state domain.BoardState) error {
		boardSaved = state
		return nil
	}

	start := domain.NewTimestamp(time.Now())
	useCase := NewGameUseCase(gameRepo, moveRepo, ids)
	game, err := useCase.New(context.Background(), sdkusecase.CreateGameCommand{
		UserID:        "user-1",
		PlayerColor:   domain.PlayerColorLight,
		IsPlayerFirst: true,
		StartAt:       &start,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if game.ID != "game-1" || saved.ID != "game-1" {
		t.Fatalf("game not persisted correctly: %+v %+v", game, saved)
	}
	if len(boardSaved.Pieces) == 0 {
		t.Fatalf("expected initial board state to be saved")
	}
}

func TestGameUseCaseProcessMoveDelegatesToMoveUseCase(t *testing.T) {
	gameRepo := &repo.GameRepositoryMock{}
	moveRepo := &repo.MoveRepositoryMock{}
	moves := &moveUseCaseStub{
		move: domain.Move{ID: "move-1"},
	}

	useCase := NewGameUseCase(gameRepo, moveRepo, &sequenceIDGen{}, moves)
	result, err := useCase.ProcessMove(context.Background(), sdkusecase.ProcessMoveCommand{
		GameID:        "game-1",
		StartPosition: domain.Position{Row: 2, Col: 2},
		Trajectory:    []domain.Position{{Row: 3, Col: 3}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "move-1" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if moves.received.GameID != "game-1" || len(moves.received.Trajectory) != 1 {
		t.Fatalf("unexpected command passed to move use case: %+v", moves.received)
	}
}

func TestGameUseCaseProcessMoveRequiresGameID(t *testing.T) {
	useCase := NewGameUseCase(&repo.GameRepositoryMock{}, &repo.MoveRepositoryMock{}, &sequenceIDGen{})
	_, err := useCase.ProcessMove(context.Background(), sdkusecase.ProcessMoveCommand{})
	if err == nil {
		t.Fatalf("expected error for missing game id")
	}
}

func TestGameUseCaseProcessMoveRequiresMoveUseCase(t *testing.T) {
	useCase := NewGameUseCase(&repo.GameRepositoryMock{}, &repo.MoveRepositoryMock{}, &sequenceIDGen{})

	_, err := useCase.ProcessMove(context.Background(), sdkusecase.ProcessMoveCommand{GameID: "game-1"})

	if err == nil {
		t.Fatalf("expected error when move usecase is not configured")
	}
}

func TestGameUseCaseProcessMoveForwardedError(t *testing.T) {
	moves := &moveUseCaseStub{err: errors.New("boom")}
	useCase := NewGameUseCase(&repo.GameRepositoryMock{}, &repo.MoveRepositoryMock{}, &sequenceIDGen{}, moves)
	_, err := useCase.ProcessMove(context.Background(), sdkusecase.ProcessMoveCommand{GameID: "game"})
	if !errors.Is(err, moves.err) {
		t.Fatalf("expected error to propagate, got %v", err)
	}
}

func TestGameUseCaseFinishGame(t *testing.T) {
	game := domain.NewGame("game-1", "user-1", domain.PlayerColorLight, true, domain.NewTimestamp(time.Now()))
	gameRepo := &repo.GameRepositoryMock{}
	gameRepo.GetByIDFunc = func(ctx context.Context, id domain.GameID) (domain.Game, error) {
		return game, nil
	}

	var updated domain.Game
	gameRepo.UpdateFunc = func(ctx context.Context, g domain.Game) error {
		updated = g
		return nil
	}

	useCase := NewGameUseCase(gameRepo, &repo.MoveRepositoryMock{}, &sequenceIDGen{})
	end := domain.NewTimestamp(time.Now().Add(time.Minute))
	result, err := useCase.FinishGame(context.Background(), sdkusecase.FinishGameCommand{GameID: game.ID, End: end})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != domain.GameStatusFinished || updated.Status != domain.GameStatusFinished {
		t.Fatalf("game not marked finished")
	}
	if result.End == nil || result.End.ToTime() != end.ToTime() {
		t.Fatalf("end timestamp not propagated")
	}
}

func TestGameUseCaseFinishGameRejectsRepeat(t *testing.T) {
	game := domain.Game{ID: "game-1", Status: domain.GameStatusFinished}
	gameRepo := &repo.GameRepositoryMock{}
	gameRepo.GetByIDFunc = func(ctx context.Context, id domain.GameID) (domain.Game, error) {
		return game, nil
	}

	useCase := NewGameUseCase(gameRepo, &repo.MoveRepositoryMock{}, &sequenceIDGen{})
	_, err := useCase.FinishGame(context.Background(), sdkusecase.FinishGameCommand{GameID: game.ID, End: domain.NewTimestamp(time.Now())})
	if !errors.Is(err, ErrGameAlreadyFinished) {
		t.Fatalf("expected ErrGameAlreadyFinished, got %v", err)
	}
}

func TestGameUseCaseGenerateHistory(t *testing.T) {
	gameRepo := &repo.GameRepositoryMock{}
	moveRepo := &repo.MoveRepositoryMock{}

	expected := []domain.Move{{ID: "m1"}}
	moveRepo.ListByGameFunc = func(ctx context.Context, id domain.GameID) ([]domain.Move, error) {
		if id != "game-1" {
			t.Fatalf("unexpected id %v", id)
		}
		return expected, nil
	}

	useCase := NewGameUseCase(gameRepo, moveRepo, &sequenceIDGen{})
	moves, err := useCase.GenerateHistory(context.Background(), "game-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moves) != len(expected) {
		t.Fatalf("unexpected moves %v", moves)
	}
}

func TestGameUseCaseGenerateHistoryPropagatesError(t *testing.T) {
	moveRepo := &repo.MoveRepositoryMock{}
	expected := errors.New("list failed")
	moveRepo.ListByGameFunc = func(ctx context.Context, id domain.GameID) ([]domain.Move, error) {
		return nil, expected
	}

	useCase := NewGameUseCase(&repo.GameRepositoryMock{}, moveRepo, &sequenceIDGen{})
	_, err := useCase.GenerateHistory(context.Background(), "game-1")
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestGameUseCaseGetChessboard(t *testing.T) {
	gameRepo := &repo.GameRepositoryMock{}
	moveRepo := &repo.MoveRepositoryMock{}

	expected := domain.NewBoardState(8)
	expected.Pieces["piece-1"] = domain.BoardPiece{ID: "piece-1", Position: domain.Position{Row: 1, Col: 1}}
	gameRepo.BoardStateFunc = func(ctx context.Context, id domain.GameID) (domain.BoardState, error) {
		if id != "game-1" {
			t.Fatalf("unexpected id %v", id)
		}
		return expected, nil
	}

	useCase := NewGameUseCase(gameRepo, moveRepo, &sequenceIDGen{})
	state, err := useCase.GetChessboard(context.Background(), "game-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.Pieces) != len(expected.Pieces) {
		t.Fatalf("unexpected state %v", state)
	}
}

func TestGameUseCaseGetChessboardReturnsError(t *testing.T) {
	gameRepo := &repo.GameRepositoryMock{}
	expected := errors.New("board error")
	gameRepo.BoardStateFunc = func(ctx context.Context, id domain.GameID) (domain.BoardState, error) {
		return domain.BoardState{}, expected
	}

	useCase := NewGameUseCase(gameRepo, &repo.MoveRepositoryMock{}, &sequenceIDGen{})
	_, err := useCase.GetChessboard(context.Background(), "game-1")
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestGameUseCaseNewRequiresUserID(t *testing.T) {
	useCase := NewGameUseCase(&repo.GameRepositoryMock{}, &repo.MoveRepositoryMock{}, &sequenceIDGen{})

	_, err := useCase.New(context.Background(), sdkusecase.CreateGameCommand{})

	if err == nil {
		t.Fatalf("expected error when user id is missing")
	}
}

func TestGameUseCaseListByUser(t *testing.T) {
	gameRepo := &repo.GameRepositoryMock{}
	expected := []domain.Game{MotherGame()}
	gameRepo.ListByUserFunc = func(ctx context.Context, userID domain.UserID) ([]domain.Game, error) {
		if userID != "user-1" {
			t.Fatalf("unexpected user id %s", userID)
		}
		return expected, nil
	}

	useCase := NewGameUseCase(gameRepo, &repo.MoveRepositoryMock{}, &sequenceIDGen{})
	games, err := useCase.ListByUser(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != len(expected) {
		t.Fatalf("unexpected games %+v", games)
	}
}

func TestGameUseCaseListByUserRequiresUserID(t *testing.T) {
	useCase := NewGameUseCase(&repo.GameRepositoryMock{}, &repo.MoveRepositoryMock{}, &sequenceIDGen{})

	_, err := useCase.ListByUser(context.Background(), "")

	if err == nil {
		t.Fatalf("expected error when user id is missing")
	}
}

func TestGameUseCaseGet(t *testing.T) {
	gameRepo := &repo.GameRepositoryMock{}
	expected := MotherGame()
	gameRepo.GetByIDFunc = func(ctx context.Context, id domain.GameID) (domain.Game, error) {
		if id != expected.ID {
			t.Fatalf("unexpected id %s", id)
		}
		return expected, nil
	}

	useCase := NewGameUseCase(gameRepo, &repo.MoveRepositoryMock{}, &sequenceIDGen{})
	game, err := useCase.Get(context.Background(), expected.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if game.ID != expected.ID {
		t.Fatalf("unexpected game %+v", game)
	}
}

func TestGameUseCaseGetReturnsError(t *testing.T) {
	gameRepo := &repo.GameRepositoryMock{}
	expected := errors.New("get failed")
	gameRepo.GetByIDFunc = func(ctx context.Context, id domain.GameID) (domain.Game, error) {
		return domain.Game{}, expected
	}

	useCase := NewGameUseCase(gameRepo, &repo.MoveRepositoryMock{}, &sequenceIDGen{})
	_, err := useCase.Get(context.Background(), "game-1")
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestGameUseCaseUpdateBoardState(t *testing.T) {
	gameRepo := &repo.GameRepositoryMock{}
	var captured domain.BoardState
	gameRepo.UpdateBoardStateFunc = func(ctx context.Context, id domain.GameID, state domain.BoardState) error {
		if id != "game-1" {
			t.Fatalf("unexpected id %s", id)
		}
		captured = state
		return nil
	}

	useCase := NewGameUseCase(gameRepo, &repo.MoveRepositoryMock{}, &sequenceIDGen{})
	state := domain.NewBoardState(8)
	state.Pieces["piece-1"] = domain.BoardPiece{ID: "piece-1", Position: domain.Position{Row: 1, Col: 1}}
	if err := useCase.UpdateBoardState(context.Background(), "game-1", state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(captured.Pieces) != 1 {
		t.Fatalf("unexpected state %+v", captured)
	}
}

func TestGameUseCaseUpdateBoardStatePropagatesError(t *testing.T) {
	gameRepo := &repo.GameRepositoryMock{}
	expected := errors.New("update failed")
	gameRepo.UpdateBoardStateFunc = func(ctx context.Context, id domain.GameID, state domain.BoardState) error {
		return expected
	}

	useCase := NewGameUseCase(gameRepo, &repo.MoveRepositoryMock{}, &sequenceIDGen{})
	err := useCase.UpdateBoardState(context.Background(), "game-1", domain.NewBoardState(8))
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestGameUseCaseDelete(t *testing.T) {
	gameRepo := &repo.GameRepositoryMock{}
	moveRepo := &repo.MoveRepositoryMock{}
	deletedMoves := false
	moveRepo.DeleteByGameFunc = func(ctx context.Context, id domain.GameID) error {
		deletedMoves = true
		return nil
	}
	deletedGame := false
	gameRepo.DeleteFunc = func(ctx context.Context, id domain.GameID) error {
		deletedGame = true
		return nil
	}

	useCase := NewGameUseCase(gameRepo, moveRepo, &sequenceIDGen{})
	if err := useCase.Delete(context.Background(), "game-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deletedMoves || !deletedGame {
		t.Fatalf("expected delete to touch moves and game")
	}
}

func TestGameUseCaseDeleteRequiresID(t *testing.T) {
	useCase := NewGameUseCase(&repo.GameRepositoryMock{}, &repo.MoveRepositoryMock{}, &sequenceIDGen{})
	if err := useCase.Delete(context.Background(), ""); err == nil {
		t.Fatalf("expected error when game id missing")
	}
}
