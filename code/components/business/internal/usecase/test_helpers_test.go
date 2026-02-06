package usecase

import (
	"context"
	"testing"
	"time"

	"ppo/sdk/domain"
	"ppo/sdk/port/repo"
)

type UserBuilder struct {
	user domain.User
}

func NewUserBuilder() *UserBuilder {
	return &UserBuilder{
		user: domain.User{
			ID:        "user-default",
			Name:      "Default User",
			Rating:    1000,
			WinStreak: 0,
		},
	}
}

func (b *UserBuilder) WithID(id domain.UserID) *UserBuilder {
	b.user.ID = id
	return b
}

func (b *UserBuilder) WithName(name domain.UserName) *UserBuilder {
	b.user.Name = name
	return b
}

func (b *UserBuilder) WithRating(rating domain.Rating) *UserBuilder {
	b.user.Rating = rating
	return b
}

func (b *UserBuilder) WithWinStreak(streak domain.WinStreak) *UserBuilder {
	b.user.WinStreak = streak
	return b
}

func (b *UserBuilder) WithLastGame(id domain.GameID) *UserBuilder {
	b.user = b.user.WithLastGame(id)
	return b
}

func (b *UserBuilder) Build() domain.User {
	return b.user
}

type GameBuilder struct {
	id           domain.GameID
	userID       domain.UserID
	color        domain.PlayerColor
	isPlayerFirst bool
	start        domain.Timestamp
}

func NewGameBuilder() *GameBuilder {
	return &GameBuilder{
		id:            "game-default",
		userID:        "user-default",
		color:         domain.PlayerColorLight,
		isPlayerFirst: true,
		start:         domain.NewTimestamp(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
	}
}

func (b *GameBuilder) WithID(id domain.GameID) *GameBuilder {
	b.id = id
	return b
}

func (b *GameBuilder) WithUserID(id domain.UserID) *GameBuilder {
	b.userID = id
	return b
}

func (b *GameBuilder) WithColor(color domain.PlayerColor) *GameBuilder {
	b.color = color
	return b
}

func (b *GameBuilder) WithPlayerFirst(first bool) *GameBuilder {
	b.isPlayerFirst = first
	return b
}

func (b *GameBuilder) WithStart(start domain.Timestamp) *GameBuilder {
	b.start = start
	return b
}

func (b *GameBuilder) Build() domain.Game {
	return domain.NewGame(b.id, b.userID, b.color, b.isPlayerFirst, b.start)
}

type MoveBuilder struct {
	id        domain.MoveID
	gameID    domain.GameID
	pieceID   domain.PieceID
	number    domain.MoveNumber
	start     domain.Position
	trajectory []domain.Position
	createdAt domain.Timestamp
}

func NewMoveBuilder() *MoveBuilder {
	return &MoveBuilder{
		id:        "move-default",
		gameID:    "game-default",
		pieceID:   "piece-default",
		number:    1,
		start:     domain.Position{Row: 5, Col: 0},
		trajectory: []domain.Position{{Row: 4, Col: 1}},
		createdAt: domain.NewTimestamp(time.Date(2024, 1, 1, 10, 5, 0, 0, time.UTC)),
	}
}

func (b *MoveBuilder) WithID(id domain.MoveID) *MoveBuilder {
	b.id = id
	return b
}

func (b *MoveBuilder) WithGameID(id domain.GameID) *MoveBuilder {
	b.gameID = id
	return b
}

func (b *MoveBuilder) WithPieceID(id domain.PieceID) *MoveBuilder {
	b.pieceID = id
	return b
}

func (b *MoveBuilder) WithNumber(number domain.MoveNumber) *MoveBuilder {
	b.number = number
	return b
}

func (b *MoveBuilder) WithTrajectory(path []domain.Position) *MoveBuilder {
	b.trajectory = path
	return b
}

func (b *MoveBuilder) Build(t *testing.T) domain.Move {
	t.Helper()
	move, err := domain.NewMove(b.id, b.gameID, b.pieceID, b.number, b.start, b.trajectory, b.createdAt)
	if err != nil {
		t.Fatalf("build move: %v", err)
	}
	return move
}

func MotherUser() domain.User {
	return NewUserBuilder().WithID("user-mother").WithName("Mother").WithRating(900).Build()
}

func MotherGame() domain.Game {
	return NewGameBuilder().WithID("game-mother").WithUserID("user-mother").Build()
}

func MotherMove(t *testing.T) domain.Move {
	return NewMoveBuilder().WithID("move-mother").WithGameID("game-mother").Build(t)
}

type inMemoryUserRepo struct {
	users map[domain.UserID]domain.User
}

func newInMemoryUserRepo() *inMemoryUserRepo {
	return &inMemoryUserRepo{users: make(map[domain.UserID]domain.User)}
}

func (r *inMemoryUserRepo) Save(ctx context.Context, user domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if user.ID == "" {
		return repo.ErrInvalidData
	}
	if _, exists := r.users[user.ID]; exists {
		return repo.ErrAlreadyExists
	}
	r.users[user.ID] = user
	return nil
}

func (r *inMemoryUserRepo) GetByID(ctx context.Context, id domain.UserID) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}
	user, ok := r.users[id]
	if !ok {
		return domain.User{}, repo.ErrNotFound
	}
	return user, nil
}

func (r *inMemoryUserRepo) Update(ctx context.Context, user domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, exists := r.users[user.ID]; !exists {
		return repo.ErrNotFound
	}
	r.users[user.ID] = user
	return nil
}

type inMemoryCredsRepo struct {
	entries map[domain.Login]domain.UserCredentials
}

func newInMemoryCredsRepo() *inMemoryCredsRepo {
	return &inMemoryCredsRepo{entries: make(map[domain.Login]domain.UserCredentials)}
}

func (r *inMemoryCredsRepo) GetByLogin(ctx context.Context, login domain.Login) (domain.UserCredentials, error) {
	if err := ctx.Err(); err != nil {
		return domain.UserCredentials{}, err
	}
	creds, ok := r.entries[login]
	if !ok {
		return domain.UserCredentials{}, repo.ErrNotFound
	}
	return creds, nil
}

func (r *inMemoryCredsRepo) Save(ctx context.Context, creds domain.UserCredentials) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, exists := r.entries[creds.Login]; exists {
		return repo.ErrAlreadyExists
	}
	r.entries[creds.Login] = creds
	return nil
}

func (r *inMemoryCredsRepo) UpdatePasswordHash(ctx context.Context, login domain.Login, hash domain.PasswordHash) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	entry, ok := r.entries[login]
	if !ok {
		return repo.ErrNotFound
	}
	entry.PasswordHash = hash
	r.entries[login] = entry
	return nil
}
