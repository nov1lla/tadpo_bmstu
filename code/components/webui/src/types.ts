export interface Position {
  row: number;
  col: number;
}

export interface Piece {
  id: string;
  row: number;
  col: number;
  color: 'light' | 'dark';
  kind: 'man' | 'king';
}

export interface BoardState {
  size: number;
  pieces: Piece[];
}

export interface Move {
  id: string;
  number: number;
  pieceId: string;
  start: Position;
  path: Position[];
  createdAt: string;
}

export interface User {
  id: string;
  name: string;
  rating: number;
  winStreak: number;
  lastGameId?: string;
}

export interface Game {
  id: string;
  number?: number;
  status: string;
  userId: string;
  playerColor: 'light' | 'dark';
  isPlayerFirst: boolean;
  startedAt: string;
  finishedAt?: string;
}

export interface GameStateResponse {
  game: Game;
  board: BoardState;
  moves: Move[];
}

export interface MoveOutcome {
  userMove?: Move;
  opponentMove?: Move;
  board: BoardState;
  history?: Move[];
  userAnimation?: Animation;
  opponentAnimation?: Animation;
}

export interface AnimationStep {
  pieceId: string;
  from: Position;
  to: Position;
}

export interface Animation {
  steps: AnimationStep[];
}
