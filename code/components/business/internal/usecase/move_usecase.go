package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"ppo/sdk/domain"
	sdkport "ppo/sdk/port"
	sdkrepo "ppo/sdk/port/repo"
	sdkusecase "ppo/sdk/usecase"
)

type moveUseCase struct {
	moveRepo sdkrepo.MoveRepository
	gameRepo sdkrepo.GameRepository
	opponent sdkport.OpponentMoveProvider
	ids      sdkport.IDGenerator
	rules    domain.CheckersRules
}

func NewMoveUseCase(moveRepo sdkrepo.MoveRepository, gameRepo sdkrepo.GameRepository, opponent sdkport.OpponentMoveProvider, ids sdkport.IDGenerator) sdkusecase.MoveUseCase {
	return &moveUseCase{moveRepo: moveRepo, gameRepo: gameRepo, opponent: opponent, ids: ids, rules: domain.CheckersRules{}}
}

func (uc *moveUseCase) AddUserMove(ctx context.Context, cmd sdkusecase.AddUserMoveCommand) (domain.Move, error) {
	if uc.ids == nil {
		return domain.Move{}, errors.New("id generator not configured")
	}
	if cmd.GameID == "" {
		return domain.Move{}, errors.New("game id is required")
	}
	game, err := uc.gameRepo.GetByID(ctx, cmd.GameID)
	if err != nil {
		return domain.Move{}, err
	}

	board, err := uc.gameRepo.BoardState(ctx, cmd.GameID)
	if err != nil {
		return domain.Move{}, err
	}
	allMoves, err := uc.moveRepo.ListByGame(ctx, cmd.GameID)
	if err != nil {
		return domain.Move{}, err
	}
	currentTurn := determineTurnColor(game, len(allMoves))
	if currentTurn != game.PlayerColor {
		return domain.Move{}, errors.New("not player's turn")
	}
	piece, ok := board.PieceAt(cmd.StartPosition)
	if !ok {
		return domain.Move{}, errors.New("no piece at specified start position")
	}
	if piece.Color != game.PlayerColor {
		return domain.Move{}, errors.New("cannot move opponent piece")
	}
	trajectory, err := buildTrajectory(uc.rules, board, piece, cmd.Trajectory, cmd.EndPosition, true)
	if err != nil {
		return domain.Move{}, err
	}
	number := domain.MoveNumber(len(allMoves) + 1)
	createdAt := timestampOrNow(cmd.PerformedAt)
	moveID := domain.MoveID(uc.ids.NewID())
	move, err := domain.NewMove(moveID, cmd.GameID, piece.ID, number, piece.Position, trajectory, createdAt)
	if err != nil {
		return domain.Move{}, err
	}
	updatedBoard, _, err := uc.rules.ValidateAndApplyMove(board, piece, trajectory, game.PlayerColor)
	if err != nil {
		return domain.Move{}, err
	}

	if err := uc.moveRepo.Add(ctx, move); err != nil {
		return domain.Move{}, err
	}
	if err := uc.gameRepo.UpdateBoardState(ctx, game.ID, updatedBoard); err != nil {
		return domain.Move{}, err
	}

	game, err = startGameIfNeeded(ctx, game, uc.gameRepo)
	if err != nil {
		return domain.Move{}, err
	}
	if isVictory(updatedBoard, game.PlayerColor.Opponent()) {
		if err := finishGame(ctx, uc.gameRepo, game); err != nil {
			return domain.Move{}, err
		}
	}

	return move, nil
}

func (uc *moveUseCase) GetOpponentMove(ctx context.Context, cmd sdkusecase.GetOpponentMoveCommand) (domain.Move, error) {
	if uc.ids == nil {
		return domain.Move{}, errors.New("id generator not configured")
	}
	game, err := uc.gameRepo.GetByID(ctx, cmd.GameID)
	if err != nil {
		return domain.Move{}, err
	}

	board, err := uc.gameRepo.BoardState(ctx, cmd.GameID)
	if err != nil {
		return domain.Move{}, err
	}
	moves, err := uc.moveRepo.ListByGame(ctx, cmd.GameID)
	if err != nil {
		return domain.Move{}, err
	}
	currentTurn := determineTurnColor(game, len(moves))
	opponentColor := game.PlayerColor.Opponent()
	if currentTurn != opponentColor {
		return domain.Move{}, errors.New("not opponent's turn")
	}

	feedback := make([]string, 0)
	var validatedMove domain.Move
	var updatedBoard domain.BoardState
	var captures []domain.PieceID
	const maxAttempts = 3

	for attempt := 0; attempt < maxAttempts; attempt++ {
		suggestion, err := uc.opponent.SuggestMove(ctx, sdkport.OpponentMoveRequest{
			Game:       game,
			Board:      board,
			Difficulty: cmd.Difficulty,
			Feedback:   append([]string(nil), feedback...),
		})
		if err != nil {
			return domain.Move{}, err
		}
		if suggestion.GameID == "" {
			suggestion.GameID = cmd.GameID
		}
		piece, ok := board.Piece(suggestion.PieceID)
		if !ok {
			pieceAt, ok := board.PieceAt(suggestion.StartPosition)
			if !ok {
				return domain.Move{}, errors.New("opponent move references unknown piece")
			}
			piece = pieceAt
			suggestion.PieceID = piece.ID
		}
		if piece.Color != opponentColor {
			return domain.Move{}, errors.New("opponent attempted to move wrong color")
		}
		log.Printf("opponent suggestion: game=%s piece=%s start=%v path=%v end=%v", cmd.GameID, suggestion.PieceID, suggestion.StartPosition, suggestion.Trajectory, suggestion.EndPosition)

		trajectory, err := buildTrajectory(uc.rules, board, piece, suggestion.Trajectory, &suggestion.EndPosition, false)
		if err != nil {
			msg := fmt.Sprintf("invalid trajectory: piece=%s start=%v path=%v end=%v err=%v", piece.ID, suggestion.StartPosition, suggestion.Trajectory, suggestion.EndPosition, err)
			feedback = append(feedback, msg)
			log.Printf("opponent trajectory invalid: %s", msg)
			continue
		}
		moveNumber := domain.MoveNumber(len(moves) + 1)
		moveID := domain.MoveID(uc.ids.NewID())
		createdAt := domain.NewTimestamp(time.Now().UTC())
		validatedMove, err = domain.NewMove(moveID, cmd.GameID, piece.ID, moveNumber, piece.Position, trajectory, createdAt)
		if err != nil {
			return domain.Move{}, err
		}
		updatedBoard, captures, err = uc.rules.ValidateAndApplyMove(board, piece, trajectory, opponentColor)
		if err != nil {
			finalSquare := domain.Position{}
			if len(trajectory) > 0 {
				finalSquare = trajectory[len(trajectory)-1]
			}
			var msg string
			if errors.Is(err, ErrTargetOccupied) {
				msg = fmt.Sprintf("destination %v occupied; select an empty square", finalSquare)
			} else if errors.Is(err, domain.ErrMoveMustCapture) {
				msg = "a capture is available; you must return a capturing move"
			} else if errors.Is(err, domain.ErrMoveMustContinue) {
				msg = "capture must continue with the same piece until no further captures are available"
			} else {
				msg = fmt.Sprintf("apply failed: piece=%s trajectory=%v err=%v", piece.ID, trajectory, err)
			}
			feedback = append(feedback, msg)
			log.Printf("opponent move apply failed: %s", msg)
			continue
		}
		if len(captures) > 0 {
			log.Printf("opponent captured pieces: game=%s captures=%v", cmd.GameID, captures)
		}
		if err := uc.moveRepo.Add(ctx, validatedMove); err != nil {
			return domain.Move{}, err
		}
		if err := uc.gameRepo.UpdateBoardState(ctx, game.ID, updatedBoard); err != nil {
			return domain.Move{}, err
		}
		game, err = startGameIfNeeded(ctx, game, uc.gameRepo)
		if err != nil {
			return domain.Move{}, err
		}
		if isVictory(updatedBoard, game.PlayerColor) {
			if err := finishGame(ctx, uc.gameRepo, game); err != nil {
				return domain.Move{}, err
			}
		}
		return validatedMove, nil
	}

	if piece, path, ok := findFallbackMove(uc.rules, board, opponentColor); ok {
		moveNumber := domain.MoveNumber(len(moves) + 1)
		moveID := domain.MoveID(uc.ids.NewID())
		createdAt := domain.NewTimestamp(time.Now().UTC())
		validatedMove, err = domain.NewMove(moveID, cmd.GameID, piece.ID, moveNumber, piece.Position, path, createdAt)
		if err != nil {
			return domain.Move{}, err
		}
		updatedBoard, captures, err = uc.rules.ValidateAndApplyMove(board, piece, path, opponentColor)
		if err != nil {
			return domain.Move{}, err
		}
		if len(captures) > 0 {
			log.Printf("opponent captured pieces (fallback): game=%s captures=%v", cmd.GameID, captures)
		}
		if err := uc.moveRepo.Add(ctx, validatedMove); err != nil {
			return domain.Move{}, err
		}
		if err := uc.gameRepo.UpdateBoardState(ctx, game.ID, updatedBoard); err != nil {
			return domain.Move{}, err
		}
		game, err = startGameIfNeeded(ctx, game, uc.gameRepo)
		if err != nil {
			return domain.Move{}, err
		}
		if isVictory(updatedBoard, game.PlayerColor) {
			if err := finishGame(ctx, uc.gameRepo, game); err != nil {
				return domain.Move{}, err
			}
		}
		log.Printf("opponent move used fallback after %d invalid attempts", maxAttempts)
		return validatedMove, nil
	}

	return domain.Move{}, fmt.Errorf("opponent move invalid after %d attempts", maxAttempts)
}
