package usecase

import (
	"testing"

	"ppo/sdk/domain"
)

type boardStub struct {
	size   domain.BoardSize
	pieces map[domain.PieceID]domain.BoardPiece
}

func newBoardStub(size domain.BoardSize, placements map[domain.PieceID]domain.Position) *boardStub {
	state := domain.NewBoardState(size)
	for id, position := range placements {
		state.Pieces[id] = domain.BoardPiece{
			ID:       id,
			Position: position,
			Color:    domain.PlayerColorLight,
			Kind:     domain.PieceKindMan,
		}
	}
	return &boardStub{size: size, pieces: state.Pieces}
}

func (b *boardStub) BoardSize() domain.BoardSize {
	return b.size
}

func (b *boardStub) PositionOf(id domain.PieceID) (domain.Position, bool) {
	piece, ok := b.pieces[id]
	if !ok {
		return domain.Position{}, false
	}
	return piece.Position, true
}

func (b *boardStub) OccupantAt(pos domain.Position) (domain.PieceID, bool) {
	for id, piece := range b.pieces {
		if piece.Position == pos {
			return id, true
		}
	}
	return "", false
}

func (b *boardStub) Piece(id domain.PieceID) (domain.BoardPiece, bool) {
	piece, ok := b.pieces[id]
	if !ok {
		return domain.BoardPiece{}, false
	}
	return piece, true
}

func (b *boardStub) withPiece(id domain.PieceID, pos domain.Position) *boardStub {
	piece, ok := b.pieces[id]
	if !ok {
		piece = domain.BoardPiece{ID: id, Color: domain.PlayerColorLight, Kind: domain.PieceKindMan}
	}
	piece.Position = pos
	b.pieces[id] = piece
	return b
}

func TestAnimatorSimpleMove(t *testing.T) {
	board := newBoardStub(8, map[domain.PieceID]domain.Position{
		"attacker": {Row: 2, Col: 2},
	})

	animator := NewAnimator(board)
	animation, err := animator.AnimateMove(AnimateMoveCommand{
		PieceID: "attacker",
		Path:    []domain.Position{{Row: 3, Col: 3}},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(animation.Steps) != 1 {
		t.Fatalf("expected single step, got %d", len(animation.Steps))
	}

	step := animation.Steps[0]
	if step.PieceID != "attacker" || step.From != (domain.Position{Row: 2, Col: 2}) || step.To != (domain.Position{Row: 3, Col: 3}) {
		t.Fatalf("unexpected step %+v", step)
	}
}

/*
func TestAnimatorCaptureProducesRemovalBeforeAttack(t *testing.T) {
	capturedID := domain.PieceID("victim")
	board := newBoardStub(8, map[domain.PieceID]domain.Position{
		"attacker": {Row: 2, Col: 2},
		capturedID: {Row: 3, Col: 3},
	})

	animator := NewAnimator(board)
	animation, err := animator.AnimateMove(AnimateMoveCommand{
		PieceID:         "attacker",
		Path:            []domain.Position{{Row: 3, Col: 3}},
		CapturedPieceID: &capturedID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(animation.Steps) < 2 {
		t.Fatalf("expected at least 2 steps, got %d", len(animation.Steps))
	}

	for i, step := range animation.Steps[:len(animation.Steps)-1] {
		if step.PieceID != capturedID {
			t.Fatalf("expected removal step for victim at %d, got %+v", i, step)
		}
		if i == 0 {
			expected := domain.Position{Row: 3, Col: 3}
			if step.From != expected {
				t.Fatalf("unexpected starting position %+v", step.From)
			}
		}
	}

	last := animation.Steps[len(animation.Steps)-1]
	if last.PieceID != "attacker" || last.To != (domain.Position{Row: 3, Col: 3}) {
		t.Fatalf("unexpected final step %+v", last)
	}
}
*/

/*
func TestAnimatorDetectsBlockedRemoval(t *testing.T) {
	capturedID := domain.PieceID("victim")
	board := newBoardStub(8, map[domain.PieceID]domain.Position{
		"attacker": {Row: 2, Col: 2},
		capturedID: {Row: 3, Col: 3},
	})

	board.withPiece("blocker", domain.Position{Row: 4, Col: 5})

	animator := NewAnimator(board)
	_, err := animator.AnimateMove(AnimateMoveCommand{
		PieceID:         "attacker",
		Path:            []domain.Position{{Row: 3, Col: 3}},
		CapturedPieceID: &capturedID,
	})

	if err != ErrRemovalBlocked {
		t.Fatalf("expected ErrRemovalBlocked, got %v", err)
	}
}
*/

/*
func TestAnimatorRejectsCaptureWhenDestinationDiffers(t *testing.T) {
	capturedID := domain.PieceID("victim")
	board := newBoardStub(8, map[domain.PieceID]domain.Position{
		"attacker": {Row: 2, Col: 2},
		capturedID: {Row: 3, Col: 3},
	})

	animator := NewAnimator(board)
	_, err := animator.AnimateMove(AnimateMoveCommand{
		PieceID:         "attacker",
		Path:            []domain.Position{{Row: 3, Col: 2}},
		CapturedPieceID: &capturedID,
	})

	if err != ErrCaptureDestinationMismatch {
		t.Fatalf("expected ErrCaptureDestinationMismatch, got %v", err)
	}
}
*/

func TestAnimatorRejectsPathLeavingBoard(t *testing.T) {
	board := newBoardStub(8, map[domain.PieceID]domain.Position{
		"attacker": {Row: 0, Col: 0},
	})

	animator := NewAnimator(board)
	_, err := animator.AnimateMove(AnimateMoveCommand{
		PieceID: "attacker",
		Path:    []domain.Position{{Row: -1, Col: -1}},
	})

	if err != ErrPathLeavesBoard {
		t.Fatalf("expected ErrPathLeavesBoard, got %v", err)
	}
}

/*
func TestAnimatorMovesBlockingPieceAside(t *testing.T) {
	capturedID := domain.PieceID("victim")
	blockerID := domain.PieceID("blocker")
	board := newBoardStub(8, map[domain.PieceID]domain.Position{
		"attacker": {Row: 3, Col: 3},
		capturedID: {Row: 2, Col: 2},
		blockerID:  {Row: 2, Col: 3},
	})

	animator := NewAnimator(board)
	animation, err := animator.AnimateMove(AnimateMoveCommand{
		PieceID:         "attacker",
		Path:            []domain.Position{{Row: 2, Col: 3}, {Row: 2, Col: 2}},
		CapturedPieceID: &capturedID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(animation.Steps) < 5 {
		t.Fatalf("unexpected steps count %d", len(animation.Steps))
	}

	var (
		blockerMoved bool
		attackerMid  bool
		attackerEnd  bool
		blockerBack  bool
	)

	for _, step := range animation.Steps {
		if step.PieceID == blockerID && step.From == (domain.Position{Row: 2, Col: 3}) {
			blockerMoved = true
		}
		if step.PieceID == "attacker" && step.To == (domain.Position{Row: 2, Col: 3}) {
			attackerMid = true
		}
		if step.PieceID == "attacker" && step.To == (domain.Position{Row: 2, Col: 2}) {
			attackerEnd = true
		}
		if step.PieceID == blockerID && step.To == (domain.Position{Row: 2, Col: 3}) {
			blockerBack = true
		}
	}

	if !blockerMoved {
		t.Fatalf("expected blocker to move aside")
	}
	if !attackerMid {
		t.Fatalf("expected attacker to move through freed cell")
	}
	if !attackerEnd {
		t.Fatalf("expected attacker to finish capture")
	}
	if !blockerBack {
		t.Fatalf("expected blocker to return afterwards")
	}
}
*/
