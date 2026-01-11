package postgres

import (
	"context"
	"testing"

	"ppo/sdk/domain"
)

func TestUserRepository_Postgres(t *testing.T) {
	env := SetupTestDB(t)
	repo := NewUserRepository(env.DB)

	user := domain.User{ID: "1", Name: "Alice", Rating: 1200, WinStreak: 3}
	if err := repo.Save(context.Background(), user); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	stored, err := repo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if stored.Name != user.Name {
		t.Fatalf("unexpected user: %+v", stored)
	}

	updated := stored.WithRating(1350)
	if err = repo.Update(context.Background(), updated); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	stored, err = repo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get after update failed: %v", err)
	}
	if stored.Rating != 1350 {
		t.Fatalf("rating not updated: %+v", stored)
	}

	if err = repo.Save(context.Background(), user); err == nil {
		t.Fatalf("expected duplicate error")
	}
}
