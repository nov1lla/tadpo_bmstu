package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"ppo/sdk/domain"
	"ppo/sdk/port/repo"
)

var _ repo.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByID(ctx context.Context, id domain.UserID) (domain.User, error) {
	const query = `SELECT id, name, rating, win_streak, last_game_id FROM users WHERE id = $1`
	var (
		user     domain.User
		lastGame sql.NullString
	)
	err := r.db.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.Name, &user.Rating, &user.WinStreak, &lastGame)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, fmt.Errorf("user %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return domain.User{}, err
	}
	if lastGame.Valid {
		gameID := domain.GameID(lastGame.String)
		user = user.WithLastGame(gameID)
	}
	return user, nil
}

func (r *UserRepository) Save(ctx context.Context, user domain.User) error {
	if user.ID == "" {
		return fmt.Errorf("user id empty: %w", ErrInvalidData)
	}
	const query = `INSERT INTO users (id, name, rating, win_streak, last_game_id) VALUES ($1,$2,$3,$4,$5)`
	var lastGame interface{}
	if user.LastGameID != nil {
		lastGame = *user.LastGameID
	}
	_, err := r.db.ExecContext(ctx, query, user.ID, user.Name, user.Rating, user.WinStreak, lastGame)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("user %s: %w", user.ID, ErrAlreadyExists)
		}
		return err
	}
	return nil
}

func (r *UserRepository) Update(ctx context.Context, user domain.User) error {
	if user.ID == "" {
		return fmt.Errorf("user id empty: %w", ErrInvalidData)
	}
	const query = `UPDATE users SET name=$2, rating=$3, win_streak=$4, last_game_id=$5 WHERE id=$1`
	var lastGame interface{}
	if user.LastGameID != nil {
		lastGame = *user.LastGameID
	}
	res, err := r.db.ExecContext(ctx, query, user.ID, user.Name, user.Rating, user.WinStreak, lastGame)
	if err != nil {
		return err
	}
	if err := checkRowsAffected(res); err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("user %s: %w", user.ID, err)
		}
		return err
	}
	return nil
}
