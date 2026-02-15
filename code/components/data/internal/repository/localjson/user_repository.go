package localjson

import (
	"context"
	"fmt"

	"ppo/sdk/domain"
	"ppo/sdk/port/repo"
)

var _ repo.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	storage *Storage
}

func NewUserRepository(storage *Storage) *UserRepository {
	if storage == nil {
		panic("nil storage")
	}
	return &UserRepository{storage: storage}
}

func (r *UserRepository) GetByID(ctx context.Context, id domain.UserID) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}
	r.storage.files.mu.Lock()
	defer r.storage.files.mu.Unlock()

	users, err := r.storage.files.loadUsers()
	if err != nil {
		return domain.User{}, err
	}
	stored, ok := users[string(id)]
	if !ok {
		return domain.User{}, fmt.Errorf("user %s: %w", id, ErrNotFound)
	}
	return toDomainUser(stored), nil
}

func (r *UserRepository) Save(ctx context.Context, user domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if user.ID == "" {
		return fmt.Errorf("user id empty: %w", ErrInvalidData)
	}

	r.storage.files.mu.Lock()
	defer r.storage.files.mu.Unlock()

	users, err := r.storage.files.loadUsers()
	if err != nil {
		return err
	}
	key := string(user.ID)
	if _, exists := users[key]; exists {
		return fmt.Errorf("user %s: %w", user.ID, ErrAlreadyExists)
	}
	users[key] = fromDomainUser(user)
	return r.storage.files.persistUsers(users)
}

func (r *UserRepository) Update(ctx context.Context, user domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if user.ID == "" {
		return fmt.Errorf("user id empty: %w", ErrInvalidData)
	}

	r.storage.files.mu.Lock()
	defer r.storage.files.mu.Unlock()

	users, err := r.storage.files.loadUsers()
	if err != nil {
		return err
	}
	key := string(user.ID)
	if _, exists := users[key]; !exists {
		return fmt.Errorf("user %s: %w", user.ID, ErrNotFound)
	}
	users[key] = fromDomainUser(user)
	return r.storage.files.persistUsers(users)
}

func toDomainUser(stored storedUser) domain.User {
	user := domain.User{
		ID:        domain.UserID(stored.ID),
		Name:      domain.UserName(stored.Name),
		Rating:    domain.Rating(stored.Rating),
		WinStreak: domain.WinStreak(stored.WinStreak),
	}
	if stored.LastGameID != nil {
		gameID := domain.GameID(*stored.LastGameID)
		user = user.WithLastGame(gameID)
	}
	return user
}

func fromDomainUser(user domain.User) storedUser {
	stored := storedUser{
		ID:        string(user.ID),
		Name:      string(user.Name),
		Rating:    int(user.Rating),
		WinStreak: int(user.WinStreak),
	}
	if user.LastGameID != nil {
		value := string(*user.LastGameID)
		stored.LastGameID = &value
	}
	return stored
}
