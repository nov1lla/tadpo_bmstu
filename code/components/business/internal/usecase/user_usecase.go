package usecase

import (
	"context"
	"errors"

	"ppo/sdk/domain"
	sdkport "ppo/sdk/port"
	sdkrepo "ppo/sdk/port/repo"
	sdkusecase "ppo/sdk/usecase"
)

var ErrInvalidUserData = errors.New("invalid user data")

type userUseCase struct {
	repository sdkrepo.UserRepository
	ids        sdkport.IDGenerator
}

func NewUserUseCase(repository sdkrepo.UserRepository, ids sdkport.IDGenerator) sdkusecase.UserUseCase {
	return &userUseCase{repository: repository, ids: ids}
}

func (uc *userUseCase) Create(ctx context.Context, cmd sdkusecase.CreateUserCommand) (domain.User, error) {
	if cmd.Name == "" {
		return domain.User{}, ErrInvalidUserData
	}
	if uc.ids == nil {
		return domain.User{}, errors.New("id generator not configured")
	}
	id := domain.UserID(uc.ids.NewID())
	user := domain.User{
		ID:        id,
		Name:      cmd.Name,
		Rating:    0,
		WinStreak: 0,
	}
	if err := uc.repository.Save(ctx, user); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (uc *userUseCase) Get(ctx context.Context, id domain.UserID) (domain.User, error) {
	return uc.repository.GetByID(ctx, id)
}

func (uc *userUseCase) UpdateFields(ctx context.Context, cmd sdkusecase.UpdateUserFieldsCommand) (domain.User, error) {
	user, err := uc.repository.GetByID(ctx, cmd.ID)
	if err != nil {
		return domain.User{}, err
	}

	if cmd.Rating != nil {
		user = user.WithRating(*cmd.Rating)
	}
	if cmd.WinStreak != nil {
		user = user.WithWinStreak(*cmd.WinStreak)
	}
	if cmd.LastGameID != nil {
		user = user.WithLastGame(*cmd.LastGameID)
	}

	if err := uc.repository.Update(ctx, user); err != nil {
		return domain.User{}, err
	}

	return user, nil
}
