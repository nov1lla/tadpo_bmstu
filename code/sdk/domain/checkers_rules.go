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
	if err := validateMoveInput(piece, pathPositions, mover); err != nil {
		return BoardState{}, nil, err
	}
	mustCapture := hasAnyCapture(board, mover)

	updated := board.Clone().RemovePiece(piece.ID)
	current, newBoard, newUpdated, captures, capturePerformed, err := applyMoveSequence(board, updated, piece, pathPositions, mover, mustCapture)
	if err != nil {
		return BoardState{}, nil, err
	}

	if err := enforceMandatoryCapture(mustCapture, capturePerformed); err != nil {
		return BoardState{}, nil, err
	}
	if err := enforceContinueCapture(newUpdated, piece, current, capturePerformed); err != nil {
		return BoardState{}, nil, err
	}

	finalPiece := finalizePieceAfterMove(piece, current, newBoard.Size)
	newUpdated = newUpdated.WithPiece(finalPiece)
	return newUpdated, captures, nil
}

type stepKind uint8

const (
	stepKindSimple stepKind = iota + 1
	stepKindCapture
)

type stepDelta struct {
	deltaRow int
	deltaCol int
}

func validateMoveInput(piece BoardPiece, pathPositions []Position, mover PlayerColor) error {
	if len(pathPositions) == 0 {
		return ErrMovePathEmpty
	}
	if piece.Color != mover {
		return ErrMoveWrongPieceColor
	}
	return nil
}

func applyMoveSequence(board BoardState, updated BoardState, piece BoardPiece, pathPositions []Position, mover PlayerColor, mustCapture bool) (Position, BoardState, BoardState, []PieceID, bool, error) {
	captures := make([]PieceID, 0)
	current := piece.Position
	capturePerformed := false

	for _, target := range pathPositions {
		next, newBoard, newUpdated, capturedID, didCapture, err := applyMoveStep(board, updated, piece, current, target, mover, mustCapture, capturePerformed)
		if err != nil {
			return Position{}, BoardState{}, BoardState{}, nil, false, err
		}
		current = next
		board = newBoard
		updated = newUpdated
		if didCapture {
			capturePerformed = true
			captures = append(captures, capturedID)
		}
	}

	return current, board, updated, captures, capturePerformed, nil
}

func enforceMandatoryCapture(mustCapture bool, capturePerformed bool) error {
	if mustCapture && !capturePerformed {
		return ErrMoveMustCapture
	}
	return nil
}

func enforceContinueCapture(board BoardState, piece BoardPiece, current Position, capturePerformed bool) error {
	if !capturePerformed {
		return nil
	}

	pieceAfter := piece
	pieceAfter.Position = current
	if canCaptureFrom(board.WithPiece(pieceAfter), pieceAfter) {
		return ErrMoveMustContinue
	}
	return nil
}

func finalizePieceAfterMove(piece BoardPiece, current Position, size BoardSize) BoardPiece {
	finalPiece := piece
	finalPiece.Position = current
	if piece.Kind == PieceKindMan && shouldPromote(piece.Color, current, size) {
		finalPiece.Kind = PieceKindKing
	}
	return finalPiece
}

func applyMoveStep(board BoardState, updated BoardState, piece BoardPiece, current Position, target Position, mover PlayerColor, mustCapture bool, capturePerformed bool) (Position, BoardState, BoardState, PieceID, bool, error) {
	if err := validateTargetSquare(board.Size, updated, target); err != nil {
		return Position{}, BoardState{}, BoardState{}, "", false, err
	}

	kind, delta, err := classifyMoveStep(current, target)
	if err != nil {
		return Position{}, BoardState{}, BoardState{}, "", false, err
	}

	switch kind {
	case stepKindSimple:
		return applySimpleMoveStep(board, updated, piece, target, mover, delta, mustCapture, capturePerformed)
	case stepKindCapture:
		return applyCaptureMoveStep(board, updated, piece, current, target, mover, delta)
	default:
		return Position{}, BoardState{}, BoardState{}, "", false, ErrMoveIllegal
	}
}

func validateTargetSquare(boardSize BoardSize, updated BoardState, target Position) error {
	if !target.IsInside(boardSize) {
		return ErrMoveIllegal
	}
	if _, occupied := updated.PieceAt(target); occupied {
		return ErrMoveTargetOccupied
	}
	return nil
}

func applySimpleMoveStep(board BoardState, updated BoardState, piece BoardPiece, target Position, mover PlayerColor, delta stepDelta, mustCapture bool, capturePerformed bool) (Position, BoardState, BoardState, PieceID, bool, error) {
	if mustCapture {
		return Position{}, BoardState{}, BoardState{}, "", false, ErrMoveMustCapture
	}
	if capturePerformed {
		return Position{}, BoardState{}, BoardState{}, "", false, ErrMoveIllegal
	}
	if err := validateManStepDirection(piece, delta.deltaRow, mover); err != nil {
		return Position{}, BoardState{}, BoardState{}, "", false, err
	}
	return target, board, updated, "", false, nil
}

func applyCaptureMoveStep(board BoardState, updated BoardState, piece BoardPiece, current Position, target Position, mover PlayerColor, delta stepDelta) (Position, BoardState, BoardState, PieceID, bool, error) {
	victimPos := captureMiddle(current, delta)
	victim, ok := board.PieceAt(victimPos)
	if !ok || victim.Color == mover {
		return Position{}, BoardState{}, BoardState{}, "", false, ErrMoveIllegal
	}
	if err := validateManCaptureDirection(piece, delta.deltaRow, mover); err != nil {
		return Position{}, BoardState{}, BoardState{}, "", false, err
	}

	board = board.RemovePiece(victim.ID)
	updated = updated.RemovePiece(victim.ID)
	return target, board, updated, victim.ID, true, nil
}

func classifyMoveStep(current Position, target Position) (stepKind, stepDelta, error) {
	deltaRow := int(target.Row - current.Row)
	deltaCol := int(target.Col - current.Col)
	absRow, absCol := abs(deltaRow), abs(deltaCol)

	if absRow == 1 && absCol == 1 {
		return stepKindSimple, stepDelta{deltaRow: deltaRow, deltaCol: deltaCol}, nil
	}
	if absRow == 2 && absCol == 2 {
		return stepKindCapture, stepDelta{deltaRow: deltaRow, deltaCol: deltaCol}, nil
	}
	return 0, stepDelta{}, ErrMoveIllegal
}

func validateManStepDirection(piece BoardPiece, deltaRow int, mover PlayerColor) error {
	if piece.Kind != PieceKindMan {
		return nil
	}
	if deltaRow == mover.ForwardDirection() {
		return nil
	}
	return ErrMoveIllegal
}

func validateManCaptureDirection(piece BoardPiece, deltaRow int, mover PlayerColor) error {
	if piece.Kind != PieceKindMan {
		return nil
	}
	if deltaRow/2 == mover.ForwardDirection() {
		return nil
	}
	return ErrMoveIllegal
}

func captureMiddle(current Position, delta stepDelta) Position {
	return Position{
		Row: current.Row + Coordinate(delta.deltaRow/2),
		Col: current.Col + Coordinate(delta.deltaCol/2),
	}
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
