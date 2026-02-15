package domain

import "errors"

var (
	ErrMoveWrongPieceColor = errors.New("piece belongs to other player")
	ErrMovePathEmpty       = errors.New("move path is empty")
	ErrMoveIllegal         = errors.New("illegal move for checkers rules")
	ErrMoveTargetOccupied  = errors.New("target square occupied")
	ErrMoveMustCapture     = errors.New("capture is mandatory when available")
	ErrMoveMustContinue    = errors.New("must continue capturing while possible")
)

type CheckersRules struct{}

func (CheckersRules) ValidateAndApplyMove(board BoardState, piece BoardPiece, pathPositions []Position, mover PlayerColor) (BoardState, []PieceID, error) {
	if len(pathPositions) == 0 {
		return BoardState{}, nil, ErrMovePathEmpty
	}
	if piece.Color != mover {
		return BoardState{}, nil, ErrMoveWrongPieceColor
	}

	mustCapture := hasAnyCapture(board, mover)

	updated := board.Clone().RemovePiece(piece.ID)
	captures := make([]PieceID, 0)
	current := piece.Position
	capturePerformed := false

	for _, target := range pathPositions {
		if !target.IsInside(board.Size) {
			return BoardState{}, nil, ErrMoveIllegal
		}
		if _, occupied := updated.PieceAt(target); occupied {
			return BoardState{}, nil, ErrMoveTargetOccupied
		}

		deltaRow := int(target.Row - current.Row)
		deltaCol := int(target.Col - current.Col)
		absRow, absCol := abs(deltaRow), abs(deltaCol)

		if absRow == 1 && absCol == 1 {
			if mustCapture {
				return BoardState{}, nil, ErrMoveMustCapture
			}
			if capturePerformed {
				return BoardState{}, nil, ErrMoveIllegal
			}
			if piece.Kind == PieceKindMan {
				if deltaRow != mover.ForwardDirection() {
					return BoardState{}, nil, ErrMoveIllegal
				}
			}
			current = target
			continue
		}

		if absRow == 2 && absCol == 2 {
			middle := Position{
				Row: current.Row + Coordinate(deltaRow/2),
				Col: current.Col + Coordinate(deltaCol/2),
			}
			victim, ok := board.PieceAt(middle)
			if !ok || victim.Color == mover {
				return BoardState{}, nil, ErrMoveIllegal
			}
			if piece.Kind == PieceKindMan {
				if deltaRow/2 != mover.ForwardDirection() {
					return BoardState{}, nil, ErrMoveIllegal
				}
			}
			captures = append(captures, victim.ID)
			board = board.RemovePiece(victim.ID)
			updated = updated.RemovePiece(victim.ID)
			current = target
			capturePerformed = true
			continue
		}

		return BoardState{}, nil, ErrMoveIllegal
	}

	if mustCapture && !capturePerformed {
		return BoardState{}, nil, ErrMoveMustCapture
	}

	if capturePerformed {
		pieceAfter := piece
		pieceAfter.Position = current
		if canCaptureFrom(updated.WithPiece(pieceAfter), pieceAfter) {
			return BoardState{}, nil, ErrMoveMustContinue
		}
	}

	finalPiece := piece
	finalPiece.Position = current
	if piece.Kind == PieceKindMan && shouldPromote(piece.Color, current, board.Size) {
		finalPiece.Kind = PieceKindKing
	}

	updated = updated.WithPiece(finalPiece)
	return updated, captures, nil
}

func shouldPromote(color PlayerColor, pos Position, size BoardSize) bool {
	if color == PlayerColorLight {
		return int(pos.Row) == 0
	}
	return int(pos.Row) == int(size)-1
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func hasAnyCapture(board BoardState, mover PlayerColor) bool {
	for _, piece := range board.Pieces {
		if piece.Color != mover {
			continue
		}
		if canCaptureFrom(board, piece) {
			return true
		}
	}
	return false
}

func canCaptureFrom(board BoardState, piece BoardPiece) bool {
	directions := []struct{ dr, dc int }{{2, 2}, {2, -2}, {-2, 2}, {-2, -2}}
	for _, dir := range directions {
		if piece.Kind == PieceKindMan {
			if dir.dr/2 != piece.Color.ForwardDirection() {
				continue
			}
		}
		landing := Position{Row: piece.Position.Row + Coordinate(dir.dr), Col: piece.Position.Col + Coordinate(dir.dc)}
		if !landing.IsInside(board.Size) {
			continue
		}
		if _, occupied := board.PieceAt(landing); occupied {
			continue
		}
		middle := Position{Row: piece.Position.Row + Coordinate(dir.dr/2), Col: piece.Position.Col + Coordinate(dir.dc/2)}
		victim, ok := board.PieceAt(middle)
		if !ok || victim.Color == piece.Color {
			continue
		}
		return true
	}
	return false
}
