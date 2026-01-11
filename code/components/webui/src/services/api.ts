import type { Game, GameStateResponse, MoveOutcome, User } from '../types';

const API_BASE = '/api';

type Difficulty = 'easy' | 'medium' | 'hard';

type PositionInput = { row: number; col: number };

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(init.headers ?? {})
    },
    ...init
  });

  const text = await response.text();
  const payload = text ? JSON.parse(text) : null;

  if (!response.ok) {
    const message = payload?.error ?? response.statusText;
    throw new Error(message || 'Request failed');
  }

  return payload as T;
}

export async function createUser(name: string): Promise<User> {
  return request<User>('/users', {
    method: 'POST',
    body: JSON.stringify({ name })
  });
}

export interface AuthResponse {
  user: User;
}

export async function register(login: string, password: string): Promise<AuthResponse> {
  return request<AuthResponse>('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ login, password })
  });
}

export async function login(login: string, password: string): Promise<AuthResponse> {
  return request<AuthResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ login, password })
  });
}

export async function getUser(id: string): Promise<User> {
  return request<User>(`/users/${id}`);
}

export async function updateUser(id: string, patch: Partial<User>): Promise<User> {
  return request<User>(`/users/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(patch)
  });
}

export interface CreateGamePayload {
  userId: string;
  playerColor: 'light' | 'dark';
  isPlayerFirst: boolean;
}

export async function createGame(payload: CreateGamePayload): Promise<GameStateResponse> {
  return request<GameStateResponse>('/games', {
    method: 'POST',
    body: JSON.stringify(payload)
  });
}

export async function getGameState(gameId: string): Promise<GameStateResponse> {
  return request<GameStateResponse>(`/games/${gameId}`);
}

export interface GamesListResponse {
  games: Game[];
}

export async function listGames(userId: string): Promise<GamesListResponse> {
  const params = new URLSearchParams({ userId });
  return request<GamesListResponse>(`/games?${params.toString()}`);
}

export interface SubmitMovePayload {
  gameId: string;
  start: PositionInput;
  steps: PositionInput[];
  requestOpponent: boolean;
  opponentDifficulty: Difficulty;
}

export async function submitMove(payload: SubmitMovePayload): Promise<MoveOutcome> {
  const { gameId, ...rest } = payload;
  return request<MoveOutcome>(`/games/${gameId}/moves`, {
    method: 'POST',
    body: JSON.stringify(rest)
  });
}

export async function requestOpponentMove(gameId: string, difficulty: Difficulty): Promise<MoveOutcome> {
  return request<MoveOutcome>(`/games/${gameId}/opponent-move`, {
    method: 'POST',
    body: JSON.stringify({ difficulty })
  });
}

export type { Difficulty, PositionInput };
