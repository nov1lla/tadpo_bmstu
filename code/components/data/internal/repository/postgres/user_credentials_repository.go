package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"ppo/sdk/domain"
	"ppo/sdk/port/repo"
)

var _ repo.UserCredentialsRepository = (*UserCredentialsRepository)(nil)

type UserCredentialsRepository struct {
	db *sql.DB
}

func NewUserCredentialsRepository(db *sql.DB) *UserCredentialsRepository {
	return &UserCredentialsRepository{db: db}
}

func (r *UserCredentialsRepository) GetByLogin(ctx context.Context, login domain.Login) (domain.UserCredentials, error) {
	const query = `SELECT user_id, login, password_hash FROM user_credentials WHERE login = $1`
	var creds domain.UserCredentials
	err := r.db.QueryRowContext(ctx, query, login).Scan(&creds.UserID, &creds.Login, &creds.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.UserCredentials{}, fmt.Errorf("login %s: %w", login, ErrNotFound)
	}
	if err != nil {
		return domain.UserCredentials{}, err
	}
	return creds, nil
}

func (r *UserCredentialsRepository) Save(ctx context.Context, creds domain.UserCredentials) error {
	if creds.UserID == "" {
		return fmt.Errorf("user id empty: %w", ErrInvalidData)
	}
	if creds.Login == "" {
		return fmt.Errorf("login empty: %w", ErrInvalidData)
	}
	if creds.PasswordHash == "" {
		return fmt.Errorf("password hash empty: %w", ErrInvalidData)
	}
	const query = `INSERT INTO user_credentials (user_id, login, password_hash) VALUES ($1,$2,$3)`
	_, err := r.db.ExecContext(ctx, query, creds.UserID, creds.Login, creds.PasswordHash)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("login %s: %w", creds.Login, ErrAlreadyExists)
		}
		return err
	}
	return nil
}
