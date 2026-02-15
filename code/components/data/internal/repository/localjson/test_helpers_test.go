package localjson

import (
	"context"
	"testing"
	"time"

	"ppo/sdk/domain"
)

func newTestStorage(t *testing.T) *Storage {
	t.Helper()
	dir := t.TempDir()
	storage, err := NewStorage(dir)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}
	return storage
}

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

func (b *UserBuilder) Build() domain.User {
	return b.user
}

type GameBuilder struct {
	id            domain.GameID
	userID        domain.UserID
	color         domain.PlayerColor
	isPlayerFirst bool
	start         domain.Timestamp
}

func NewGameBuilder() *GameBuilder {
	return &GameBuilder{
		id:            "game-default",
		userID:        "user-default",
		color:         domain.PlayerColorLight,
		isPlayerFirst: true,
		start:         domain.NewTimestamp(time.Date(2024, 2, 1, 10, 0, 0, 0, time.UTC)),
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

func (b *GameBuilder) WithStart(start domain.Timestamp) *GameBuilder {
	b.start = start
	return b
}

func (b *GameBuilder) Build() domain.Game {
	return domain.NewGame(b.id, b.userID, b.color, b.isPlayerFirst, b.start)
}

type MoveBuilder struct {
	id         domain.MoveID
	gameID     domain.GameID
	pieceID    domain.PieceID
	number     domain.MoveNumber
	start      domain.Position
	trajectory []domain.Position
	createdAt  domain.Timestamp
}

func NewMoveBuilder() *MoveBuilder {
	return &MoveBuilder{
		id:         "move-default",
		gameID:     "game-default",
		pieceID:    "piece-default",
		number:     1,
		start:      domain.Position{Row: 5, Col: 0},
		trajectory: []domain.Position{{Row: 4, Col: 1}},
		createdAt:  domain.NewTimestamp(time.Date(2024, 2, 1, 10, 5, 0, 0, time.UTC)),
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

func (b *MoveBuilder) WithNumber(number domain.MoveNumber) *MoveBuilder {
	b.number = number
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

type CredentialsBuilder struct {
	creds domain.UserCredentials
}

func NewCredentialsBuilder() *CredentialsBuilder {
	return &CredentialsBuilder{
		creds: domain.UserCredentials{
			UserID:       "user-default",
			Login:        "login-default",
			PasswordHash: "hash-default",
		},
	}
}

func (b *CredentialsBuilder) WithLogin(login domain.Login) *CredentialsBuilder {
	b.creds.Login = login
	return b
}

func (b *CredentialsBuilder) WithUserID(id domain.UserID) *CredentialsBuilder {
	b.creds.UserID = id
	return b
}

func (b *CredentialsBuilder) Build() domain.UserCredentials {
	return b.creds
}

func MotherUser() domain.User {
	return NewUserBuilder().WithID("user-mother").WithName("Mother").Build()
}

func MotherGame() domain.Game {
	return NewGameBuilder().WithID("game-mother").WithUserID("user-mother").Build()
}

func MotherMove(t *testing.T) domain.Move {
	return NewMoveBuilder().WithID("move-mother").WithGameID("game-mother").Build(t)
}

func MotherCredentials() domain.UserCredentials {
	return NewCredentialsBuilder().WithLogin("mother-login").WithUserID("user-mother").Build()
}

type userRepoFixture struct {
	repo *UserRepository
	ctx  context.Context
}

func newUserRepoFixture(t *testing.T) *userRepoFixture {
	storage := newTestStorage(t)
	return &userRepoFixture{repo: NewUserRepository(storage), ctx: context.Background()}
}

type gameRepoFixture struct {
	repo *GameRepository
	ctx  context.Context
}

func newGameRepoFixture(t *testing.T) *gameRepoFixture {
	storage := newTestStorage(t)
	return &gameRepoFixture{repo: NewGameRepository(storage), ctx: context.Background()}
}

type moveRepoFixture struct {
	repo *MoveRepository
	ctx  context.Context
}

func newMoveRepoFixture(t *testing.T) *moveRepoFixture {
	storage := newTestStorage(t)
	return &moveRepoFixture{repo: NewMoveRepository(storage), ctx: context.Background()}
}

type credentialsRepoFixture struct {
	repo *UserCredentialsRepository
	ctx  context.Context
}

func newCredentialsRepoFixture(t *testing.T) *credentialsRepoFixture {
	storage := newTestStorage(t)
	return &credentialsRepoFixture{repo: NewUserCredentialsRepository(storage), ctx: context.Background()}
}

func canceledContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
