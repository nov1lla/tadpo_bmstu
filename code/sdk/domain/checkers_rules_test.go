package domain

import (
	"errors"
	"testing"
)

func TestCheckersRules_MustCaptureWhenAvailable(t *testing.T) {
	board := NewBoardState(DefaultBoardSize)
	lightCapturer := BoardPiece{ID: "l-1", Position: Position{Row: 5, Col: 0}, Color: PlayerColorLight, Kind: PieceKindMan}
	lightOther := BoardPiece{ID: "l-2", Position: Position{Row: 5, Col: 2}, Color: PlayerColorLight, Kind: PieceKindMan}
	darkVictim := BoardPiece{ID: "d-1", Position: Position{Row: 4, Col: 1}, Color: PlayerColorDark, Kind: PieceKindMan}
	board = board.WithPiece(lightCapturer).WithPiece(lightOther).WithPiece(darkVictim)

	rules := CheckersRules{}
	_, _, err := rules.ValidateAndApplyMove(board, lightOther, []Position{{Row: 4, Col: 3}}, PlayerColorLight)
	if !errors.Is(err, ErrMoveMustCapture) {
		t.Fatalf("expected ErrMoveMustCapture, got %v", err)
	}
}

func TestCheckersRules_MustContinueCaptureWhilePossible(t *testing.T) {
	board := NewBoardState(DefaultBoardSize)
	light := BoardPiece{ID: "l-1", Position: Position{Row: 5, Col: 0}, Color: PlayerColorLight, Kind: PieceKindMan}
	dark1 := BoardPiece{ID: "d-1", Position: Position{Row: 4, Col: 1}, Color: PlayerColorDark, Kind: PieceKindMan}
	dark2 := BoardPiece{ID: "d-2", Position: Position{Row: 2, Col: 3}, Color: PlayerColorDark, Kind: PieceKindMan}
	board = board.WithPiece(light).WithPiece(dark1).WithPiece(dark2)

	rules := CheckersRules{}

	_, _, err := rules.ValidateAndApplyMove(board, light, []Position{{Row: 3, Col: 2}}, PlayerColorLight)
	if !errors.Is(err, ErrMoveMustContinue) {
		t.Fatalf("expected ErrMoveMustContinue, got %v", err)
	}

	updated, captures, err := rules.ValidateAndApplyMove(board, light, []Position{{Row: 3, Col: 2}, {Row: 1, Col: 4}}, PlayerColorLight)
	if err != nil {
		t.Fatalf("expected move to succeed, got %v", err)
	}
	if len(captures) != 2 {
		t.Fatalf("expected 2 captures, got %d (%v)", len(captures), captures)
	}
	if _, ok := updated.Piece(dark1.ID); ok {
		t.Fatalf("expected victim %s removed", dark1.ID)
	}
	if _, ok := updated.Piece(dark2.ID); ok {
		t.Fatalf("expected victim %s removed", dark2.ID)
	}
}
