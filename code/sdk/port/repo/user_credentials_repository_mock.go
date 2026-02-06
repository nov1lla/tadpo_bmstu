package repo

import (
	"context"

	"ppo/sdk/domain"
)

type UserCredentialsRepositoryMock struct {
	GetByLoginFunc func(ctx context.Context, login domain.Login) (domain.UserCredentials, error)
	SaveFunc       func(ctx context.Context, creds domain.UserCredentials) error
	UpdatePasswordHashFunc func(ctx context.Context, login domain.Login, hash domain.PasswordHash) error
}

func (m *UserCredentialsRepositoryMock) GetByLogin(ctx context.Context, login domain.Login) (domain.UserCredentials, error) {
	if m.GetByLoginFunc != nil {
		return m.GetByLoginFunc(ctx, login)
	}
	return domain.UserCredentials{}, nil
}

func (m *UserCredentialsRepositoryMock) Save(ctx context.Context, creds domain.UserCredentials) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, creds)
	}
	return nil
}

func (m *UserCredentialsRepositoryMock) UpdatePasswordHash(ctx context.Context, login domain.Login, hash domain.PasswordHash) error {
	if m.UpdatePasswordHashFunc != nil {
		return m.UpdatePasswordHashFunc(ctx, login, hash)
	}
	return nil
}
