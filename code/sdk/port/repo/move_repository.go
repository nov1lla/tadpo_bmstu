package repo

import (
	"context"

	"ppo/sdk/domain"
)

type MoveRepository interface {
	GetByID(ctx context.Context, id domain.MoveID) (domain.Move, error)
	Add(ctx context.Context, move domain.Move) error
	ListByGame(ctx context.Context, id domain.GameID) ([]domain.Move, error)
	DeleteByGame(ctx context.Context, id domain.GameID) error
}
