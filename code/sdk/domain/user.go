package domain

type Rating int

type WinStreak int

type User struct {
	ID         UserID
	Name       UserName
	Rating     Rating
	WinStreak  WinStreak
	LastGameID *GameID
}

func (u User) WithRating(r Rating) User {
	u.Rating = r
	return u
}

func (u User) WithWinStreak(streak WinStreak) User {
	u.WinStreak = streak
	return u
}

func (u User) WithLastGame(gameID GameID) User {
	u.LastGameID = &gameID
	return u
}

func (u User) WithoutLastGame() User {
	u.LastGameID = nil
	return u
}
