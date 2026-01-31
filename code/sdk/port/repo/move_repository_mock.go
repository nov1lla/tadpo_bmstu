package repo

import (
	"context"

	"ppo/sdk/domain"
)

type MoveRepositoryMock struct {
	GetByIDFunc      func(ctx context.Context, id domain.MoveID) (domain.Move, error)
	AddFunc          func(ctx context.Context, move domain.Move) error
	ListByGameFunc   func(ctx context.Context, id domain.GameID) ([]domain.Move, error)
	DeleteByGameFunc func(ctx context.Context, id domain.GameID) error
}

func (m *MoveRepositoryMock) GetByID(ctx context.Context, id domain.MoveID) (domain.Move, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return domain.Move{}, nil
}

func (m *MoveRepositoryMock) Add(ctx context.Context, move domain.Move) error {
	if m.AddFunc != nil {
		return m.AddFunc(ctx, move)
	}
	return nil
}

func (m *MoveRepositoryMock) ListByGame(ctx context.Context, id domain.GameID) ([]domain.Move, error) {
	if m.ListByGameFunc != nil {
		return m.ListByGameFunc(ctx, id)
	}
	return nil, nil
}

func (m *MoveRepositoryMock) DeleteByGame(ctx context.Context, id domain.GameID) error {
	if m.DeleteByGameFunc != nil {
		return m.DeleteByGameFunc(ctx, id)
	}
	return nil
}
