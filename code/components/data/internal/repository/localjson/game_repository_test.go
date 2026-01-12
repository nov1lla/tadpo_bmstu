package localjson

import (
	"errors"
	"testing"
	"time"

	"ppo/sdk/domain"
	"ppo/sdk/port/repo"
)

func TestGameRepositorySave_OK(t *testing.T) {
	fixture := newGameRepoFixture(t)
	game := NewGameBuilder().WithID("game-1").WithUserID("user-1").Build()

	err := fixture.repo.Save(fixture.ctx, game)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	loaded, err := fixture.repo.GetByID(fixture.ctx, game.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if loaded.ID != game.ID {
		t.Fatalf("unexpected game: %+v", loaded)
	}
}

func TestGameRepositorySave_RejectsInvalidID(t *testing.T) {
	fixture := newGameRepoFixture(t)
	game := NewGameBuilder().WithID("").Build()

	err := fixture.repo.Save(fixture.ctx, game)

	if !errors.Is(err, repo.ErrInvalidData) {
		t.Fatalf("expected ErrInvalidData, got %v", err)
	}
}

func TestGameRepositoryGetByID_NotFound(t *testing.T) {
	fixture := newGameRepoFixture(t)

	_, err := fixture.repo.GetByID(fixture.ctx, "missing")

	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGameRepositoryListByUser_OK(t *testing.T) {
	fixture := newGameRepoFixture(t)
	first := NewGameBuilder().WithID("game-1").WithUserID("user-1").Build()
	secondStart := domain.NewTimestamp(time.Date(2024, 2, 1, 11, 0, 0, 0, time.UTC))
	second := NewGameBuilder().WithID("game-2").WithUserID("user-1").WithStart(secondStart).Build()
	if err := fixture.repo.Save(fixture.ctx, first); err != nil {
		t.Fatalf("save first: %v", err)
	}
	if err := fixture.repo.Save(fixture.ctx, second); err != nil {
		t.Fatalf("save second: %v", err)
	}

	games, err := fixture.repo.ListByUser(fixture.ctx, "user-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(games) != 2 || games[0].ID != second.ID {
		t.Fatalf("unexpected list: %+v", games)
	}
}

func TestGameRepositoryListByUser_ContextCanceled(t *testing.T) {
	fixture := newGameRepoFixture(t)
	ctx := canceledContext(t)

	_, err := fixture.repo.ListByUser(ctx, "user-1")

	if err == nil {
		t.Fatalf("expected context error")
	}
}

func TestGameRepositoryUpdate_OK(t *testing.T) {
	fixture := newGameRepoFixture(t)
	game := MotherGame()
	if err := fixture.repo.Save(fixture.ctx, game); err != nil {
		t.Fatalf("save: %v", err)
	}
	game.Status = domain.GameStatusInProgress

	err := fixture.repo.Update(fixture.ctx, game)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	loaded, err := fixture.repo.GetByID(fixture.ctx, game.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if loaded.Status != domain.GameStatusInProgress {
		t.Fatalf("unexpected status: %+v", loaded)
	}
}

func TestGameRepositoryUpdate_NotFound(t *testing.T) {
	fixture := newGameRepoFixture(t)
	game := NewGameBuilder().WithID("missing").Build()

	err := fixture.repo.Update(fixture.ctx, game)

	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGameRepositoryDelete_OK(t *testing.T) {
	fixture := newGameRepoFixture(t)
	game := NewGameBuilder().WithID("game-delete").Build()
	if err := fixture.repo.Save(fixture.ctx, game); err != nil {
		t.Fatalf("save: %v", err)
	}

	err := fixture.repo.Delete(fixture.ctx, game.ID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = fixture.repo.GetByID(fixture.ctx, game.ID)
	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestGameRepositoryDelete_NotFound(t *testing.T) {
	fixture := newGameRepoFixture(t)

	err := fixture.repo.Delete(fixture.ctx, "missing")

	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGameRepositoryBoardState_OK(t *testing.T) {
	fixture := newGameRepoFixture(t)
	game := NewGameBuilder().WithID("game-board").Build()
	if err := fixture.repo.Save(fixture.ctx, game); err != nil {
		t.Fatalf("save: %v", err)
	}
	state := domain.NewBoardState(8)
	state.Pieces["piece-1"] = domain.BoardPiece{ID: "piece-1", Position: domain.Position{Row: 1, Col: 1}}
	if err := fixture.repo.UpdateBoardState(fixture.ctx, game.ID, state); err != nil {
		t.Fatalf("update board: %v", err)
	}

	loaded, err := fixture.repo.BoardState(fixture.ctx, game.ID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded.Pieces) != 1 {
		t.Fatalf("unexpected board state: %+v", loaded)
	}
}

func TestGameRepositoryBoardState_NotFound(t *testing.T) {
	fixture := newGameRepoFixture(t)

	_, err := fixture.repo.BoardState(fixture.ctx, "missing")

	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGameRepositoryUpdateBoardState_OK(t *testing.T) {
	fixture := newGameRepoFixture(t)
	game := NewGameBuilder().WithID("game-board-update").Build()
	if err := fixture.repo.Save(fixture.ctx, game); err != nil {
		t.Fatalf("save: %v", err)
	}
	state := domain.NewBoardState(8)
	state.Pieces["piece-1"] = domain.BoardPiece{ID: "piece-1", Position: domain.Position{Row: 2, Col: 2}}

	err := fixture.repo.UpdateBoardState(fixture.ctx, game.ID, state)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	loaded, err := fixture.repo.BoardState(fixture.ctx, game.ID)
	if err != nil {
		t.Fatalf("board state: %v", err)
	}
	if loaded.Size != state.Size {
		t.Fatalf("unexpected board size: %v", loaded.Size)
	}
}

func TestGameRepositoryUpdateBoardState_NotFound(t *testing.T) {
	fixture := newGameRepoFixture(t)
	state := domain.NewBoardState(8)

	err := fixture.repo.UpdateBoardState(fixture.ctx, "missing", state)

	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
