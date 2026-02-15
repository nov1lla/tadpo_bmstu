package domain

import "fmt"

type Position struct {
	Row Coordinate
	Col Coordinate
}

func (p Position) String() string {
	return fmt.Sprintf("(%d,%d)", p.Row, p.Col)
}

func (p Position) IsAdjacent(target Position) bool {
	rowDiff := int(p.Row - target.Row)
	if rowDiff < 0 {
		rowDiff = -rowDiff
	}

	colDiff := int(p.Col - target.Col)
	if colDiff < 0 {
		colDiff = -colDiff
	}

	if rowDiff == 0 && colDiff == 0 {
		return false
	}

	return rowDiff <= 1 && colDiff <= 1
}

func (p Position) IsInside(boardSize BoardSize) bool {
	size := int(boardSize)
	return int(p.Row) >= 0 && int(p.Col) >= 0 && int(p.Row) < size && int(p.Col) < size
}

func (p Position) IsOutside(boardSize BoardSize) bool {
	return !p.IsInside(boardSize)
}
