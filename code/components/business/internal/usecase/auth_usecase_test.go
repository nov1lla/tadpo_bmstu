package usecase

import (
	"context"
	"testing"

	"ppo/sdk/domain"
	sdkusecase "ppo/sdk/usecase"
)

func TestAuthUseCaseRegisterAndLogin_Classic(t *testing.T) {
	users := newInMemoryUserRepo()
	creds := newInMemoryCredsRepo()
	uc := NewAuthUseCase(users, creds, fixedIDGen{id: "user-auth"})

	user, err := uc.Register(context.Background(), sdkusecase.RegisterCommand{
		Login:    "Student",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "user-auth" || user.Name != domain.UserName("student") {
		t.Fatalf("unexpected user: %+v", user)
	}

	loaded, err := uc.Login(context.Background(), sdkusecase.LoginCommand{
		Login:    "Student",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.ID != user.ID {
		t.Fatalf("unexpected login result: %+v", loaded)
	}
}

func TestAuthUseCaseRegisterRejectsInvalidData(t *testing.T) {
	uc := NewAuthUseCase(newInMemoryUserRepo(), newInMemoryCredsRepo(), fixedIDGen{id: "user-auth"})

	_, err := uc.Register(context.Background(), sdkusecase.RegisterCommand{Login: "", Password: ""})

	if err != ErrInvalidAuthData {
		t.Fatalf("expected ErrInvalidAuthData, got %v", err)
	}
}

func TestAuthUseCaseLoginRejectsInvalidCredentials(t *testing.T) {
	users := newInMemoryUserRepo()
	creds := newInMemoryCredsRepo()
	uc := NewAuthUseCase(users, creds, fixedIDGen{id: "user-auth"})

	_, err := uc.Register(context.Background(), sdkusecase.RegisterCommand{
		Login:    "student",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err = uc.Login(context.Background(), sdkusecase.LoginCommand{
		Login:    "student",
		Password: "wrong",
	})

	if err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}
