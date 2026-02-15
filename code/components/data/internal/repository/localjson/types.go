package localjson

import "time"

type storedUser struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Rating     int     `json:"rating"`
	WinStreak  int     `json:"win_streak"`
	LastGameID *string `json:"last_game_id,omitempty"`
}

type storedUserCredentials struct {
	UserID       string `json:"user_id"`
	PasswordHash string `json:"password_hash"`
}

type storedGame struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	Start         time.Time  `json:"start"`
	End           *time.Time `json:"end,omitempty"`
	Status        string     `json:"status"`
	PlayerColor   string     `json:"player_color"`
	IsPlayerFirst bool       `json:"is_player_first"`
	BoardSize     int        `json:"board_size"`
}

type storedBoardState struct {
	Size   int                         `json:"size"`
	Pieces map[string]storedBoardPiece `json:"pieces"`
}

type storedBoardPiece struct {
	Row   int    `json:"row"`
	Col   int    `json:"col"`
	Color string `json:"color"`
	Kind  string `json:"kind"`
}

type storedMove struct {
	ID         string           `json:"id"`
	GameID     string           `json:"game_id"`
	PieceID    string           `json:"piece_id"`
	Number     int              `json:"number"`
	Start      storedPosition   `json:"start"`
	Trajectory []storedPosition `json:"trajectory"`
	CreatedAt  time.Time        `json:"created_at"`
}

type storedPosition struct {
	Row int `json:"row"`
	Col int `json:"col"`
}
