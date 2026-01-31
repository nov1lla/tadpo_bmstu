package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"

	"ppo/sdk/domain"
	sdkusecase "ppo/sdk/usecase"
)

var ErrAnimationMoveMismatch = errors.New("move does not match board state")

type moveAnimationUseCase struct{}

func NewMoveAnimationUseCase() sdkusecase.MoveAnimationUseCase {
	return &moveAnimationUseCase{}
}

func (uc *moveAnimationUseCase) Build(ctx context.Context, cmd sdkusecase.MoveAnimationCommand) (domain.Animation, error) {
	if err := ctx.Err(); err != nil {
		return domain.Animation{}, err
	}
	board := cmd.Board
	move := cmd.Move

	piece, err := resolveMovePiece(board, move)
	if err != nil {
		return domain.Animation{}, err
	}
	if err := ensureMoveStart(piece, move); err != nil {
		return domain.Animation{}, err
	}

	steps, err := buildMoveAnimationSteps(board, piece, move)
	if err != nil {
		return domain.Animation{}, err
	}
	return domain.Animation{Steps: steps}, nil
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func resolveMovePiece(board domain.BoardState, move domain.Move) (domain.BoardPiece, error) {
	if piece, ok := board.Piece(move.PieceID); ok {
		return piece, nil
	}
	if pieceAt, ok := board.PieceAt(move.StartPosition); ok {
		return pieceAt, nil
	}
	return domain.BoardPiece{}, ErrAnimationMoveMismatch
}

func ensureMoveStart(piece domain.BoardPiece, move domain.Move) error {
	if piece.Position != move.StartPosition {
		return ErrAnimationMoveMismatch
	}
	return nil
}

func buildMoveAnimationSteps(board domain.BoardState, piece domain.BoardPiece, move domain.Move) ([]domain.AnimationStep, error) {
	steps := make([]domain.AnimationStep, 0, len(move.Trajectory)*3)
	working := board.Clone()
	current := piece.Position

	for _, target := range move.Trajectory {
		if !target.IsInside(working.Size) {
			return nil, fmt.Errorf("target outside board: %v", target)
		}

		isCapture, middle := captureMiddle(current, target)
		if !isCapture {
			steps = append(steps, domain.AnimationStep{PieceID: piece.ID, From: current, To: target})
			working = working.MovePiece(piece.ID, target)
			current = target
			continue
		}

		var err error
		working, current, err = applyCaptureAnimation(&steps, working, piece, current, middle, target)
		if err != nil {
			return nil, err
		}
	}
	return steps, nil
}

func captureMiddle(from, to domain.Position) (bool, domain.Position) {
	deltaRow := int(to.Row - from.Row)
	deltaCol := int(to.Col - from.Col)
	if abs(deltaRow) != 2 || abs(deltaCol) != 2 {
		return false, domain.Position{}
	}
	return true, domain.Position{
		Row: from.Row + domain.Coordinate(deltaRow/2),
		Col: from.Col + domain.Coordinate(deltaCol/2),
	}
}

func applyCaptureAnimation(steps *[]domain.AnimationStep, board domain.BoardState, piece domain.BoardPiece, current, middle, target domain.Position) (domain.BoardState, domain.Position, error) {
	victim, ok := board.PieceAt(middle)
	if !ok || victim.Color == piece.Color {
		return board, current, ErrAnimationMoveMismatch
	}

	exitPath := domain.ComputeCaptureExitPath(middle, board.Size, victim.Color)
	log.Printf("capture exit path piece=%s color=%s from=%v path=%v", victim.ID, victim.Color, middle, exitPath)

	prev := middle
	for _, next := range exitPath {
		*steps = append(*steps, domain.AnimationStep{PieceID: victim.ID, From: prev, To: next})
		prev = next
	}
	board = board.RemovePiece(victim.ID)

	*steps = append(*steps, domain.AnimationStep{PieceID: piece.ID, From: current, To: middle})
	current = middle

	*steps = append(*steps, domain.AnimationStep{PieceID: piece.ID, From: current, To: target})
	board = board.MovePiece(piece.ID, target)
	return board, target, nil
}
