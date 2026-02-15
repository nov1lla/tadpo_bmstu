package postgres

import (
	"context"
	"testing"
	"time"

	"ppo/sdk/domain"
)

func TestGameRepository_Postgres(t *testing.T) {
	env := SetupTestDB(t)
	userRepo := NewUserRepository(env.DB)
	gameRepo := NewGameRepository(env.DB)

	user := domain.User{ID: "user-1", Name: "Alice", Rating: 1200, WinStreak: 2}
	if err := userRepo.Save(context.Background(), user); err != nil {
		t.Fatalf("save user failed: %v", err)
	}

	game := domain.NewGame("game-1", user.ID, domain.PlayerColorLight, true, domain.NewTimestamp(time.Now()))
	if err := gameRepo.Save(context.Background(), game); err != nil {
		t.Fatalf("save game failed: %v", err)
	}

	games, err := gameRepo.ListByUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("list by user failed: %v", err)
	}
	if len(games) != 1 || games[0].ID != game.ID {
		t.Fatalf("unexpected games: %+v", games)
	}

	stored, err := gameRepo.GetByID(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("get game failed: %v", err)
	}
	if stored.ID != game.ID {
		t.Fatalf("unexpected game %+v", stored)
	}

	board := domain.NewBoardState(8)
	board.Pieces["piece-1"] = domain.BoardPiece{
		ID:       "piece-1",
		Position: domain.Position{Row: 2, Col: 2},
		Color:    domain.PlayerColorLight,
		Kind:     domain.PieceKindMan,
	}
	if err := gameRepo.UpdateBoardState(context.Background(), game.ID, board); err != nil {
		t.Fatalf("update board failed: %v", err)
	}

	storedBoard, err := gameRepo.BoardState(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("board state failed: %v", err)
	}
	if storedBoard.Size != 8 || storedBoard.Pieces["piece-1"].Position.Row != 2 {
		t.Fatalf("unexpected board %+v", storedBoard)
	}

	finished, err := stored.Finish(domain.NewTimestamp(stored.Start.ToTime().Add(time.Hour)))
	if err != nil {
		t.Fatalf("finish error: %v", err)
	}

	if err := gameRepo.Update(context.Background(), finished); err != nil {
		t.Fatalf("update game failed: %v", err)
	}

	stored, err = gameRepo.GetByID(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("get after update failed: %v", err)
	}
	if stored.Status != domain.GameStatusFinished {
		t.Fatalf("status not updated: %+v", stored)
	}
}
