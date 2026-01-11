package repo

import (
	"context"

	"ppo/sdk/domain"
)

type UserRepository interface {
	GetByID(ctx context.Context, id domain.UserID) (domain.User, error)
	Save(ctx context.Context, user domain.User) error
	Update(ctx context.Context, user domain.User) error
}
