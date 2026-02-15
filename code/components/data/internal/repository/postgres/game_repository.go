package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ppo/sdk/domain"
	"ppo/sdk/port/repo"
)

var _ repo.GameRepository = (*GameRepository)(nil)

type GameRepository struct {
	db *sql.DB
}

type persistedBoardPiece struct {
	Row   int    `json:"row"`
	Col   int    `json:"col"`
	Color string `json:"color"`
	Kind  string `json:"kind"`
}

func NewGameRepository(db *sql.DB) *GameRepository {
	return &GameRepository{db: db}
}

func (r *GameRepository) Save(ctx context.Context, game domain.Game) error {
	if game.ID == "" {
		return fmt.Errorf("game id empty: %w", ErrInvalidData)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const query = `INSERT INTO games (id, user_id, start_time, end_time, status, player_color, is_player_first, board_size)
	                   VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	var end interface{}
	if game.End != nil {
		end = game.End.ToTime()
	}
	_, err = tx.ExecContext(ctx, query,
		game.ID,
		game.UserID,
		game.Start.ToTime(),
		end,
		game.Status,
		game.PlayerColor,
		game.IsPlayerFirst,
		int(domain.DefaultBoardSize),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("game %s: %w", game.ID, ErrAlreadyExists)
		}
		return err
	}

	const boardQuery = `INSERT INTO game_board_states (game_id, board_size, pieces_json) VALUES ($1,$2,$3)`
	emptyPieces, _ := json.Marshal(map[string]persistedBoardPiece{})
	if _, err := tx.ExecContext(ctx, boardQuery, game.ID, int(domain.DefaultBoardSize), emptyPieces); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *GameRepository) Update(ctx context.Context, game domain.Game) error {
	if game.ID == "" {
		return fmt.Errorf("game id empty: %w", ErrInvalidData)
	}
	const query = `UPDATE games SET user_id=$2, start_time=$3, end_time=$4, status=$5, player_color=$6, is_player_first=$7 WHERE id=$1`
	var end interface{}
	if game.End != nil {
		end = game.End.ToTime()
	}
	res, err := r.db.ExecContext(ctx, query,
		game.ID,
		game.UserID,
		game.Start.ToTime(),
		end,
		game.Status,
		game.PlayerColor,
		game.IsPlayerFirst,
	)
	if err != nil {
		return err
	}
	if err := checkRowsAffected(res); err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("game %s: %w", game.ID, err)
		}
		return err
	}
	return nil
}

func (r *GameRepository) Delete(ctx context.Context, id domain.GameID) error {
	const boardQuery = `DELETE FROM game_board_states WHERE game_id=$1`
	const gameQuery = `DELETE FROM games WHERE id=$1`
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, boardQuery, id); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, gameQuery, id)
	if err != nil {
		return err
	}
	if err := checkRowsAffected(res); err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("game %s: %w", id, err)
		}
		return err
	}
	return tx.Commit()
}

func (r *GameRepository) GetByID(ctx context.Context, id domain.GameID) (domain.Game, error) {
	const query = `SELECT id, user_id, start_time, end_time, status, player_color, is_player_first FROM games WHERE id=$1`
	var (
		game domain.Game
		end  sql.NullTime
	)
	var start time.Time
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&game.ID,
		&game.UserID,
		&start,
		&end,
		&game.Status,
		&game.PlayerColor,
		&game.IsPlayerFirst,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Game{}, fmt.Errorf("game %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return domain.Game{}, err
	}
	game.Start = domain.NewTimestamp(start)
	if end.Valid {
		ts := domain.NewTimestamp(end.Time)
		game.End = &ts
	}
	return game, nil
}

func (r *GameRepository) ListByUser(ctx context.Context, userID domain.UserID) ([]domain.Game, error) {
	const query = `SELECT id, user_id, start_time, end_time, status, player_color, is_player_first FROM games WHERE user_id=$1 ORDER BY start_time DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var games []domain.Game
	for rows.Next() {
		var (
			g     domain.Game
			end   sql.NullTime
			start time.Time
		)
		if err := rows.Scan(&g.ID, &g.UserID, &start, &end, &g.Status, &g.PlayerColor, &g.IsPlayerFirst); err != nil {
			return nil, err
		}
		g.Start = domain.NewTimestamp(start)
		if end.Valid {
			ts := domain.NewTimestamp(end.Time)
			g.End = &ts
		}
		games = append(games, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return games, nil
}

func (r *GameRepository) BoardState(ctx context.Context, id domain.GameID) (domain.BoardState, error) {
	const query = `SELECT board_size, pieces_json FROM game_board_states WHERE game_id=$1`
	var (
		size        int
		piecesBytes []byte
	)
	err := r.db.QueryRowContext(ctx, query, id).Scan(&size, &piecesBytes)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.BoardState{}, fmt.Errorf("game %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return domain.BoardState{}, err
	}
	state := domain.NewBoardState(domain.BoardSize(size))
	if len(piecesBytes) > 0 && string(piecesBytes) != "{}" {
		decoded := make(map[string]persistedBoardPiece)
		if err := json.Unmarshal(piecesBytes, &decoded); err != nil {
			return domain.BoardState{}, err
		}
		for key, piece := range decoded {
			state.Pieces[domain.PieceID(key)] = domain.BoardPiece{
				ID:       domain.PieceID(key),
				Position: domain.Position{Row: domain.Coordinate(piece.Row), Col: domain.Coordinate(piece.Col)},
				Color:    domain.PlayerColor(piece.Color),
				Kind:     domain.PieceKind(piece.Kind),
			}
		}
	}
	return state, nil
}

func (r *GameRepository) UpdateBoardState(ctx context.Context, id domain.GameID, state domain.BoardState) error {
	if state.Size < 0 {
		return fmt.Errorf("board size negative: %w", ErrInvalidData)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const query = `UPDATE game_board_states SET board_size=$2, pieces_json=$3 WHERE game_id=$1`
	encoded := make(map[string]persistedBoardPiece, len(state.Pieces))
	for key, value := range state.Pieces {
		encoded[string(key)] = persistedBoardPiece{
			Row:   int(value.Position.Row),
			Col:   int(value.Position.Col),
			Color: string(value.Color),
			Kind:  string(value.Kind),
		}
	}
	bytes, err := json.Marshal(encoded)
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, query, id, int(state.Size), bytes)
	if err != nil {
		return err
	}
	if err := checkRowsAffected(res); err != nil {
		if errors.Is(err, ErrNotFound) {
			return fmt.Errorf("game %s: %w", id, err)
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE games SET board_size=$2 WHERE id=$1`, id, int(state.Size)); err != nil {
		return err
	}
	return tx.Commit()
}
