package domain

type BoardPiece struct {
	ID       PieceID
	Position Position
	Color    PlayerColor
	Kind     PieceKind
}

type BoardState struct {
	Size   BoardSize
	Pieces map[PieceID]BoardPiece
}

func NewBoardState(size BoardSize) BoardState {
	return BoardState{
		Size:   size,
		Pieces: make(map[PieceID]BoardPiece),
	}
}

func (bs BoardState) Clone() BoardState {
	copyPieces := make(map[PieceID]BoardPiece, len(bs.Pieces))
	for k, v := range bs.Pieces {
		copyPieces[k] = v
	}
	return BoardState{Size: bs.Size, Pieces: copyPieces}
}

func (bs BoardState) WithPiece(piece BoardPiece) BoardState {
	clone := bs.Clone()
	clone.Pieces[piece.ID] = piece
	return clone
}

func (bs BoardState) MovePiece(pieceID PieceID, position Position) BoardState {
	clone := bs.Clone()
	if piece, ok := clone.Pieces[pieceID]; ok {
		piece.Position = position
		clone.Pieces[pieceID] = piece
	}
	return clone
}

func (bs BoardState) RemovePiece(pieceID PieceID) BoardState {
	clone := bs.Clone()
	delete(clone.Pieces, pieceID)
	return clone
}

func (bs BoardState) Piece(id PieceID) (BoardPiece, bool) {
	piece, ok := bs.Pieces[id]
	return piece, ok
}

func (bs BoardState) PieceAt(pos Position) (BoardPiece, bool) {
	for _, piece := range bs.Pieces {
		if piece.Position == pos {
			return piece, true
		}
	}
	return BoardPiece{}, false
}

func (bs BoardState) Positions() map[PieceID]Position {
	positions := make(map[PieceID]Position, len(bs.Pieces))
	for id, piece := range bs.Pieces {
		positions[id] = piece.Position
	}
	return positions
}
