package domain

type AnimationStep struct {
	PieceID PieceID
	From    Position
	To      Position
}

type Animation struct {
	Steps []AnimationStep
}
