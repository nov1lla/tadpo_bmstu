package repo

import (
	"context"

	"ppo/sdk/domain"
)

type GameRepository interface {
	GetByID(ctx context.Context, id domain.GameID) (domain.Game, error)
	ListByUser(ctx context.Context, userID domain.UserID) ([]domain.Game, error)
	Save(ctx context.Context, game domain.Game) error
	Update(ctx context.Context, game domain.Game) error
	BoardState(ctx context.Context, id domain.GameID) (domain.BoardState, error)
	UpdateBoardState(ctx context.Context, id domain.GameID, state domain.BoardState) error
}
