package port

import (
	"context"

	"ppo/sdk/domain"
)

type OpponentMoveRequest struct {
	Game       domain.Game
	Board      domain.BoardState
	Difficulty domain.DifficultyLevel
	Feedback   []string
}

type OpponentMoveProvider interface {
	SuggestMove(ctx context.Context, req OpponentMoveRequest) (domain.Move, error)
}
