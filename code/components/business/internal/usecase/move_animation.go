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
	piece, ok := board.Piece(move.PieceID)
	if !ok {
		if pieceAt, ok := board.PieceAt(move.StartPosition); ok {
			piece = pieceAt
		} else {
			return domain.Animation{}, ErrAnimationMoveMismatch
		}
	}
	current := piece.Position
	if current != move.StartPosition {
		return domain.Animation{}, ErrAnimationMoveMismatch
	}

	steps := make([]domain.AnimationStep, 0, len(move.Trajectory)*3)
	working := board.Clone()

	for _, target := range move.Trajectory {
		if !target.IsInside(working.Size) {
			return domain.Animation{}, fmt.Errorf("target outside board: %v", target)
		}
		deltaRow := int(target.Row - current.Row)
		deltaCol := int(target.Col - current.Col)
		absRow := abs(deltaRow)
		absCol := abs(deltaCol)

		if absRow == 2 && absCol == 2 {
			middle := domain.Position{
				Row: current.Row + domain.Coordinate(deltaRow/2),
				Col: current.Col + domain.Coordinate(deltaCol/2),
			}
			victim, ok := working.PieceAt(middle)
			if !ok || victim.Color == piece.Color {
				return domain.Animation{}, ErrAnimationMoveMismatch
			}

			exitPath := domain.ComputeCaptureExitPath(middle, working.Size, victim.Color)
			log.Printf("capture exit path piece=%s color=%s from=%v path=%v", victim.ID, victim.Color, middle, exitPath)
			previous := middle
			for _, next := range exitPath {
				steps = append(steps, domain.AnimationStep{
					PieceID: victim.ID,
					From:    previous,
					To:      next,
				})
				previous = next
			}
			working = working.RemovePiece(victim.ID)

			steps = append(steps, domain.AnimationStep{
				PieceID: piece.ID,
				From:    current,
				To:      middle,
			})
			current = middle

			steps = append(steps, domain.AnimationStep{
				PieceID: piece.ID,
				From:    current,
				To:      target,
			})
			working = working.MovePiece(piece.ID, target)
			current = target
			continue
		}

		steps = append(steps, domain.AnimationStep{
			PieceID: piece.ID,
			From:    current,
			To:      target,
		})
		working = working.MovePiece(piece.ID, target)
		current = target
	}

	return domain.Animation{Steps: steps}, nil
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
