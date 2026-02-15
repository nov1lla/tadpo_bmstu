package localjson

import (
	"context"
	"fmt"
	"sort"

	"ppo/sdk/domain"
	"ppo/sdk/port/repo"
)

var _ repo.GameRepository = (*GameRepository)(nil)

type GameRepository struct {
	storage *Storage
}

func NewGameRepository(storage *Storage) *GameRepository {
	if storage == nil {
		panic("nil storage")
	}
	return &GameRepository{storage: storage}
}

func (r *GameRepository) Save(ctx context.Context, game domain.Game) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if game.ID == "" {
		return fmt.Errorf("game id empty: %w", ErrInvalidData)
	}

	files := r.storage.files
	files.mu.Lock()
	defer files.mu.Unlock()

	games, err := files.loadGames()
	if err != nil {
		return err
	}
	key := string(game.ID)
	if _, exists := games[key]; exists {
		return fmt.Errorf("game %s: %w", game.ID, ErrAlreadyExists)
	}
	boardStates, err := files.loadBoardStates()
	if err != nil {
		return err
	}

	updatedGames := cloneStoredGames(games)
	stored := fromDomainGame(game)
	stored.BoardSize = int(domain.DefaultBoardSize)
	updatedGames[key] = stored

	updatedStates := cloneStoredBoardStates(boardStates)
	updatedStates[key] = storedBoardState{Size: int(domain.DefaultBoardSize), Pieces: map[string]storedBoardPiece{}}

	if err := files.persistGames(updatedGames); err != nil {
		return err
	}
	if err := files.persistBoardStates(updatedStates); err != nil {
		_ = files.persistGames(games)
		return err
	}
	return nil
}

func (r *GameRepository) Update(ctx context.Context, game domain.Game) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if game.ID == "" {
		return fmt.Errorf("game id empty: %w", ErrInvalidData)
	}

	files := r.storage.files
	files.mu.Lock()
	defer files.mu.Unlock()

	games, err := files.loadGames()
	if err != nil {
		return err
	}
	key := string(game.ID)
	stored, exists := games[key]
	if !exists {
		return fmt.Errorf("game %s: %w", game.ID, ErrNotFound)
	}

	updatedGames := cloneStoredGames(games)
	replacement := fromDomainGame(game)
	replacement.BoardSize = stored.BoardSize
	updatedGames[key] = replacement

	return files.persistGames(updatedGames)
}

func (r *GameRepository) Delete(ctx context.Context, id domain.GameID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("game id empty: %w", ErrInvalidData)
	}

	files := r.storage.files
	files.mu.Lock()
	defer files.mu.Unlock()

	games, err := files.loadGames()
	if err != nil {
		return err
	}
	if _, exists := games[string(id)]; !exists {
		return fmt.Errorf("game %s: %w", id, ErrNotFound)
	}
	boardStates, err := files.loadBoardStates()
	if err != nil {
		return err
	}

	updatedGames := cloneStoredGames(games)
	delete(updatedGames, string(id))

	updatedStates := cloneStoredBoardStates(boardStates)
	delete(updatedStates, string(id))

	if err := files.persistGames(updatedGames); err != nil {
		return err
	}
	if err := files.persistBoardStates(updatedStates); err != nil {
		_ = files.persistGames(games)
		return err
	}
	return nil
}

func (r *GameRepository) GetByID(ctx context.Context, id domain.GameID) (domain.Game, error) {
	if err := ctx.Err(); err != nil {
		return domain.Game{}, err
	}

	files := r.storage.files
	files.mu.Lock()
	defer files.mu.Unlock()

	games, err := files.loadGames()
	if err != nil {
		return domain.Game{}, err
	}
	stored, ok := games[string(id)]
	if !ok {
		return domain.Game{}, fmt.Errorf("game %s: %w", id, ErrNotFound)
	}
	return toDomainGame(stored), nil
}

func (r *GameRepository) ListByUser(ctx context.Context, userID domain.UserID) ([]domain.Game, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	files := r.storage.files
	files.mu.Lock()
	defer files.mu.Unlock()

	games, err := files.loadGames()
	if err != nil {
		return nil, err
	}
	var result []domain.Game
	for _, stored := range games {
		if stored.UserID == string(userID) {
			result = append(result, toDomainGame(stored))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Start.ToTime().After(result[j].Start.ToTime())
	})
	return result, nil
}

func (r *GameRepository) BoardState(ctx context.Context, id domain.GameID) (domain.BoardState, error) {
	if err := ctx.Err(); err != nil {
		return domain.BoardState{}, err
	}

	files := r.storage.files
	files.mu.Lock()
	defer files.mu.Unlock()

	states, err := files.loadBoardStates()
	if err != nil {
		return domain.BoardState{}, err
	}
	stored, ok := states[string(id)]
	if !ok {
		return domain.BoardState{}, fmt.Errorf("game %s: %w", id, ErrNotFound)
	}
	return toDomainBoardState(stored), nil
}

func (r *GameRepository) UpdateBoardState(ctx context.Context, id domain.GameID, state domain.BoardState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if state.Size < 0 {
		return fmt.Errorf("board size negative: %w", ErrInvalidData)
	}

	files := r.storage.files
	files.mu.Lock()
	defer files.mu.Unlock()

	states, err := files.loadBoardStates()
	if err != nil {
		return err
	}
	if _, exists := states[string(id)]; !exists {
		return fmt.Errorf("game %s: %w", id, ErrNotFound)
	}
	games, err := files.loadGames()
	if err != nil {
		return err
	}
	game, exists := games[string(id)]
	if !exists {
		return fmt.Errorf("game %s: %w", id, ErrNotFound)
	}

	updatedStates := cloneStoredBoardStates(states)
	updatedStates[string(id)] = fromDomainBoardState(state)

	if err := files.persistBoardStates(updatedStates); err != nil {
		return err
	}

	updatedGames := cloneStoredGames(games)
	game.BoardSize = int(state.Size)
	updatedGames[string(id)] = game
	if err := files.persistGames(updatedGames); err != nil {
		_ = files.persistBoardStates(states)
		return err
	}
	return nil
}

func cloneStoredGames(src map[string]storedGame) map[string]storedGame {
	copy := make(map[string]storedGame, len(src))
	for k, v := range src {
		clone := v
		if v.End != nil {
			ts := *v.End
			clone.End = &ts
		}
		copy[k] = clone
	}
	return copy
}

func cloneStoredBoardStates(src map[string]storedBoardState) map[string]storedBoardState {
	copy := make(map[string]storedBoardState, len(src))
	for k, v := range src {
		pieces := make(map[string]storedBoardPiece, len(v.Pieces))
		for pk, pv := range v.Pieces {
			pieces[pk] = pv
		}
		copy[k] = storedBoardState{Size: v.Size, Pieces: pieces}
	}
	return copy
}

func toDomainGame(stored storedGame) domain.Game {
	game := domain.Game{
		ID:            domain.GameID(stored.ID),
		UserID:        domain.UserID(stored.UserID),
		Status:        domain.GameStatus(stored.Status),
		PlayerColor:   domain.PlayerColor(stored.PlayerColor),
		IsPlayerFirst: stored.IsPlayerFirst,
	}
	game.Start = domain.NewTimestamp(stored.Start)
	if stored.End != nil {
		ts := domain.NewTimestamp(*stored.End)
		game.End = &ts
	}
	return game
}

func fromDomainGame(game domain.Game) storedGame {
	stored := storedGame{
		ID:            string(game.ID),
		UserID:        string(game.UserID),
		Start:         game.Start.ToTime(),
		Status:        string(game.Status),
		PlayerColor:   string(game.PlayerColor),
		IsPlayerFirst: game.IsPlayerFirst,
	}
	if game.End != nil {
		end := game.End.ToTime()
		stored.End = &end
	}
	return stored
}

func toDomainBoardState(stored storedBoardState) domain.BoardState {
	state := domain.BoardState{
		Size:   domain.BoardSize(stored.Size),
		Pieces: make(map[domain.PieceID]domain.BoardPiece, len(stored.Pieces)),
	}
	for key, piece := range stored.Pieces {
		state.Pieces[domain.PieceID(key)] = domain.BoardPiece{
			ID:       domain.PieceID(key),
			Position: domain.Position{Row: domain.Coordinate(piece.Row), Col: domain.Coordinate(piece.Col)},
			Color:    domain.PlayerColor(piece.Color),
			Kind:     domain.PieceKind(piece.Kind),
		}
	}
	return state
}

func fromDomainBoardState(state domain.BoardState) storedBoardState {
	stored := storedBoardState{
		Size:   int(state.Size),
		Pieces: make(map[string]storedBoardPiece, len(state.Pieces)),
	}
	for key, piece := range state.Pieces {
		stored.Pieces[string(key)] = storedBoardPiece{
			Row:   int(piece.Position.Row),
			Col:   int(piece.Position.Col),
			Color: string(piece.Color),
			Kind:  string(piece.Kind),
		}
	}
	return stored
}
