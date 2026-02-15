package repo

import (
	"context"

	"ppo/sdk/domain"
)

type GameRepositoryMock struct {
	GetByIDFunc          func(ctx context.Context, id domain.GameID) (domain.Game, error)
	ListByUserFunc       func(ctx context.Context, userID domain.UserID) ([]domain.Game, error)
	SaveFunc             func(ctx context.Context, game domain.Game) error
	UpdateFunc           func(ctx context.Context, game domain.Game) error
	DeleteFunc           func(ctx context.Context, id domain.GameID) error
	BoardStateFunc       func(ctx context.Context, id domain.GameID) (domain.BoardState, error)
	UpdateBoardStateFunc func(ctx context.Context, id domain.GameID, state domain.BoardState) error
}

func (m *GameRepositoryMock) GetByID(ctx context.Context, id domain.GameID) (domain.Game, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return domain.Game{}, nil
}

func (m *GameRepositoryMock) ListByUser(ctx context.Context, userID domain.UserID) ([]domain.Game, error) {
	if m.ListByUserFunc != nil {
		return m.ListByUserFunc(ctx, userID)
	}
	return nil, nil
}

func (m *GameRepositoryMock) Save(ctx context.Context, game domain.Game) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, game)
	}
	return nil
}

func (m *GameRepositoryMock) Update(ctx context.Context, game domain.Game) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, game)
	}
	return nil
}

func (m *GameRepositoryMock) Delete(ctx context.Context, id domain.GameID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *GameRepositoryMock) BoardState(ctx context.Context, id domain.GameID) (domain.BoardState, error) {
	if m.BoardStateFunc != nil {
		return m.BoardStateFunc(ctx, id)
	}
	return domain.BoardState{}, nil
}

func (m *GameRepositoryMock) UpdateBoardState(ctx context.Context, id domain.GameID, state domain.BoardState) error {
	if m.UpdateBoardStateFunc != nil {
		return m.UpdateBoardStateFunc(ctx, id, state)
	}
	return nil
}
