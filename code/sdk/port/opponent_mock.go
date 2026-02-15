package port

import (
	"context"

	"ppo/sdk/domain"
)

type OpponentMoveProviderMock struct {
	SuggestMoveFunc func(ctx context.Context, req OpponentMoveRequest) (domain.Move, error)
}

func (m *OpponentMoveProviderMock) SuggestMove(ctx context.Context, req OpponentMoveRequest) (domain.Move, error) {
	if m.SuggestMoveFunc != nil {
		return m.SuggestMoveFunc(ctx, req)
	}
	return domain.Move{}, nil
}
