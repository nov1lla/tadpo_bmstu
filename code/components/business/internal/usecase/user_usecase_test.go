package usecase

import (
	"context"
	"errors"
	"testing"

	"ppo/sdk/domain"
	"ppo/sdk/port/repo"
	sdkusecase "ppo/sdk/usecase"
)

type fixedIDGen struct {
	id string
}

func (g fixedIDGen) NewID() string {
	return g.id
}

func TestUserUseCaseUpdateFields(t *testing.T) {
	repoMock := &repo.UserRepositoryMock{}
	initial := domain.User{
		ID:        "user-1",
		Name:      "Alice",
		Rating:    1200,
		WinStreak: 3,
	}

	repoMock.GetByIDFunc = func(ctx context.Context, id domain.UserID) (domain.User, error) {
		if id != initial.ID {
			t.Fatalf("unexpected id %v", id)
		}
		return initial, nil
	}

	captured := domain.User{}
	repoMock.UpdateFunc = func(ctx context.Context, user domain.User) error {
		captured = user
		return nil
	}

	uc := NewUserUseCase(repoMock, fixedIDGen{id: "ignored"})
	newRating := domain.Rating(1250)
	streak := domain.WinStreak(4)
	lastGame := domain.GameID("game-42")

	updated, err := uc.UpdateFields(context.Background(), sdkusecase.UpdateUserFieldsCommand{
		ID:         initial.ID,
		Rating:     &newRating,
		WinStreak:  &streak,
		LastGameID: &lastGame,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.Rating != newRating || captured.WinStreak != streak {
		t.Fatalf("unexpected captured user %+v", captured)
	}
	if captured.LastGameID == nil || *captured.LastGameID != lastGame {
		t.Fatalf("last game mismatch %+v", captured.LastGameID)
	}

	if updated != captured {
		t.Fatalf("expected returned user to match updated state")
	}
}

func TestUserUseCasePropagatesUpdateError(t *testing.T) {
	repoMock := &repo.UserRepositoryMock{}
	repoMock.GetByIDFunc = func(ctx context.Context, id domain.UserID) (domain.User, error) {
		return domain.User{ID: id}, nil
	}

	expectedErr := errors.New("update failed")
	repoMock.UpdateFunc = func(ctx context.Context, user domain.User) error {
		return expectedErr
	}

	uc := NewUserUseCase(repoMock, fixedIDGen{id: "ignored"})
	_, err := uc.UpdateFields(context.Background(), sdkusecase.UpdateUserFieldsCommand{ID: "user"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestUserUseCaseCreateAndGet(t *testing.T) {
	repository := &repo.UserRepositoryMock{}
	var captured domain.User
	repository.SaveFunc = func(ctx context.Context, user domain.User) error {
		captured = user
		return nil
	}
	repository.GetByIDFunc = func(ctx context.Context, id domain.UserID) (domain.User, error) {
		if id != captured.ID {
			t.Fatalf("unexpected id %s", id)
		}
		return captured, nil
	}

	uc := NewUserUseCase(repository, fixedIDGen{id: "user-create"})
	cmd := sdkusecase.CreateUserCommand{Name: "Carol"}
	created, err := uc.Create(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if created.ID != "user-create" || created.Name != cmd.Name {
		t.Fatalf("unexpected created user: %+v", created)
	}

	loaded, err := uc.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get returned error: %v", err)
	}
	if loaded.ID != created.ID {
		t.Fatalf("unexpected loaded user: %+v", loaded)
	}

	_, err = uc.Create(context.Background(), sdkusecase.CreateUserCommand{})
	if !errors.Is(err, ErrInvalidUserData) {
		t.Fatalf("expected ErrInvalidUserData, got %v", err)
	}
}
