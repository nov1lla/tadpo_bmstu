package localjson

import (
	"context"
	"testing"
	"time"

	"ppo/sdk/domain"
)

func newTestStorage(t *testing.T) *Storage {
	t.Helper()
	dir := t.TempDir()
	storage, err := NewStorage(dir)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	return storage
}

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

	updated := user.WithRating(1300).WithWinStreak(4)
	if err := repo.Update(ctx, updated); err != nil {
		t.Fatalf("update user: %v", err)
	}

	fetched, err = repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user after update: %v", err)
	}
	if fetched != updated {
		t.Fatalf("expected %+v, got %+v", updated, fetched)
	}

	duplicateErr := repo.Save(ctx, user)
	if duplicateErr == nil {
		t.Fatalf("expected duplicate error")
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

	fetched, err := repo.GetByID(ctx, game.ID)
	if err != nil {
		t.Fatalf("get game: %v", err)
	}
	if fetched.ID != game.ID || fetched.UserID != game.UserID || fetched.Status != game.Status {
		t.Fatalf("unexpected game: %+v", fetched)
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

	secondStart := domain.NewTimestamp(time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC))
	secondGame := domain.NewGame("game-2", "user-1", domain.PlayerColorDark, false, secondStart)
	if err := repo.Save(ctx, secondGame); err != nil {
		t.Fatalf("save second game: %v", err)
	}

	list, err := repo.ListByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("list games: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 games, got %d", len(list))
	}
	if list[0].ID != secondGame.ID {
		t.Fatalf("expected most recent game first, got %s", list[0].ID)
	}

	// Update board state with a captured piece
	state := domain.NewBoardState(domain.DefaultBoardSize)
	state = state.WithPiece(domain.BoardPiece{
		ID:       "piece-1",
		Position: domain.Position{Row: 5, Col: 2},
		Color:    domain.PlayerColorLight,
		Kind:     domain.PieceKindMan,
	})

	if err := repo.UpdateBoardState(ctx, game.ID, state); err != nil {
		t.Fatalf("update board state: %v", err)
	}

	board, err = repo.BoardState(ctx, game.ID)
	if err != nil {
		t.Fatalf("board state after update: %v", err)
	}
	if len(board.Pieces) != 1 {
		t.Fatalf("expected 1 piece, got %d", len(board.Pieces))
	}
	piece, ok := board.Pieces["piece-1"]
	if !ok {
		t.Fatalf("piece not found in board state")
	}
	if piece.Position != (domain.Position{Row: 5, Col: 2}) {
		t.Fatalf("unexpected piece position: %+v", piece)
	}

	game.Status = domain.GameStatusInProgress
	if err := repo.Update(ctx, game); err != nil {
		t.Fatalf("update game: %v", err)
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

	fetched, err := repo.GetByID(ctx, move.ID)
	if err != nil {
		t.Fatalf("get move: %v", err)
	}
	if fetched.ID != move.ID || len(fetched.Trajectory) != len(move.Trajectory) {
		t.Fatalf("unexpected move: %+v", fetched)
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

	if err := repo.Add(ctx, move); err == nil {
		t.Fatalf("expected duplicate move error")
	}
}
