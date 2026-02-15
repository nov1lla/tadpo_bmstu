package domain

func ComputeExitPath(start Position, boardSize BoardSize) []Position {
	return ComputeCaptureExitPath(start, boardSize, PlayerColorLight)
}

func ComputeCaptureExitPath(start Position, boardSize BoardSize, owner PlayerColor) []Position {
	if boardSize <= 0 {
		return []Position{start}
	}

	directionRow := -1
	if owner == PlayerColorLight {
		directionRow = 1
	}

	leftPath, leftRow, leftLen := captureExitPath(start, boardSize, directionRow, -1)
	rightPath, rightRow, rightLen := captureExitPath(start, boardSize, directionRow, 1)

	if len(leftPath) == 0 {
		return rightPath
	}
	if len(rightPath) == 0 {
		return leftPath
	}

	if owner == PlayerColorDark {
		if leftRow < rightRow {
			return leftPath
		}
		if rightRow < leftRow {
			return rightPath
		}
	} else {
		if leftRow > rightRow {
			return leftPath
		}
		if rightRow > leftRow {
			return rightPath
		}
	}

	if leftLen <= rightLen {
		return leftPath
	}
	return rightPath
}

func captureExitPath(start Position, boardSize BoardSize, directionRow int, directionCol int) ([]Position, int, int) {
	path := make([]Position, 0, int(boardSize)+2)
	lastInsideRow := int(start.Row)
	current := start

	first := Position{Row: current.Row, Col: current.Col + Coordinate(directionCol)}
	path = append(path, first)
	if !first.IsInside(boardSize) {
		return path, lastInsideRow, len(path)
	}
	current = first
	lastInsideRow = int(current.Row)

	for {
		next := Position{
			Row: current.Row + Coordinate(directionRow),
			Col: current.Col + Coordinate(directionCol),
		}
		path = append(path, next)
		if !next.IsInside(boardSize) {
			break
		}
		current = next
		lastInsideRow = int(current.Row)
	}

	return path, lastInsideRow, len(path)
}
