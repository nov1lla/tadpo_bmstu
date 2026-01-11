package domain

import "fmt"

func NewCheckersInitialState() BoardState {
	board := NewBoardState(DefaultBoardSize)

	place := func(color PlayerColor, rows []int) {
		index := 1
		for _, row := range rows {
			startCol := 0
			if row%2 == 0 {
				startCol = 1
			}
			for col := startCol; col < int(board.Size); col += 2 {
				id := PieceID(fmt.Sprintf("%s-%02d", color, index))
				piece := BoardPiece{
					ID:       id,
					Position: Position{Row: Coordinate(row), Col: Coordinate(col)},
					Color:    color,
					Kind:     PieceKindMan,
				}
				board.Pieces[id] = piece
				index++
			}
		}
	}

	place(PlayerColorDark, []int{0, 1, 2})
	place(PlayerColorLight, []int{5, 6, 7})

	return board
}
