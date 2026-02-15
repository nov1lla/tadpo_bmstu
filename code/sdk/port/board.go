package port

import "ppo/sdk/domain"

type BoardStateReader interface {
	BoardSize() domain.BoardSize

	PositionOf(id domain.PieceID) (domain.Position, bool)

	OccupantAt(pos domain.Position) (domain.PieceID, bool)

	Piece(id domain.PieceID) (domain.BoardPiece, bool)
}
