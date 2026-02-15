package localjson

import (
	"context"
	"errors"
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

func newTestUser(id domain.UserID) domain.User {
	return domain.User{
		ID:        id,
		Name:      "Test User",
		Rating:    1200,
		WinStreak: 1,
	}
}

func newTestGame(id domain.GameID, userID domain.UserID) domain.Game {
	start := domain.NewTimestamp(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	return newTestGameWithStart(id, userID, start)
}

func newTestGameWithStart(id domain.GameID, userID domain.UserID, start domain.Timestamp) domain.Game {
	return domain.NewGame(id, userID, domain.PlayerColorLight, true, start)
}

func newTestMove(t *testing.T, id domain.MoveID, gameID domain.GameID, number domain.MoveNumber) domain.Move {
	t.Helper()
	start := domain.Position{Row: 5, Col: 0}
	trajectory := []domain.Position{{Row: 4, Col: 1}}
	created := domain.NewTimestamp(time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC))

	move, err := domain.NewMove(id, gameID, "piece-1", number, start, trajectory, created)

	if err != nil {
		t.Fatalf("new move: %v", err)
	}

	return move
}

func TestUserRepositorySaveAndGetByID(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewUserRepository(storage)
	ctx := context.Background()

	user := newTestUser("user-1")

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

func TestUserRepositorySaveRejectsDuplicate(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewUserRepository(storage)
	ctx := context.Background()

	user := newTestUser("user-1")

	if err := repo.Save(ctx, user); err != nil {
		t.Fatalf("save user: %v", err)
	}

	if err := repo.Save(ctx, user); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestUserRepositoryUpdate(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewUserRepository(storage)
	ctx := context.Background()
	user := newTestUser("user-1")

	if err := repo.Save(ctx, user); err != nil {
		t.Fatalf("save user: %v", err)
	}

	updated := user.WithRating(1300).WithWinStreak(4)
	if err := repo.Update(ctx, updated); err != nil {
		t.Fatalf("update user: %v", err)
	}

	fetched, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user after update: %v", err)
	}
	if fetched != updated {
		t.Fatalf("expected %+v, got %+v", updated, fetched)
	}
}

func TestUserRepositoryUpdateMissing(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewUserRepository(storage)
	ctx := context.Background()
	user := newTestUser("missing")

	if err := repo.Update(ctx, user); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepositoryGetByIDMissing(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewUserRepository(storage)
	ctx := context.Background()

	if _, err := repo.GetByID(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGameRepositorySave(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewGameRepository(storage)
	ctx := context.Background()
	game := newTestGame("game-1", "user-1")

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
}

func TestGameRepositorySaveRejectsDuplicate(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewGameRepository(storage)
	ctx := context.Background()
	game := newTestGame("game-1", "user-1")

	if err := repo.Save(ctx, game); err != nil {
		t.Fatalf("save game: %v", err)
	}

	if err := repo.Save(ctx, game); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestGameRepositoryGetByID(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewGameRepository(storage)
	ctx := context.Background()
	game := newTestGame("game-1", "user-1")

	if err := repo.Save(ctx, game); err != nil {
		t.Fatalf("save game: %v", err)
	}

	fetched, err := repo.GetByID(ctx, game.ID)
	if err != nil {
		t.Fatalf("get game: %v", err)
	}
	if fetched.ID != game.ID || fetched.UserID != game.UserID {
		t.Fatalf("unexpected game: %+v", fetched)
	}
}

func TestGameRepositoryGetByIDMissing(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewGameRepository(storage)
	ctx := context.Background()

	if _, err := repo.GetByID(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGameRepositoryListByUserOrdersByStart(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewGameRepository(storage)
	ctx := context.Background()

	firstStart := domain.NewTimestamp(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	secondStart := domain.NewTimestamp(time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC))
	first := newTestGameWithStart("game-1", "user-1", firstStart)
	second := newTestGameWithStart("game-2", "user-1", secondStart)

	if err := repo.Save(ctx, first); err != nil {
		t.Fatalf("save game: %v", err)
	}
	if err := repo.Save(ctx, second); err != nil {
		t.Fatalf("save game: %v", err)
	}

	list, err := repo.ListByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("list games: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 games, got %d", len(list))
	}
	if list[0].ID != second.ID {
		t.Fatalf("expected most recent game first, got %s", list[0].ID)
	}
}

func TestGameRepositoryListByUserEmpty(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewGameRepository(storage)
	ctx := context.Background()

	game := newTestGame("game-1", "user-1")
	if err := repo.Save(ctx, game); err != nil {
		t.Fatalf("save game: %v", err)
	}

	list, err := repo.ListByUser(ctx, "user-2")
	if err != nil {
		t.Fatalf("list games: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}
}

func TestGameRepositoryBoardState(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewGameRepository(storage)
	ctx := context.Background()
	game := newTestGame("game-1", "user-1")

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

func TestGameRepositoryBoardStateMissing(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewGameRepository(storage)
	ctx := context.Background()

	if _, err := repo.BoardState(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGameRepositoryUpdateBoardState(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewGameRepository(storage)
	ctx := context.Background()
	game := newTestGame("game-1", "user-1")

	if err := repo.Save(ctx, game); err != nil {
		t.Fatalf("save game: %v", err)
	}

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

	board, err := repo.BoardState(ctx, game.ID)
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
}

func TestGameRepositoryUpdateBoardStateMissing(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewGameRepository(storage)
	ctx := context.Background()
	state := domain.NewBoardState(domain.DefaultBoardSize)

	if err := repo.UpdateBoardState(ctx, "missing", state); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGameRepositoryUpdate(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewGameRepository(storage)
	ctx := context.Background()
	game := newTestGame("game-1", "user-1")

	if err := repo.Save(ctx, game); err != nil {
		t.Fatalf("save game: %v", err)
	}

	updated, err := game.StartPlay()
	if err != nil {
		t.Fatalf("start play: %v", err)
	}

	if err := repo.Update(ctx, updated); err != nil {
		t.Fatalf("update game: %v", err)
	}

	fetched, err := repo.GetByID(ctx, game.ID)
	if err != nil {
		t.Fatalf("get game after update: %v", err)
	}
	if fetched.Status != domain.GameStatusInProgress {
		t.Fatalf("expected status in_progress, got %v", fetched.Status)
	}
}

func TestGameRepositoryUpdateMissing(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewGameRepository(storage)
	ctx := context.Background()
	game := newTestGame("missing", "user-1")

	if err := repo.Update(ctx, game); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMoveRepositoryAddAndGetByID(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewMoveRepository(storage)
	ctx := context.Background()
	move := newTestMove(t, "move-1", "game-1", 1)

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
}

func TestMoveRepositoryAddRejectsDuplicate(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewMoveRepository(storage)
	ctx := context.Background()
	move := newTestMove(t, "move-1", "game-1", 1)

	if err := repo.Add(ctx, move); err != nil {
		t.Fatalf("add move: %v", err)
	}

	if err := repo.Add(ctx, move); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestMoveRepositoryGetByIDMissing(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewMoveRepository(storage)
	ctx := context.Background()

	if _, err := repo.GetByID(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMoveRepositoryListByGame(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewMoveRepository(storage)
	ctx := context.Background()
	move1 := newTestMove(t, "move-1", "game-1", 1)
	move2 := newTestMove(t, "move-2", "game-1", 2)

	if err := repo.Add(ctx, move1); err != nil {
		t.Fatalf("add move: %v", err)
	}
	if err := repo.Add(ctx, move2); err != nil {
		t.Fatalf("add move: %v", err)
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

func TestMoveRepositoryListByGameEmpty(t *testing.T) {
	storage := newTestStorage(t)
	repo := NewMoveRepository(storage)
	ctx := context.Background()
	move := newTestMove(t, "move-1", "game-1", 1)

	if err := repo.Add(ctx, move); err != nil {
		t.Fatalf("add move: %v", err)
	}

	list, err := repo.ListByGame(ctx, "game-2")
	if err != nil {
		t.Fatalf("list moves: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}
}
