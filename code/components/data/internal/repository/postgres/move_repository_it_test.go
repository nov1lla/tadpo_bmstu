//go:build integration
// +build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"ppo/sdk/domain"
)

func TestMoveRepository_Postgres(t *testing.T) {
	env := SetupTestDB(t)
	userRepo := NewUserRepository(env.DB)
	gameRepo := NewGameRepository(env.DB)
	moveRepo := NewMoveRepository(env.DB)

	user := domain.User{ID: "user-1", Name: "Alice", Rating: 1200, WinStreak: 1}
	if err := userRepo.Save(context.Background(), user); err != nil {
		t.Fatalf("save user failed: %v", err)
	}

	game := domain.NewGame("game-1", user.ID, domain.PlayerColorDark, false, domain.NewTimestamp(time.Now()))
	if err := gameRepo.Save(context.Background(), game); err != nil {
		t.Fatalf("save game failed: %v", err)
	}

	move1, err := domain.NewMove(
		"move-1",
		game.ID,
		"piece-1",
		1,
		domain.Position{Row: 2, Col: 2},
		[]domain.Position{{Row: 3, Col: 3}},
		domain.NewTimestamp(time.Now()),
	)
	if err != nil {
		t.Fatalf("new move1 failed: %v", err)
	}
	move2, err := domain.NewMove(
		"move-2",
		game.ID,
		"piece-2",
		2,
		domain.Position{Row: 5, Col: 5},
		[]domain.Position{{Row: 4, Col: 4}},
		domain.NewTimestamp(time.Now()),
	)
	if err != nil {
		t.Fatalf("new move2 failed: %v", err)
	}

	if err := moveRepo.Add(context.Background(), move1); err != nil {
		t.Fatalf("add move1 failed: %v", err)
	}
	if err := moveRepo.Add(context.Background(), move2); err != nil {
		t.Fatalf("add move2 failed: %v", err)
	}

	stored, err := moveRepo.GetByID(context.Background(), move1.ID)
	if err != nil {
		t.Fatalf("get move failed: %v", err)
	}
	if stored.ID != move1.ID {
		t.Fatalf("unexpected move %+v", stored)
	}
	if len(stored.Trajectory) != 1 || stored.EndPosition != move1.EndPosition {
		t.Fatalf("trajectory not persisted: %+v", stored)
	}

	list, err := moveRepo.ListByGame(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("list by game failed: %v", err)
	}
	if len(list) != 2 || list[0].Number != 1 || list[1].Number != 2 {
		t.Fatalf("unexpected list %+v", list)
	}
	if len(list[0].Trajectory) != 1 || len(list[1].Trajectory) != 1 {
		t.Fatalf("expected trajectories to be populated: %+v", list)
	}

	if err := moveRepo.Add(context.Background(), move1); err == nil {
		t.Fatalf("expected duplicate add error")
	}
}
