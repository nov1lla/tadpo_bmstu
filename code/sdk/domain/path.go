package domain

import "errors"

var (
	ErrEmptyPath = errors.New("path must contain at least one position")

	ErrNonAdjacentHop = errors.New("path contains non-adjacent positions")
)

type Path struct {
	positions []Position
}

func NewPath(positions []Position, start Position) (Path, error) {
	if len(positions) == 0 {
		return Path{}, ErrEmptyPath
	}

	previous := start
	for _, current := range positions {
		if !previous.IsAdjacent(current) {
			return Path{}, ErrNonAdjacentHop
		}
		previous = current
	}

	return Path{positions: append([]Position(nil), positions...)}, nil
}

func (p Path) Positions() []Position {
	return append([]Position(nil), p.positions...)
}

func (p Path) Last() Position {
	return p.positions[len(p.positions)-1]
}
