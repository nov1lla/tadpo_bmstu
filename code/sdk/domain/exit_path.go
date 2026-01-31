package domain

func ComputeExitPath(start Position, boardSize BoardSize) []Position {
	return ComputeCaptureExitPath(start, boardSize, PlayerColorLight)
}

func ComputeCaptureExitPath(start Position, boardSize BoardSize, owner PlayerColor) []Position {
	if boardSize <= 0 {
		return []Position{start}
	}

	directionRow := captureDirectionRow(owner)
	leftPath, leftRow, leftLen := captureExitPath(start, boardSize, directionRow, -1)
	rightPath, rightRow, rightLen := captureExitPath(start, boardSize, directionRow, 1)

	return chooseCaptureExitPath(owner, leftPath, rightPath, leftRow, rightRow, leftLen, rightLen)
}

func captureDirectionRow(owner PlayerColor) int {
	if owner == PlayerColorLight {
		return 1
	}
	return -1
}

func chooseCaptureExitPath(owner PlayerColor, leftPath []Position, rightPath []Position, leftRow int, rightRow int, leftLen int, rightLen int) []Position {
	if len(leftPath) == 0 {
		return rightPath
	}
	if len(rightPath) == 0 {
		return leftPath
	}

	preferred := compareCaptureExitLastInsideRow(owner, leftPath, rightPath, leftRow, rightRow)
	if preferred != nil {
		return preferred
	}

	if leftLen <= rightLen {
		return leftPath
	}
	return rightPath
}

func compareCaptureExitLastInsideRow(owner PlayerColor, leftPath []Position, rightPath []Position, leftRow int, rightRow int) []Position {
	if owner == PlayerColorDark {
		if leftRow < rightRow {
			return leftPath
		}
		if rightRow < leftRow {
			return rightPath
		}
		return nil
	}

	if leftRow > rightRow {
		return leftPath
	}
	if rightRow > leftRow {
		return rightPath
	}
	return nil
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
