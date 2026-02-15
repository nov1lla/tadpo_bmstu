package usecase

import (
	"errors"
	"log"

	"ppo/sdk/domain"
	sdkport "ppo/sdk/port"
)

var (
	ErrPieceNotFound = errors.New("piece not found on the board")

	ErrCapturedPieceNotFound = errors.New("captured piece not found on the board")

	ErrTargetOccupied = errors.New("target cell is already occupied")

	ErrPathLeavesBoard = errors.New("path leaves the board boundaries")

	ErrRemovalBlocked = errors.New("captured piece removal path is blocked")

	ErrCaptureDestinationMismatch = errors.New("capture destination does not match captured position")

	ErrObstacleBlocked = errors.New("blocking piece cannot be moved aside")
)

type AnimateMoveCommand struct {
	PieceID         domain.PieceID
	Path            []domain.Position
	CapturedPieceID *domain.PieceID
}

type AnimationUseCase interface {
	AnimateMove(cmd AnimateMoveCommand) (domain.Animation, error)
}

type animator struct {
	board sdkport.BoardStateReader
}

func NewAnimator(board sdkport.BoardStateReader) AnimationUseCase {
	return &animator{board: board}
}

func (a *animator) AnimateMove(cmd AnimateMoveCommand) (domain.Animation, error) {
	attackerPos, ok := a.board.PositionOf(cmd.PieceID)
	if !ok {
		return domain.Animation{}, ErrPieceNotFound
	}

	path, err := domain.NewPath(cmd.Path, attackerPos)
	if err != nil {
		return domain.Animation{}, err
	}

	pathPositions := path.Positions()

	var (
		capture        bool
		capturedID     domain.PieceID
		capturedPos    domain.Position
		capturedExists bool
	)

	if cmd.CapturedPieceID != nil {
		capture = true
		capturedID = *cmd.CapturedPieceID
		capturedPos, capturedExists = a.board.PositionOf(capturedID)
		if !capturedExists {
			return domain.Animation{}, ErrCapturedPieceNotFound
		}
	}

	boardSize := a.board.BoardSize()
	pathSet := make(map[domain.Position]struct{}, len(pathPositions))
	for _, pos := range pathPositions {
		pathSet[pos] = struct{}{}
	}

	occupancy := make(map[domain.Position]*domain.PieceID)
	setOccupant(occupancy, attackerPos, cmd.PieceID)

	if capture {
		setOccupant(occupancy, capturedPos, capturedID)
	}

	preSteps := make([]domain.AnimationStep, 0, len(pathPositions))
	postSteps := make([]domain.AnimationStep, 0, len(pathPositions))

	for idx, next := range pathPositions {
		if !next.IsInside(boardSize) {

			return domain.Animation{}, ErrPathLeavesBoard
		}

		occupant, occupied := a.lookupOccupant(occupancy, next)
		isFinalStep := idx == len(pathPositions)-1
		if occupied {
			if capture && next == capturedPos && isFinalStep {
				if occupant != capturedID {
					return domain.Animation{}, ErrTargetOccupied
				}
			} else {
				moved, returns, err := a.moveBlockingPiece(occupancy, occupant, next, boardSize, pathSet)
				if err != nil {
					return domain.Animation{}, err
				}
				preSteps = append(preSteps, moved...)
				postSteps = append(returns, postSteps...)
			}
		}
	}

	if capture && path.Last() != capturedPos {
		return domain.Animation{}, ErrCaptureDestinationMismatch
	}

	steps := make([]domain.AnimationStep, 0, len(pathPositions)+len(preSteps)+len(postSteps)+4)

	if capture {
		stepsRemoval, err := a.animateRemoval(capturedID, capturedPos, boardSize, occupancy)
		if err != nil {
			return domain.Animation{}, err
		}
		steps = append(steps, stepsRemoval...)
	}

	steps = append(steps, preSteps...)

	previous := attackerPos
	for _, next := range pathPositions {
		steps = append(steps, domain.AnimationStep{
			PieceID: cmd.PieceID,
			From:    previous,
			To:      next,
		})
		setOccupant(occupancy, previous, domain.PieceID(""))
		setOccupant(occupancy, next, cmd.PieceID)
		previous = next
	}

	steps = append(steps, postSteps...)

	return domain.Animation{Steps: steps}, nil
}

func (a *animator) animateRemoval(pieceID domain.PieceID, start domain.Position, boardSize domain.BoardSize, occupancy map[domain.Position]*domain.PieceID) ([]domain.AnimationStep, error) {
	piece, ok := a.board.Piece(pieceID)
	if !ok {
		return nil, ErrCapturedPieceNotFound
	}
	exitPath := domain.ComputeCaptureExitPath(start, boardSize, piece.Color)
	log.Printf("capture exit path piece=%s color=%s from=%v path=%v", pieceID, piece.Color, start, exitPath)
	steps := make([]domain.AnimationStep, 0, len(exitPath))

	previous := start
	for _, next := range exitPath {
		if next.IsInside(boardSize) {
			if occupant, occupied := a.lookupOccupant(occupancy, next); occupied && occupant != pieceID {
				return nil, ErrRemovalBlocked
			}
		}

		steps = append(steps, domain.AnimationStep{
			PieceID: pieceID,
			From:    previous,
			To:      next,
		})
		setOccupant(occupancy, previous, domain.PieceID(""))
		if next.IsInside(boardSize) {
			setOccupant(occupancy, next, pieceID)
		}
		previous = next
	}

	setOccupant(occupancy, previous, domain.PieceID(""))

	return steps, nil
}

func (a *animator) moveBlockingPiece(occupancy map[domain.Position]*domain.PieceID, pieceID domain.PieceID, start domain.Position, boardSize domain.BoardSize, forbidden map[domain.Position]struct{}) ([]domain.AnimationStep, []domain.AnimationStep, error) {
	temp, ok := a.findTemporarySpot(occupancy, start, boardSize, forbidden)
	if !ok {
		return nil, nil, ErrObstacleBlocked
	}

	preStep := domain.AnimationStep{PieceID: pieceID, From: start, To: temp}
	setOccupant(occupancy, start, domain.PieceID(""))
	setOccupant(occupancy, temp, pieceID)
	returnStep := domain.AnimationStep{PieceID: pieceID, From: temp, To: start}

	return []domain.AnimationStep{preStep}, []domain.AnimationStep{returnStep}, nil
}

func (a *animator) findTemporarySpot(occupancy map[domain.Position]*domain.PieceID, start domain.Position, boardSize domain.BoardSize, forbidden map[domain.Position]struct{}) (domain.Position, bool) {
	intDirs := [][2]int{
		{-1, 0}, {1, 0}, {0, -1}, {0, 1},
		{-1, -1}, {-1, 1}, {1, -1}, {1, 1},
	}

	for _, delta := range intDirs {
		candidate := domain.Position{
			Row: domain.Coordinate(int(start.Row) + delta[0]),
			Col: domain.Coordinate(int(start.Col) + delta[1]),
		}
		if !candidate.IsInside(boardSize) {
			continue
		}
		if _, restricted := forbidden[candidate]; restricted {
			continue
		}
		if _, occupied := a.lookupOccupant(occupancy, candidate); occupied {
			continue
		}
		return candidate, true
	}

	return domain.Position{}, false
}

func (a *animator) lookupOccupant(occupancy map[domain.Position]*domain.PieceID, pos domain.Position) (domain.PieceID, bool) {
	if cached, ok := occupancy[pos]; ok {
		if cached == nil {
			return "", false
		}
		return *cached, true
	}

	id, ok := a.board.OccupantAt(pos)
	if ok {
		setOccupant(occupancy, pos, id)
		return id, true
	}

	setOccupant(occupancy, pos, domain.PieceID(""))
	return "", false
}

func setOccupant(occupancy map[domain.Position]*domain.PieceID, pos domain.Position, pieceID domain.PieceID) {
	if pieceID == "" {
		occupancy[pos] = nil
		return
	}
	copyID := pieceID
	occupancy[pos] = &copyID
}
