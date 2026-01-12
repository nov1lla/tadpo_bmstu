package localjson

import (
	"errors"
	"testing"

	"ppo/sdk/port/repo"
)

func TestMoveRepositoryAdd_OK(t *testing.T) {
	fixture := newMoveRepoFixture(t)
	move := NewMoveBuilder().WithID("move-1").WithGameID("game-1").Build(t)

	err := fixture.repo.Add(fixture.ctx, move)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	loaded, err := fixture.repo.GetByID(fixture.ctx, move.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if loaded.ID != move.ID {
		t.Fatalf("unexpected move: %+v", loaded)
	}
}

func TestMoveRepositoryAdd_RejectsDuplicate(t *testing.T) {
	fixture := newMoveRepoFixture(t)
	move := MotherMove(t)
	if err := fixture.repo.Add(fixture.ctx, move); err != nil {
		t.Fatalf("add: %v", err)
	}

	err := fixture.repo.Add(fixture.ctx, move)

	if !errors.Is(err, repo.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestMoveRepositoryGetByID_NotFound(t *testing.T) {
	fixture := newMoveRepoFixture(t)

	_, err := fixture.repo.GetByID(fixture.ctx, "missing")

	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMoveRepositoryListByGame_OK(t *testing.T) {
	fixture := newMoveRepoFixture(t)
	first := NewMoveBuilder().WithID("move-1").WithGameID("game-1").WithNumber(1).Build(t)
	second := NewMoveBuilder().WithID("move-2").WithGameID("game-1").WithNumber(2).Build(t)
	if err := fixture.repo.Add(fixture.ctx, first); err != nil {
		t.Fatalf("add first: %v", err)
	}
	if err := fixture.repo.Add(fixture.ctx, second); err != nil {
		t.Fatalf("add second: %v", err)
	}

	moves, err := fixture.repo.ListByGame(fixture.ctx, "game-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moves) != 2 || moves[0].ID != first.ID || moves[1].ID != second.ID {
		t.Fatalf("unexpected moves: %+v", moves)
	}
}

func TestMoveRepositoryListByGame_ContextCanceled(t *testing.T) {
	fixture := newMoveRepoFixture(t)
	ctx := canceledContext(t)

	_, err := fixture.repo.ListByGame(ctx, "game-1")

	if err == nil {
		t.Fatalf("expected context error")
	}
}

func TestMoveRepositoryDeleteByGame_OK(t *testing.T) {
	fixture := newMoveRepoFixture(t)
	move := NewMoveBuilder().WithID("move-1").WithGameID("game-1").Build(t)
	if err := fixture.repo.Add(fixture.ctx, move); err != nil {
		t.Fatalf("add: %v", err)
	}

	err := fixture.repo.DeleteByGame(fixture.ctx, "game-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	moves, err := fixture.repo.ListByGame(fixture.ctx, "game-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(moves) != 0 {
		t.Fatalf("expected empty list, got %+v", moves)
	}
}

func TestMoveRepositoryDeleteByGame_ContextCanceled(t *testing.T) {
	fixture := newMoveRepoFixture(t)
	ctx := canceledContext(t)

	err := fixture.repo.DeleteByGame(ctx, "game-1")

	if err == nil {
		t.Fatalf("expected context error")
	}
}
