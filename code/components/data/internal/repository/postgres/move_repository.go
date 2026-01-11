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

var _ repo.MoveRepository = (*MoveRepository)(nil)

type MoveRepository struct {
	db *sql.DB
}

func NewMoveRepository(db *sql.DB) *MoveRepository {
	return &MoveRepository{db: db}
}

func (r *MoveRepository) Add(ctx context.Context, move domain.Move) error {
	if move.ID == "" {
		return fmt.Errorf("move id empty: %w", ErrInvalidData)
	}
	pathBytes, err := marshalPath(move.Trajectory)
	if err != nil {
		return err
	}
	const query = `INSERT INTO moves (id, game_id, piece_id, number, start_row, start_col, end_row, end_col, path_json, created_at)
                   VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`
	_, err = r.db.ExecContext(ctx, query,
		move.ID,
		move.GameID,
		move.PieceID,
		int(move.Number),
		int(move.StartPosition.Row),
		int(move.StartPosition.Col),
		int(move.EndPosition.Row),
		int(move.EndPosition.Col),
		pathBytes,
		move.CreatedAt.ToTime(),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("move %s: %w", move.ID, ErrAlreadyExists)
		}
		return err
	}
	return nil
}

func (r *MoveRepository) GetByID(ctx context.Context, id domain.MoveID) (domain.Move, error) {
	const query = `SELECT id, game_id, piece_id, number, start_row, start_col, end_row, end_col, path_json, created_at FROM moves WHERE id=$1`
	var (
		storedID                           string
		storedGameID                       string
		storedPieceID                      string
		number                             int
		startRow, startCol, endRow, endCol int
		pathBytes                          []byte
		createdAt                          sql.NullTime
	)
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&storedID,
		&storedGameID,
		&storedPieceID,
		&number,
		&startRow,
		&startCol,
		&endRow,
		&endCol,
		&pathBytes,
		&createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Move{}, fmt.Errorf("move %s: %w", id, ErrNotFound)
	}
	if err != nil {
		return domain.Move{}, err
	}
	trajectory, err := unmarshalPath(pathBytes)
	if err != nil {
		return domain.Move{}, err
	}
	start := domain.Position{Row: domain.Coordinate(startRow), Col: domain.Coordinate(startCol)}
	created := domain.NewTimestamp(createdAt.Time)
	move, err := domain.NewMove(domain.MoveID(storedID), domain.GameID(storedGameID), domain.PieceID(storedPieceID), domain.MoveNumber(number), start, trajectory, created)
	if err != nil {
		return domain.Move{}, err
	}
	return move, nil
}

func (r *MoveRepository) ListByGame(ctx context.Context, id domain.GameID) ([]domain.Move, error) {
	const query = `SELECT id, game_id, piece_id, number, start_row, start_col, end_row, end_col, path_json, created_at
                   FROM moves WHERE game_id=$1 ORDER BY number`
	rows, err := r.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var moves []domain.Move
	for rows.Next() {
		var (
			storedID                           string
			storedGameID                       string
			storedPieceID                      string
			number                             int
			startRow, startCol, endRow, endCol int
			pathBytes                          []byte
			createdAt                          time.Time
		)
		if err := rows.Scan(
			&storedID,
			&storedGameID,
			&storedPieceID,
			&number,
			&startRow,
			&startCol,
			&endRow,
			&endCol,
			&pathBytes,
			&createdAt,
		); err != nil {
			return nil, err
		}
		trajectory, err := unmarshalPath(pathBytes)
		if err != nil {
			return nil, err
		}
		start := domain.Position{Row: domain.Coordinate(startRow), Col: domain.Coordinate(startCol)}
		move, err := domain.NewMove(domain.MoveID(storedID), domain.GameID(storedGameID), domain.PieceID(storedPieceID), domain.MoveNumber(number), start, trajectory, domain.NewTimestamp(createdAt))
		if err != nil {
			return nil, err
		}
		moves = append(moves, move)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return moves, nil
}

type persistedPosition struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

func marshalPath(path []domain.Position) ([]byte, error) {
	data := make([]persistedPosition, 0, len(path))
	for _, pos := range path {
		data = append(data, persistedPosition{Row: int(pos.Row), Col: int(pos.Col)})
	}
	return json.Marshal(data)
}

func unmarshalPath(raw []byte) ([]domain.Position, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty path payload")
	}
	var data []persistedPosition
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("path payload missing entries")
	}
	result := make([]domain.Position, 0, len(data))
	for _, item := range data {
		result = append(result, domain.Position{Row: domain.Coordinate(item.Row), Col: domain.Coordinate(item.Col)})
	}
	return result, nil
}
