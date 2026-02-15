package domain

import "errors"

type GameStatus string

const (
	GameStatusPending    GameStatus = "pending"
	GameStatusInProgress GameStatus = "in_progress"
	GameStatusFinished   GameStatus = "finished"
)

type Game struct {
	ID            GameID
	Start         Timestamp
	End           *Timestamp
	Status        GameStatus
	UserID        UserID
	PlayerColor   PlayerColor
	IsPlayerFirst bool
}

var ErrInvalidGameTransition = errors.New("invalid game status transition")

var ErrInvalidGameTime = errors.New("invalid game timestamps")

func NewGame(id GameID, userID UserID, playerColor PlayerColor, isPlayerFirst bool, start Timestamp) Game {
	return Game{
		ID:            id,
		Start:         start,
		Status:        GameStatusPending,
		UserID:        userID,
		PlayerColor:   playerColor,
		IsPlayerFirst: isPlayerFirst,
	}
}

func (g Game) StartPlay() (Game, error) {
	if g.Status != GameStatusPending {
		return Game{}, ErrInvalidGameTransition
	}
	g.Status = GameStatusInProgress
	return g, nil
}

func (g Game) Finish(end Timestamp) (Game, error) {
	if end.Before(g.Start.Time) {
		return Game{}, ErrInvalidGameTime
	}
	g.Status = GameStatusFinished
	g.End = &end
	return g, nil
}
