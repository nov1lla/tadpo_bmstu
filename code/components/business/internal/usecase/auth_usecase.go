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
	ErrPasswordUnchanged  = errors.New("password unchanged")
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
	login := normalizeLogin(cmd.Login)
	if login == "" || strings.TrimSpace(cmd.Password) == "" {
		return domain.User{}, ErrInvalidAuthData
	}
	if uc.ids == nil || uc.users == nil || uc.creds == nil {
		return domain.User{}, ErrAuthNotConfigured
	}
	_, err := uc.creds.GetByLogin(ctx, login)
	if err == nil {
		return domain.User{}, ErrLoginAlreadyTaken
	}
	if !errors.Is(err, sdkrepo.ErrNotFound) {
		return domain.User{}, err
	}

	id := domain.UserID(uc.ids.NewID())
	user := domain.User{
		ID:        id,
		Name:      domain.UserName(login),
		Rating:    0,
		WinStreak: 0,
	}
	if err := uc.users.Save(ctx, user); err != nil {
		return domain.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}
	if err := uc.creds.Save(ctx, domain.UserCredentials{
		UserID:       id,
		Login:        login,
		PasswordHash: domain.PasswordHash(string(hash)),
	}); err != nil {
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

func (uc *authUseCase) ChangePassword(ctx context.Context, cmd sdkusecase.ChangePasswordCommand) error {
	login := normalizeLogin(cmd.Login)
	oldPassword := strings.TrimSpace(cmd.OldPassword)
	newPassword := strings.TrimSpace(cmd.NewPassword)
	if login == "" || oldPassword == "" || newPassword == "" {
		return ErrInvalidAuthData
	}
	if newPassword == oldPassword {
		return ErrPasswordUnchanged
	}
	if uc.creds == nil {
		return ErrAuthNotConfigured
	}

	creds, err := uc.creds.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, sdkrepo.ErrNotFound) {
			return ErrLoginNotRegistered
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(creds.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return uc.creds.UpdatePasswordHash(ctx, login, domain.PasswordHash(string(hash)))
}

func (uc *authUseCase) ResetPassword(ctx context.Context, cmd sdkusecase.ResetPasswordCommand) error {
	login := normalizeLogin(cmd.Login)
	newPassword := strings.TrimSpace(cmd.NewPassword)
	if login == "" || newPassword == "" {
		return ErrInvalidAuthData
	}
	if uc.creds == nil {
		return ErrAuthNotConfigured
	}

	_, err := uc.creds.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, sdkrepo.ErrNotFound) {
			return ErrLoginNotRegistered
		}
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return uc.creds.UpdatePasswordHash(ctx, login, domain.PasswordHash(string(hash)))
}

func normalizeLogin(login domain.Login) domain.Login {
	return domain.Login(strings.TrimSpace(strings.ToLower(string(login))))
}
