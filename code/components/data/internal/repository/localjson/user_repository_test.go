package localjson

import (
	"errors"
	"testing"

	"ppo/sdk/domain"
	"ppo/sdk/port/repo"
)

func TestUserRepositorySave_OK(t *testing.T) {
	fixture := newUserRepoFixture(t)
	user := NewUserBuilder().WithID("user-1").WithName("Alice").Build()

	err := fixture.repo.Save(fixture.ctx, user)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	loaded, err := fixture.repo.GetByID(fixture.ctx, user.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if loaded != user {
		t.Fatalf("unexpected user: %+v", loaded)
	}
}

func TestUserRepositorySave_RejectsDuplicate(t *testing.T) {
	fixture := newUserRepoFixture(t)
	user := MotherUser()
	if err := fixture.repo.Save(fixture.ctx, user); err != nil {
		t.Fatalf("save: %v", err)
	}

	err := fixture.repo.Save(fixture.ctx, user)

	if !errors.Is(err, repo.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestUserRepositoryGetByID_NotFound(t *testing.T) {
	fixture := newUserRepoFixture(t)

	_, err := fixture.repo.GetByID(fixture.ctx, "missing")

	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepositoryUpdate_OK(t *testing.T) {
	fixture := newUserRepoFixture(t)
	user := NewUserBuilder().WithID("user-2").WithName("Bob").Build()
	if err := fixture.repo.Save(fixture.ctx, user); err != nil {
		t.Fatalf("save: %v", err)
	}
	updated := user.WithRating(1201).WithWinStreak(2)

	err := fixture.repo.Update(fixture.ctx, updated)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	loaded, err := fixture.repo.GetByID(fixture.ctx, updated.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if loaded.Rating != updated.Rating || loaded.WinStreak != updated.WinStreak {
		t.Fatalf("unexpected user: %+v", loaded)
	}
}

func TestUserRepositoryUpdate_NotFound(t *testing.T) {
	fixture := newUserRepoFixture(t)
	user := domain.User{ID: "missing"}

	err := fixture.repo.Update(fixture.ctx, user)

	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
