package repo

import (
	"context"

	"ppo/sdk/domain"
)

type UserCredentialsRepository interface {
	GetByLogin(ctx context.Context, login domain.Login) (domain.UserCredentials, error)
	Save(ctx context.Context, creds domain.UserCredentials) error
	UpdatePasswordHash(ctx context.Context, login domain.Login, hash domain.PasswordHash) error
}
