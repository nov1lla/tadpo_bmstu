package domain

import "time"

type Coordinate int

type BoardSize int

type Timestamp struct {
	time.Time
}

func NewTimestamp(t time.Time) Timestamp {
	return Timestamp{Time: t}
}

func (ts Timestamp) ToTime() time.Time {
	return ts.Time
}

type DifficultyLevel string

const (
	DifficultyEasy   DifficultyLevel = "easy"
	DifficultyMedium DifficultyLevel = "medium"
	DifficultyHard   DifficultyLevel = "hard"
)
