package repo

import (
	"context"

	"ppo/sdk/domain"
)

type UserRepositoryMock struct {
	GetByIDFunc func(ctx context.Context, id domain.UserID) (domain.User, error)
	SaveFunc    func(ctx context.Context, user domain.User) error
	UpdateFunc  func(ctx context.Context, user domain.User) error
}

func (m *UserRepositoryMock) GetByID(ctx context.Context, id domain.UserID) (domain.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return domain.User{}, nil
}

func (m *UserRepositoryMock) Save(ctx context.Context, user domain.User) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, user)
	}
	return nil
}

func (m *UserRepositoryMock) Update(ctx context.Context, user domain.User) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, user)
	}
	return nil
}
