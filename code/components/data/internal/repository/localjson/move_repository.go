package localjson

import (
	"context"
	"fmt"
	"sort"

	"ppo/sdk/domain"
	"ppo/sdk/port/repo"
)

var _ repo.MoveRepository = (*MoveRepository)(nil)

type MoveRepository struct {
	storage *Storage
}

func NewMoveRepository(storage *Storage) *MoveRepository {
	if storage == nil {
		panic("nil storage")
	}
	return &MoveRepository{storage: storage}
}

func (r *MoveRepository) Add(ctx context.Context, move domain.Move) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if move.ID == "" {
		return fmt.Errorf("move id empty: %w", ErrInvalidData)
	}

	files := r.storage.files
	files.mu.Lock()
	defer files.mu.Unlock()

	moves, err := files.loadMoves()
	if err != nil {
		return err
	}
	key := string(move.ID)
	if _, exists := moves[key]; exists {
		return fmt.Errorf("move %s: %w", move.ID, ErrAlreadyExists)
	}

	updated := cloneStoredMoves(moves)
	updated[key] = fromDomainMove(move)

	return files.persistMoves(updated)
}

func (r *MoveRepository) GetByID(ctx context.Context, id domain.MoveID) (domain.Move, error) {
	if err := ctx.Err(); err != nil {
		return domain.Move{}, err
	}

	files := r.storage.files
	files.mu.Lock()
	defer files.mu.Unlock()

	moves, err := files.loadMoves()
	if err != nil {
		return domain.Move{}, err
	}
	stored, ok := moves[string(id)]
	if !ok {
		return domain.Move{}, fmt.Errorf("move %s: %w", id, ErrNotFound)
	}
	return toDomainMove(stored)
}

func (r *MoveRepository) ListByGame(ctx context.Context, id domain.GameID) ([]domain.Move, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	files := r.storage.files
	files.mu.Lock()
	defer files.mu.Unlock()

	moves, err := files.loadMoves()
	if err != nil {
		return nil, err
	}
	var result []domain.Move
	for _, stored := range moves {
		if stored.GameID == string(id) {
			move, err := toDomainMove(stored)
			if err != nil {
				return nil, err
			}
			result = append(result, move)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Number < result[j].Number
	})
	return result, nil
}

func (r *MoveRepository) DeleteByGame(ctx context.Context, id domain.GameID) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	files := r.storage.files
	files.mu.Lock()
	defer files.mu.Unlock()

	moves, err := files.loadMoves()
	if err != nil {
		return err
	}

	updated := cloneStoredMoves(moves)
	for key, move := range updated {
		if move.GameID == string(id) {
			delete(updated, key)
		}
	}
	return files.persistMoves(updated)
}

func cloneStoredMoves(src map[string]storedMove) map[string]storedMove {
	duplicated := make(map[string]storedMove, len(src))
	for k, v := range src {
		trajectory := make([]storedPosition, len(v.Trajectory))
		copy(trajectory, v.Trajectory)
		clone := v
		clone.Trajectory = trajectory
		duplicated[k] = clone
	}
	return duplicated
}

func fromDomainMove(move domain.Move) storedMove {
	trajectory := make([]storedPosition, 0, len(move.Trajectory))
	for _, pos := range move.Trajectory {
		trajectory = append(trajectory, storedPosition{Row: int(pos.Row), Col: int(pos.Col)})
	}
	return storedMove{
		ID:         string(move.ID),
		GameID:     string(move.GameID),
		PieceID:    string(move.PieceID),
		Number:     int(move.Number),
		Start:      storedPosition{Row: int(move.StartPosition.Row), Col: int(move.StartPosition.Col)},
		Trajectory: trajectory,
		CreatedAt:  move.CreatedAt.ToTime(),
	}
}

func toDomainMove(stored storedMove) (domain.Move, error) {
	trajectory := make([]domain.Position, 0, len(stored.Trajectory))
	for _, pos := range stored.Trajectory {
		trajectory = append(trajectory, domain.Position{Row: domain.Coordinate(pos.Row), Col: domain.Coordinate(pos.Col)})
	}
	start := domain.Position{Row: domain.Coordinate(stored.Start.Row), Col: domain.Coordinate(stored.Start.Col)}
	created := domain.NewTimestamp(stored.CreatedAt)
	return domain.NewMove(domain.MoveID(stored.ID), domain.GameID(stored.GameID), domain.PieceID(stored.PieceID), domain.MoveNumber(stored.Number), start, trajectory, created)
}
