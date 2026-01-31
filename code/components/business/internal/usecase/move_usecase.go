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
	ctxMove, err := uc.loadMoveContext(ctx, cmd.GameID)
	if err != nil {
		return domain.Move{}, err
	}
	if err := ensureTurn(ctxMove.game, len(ctxMove.moves), ctxMove.game.PlayerColor, "not player's turn"); err != nil {
		return domain.Move{}, err
	}
	piece, err := pieceAtColor(ctxMove.board, cmd.StartPosition, ctxMove.game.PlayerColor, "cannot move opponent piece")
	if err != nil {
		return domain.Move{}, err
	}
	move, updatedBoard, err := uc.buildApplyMove(ctxMove, piece, cmd.Trajectory, cmd.EndPosition, cmd.PerformedAt, ctxMove.game.PlayerColor)
	if err != nil {
		return domain.Move{}, err
	}
	if err := uc.persistMoveAndBoard(ctx, ctxMove.game.ID, move, updatedBoard); err != nil {
		return domain.Move{}, err
	}
	if err := uc.finalizeGameAfterMove(ctx, ctxMove.game, updatedBoard, ctxMove.game.PlayerColor); err != nil {
		return domain.Move{}, err
	}
	return move, nil
}

func (uc *moveUseCase) AddOpponentMove(ctx context.Context, cmd sdkusecase.AddOpponentMoveCommand) (domain.Move, error) {
	ctxMove, err := uc.loadMoveContext(ctx, cmd.GameID)
	if err != nil {
		return domain.Move{}, err
	}
	opponentColor := ctxMove.game.PlayerColor.Opponent()
	if err := ensureTurn(ctxMove.game, len(ctxMove.moves), opponentColor, "not opponent's turn"); err != nil {
		return domain.Move{}, err
	}
	piece, err := pieceAtColor(ctxMove.board, cmd.StartPosition, opponentColor, "cannot move player piece")
	if err != nil {
		return domain.Move{}, err
	}
	move, updatedBoard, err := uc.buildApplyMove(ctxMove, piece, cmd.Trajectory, cmd.EndPosition, cmd.PerformedAt, opponentColor)
	if err != nil {
		return domain.Move{}, err
	}
	if err := uc.persistMoveAndBoard(ctx, ctxMove.game.ID, move, updatedBoard); err != nil {
		return domain.Move{}, err
	}
	if err := uc.finalizeGameAfterMove(ctx, ctxMove.game, updatedBoard, opponentColor); err != nil {
		return domain.Move{}, err
	}
	return move, nil
}

func (uc *moveUseCase) GetOpponentMove(ctx context.Context, cmd sdkusecase.GetOpponentMoveCommand) (domain.Move, error) {
	ctxMove, err := uc.loadMoveContext(ctx, cmd.GameID)
	if err != nil {
		return domain.Move{}, err
	}
	opponentColor := ctxMove.game.PlayerColor.Opponent()
	if err := ensureTurn(ctxMove.game, len(ctxMove.moves), opponentColor, "not opponent's turn"); err != nil {
		return domain.Move{}, err
	}
	if uc.opponent == nil {
		return domain.Move{}, errors.New("opponent provider not configured")
	}

	move, updatedBoard, err := uc.tryOpponentSuggestions(ctx, ctxMove, opponentColor, cmd.Difficulty)
	if err == nil {
		if err := uc.finalizeGameAfterMove(ctx, ctxMove.game, updatedBoard, opponentColor); err != nil {
			return domain.Move{}, err
		}
		return move, nil
	}

	fallback, updatedBoard, err := uc.applyOpponentFallback(ctx, ctxMove, opponentColor)
	if err != nil {
		return domain.Move{}, err
	}
	if err := uc.finalizeGameAfterMove(ctx, ctxMove.game, updatedBoard, opponentColor); err != nil {
		return domain.Move{}, err
	}
	return fallback, nil
}

type moveContext struct {
	game  domain.Game
	board domain.BoardState
	moves []domain.Move
}

func (uc *moveUseCase) loadMoveContext(ctx context.Context, gameID domain.GameID) (moveContext, error) {
	if uc.ids == nil {
		return moveContext{}, errors.New("id generator not configured")
	}
	if gameID == "" {
		return moveContext{}, errors.New("game id is required")
	}
	game, err := uc.gameRepo.GetByID(ctx, gameID)
	if err != nil {
		return moveContext{}, err
	}
	board, err := uc.gameRepo.BoardState(ctx, gameID)
	if err != nil {
		return moveContext{}, err
	}
	moves, err := uc.moveRepo.ListByGame(ctx, gameID)
	if err != nil {
		return moveContext{}, err
	}
	return moveContext{game: game, board: board, moves: moves}, nil
}

func ensureTurn(game domain.Game, movesCount int, expected domain.PlayerColor, msg string) error {
	current := determineTurnColor(game, movesCount)
	if current != expected {
		return errors.New(msg)
	}
	return nil
}

func pieceAtColor(board domain.BoardState, start domain.Position, expected domain.PlayerColor, wrongColorMsg string) (domain.BoardPiece, error) {
	piece, ok := board.PieceAt(start)
	if !ok {
		return domain.BoardPiece{}, errors.New("no piece at specified start position")
	}
	if piece.Color != expected {
		return domain.BoardPiece{}, errors.New(wrongColorMsg)
	}
	return piece, nil
}

func (uc *moveUseCase) buildApplyMove(ctxMove moveContext, piece domain.BoardPiece, path []domain.Position, end *domain.Position, performedAt *domain.Timestamp, mover domain.PlayerColor) (domain.Move, domain.BoardState, error) {
	trajectory, err := buildTrajectory(uc.rules, ctxMove.board, piece, path, end, true)
	if err != nil {
		return domain.Move{}, domain.BoardState{}, err
	}
	number := domain.MoveNumber(len(ctxMove.moves) + 1)
	createdAt := timestampOrNow(performedAt)
	moveID := domain.MoveID(uc.ids.NewID())
	move, err := domain.NewMove(moveID, ctxMove.game.ID, piece.ID, number, piece.Position, trajectory, createdAt)
	if err != nil {
		return domain.Move{}, domain.BoardState{}, err
	}
	updatedBoard, _, err := uc.rules.ValidateAndApplyMove(ctxMove.board, piece, trajectory, mover)
	if err != nil {
		return domain.Move{}, domain.BoardState{}, err
	}
	return move, updatedBoard, nil
}

func (uc *moveUseCase) persistMoveAndBoard(ctx context.Context, gameID domain.GameID, move domain.Move, board domain.BoardState) error {
	if err := uc.moveRepo.Add(ctx, move); err != nil {
		return err
	}
	return uc.gameRepo.UpdateBoardState(ctx, gameID, board)
}

func (uc *moveUseCase) finalizeGameAfterMove(ctx context.Context, game domain.Game, board domain.BoardState, mover domain.PlayerColor) error {
	updatedGame, err := startGameIfNeeded(ctx, game, uc.gameRepo)
	if err != nil {
		return err
	}
	if isVictory(board, mover.Opponent()) {
		return finishGame(ctx, uc.gameRepo, updatedGame)
	}
	return nil
}

func (uc *moveUseCase) tryOpponentSuggestions(ctx context.Context, ctxMove moveContext, opponentColor domain.PlayerColor, difficulty domain.DifficultyLevel) (domain.Move, domain.BoardState, error) {
	feedback := make([]string, 0)
	const maxAttempts = 3

	for attempt := 0; attempt < maxAttempts; attempt++ {
		suggestion, err := uc.opponent.SuggestMove(ctx, sdkport.OpponentMoveRequest{
			Game:       ctxMove.game,
			Board:      ctxMove.board,
			Difficulty: difficulty,
			Feedback:   append([]string(nil), feedback...),
		})
		if err != nil {
			return domain.Move{}, domain.BoardState{}, err
		}

		move, updatedBoard, ok, msg, err := uc.applyOpponentSuggestion(ctx, ctxMove, opponentColor, suggestion)
		if err != nil {
			return domain.Move{}, domain.BoardState{}, err
		}
		if ok {
			return move, updatedBoard, nil
		}
		feedback = append(feedback, msg)
	}

	return domain.Move{}, domain.BoardState{}, fmt.Errorf("opponent move invalid after %d attempts", maxAttempts)
}

func (uc *moveUseCase) applyOpponentSuggestion(ctx context.Context, ctxMove moveContext, opponentColor domain.PlayerColor, suggestion domain.Move) (domain.Move, domain.BoardState, bool, string, error) {
	if suggestion.GameID == "" {
		suggestion.GameID = ctxMove.game.ID
	}

	piece, err := resolveSuggestedPiece(ctxMove.board, suggestion)
	if err != nil {
		return domain.Move{}, domain.BoardState{}, false, "", err
	}
	if piece.Color != opponentColor {
		return domain.Move{}, domain.BoardState{}, false, "", errors.New("opponent attempted to move wrong color")
	}
	log.Printf("opponent suggestion: game=%s piece=%s start=%v path=%v end=%v", ctxMove.game.ID, piece.ID, suggestion.StartPosition, suggestion.Trajectory, suggestion.EndPosition)

	trajectory, err := buildTrajectory(uc.rules, ctxMove.board, piece, suggestion.Trajectory, &suggestion.EndPosition, false)
	if err != nil {
		msg := fmt.Sprintf("invalid trajectory: piece=%s start=%v path=%v end=%v err=%v", piece.ID, suggestion.StartPosition, suggestion.Trajectory, suggestion.EndPosition, err)
		log.Printf("opponent trajectory invalid: %s", msg)
		return domain.Move{}, domain.BoardState{}, false, msg, nil
	}

	move, updatedBoard, msg, ok, err := uc.createApplyPersistOpponentMove(ctx, ctxMove, piece, trajectory, opponentColor)
	if err != nil {
		return domain.Move{}, domain.BoardState{}, false, "", err
	}
	if ok {
		return move, updatedBoard, true, "", nil
	}
	log.Printf("opponent move apply failed: %s", msg)
	return domain.Move{}, domain.BoardState{}, false, msg, nil
}

func resolveSuggestedPiece(board domain.BoardState, suggestion domain.Move) (domain.BoardPiece, error) {
	if piece, ok := board.Piece(suggestion.PieceID); ok {
		return piece, nil
	}
	if pieceAt, ok := board.PieceAt(suggestion.StartPosition); ok {
		return pieceAt, nil
	}
	return domain.BoardPiece{}, errors.New("opponent move references unknown piece")
}

func (uc *moveUseCase) createApplyPersistOpponentMove(ctx context.Context, ctxMove moveContext, piece domain.BoardPiece, trajectory []domain.Position, opponentColor domain.PlayerColor) (domain.Move, domain.BoardState, string, bool, error) {
	moveNumber := domain.MoveNumber(len(ctxMove.moves) + 1)
	moveID := domain.MoveID(uc.ids.NewID())
	createdAt := domain.NewTimestamp(time.Now().UTC())
	move, err := domain.NewMove(moveID, ctxMove.game.ID, piece.ID, moveNumber, piece.Position, trajectory, createdAt)
	if err != nil {
		return domain.Move{}, domain.BoardState{}, "", false, err
	}
	updatedBoard, captures, err := uc.rules.ValidateAndApplyMove(ctxMove.board, piece, trajectory, opponentColor)
	if err != nil {
		return domain.Move{}, domain.BoardState{}, formatOpponentApplyError(err, piece.ID, trajectory), false, nil
	}
	if len(captures) > 0 {
		log.Printf("opponent captured pieces: game=%s captures=%v", ctxMove.game.ID, captures)
	}
	if err := uc.persistMoveAndBoard(ctx, ctxMove.game.ID, move, updatedBoard); err != nil {
		return domain.Move{}, domain.BoardState{}, "", false, err
	}
	return move, updatedBoard, "", true, nil
}

func formatOpponentApplyError(err error, pieceID domain.PieceID, trajectory []domain.Position) string {
	finalSquare := domain.Position{}
	if len(trajectory) > 0 {
		finalSquare = trajectory[len(trajectory)-1]
	}
	if errors.Is(err, ErrTargetOccupied) {
		return fmt.Sprintf("destination %v occupied; select an empty square", finalSquare)
	}
	if errors.Is(err, domain.ErrMoveMustCapture) {
		return "a capture is available; you must return a capturing move"
	}
	if errors.Is(err, domain.ErrMoveMustContinue) {
		return "capture must continue with the same piece until no further captures are available"
	}
	return fmt.Sprintf("apply failed: piece=%s trajectory=%v err=%v", pieceID, trajectory, err)
}

func (uc *moveUseCase) applyOpponentFallback(ctx context.Context, ctxMove moveContext, opponentColor domain.PlayerColor) (domain.Move, domain.BoardState, error) {
	piece, path, ok := findFallbackMove(uc.rules, ctxMove.board, opponentColor)
	if !ok {
		return domain.Move{}, domain.BoardState{}, errors.New("no fallback move available")
	}
	moveNumber := domain.MoveNumber(len(ctxMove.moves) + 1)
	moveID := domain.MoveID(uc.ids.NewID())
	createdAt := domain.NewTimestamp(time.Now().UTC())
	move, err := domain.NewMove(moveID, ctxMove.game.ID, piece.ID, moveNumber, piece.Position, path, createdAt)
	if err != nil {
		return domain.Move{}, domain.BoardState{}, err
	}
	updatedBoard, captures, err := uc.rules.ValidateAndApplyMove(ctxMove.board, piece, path, opponentColor)
	if err != nil {
		return domain.Move{}, domain.BoardState{}, err
	}
	if len(captures) > 0 {
		log.Printf("opponent captured pieces (fallback): game=%s captures=%v", ctxMove.game.ID, captures)
	}
	if err := uc.persistMoveAndBoard(ctx, ctxMove.game.ID, move, updatedBoard); err != nil {
		return domain.Move{}, domain.BoardState{}, err
	}
	log.Printf("opponent move used fallback")
	return move, updatedBoard, nil
}
