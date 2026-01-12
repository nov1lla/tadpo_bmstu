package localjson

import (
	"context"
	"testing"
	"time"

	"ppo/sdk/domain"
)

func TestUserRepository_SaveGetUpdate(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewUserRepository(storage)
	ctx := context.Background()

	user := domain.User{
		ID:        "user-1",
		Name:      "Alice",
		Rating:    1200,
		WinStreak: 3,
	}
	if err := repo.Save(ctx, user); err != nil {
		t.Fatalf("save user: %v", err)
	}

	fetched, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if fetched != user {
		t.Fatalf("unexpected user: %+v", fetched)
	}
}

func TestGameRepository_Flow(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewGameRepository(storage)
	ctx := context.Background()

	start := domain.NewTimestamp(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	game := domain.NewGame("game-1", "user-1", domain.PlayerColorLight, true, start)

	if err := repo.Save(ctx, game); err != nil {
		t.Fatalf("save game: %v", err)
	}

	board, err := repo.BoardState(ctx, game.ID)
	if err != nil {
		t.Fatalf("board state: %v", err)
	}
	if board.Size != domain.DefaultBoardSize {
		t.Fatalf("expected board size %d, got %d", domain.DefaultBoardSize, board.Size)
	}
	if len(board.Pieces) != 0 {
		t.Fatalf("expected empty pieces, got %d", len(board.Pieces))
	}
}

func TestMoveRepository_AddAndList(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewMoveRepository(storage)
	ctx := context.Background()

	start := domain.Position{Row: 5, Col: 0}
	trajectory := []domain.Position{{Row: 4, Col: 1}}
	created := domain.NewTimestamp(time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC))
	move, err := domain.NewMove("move-1", "game-1", "piece-1", 1, start, trajectory, created)
	if err != nil {
		t.Fatalf("new move: %v", err)
	}

	if err := repo.Add(ctx, move); err != nil {
		t.Fatalf("add move: %v", err)
	}

	trajectory2 := []domain.Position{{Row: 2, Col: 3}}
	created2 := domain.NewTimestamp(time.Date(2024, 1, 3, 12, 5, 0, 0, time.UTC))
	move2, err := domain.NewMove("move-2", "game-1", "piece-2", 2, domain.Position{Row: 2, Col: 1}, trajectory2, created2)
	if err != nil {
		t.Fatalf("new move 2: %v", err)
	}
	if err := repo.Add(ctx, move2); err != nil {
		t.Fatalf("add move 2: %v", err)
	}

	list, err := repo.ListByGame(ctx, "game-1")
	if err != nil {
		t.Fatalf("list moves: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 moves, got %d", len(list))
	}
	if list[0].Number != 1 || list[1].Number != 2 {
		t.Fatalf("unexpected order: %+v", list)
	}
}
