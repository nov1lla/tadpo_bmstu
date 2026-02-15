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
	initial := NewUserBuilder().
		WithID("user-1").
		WithName("Alice").
		WithRating(1200).
		WithWinStreak(3).
		Build()

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

func TestUserUseCaseCreate(t *testing.T) {
	repository := &repo.UserRepositoryMock{}
	var captured domain.User
	repository.SaveFunc = func(ctx context.Context, user domain.User) error {
		captured = user
		return nil
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
	if captured.ID != created.ID {
		t.Fatalf("expected saved user to match created user")
	}
}

func TestUserUseCaseCreateRequiresIDGenerator(t *testing.T) {
	uc := NewUserUseCase(&repo.UserRepositoryMock{}, nil)

	_, err := uc.Create(context.Background(), sdkusecase.CreateUserCommand{Name: "Alice"})

	if err == nil {
		t.Fatalf("expected error when id generator is missing")
	}
}

func TestUserUseCaseCreateRejectsInvalidData(t *testing.T) {
	uc := NewUserUseCase(&repo.UserRepositoryMock{}, fixedIDGen{id: "ignored"})

	_, err := uc.Create(context.Background(), sdkusecase.CreateUserCommand{})

	if !errors.Is(err, ErrInvalidUserData) {
		t.Fatalf("expected ErrInvalidUserData, got %v", err)
	}
}

func TestUserUseCaseGet(t *testing.T) {
	repository := &repo.UserRepositoryMock{}
	expected := NewUserBuilder().WithID("user-42").WithName("Carol").Build()
	repository.GetByIDFunc = func(ctx context.Context, id domain.UserID) (domain.User, error) {
		if id != expected.ID {
			t.Fatalf("unexpected id %s", id)
		}
		return expected, nil
	}

	uc := NewUserUseCase(repository, fixedIDGen{id: "ignored"})
	loaded, err := uc.Get(context.Background(), expected.ID)
	if err != nil {
		t.Fatalf("get returned error: %v", err)
	}
	if loaded.ID != expected.ID {
		t.Fatalf("unexpected loaded user: %+v", loaded)
	}
}

func TestUserUseCaseGetReturnsRepositoryError(t *testing.T) {
	expected := errors.New("missing user")
	repository := &repo.UserRepositoryMock{
		GetByIDFunc: func(ctx context.Context, id domain.UserID) (domain.User, error) {
			return domain.User{}, expected
		},
	}
	uc := NewUserUseCase(repository, fixedIDGen{id: "ignored"})

	_, err := uc.Get(context.Background(), "user-1")

	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestUserUseCaseCreate_Classic(t *testing.T) {
	repository := newInMemoryUserRepo()
	uc := NewUserUseCase(repository, fixedIDGen{id: "user-classic"})

	created, err := uc.Create(context.Background(), sdkusecase.CreateUserCommand{Name: "Classic"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID != "user-classic" {
		t.Fatalf("unexpected user: %+v", created)
	}
}

func TestUserUseCaseGet_Classic(t *testing.T) {
	repository := newInMemoryUserRepo()
	user := NewUserBuilder().WithID("user-classic").WithName("Classic").Build()
	if err := repository.Save(context.Background(), user); err != nil {
		t.Fatalf("save: %v", err)
	}
	uc := NewUserUseCase(repository, fixedIDGen{id: "ignored"})

	loaded, err := uc.Get(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.ID != user.ID {
		t.Fatalf("unexpected user: %+v", loaded)
	}
}
