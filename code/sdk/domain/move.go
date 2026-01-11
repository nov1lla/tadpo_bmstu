package domain

import "errors"

type Move struct {
	ID            MoveID
	GameID        GameID
	PieceID       PieceID
	Number        MoveNumber
	StartPosition Position
	Trajectory    []Position
	EndPosition   Position
	CreatedAt     Timestamp
}

var ErrInvalidMoveNumber = errors.New("move number must be positive")

var ErrInvalidMovePath = errors.New("move must change position")
var ErrInvalidMoveTrajectory = errors.New("move trajectory must contain target positions")

func NewMove(id MoveID, gameID GameID, pieceID PieceID, number MoveNumber, start Position, trajectory []Position, createdAt Timestamp) (Move, error) {
	if number <= 0 {
		return Move{}, ErrInvalidMoveNumber
	}
	if len(trajectory) == 0 {
		return Move{}, ErrInvalidMoveTrajectory
	}
	end := trajectory[len(trajectory)-1]
	if start == end {
		return Move{}, ErrInvalidMovePath
	}
	return Move{
		ID:            id,
		GameID:        gameID,
		PieceID:       pieceID,
		Number:        number,
		StartPosition: start,
		Trajectory:    append([]Position(nil), trajectory...),
		EndPosition:   end,
		CreatedAt:     createdAt,
	}, nil
}
