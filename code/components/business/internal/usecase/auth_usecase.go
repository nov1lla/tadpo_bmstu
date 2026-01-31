package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"ppo/sdk/domain"
	sdkport "ppo/sdk/port"
	sdkrepo "ppo/sdk/port/repo"
	sdkusecase "ppo/sdk/usecase"
)

var (
	ErrInvalidAuthData    = errors.New("invalid auth data")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrLoginAlreadyTaken  = errors.New("login already taken")
	ErrLoginNotRegistered = errors.New("login not registered")
	ErrAuthNotConfigured  = errors.New("auth not configured")
)

type authUseCase struct {
	users sdkrepo.UserRepository
	creds sdkrepo.UserCredentialsRepository
	ids   sdkport.IDGenerator
}

func NewAuthUseCase(users sdkrepo.UserRepository, creds sdkrepo.UserCredentialsRepository, ids sdkport.IDGenerator) sdkusecase.AuthUseCase {
	return &authUseCase{users: users, creds: creds, ids: ids}
}

func (uc *authUseCase) Register(ctx context.Context, cmd sdkusecase.RegisterCommand) (domain.User, error) {
	login, password, err := validateAuthInput(cmd.Login, cmd.Password)
	if err != nil {
		return domain.User{}, err
	}
	if err := uc.ensureRegisterConfigured(); err != nil {
		return domain.User{}, err
	}
	if err := uc.ensureLoginAvailable(ctx, login); err != nil {
		return domain.User{}, err
	}

	id := domain.UserID(uc.ids.NewID())
	user := domain.User{ID: id, Name: domain.UserName(login), Rating: 0, WinStreak: 0}
	if err := uc.users.Save(ctx, user); err != nil {
		return domain.User{}, err
	}
	creds, err := buildCredentials(id, login, password)
	if err != nil {
		return domain.User{}, err
	}
	if err := uc.creds.Save(ctx, creds); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (uc *authUseCase) Login(ctx context.Context, cmd sdkusecase.LoginCommand) (domain.User, error) {
	login := normalizeLogin(cmd.Login)
	password := strings.TrimSpace(cmd.Password)
	if login == "" || password == "" {
		return domain.User{}, ErrInvalidAuthData
	}
	if uc.users == nil || uc.creds == nil {
		return domain.User{}, ErrAuthNotConfigured
	}
	creds, err := uc.creds.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, sdkrepo.ErrNotFound) {
			return domain.User{}, ErrLoginNotRegistered
		}
		return domain.User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(creds.PasswordHash), []byte(password)); err != nil {
		return domain.User{}, ErrInvalidCredentials
	}
	return uc.users.GetByID(ctx, creds.UserID)
}

func normalizeLogin(login domain.Login) domain.Login {
	return domain.Login(strings.TrimSpace(strings.ToLower(string(login))))
}

func validateAuthInput(login domain.Login, password string) (domain.Login, string, error) {
	normalized := normalizeLogin(login)
	trimmedPassword := strings.TrimSpace(password)
	if normalized == "" || trimmedPassword == "" {
		return "", "", ErrInvalidAuthData
	}
	return normalized, trimmedPassword, nil
}

func (uc *authUseCase) ensureRegisterConfigured() error {
	if uc.ids == nil || uc.users == nil || uc.creds == nil {
		return ErrAuthNotConfigured
	}
	return nil
}

func (uc *authUseCase) ensureLoginAvailable(ctx context.Context, login domain.Login) error {
	_, err := uc.creds.GetByLogin(ctx, login)
	if err == nil {
		return ErrLoginAlreadyTaken
	}
	if !errors.Is(err, sdkrepo.ErrNotFound) {
		return err
	}
	return nil
}

func buildCredentials(id domain.UserID, login domain.Login, password string) (domain.UserCredentials, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.UserCredentials{}, fmt.Errorf("hash password: %w", err)
	}
	return domain.UserCredentials{
		UserID:       id,
		Login:        login,
		PasswordHash: domain.PasswordHash(string(hash)),
	}, nil
}
