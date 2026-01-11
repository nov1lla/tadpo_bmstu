package domain

type UserID string

type GameID string

type MoveID string

type MoveNumber int

type UserName string

type PlayerColor string

const (
	PlayerColorLight PlayerColor = "light"
	PlayerColorDark  PlayerColor = "dark"
)

func (c PlayerColor) Opponent() PlayerColor {
	if c == PlayerColorLight {
		return PlayerColorDark
	}
	return PlayerColorLight
}

func (c PlayerColor) ForwardDirection() int {
	if c == PlayerColorLight {
		return -1
	}
	return 1
}
