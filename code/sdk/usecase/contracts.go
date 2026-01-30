package usecase

import (
	"context"

	"ppo/sdk/domain"
)

type UserUseCase interface {
	Create(ctx context.Context, cmd CreateUserCommand) (domain.User, error)
	Get(ctx context.Context, id domain.UserID) (domain.User, error)
	UpdateFields(ctx context.Context, cmd UpdateUserFieldsCommand) (domain.User, error)
}

type CreateUserCommand struct {
	Name domain.UserName
}

type UpdateUserFieldsCommand struct {
	ID         domain.UserID
	Rating     *domain.Rating
	WinStreak  *domain.WinStreak
	LastGameID *domain.GameID
}

type GameUseCase interface {
	New(ctx context.Context, cmd CreateGameCommand) (domain.Game, error)
	ListByUser(ctx context.Context, userID domain.UserID) ([]domain.Game, error)
	Get(ctx context.Context, id domain.GameID) (domain.Game, error)
	GenerateHistory(ctx context.Context, id domain.GameID) ([]domain.Move, error)
	GetChessboard(ctx context.Context, id domain.GameID) (domain.BoardState, error)
	ProcessMove(ctx context.Context, cmd ProcessMoveCommand) (domain.Move, error)
	FinishGame(ctx context.Context, cmd FinishGameCommand) (domain.Game, error)
	UpdateBoardState(ctx context.Context, id domain.GameID, state domain.BoardState) error
	Delete(ctx context.Context, id domain.GameID) error
}

type CreateGameCommand struct {
	UserID        domain.UserID
	PlayerColor   domain.PlayerColor
	IsPlayerFirst bool
	StartAt       *domain.Timestamp
}

type ProcessMoveCommand struct {
	GameID        domain.GameID
	StartPosition domain.Position
	Trajectory    []domain.Position
	EndPosition   *domain.Position
	PerformedAt   *domain.Timestamp
}

type FinishGameCommand struct {
	GameID domain.GameID
	End    domain.Timestamp
}

type MoveUseCase interface {
	AddUserMove(ctx context.Context, cmd AddUserMoveCommand) (domain.Move, error)
	AddOpponentMove(ctx context.Context, cmd AddOpponentMoveCommand) (domain.Move, error)
	GetOpponentMove(ctx context.Context, cmd GetOpponentMoveCommand) (domain.Move, error)
}

type AddUserMoveCommand struct {
	GameID        domain.GameID
	StartPosition domain.Position
	Trajectory    []domain.Position
	EndPosition   *domain.Position
	PerformedAt   *domain.Timestamp
}

type AddOpponentMoveCommand struct {
	GameID        domain.GameID
	StartPosition domain.Position
	Trajectory    []domain.Position
	EndPosition   *domain.Position
	PerformedAt   *domain.Timestamp
}

type GetOpponentMoveCommand struct {
	GameID     domain.GameID
	Difficulty domain.DifficultyLevel
}

type MoveAnimationUseCase interface {
	Build(ctx context.Context, cmd MoveAnimationCommand) (domain.Animation, error)
}

type MoveAnimationCommand struct {
	Board domain.BoardState
	Move  domain.Move
}

type AuthUseCase interface {
	Register(ctx context.Context, cmd RegisterCommand) (domain.User, error)
	Login(ctx context.Context, cmd LoginCommand) (domain.User, error)
}

type RegisterCommand struct {
	Login    domain.Login
	Password string
}

type LoginCommand struct {
	Login    domain.Login
	Password string
}
