package usecase

import (
	"context"
	"errors"
	"time"

	"ppo/sdk/domain"
	"ppo/sdk/port/repo"
)

func startGameIfNeeded(ctx context.Context, game domain.Game, gameRepo repo.GameRepository) (domain.Game, error) {
	if game.Status != domain.GameStatusPending {
		return game, nil
	}
	started, err := game.StartPlay()
	if err != nil {
		return domain.Game{}, err
	}
	if err := gameRepo.Update(ctx, started); err != nil {
		return domain.Game{}, err
	}
	return started, nil
}

func timestampOrNow(ts *domain.Timestamp) domain.Timestamp {
	if ts != nil {
		return *ts
	}
	return domain.NewTimestamp(time.Now().UTC())
}

func determineTurnColor(game domain.Game, movesCount int) domain.PlayerColor {
	playerColor := game.PlayerColor
	if game.IsPlayerFirst {
		if movesCount%2 == 0 {
			return playerColor
		}
		return playerColor.Opponent()
	}
	if movesCount%2 == 0 {
		return playerColor.Opponent()
	}
	return playerColor
}

func isVictory(board domain.BoardState, color domain.PlayerColor) bool {
	for _, piece := range board.Pieces {
		if piece.Color == color {
			return false
		}
	}
	return true
}

func finishGame(ctx context.Context, gameRepo repo.GameRepository, game domain.Game) error {
	end := domain.NewTimestamp(time.Now().UTC())
	finished, err := game.Finish(end)
	if err != nil {
		return err
	}
	return gameRepo.Update(ctx, finished)
}

func buildTrajectory(rules domain.CheckersRules, board domain.BoardState, piece domain.BoardPiece, provided []domain.Position, end *domain.Position, strict bool) ([]domain.Position, error) {
	if len(provided) > 0 {
		trajectory := append([]domain.Position(nil), provided...)
		if _, _, err := rules.ValidateAndApplyMove(board, piece, trajectory, piece.Color); err != nil {
			if !strict && end != nil {
				if fallback, ferr := resolveTrajectory(board, piece, *end, piece.Color); ferr == nil {
					return fallback, nil
				}
			}
			return nil, err
		}
		return trajectory, nil
	}
	if end == nil {
		return nil, errors.New("trajectory or end position required")
	}
	trajectory, err := resolveTrajectory(board, piece, *end, piece.Color)
	if err != nil {
		return nil, err
	}
	if _, _, err := rules.ValidateAndApplyMove(board, piece, trajectory, piece.Color); err != nil {
		return nil, err
	}
	return trajectory, nil
}

func resolveTrajectory(board domain.BoardState, piece domain.BoardPiece, target domain.Position, mover domain.PlayerColor) ([]domain.Position, error) {
	if !target.IsInside(board.Size) {
		return nil, errors.New("target outside board")
	}
	if _, occupied := board.PieceAt(target); occupied {
		return nil, errors.New("target occupied")
	}
	rules := domain.CheckersRules{}
	simple := []domain.Position{target}
	if _, _, err := rules.ValidateAndApplyMove(board, piece, simple, mover); err == nil {
		return simple, nil
	}
	if path, ok := searchCapturePath(board, piece, target); ok {
		return path, nil
	}
	return nil, errors.New("unable to build trajectory")
}

func searchCapturePath(board domain.BoardState, piece domain.BoardPiece, target domain.Position) ([]domain.Position, bool) {
	return captureDFS(board, piece, target, nil)
}

func captureDFS(board domain.BoardState, piece domain.BoardPiece, target domain.Position, path []domain.Position) ([]domain.Position, bool) {
	directions := []struct{ dr, dc int }{{2, 2}, {2, -2}, {-2, 2}, {-2, -2}}
	for _, dir := range directions {
		if piece.Kind == domain.PieceKindMan {
			if dir.dr/2 != piece.Color.ForwardDirection() {
				continue
			}
		}
		landing := domain.Position{Row: piece.Position.Row + domain.Coordinate(dir.dr), Col: piece.Position.Col + domain.Coordinate(dir.dc)}
		if !landing.IsInside(board.Size) {
			continue
		}
		if _, occupied := board.PieceAt(landing); occupied {
			continue
		}
		middle := domain.Position{Row: piece.Position.Row + domain.Coordinate(dir.dr/2), Col: piece.Position.Col + domain.Coordinate(dir.dc/2)}
		victim, ok := board.PieceAt(middle)
		if !ok || victim.Color == piece.Color {
			continue
		}
		newBoard := board.RemovePiece(piece.ID)
		newBoard = newBoard.RemovePiece(victim.ID)
		newPiece := piece
		newPiece.Position = landing
		newBoard = newBoard.WithPiece(newPiece)
		newPath := append(path, landing)
		if landing == target {
			return newPath, true
		}
		if result, ok := captureDFS(newBoard, newPiece, target, newPath); ok {
			return result, true
		}
	}
	return nil, false
}

func findFallbackMove(rules domain.CheckersRules, board domain.BoardState, mover domain.PlayerColor) (domain.BoardPiece, []domain.Position, bool) {
	if piece, path, ok := findCaptureMove(board, mover); ok {
		return piece, path, true
	}
	if piece, path, ok := findSimpleMove(rules, board, mover); ok {
		return piece, path, true
	}
	return domain.BoardPiece{}, nil, false
}

func findCaptureMove(board domain.BoardState, mover domain.PlayerColor) (domain.BoardPiece, []domain.Position, bool) {
	for _, piece := range board.Pieces {
		if piece.Color != mover {
			continue
		}
		if path, ok := findCaptureSequence(board, piece); ok {
			return piece, path, true
		}
	}
	return domain.BoardPiece{}, nil, false
}

func findCaptureSequence(board domain.BoardState, piece domain.BoardPiece) ([]domain.Position, bool) {
	var result []domain.Position
	var dfs func(domain.BoardState, domain.BoardPiece, []domain.Position) bool
	dfs = func(state domain.BoardState, current domain.BoardPiece, path []domain.Position) bool {
		found := false
		directions := []struct{ dr, dc int }{{2, 2}, {2, -2}, {-2, 2}, {-2, -2}}
		for _, dir := range directions {
			landing, victim, ok := captureCandidate(state, current, dir.dr, dir.dc)
			if !ok {
				continue
			}
			newBoard, newPiece := applyCapture(state, current, victim, landing)
			newPath := append(path, landing)
			if dfs(newBoard, newPiece, newPath) {
				return true
			}
			found = true
			if result == nil {
				result = newPath
			}
		}
		if !found && len(path) > 0 {
			result = path
			return true
		}
		return false
	}
	if dfs(board, piece, nil) && len(result) > 0 {
		return result, true
	}
	return nil, false
}

func captureCandidate(state domain.BoardState, current domain.BoardPiece, dr, dc int) (domain.Position, domain.BoardPiece, bool) {
	if current.Kind == domain.PieceKindMan && dr/2 != current.Color.ForwardDirection() {
		return domain.Position{}, domain.BoardPiece{}, false
	}
	landing := domain.Position{Row: current.Position.Row + domain.Coordinate(dr), Col: current.Position.Col + domain.Coordinate(dc)}
	if !landing.IsInside(state.Size) {
		return domain.Position{}, domain.BoardPiece{}, false
	}
	if _, occupied := state.PieceAt(landing); occupied {
		return domain.Position{}, domain.BoardPiece{}, false
	}
	middle := domain.Position{Row: current.Position.Row + domain.Coordinate(dr/2), Col: current.Position.Col + domain.Coordinate(dc/2)}
	victim, ok := state.PieceAt(middle)
	if !ok || victim.Color == current.Color {
		return domain.Position{}, domain.BoardPiece{}, false
	}
	return landing, victim, true
}

func applyCapture(state domain.BoardState, current domain.BoardPiece, victim domain.BoardPiece, landing domain.Position) (domain.BoardState, domain.BoardPiece) {
	newBoard := state.RemovePiece(current.ID).RemovePiece(victim.ID)
	newPiece := current
	newPiece.Position = landing
	newBoard = newBoard.WithPiece(newPiece)
	return newBoard, newPiece
}

func findSimpleMove(rules domain.CheckersRules, board domain.BoardState, mover domain.PlayerColor) (domain.BoardPiece, []domain.Position, bool) {
	for _, piece := range board.Pieces {
		if piece.Color != mover {
			continue
		}
		directions := []struct{ dr, dc int }{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
		for _, dir := range directions {
			if piece.Kind == domain.PieceKindMan && dir.dr != piece.Color.ForwardDirection() {
				continue
			}
			target := domain.Position{Row: piece.Position.Row + domain.Coordinate(dir.dr), Col: piece.Position.Col + domain.Coordinate(dir.dc)}
			if !target.IsInside(board.Size) {
				continue
			}
			if _, occupied := board.PieceAt(target); occupied {
				continue
			}
			path := []domain.Position{target}
			if _, _, err := rules.ValidateAndApplyMove(board, piece, path, mover); err == nil {
				return piece, path, true
			}
		}
	}
	return domain.BoardPiece{}, nil, false
}
