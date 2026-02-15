package domain

import "testing"

func TestNewPathValidatesAdjacency(t *testing.T) {
	start := Position{Row: 2, Col: 2}
	path, err := NewPath([]Position{{Row: 3, Col: 3}, {Row: 4, Col: 4}}, start)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	positions := path.Positions()
	if len(positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(positions))
	}

	if positions[1] != (Position{Row: 4, Col: 4}) {
		t.Fatalf("unexpected final position: %v", positions[1])
	}
}

func TestNewPathRejectsNonAdjacentHop(t *testing.T) {
	start := Position{Row: 2, Col: 2}
	_, err := NewPath([]Position{{Row: 4, Col: 4}}, start)
	if err != ErrNonAdjacentHop {
		t.Fatalf("expected ErrNonAdjacentHop, got %v", err)
	}
}

func TestComputeExitPathLeavesBoard(t *testing.T) {
	start := Position{Row: 3, Col: 3}
	path := ComputeExitPath(start, 8)

	if len(path) == 0 {
		t.Fatalf("exit path should contain steps")
	}

	last := path[len(path)-1]
	if !last.IsOutside(8) {
		t.Fatalf("expected the last position to be outside the board, got %v", last)
	}
}
