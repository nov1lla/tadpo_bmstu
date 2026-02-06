package localjson

import (
	"context"
	"fmt"

	"ppo/sdk/domain"
	"ppo/sdk/port/repo"
)

var _ repo.UserCredentialsRepository = (*UserCredentialsRepository)(nil)

type UserCredentialsRepository struct {
	storage *Storage
}

func NewUserCredentialsRepository(storage *Storage) *UserCredentialsRepository {
	if storage == nil {
		panic("nil storage")
	}
	return &UserCredentialsRepository{storage: storage}
}

func (r *UserCredentialsRepository) GetByLogin(ctx context.Context, login domain.Login) (domain.UserCredentials, error) {
	if err := ctx.Err(); err != nil {
		return domain.UserCredentials{}, err
	}
	if login == "" {
		return domain.UserCredentials{}, fmt.Errorf("login empty: %w", ErrInvalidData)
	}

	r.storage.files.mu.Lock()
	defer r.storage.files.mu.Unlock()

	credentials, err := r.storage.files.loadCredentials()
	if err != nil {
		return domain.UserCredentials{}, err
	}
	entry, ok := credentials[string(login)]
	if !ok {
		return domain.UserCredentials{}, fmt.Errorf("login %s: %w", login, ErrNotFound)
	}
	return domain.UserCredentials{
		UserID:       domain.UserID(entry.UserID),
		Login:        login,
		PasswordHash: domain.PasswordHash(entry.PasswordHash),
	}, nil
}

func (r *UserCredentialsRepository) Save(ctx context.Context, creds domain.UserCredentials) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if creds.UserID == "" {
		return fmt.Errorf("user id empty: %w", ErrInvalidData)
	}
	if creds.Login == "" {
		return fmt.Errorf("login empty: %w", ErrInvalidData)
	}
	if creds.PasswordHash == "" {
		return fmt.Errorf("password hash empty: %w", ErrInvalidData)
	}

	r.storage.files.mu.Lock()
	defer r.storage.files.mu.Unlock()

	credentials, err := r.storage.files.loadCredentials()
	if err != nil {
		return err
	}
	key := string(creds.Login)
	if _, exists := credentials[key]; exists {
		return fmt.Errorf("login %s: %w", creds.Login, ErrAlreadyExists)
	}
	credentials[key] = storedUserCredentials{
		UserID:       string(creds.UserID),
		PasswordHash: string(creds.PasswordHash),
	}
	return r.storage.files.persistCredentials(credentials)
}

func (r *UserCredentialsRepository) UpdatePasswordHash(ctx context.Context, login domain.Login, hash domain.PasswordHash) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if login == "" {
		return fmt.Errorf("login empty: %w", ErrInvalidData)
	}
	if hash == "" {
		return fmt.Errorf("password hash empty: %w", ErrInvalidData)
	}

	r.storage.files.mu.Lock()
	defer r.storage.files.mu.Unlock()

	credentials, err := r.storage.files.loadCredentials()
	if err != nil {
		return err
	}
	key := string(login)
	entry, ok := credentials[key]
	if !ok {
		return fmt.Errorf("login %s: %w", login, ErrNotFound)
	}
	entry.PasswordHash = string(hash)
	credentials[key] = entry
	return r.storage.files.persistCredentials(credentials)
}
