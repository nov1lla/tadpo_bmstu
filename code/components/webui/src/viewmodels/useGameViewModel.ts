import { computed, reactive, ref } from 'vue';
import type { AnimationStep, BoardState, Game, Move, User } from '../types';
import {
  createGame,
  createUser,
  getGameState,
  getUser,
  login,
  listGames,
  register,
  requestOpponentMove,
  submitMove,
  type CreateGamePayload,
  type Difficulty,
  type PositionInput,
  type SubmitMovePayload
} from '../services/api';

const USER_STORAGE_KEY = 'ppo:user-id';
const GAME_STORAGE_KEY = 'ppo:game-id';
const DEFAULT_ANIMATION_DELAY_MS = 350;
const CONFIGURED_ANIMATION_DELAY_MS = (() => {
  const raw = import.meta.env.VITE_ANIMATION_DELAY_MS;
  if (!raw) {
    return DEFAULT_ANIMATION_DELAY_MS;
  }
  const parsed = Number(raw);
  if (Number.isFinite(parsed) && parsed >= 0) {
    return parsed;
  }
  return DEFAULT_ANIMATION_DELAY_MS;
})();

type GuardAction<T> = () => Promise<T>;

type SubmitOptions = { auto: boolean; difficulty: Difficulty };

type GameStore = ReturnType<typeof createGameStore>;

const store: GameStore = createGameStore();

export function useGameStore() {
  return store;
}

function logDebug(message: string, payload?: unknown) {
  if (typeof window !== 'undefined') {
    console.debug(`[game-store] ${message}`, payload ?? '');
  }
}

function createGameStore() {
  const loading = ref(false);
  const error = ref<string | null>(null);

  const user = ref<User | null>(null);
  const game = ref<Game | null>(null);
  const board = ref<BoardState | null>(null);
  const history = ref<Move[]>([]);
  const availableGames = ref<Game[]>([]);

  const opponentPreference = reactive({
    difficulty: 'medium' as Difficulty,
    auto: true
  });

  const hasGame = computed(() => !!game.value);

  async function guard<T>(action: GuardAction<T>, label?: string): Promise<T | undefined> {
    loading.value = true;
    error.value = null;
    label && logDebug(`${label}:start`);
    try {
      const result = await action();
      label && logDebug(`${label}:success`);
      return result;
    } catch (err) {
      console.error('GameStore action failed', err);
      error.value = err instanceof Error ? err.message : 'Unexpected error';
      label && logDebug(`${label}:error`, err);
      return undefined;
    } finally {
      loading.value = false;
      label && logDebug(`${label}:end`);
    }
  }

  function rememberGame(id: string) {
    if (id) {
      localStorage.setItem(GAME_STORAGE_KEY, id);
    }
  }

  function clearForUserSwitch() {
    clearGame();
    availableGames.value = [];
  }

  async function ensureUser(name: string) {
    await guard(async () => {
      const trimmed = name.trim();
      if (!trimmed) {
        throw new Error('Provide non-empty user name');
      }
      // Legacy helper: still supported by backend, but the main UI uses auth (login/register).
      const created = await createUser(trimmed);
      const userChanged = user.value?.id !== created.id;
      user.value = created;
      localStorage.setItem(USER_STORAGE_KEY, created.id);
      if (userChanged || game.value) {
        clearForUserSwitch();
      }
      await loadGames();
    }, 'ensureUser');
  }

  async function registerUser(loginValue: string, password: string) {
    await guard(async () => {
      const response = await register(loginValue, password);
      const userChanged = user.value?.id !== response.user.id;
      user.value = response.user;
      localStorage.setItem(USER_STORAGE_KEY, response.user.id);
      if (userChanged || game.value) {
        clearForUserSwitch();
      }
      await loadGames();
    }, 'register');
  }

  async function loginUser(loginValue: string, password: string) {
    await guard(async () => {
      const response = await login(loginValue, password);
      const userChanged = user.value?.id !== response.user.id;
      user.value = response.user;
      localStorage.setItem(USER_STORAGE_KEY, response.user.id);
      if (userChanged || game.value) {
        clearForUserSwitch();
      }
      await loadGames();
    }, 'login');
  }

  function logout() {
    user.value = null;
    localStorage.removeItem(USER_STORAGE_KEY);
    clearForUserSwitch();
  }

  function clearError() {
    error.value = null;
  }

  async function refreshUser() {
    const current = user.value?.id ?? localStorage.getItem(USER_STORAGE_KEY);
    if (!current) {
      return;
    }
    await guard(async () => {
      user.value = await getUser(current);
    }, 'refreshUser');
  }

  async function loadGames() {
    if (!user.value) {
      availableGames.value = [];
      return;
    }
    await guard(async () => {
      const response = await listGames(user.value!.id);
      availableGames.value = response.games;
    }, 'loadGames');
  }

  async function startGame(payload: CreateGamePayload) {
    if (!user.value) {
      throw new Error('Login before starting a game');
    }
    await guard(async () => {
      const response = await createGame(payload);
      game.value = response.game;
      board.value = response.board;
      history.value = response.moves;
      rememberGame(response.game.id);
      await refreshUser();
      await loadGames();
    }, 'startGame');
  }

  async function loadGame(gameId: string, options?: { remember?: boolean }) {
    const remember = options?.remember ?? false;
    await guard(async () => {
      const state = await getGameState(gameId);
      game.value = state.game;
      board.value = state.board;
      history.value = state.moves;
      if (remember) {
        rememberGame(state.game.id);
      }
    }, 'loadGame');
  }

  async function refreshGame() {
    const currentId = game.value?.id ?? localStorage.getItem(GAME_STORAGE_KEY);
    if (!currentId) {
      return;
    }
    await loadGame(currentId);
    if (!user.value) {
      await refreshUser();
    }
  }

  function restoreGameShell(): Game | null {
    const stored = localStorage.getItem(GAME_STORAGE_KEY);
    if (!stored) {
      return null;
    }
    game.value = {
      id: stored,
      status: 'pending',
      userId: user.value?.id ?? '',
      playerColor: 'light',
      isPlayerFirst: true,
      startedAt: new Date().toISOString()
    };
    return game.value;
  }

  async function submitUserMove(start: PositionInput, steps: PositionInput[], options?: SubmitOptions) {
    if (!game.value) {
      throw new Error('No active game');
    }
    if (steps.length === 0) {
      throw new Error('Provide at least one destination square');
    }
    if (options) {
      opponentPreference.auto = options.auto;
      opponentPreference.difficulty = options.difficulty;
    }
    await guard(async () => {
      const payload: SubmitMovePayload = {
        gameId: game.value!.id,
        start,
        steps,
        requestOpponent: false,
        opponentDifficulty: opponentPreference.difficulty
      };
      const outcome = await submitMove(payload);
      if (outcome.userMove) {
        history.value = history.value.concat(outcome.userMove);
      }
      if (outcome.userAnimation?.steps?.length) {
        await playAnimation(outcome.userAnimation.steps);
      }
      board.value = outcome.board;
      await refreshGame();
      await refreshUser();

      if (opponentPreference.auto) {
        await triggerOpponentMove(opponentPreference.difficulty);
      }
    }, 'submitMove');
  }

  async function triggerOpponentMove(difficulty?: Difficulty) {
    if (!game.value) {
      throw new Error('No active game');
    }
    await guard(async () => {
      const outcome = await requestOpponentMove(game.value!.id, difficulty ?? opponentPreference.difficulty);
      if (outcome.opponentMove) {
        history.value = history.value.concat(outcome.opponentMove);
      }
      if (outcome.opponentAnimation?.steps?.length) {
        await playAnimation(outcome.opponentAnimation.steps);
      }
      board.value = outcome.board;
      await refreshGame();
      await refreshUser();
    }, 'opponentMove');
  }

  async function playAnimation(steps: AnimationStep[], delayMs = CONFIGURED_ANIMATION_DELAY_MS) {
    if (!board.value || steps.length === 0) {
      return;
    }
    let current: BoardState = JSON.parse(JSON.stringify(board.value)) as BoardState;
    for (const step of steps) {
      current = await animateStep(current, step, delayMs);
      board.value = JSON.parse(JSON.stringify(current)) as BoardState;
    }
  }

  function applyAnimationStep(state: BoardState, step: AnimationStep): BoardState {
    const size = state.size ?? 8;
    const nextPieces = state.pieces.map((piece) => {
      if (piece.id !== step.pieceId) {
        return piece;
      }
      return { ...piece, row: step.to.row, col: step.to.col };
    });
    const inside = step.to.row >= 0 && step.to.col >= 0 && step.to.row < size && step.to.col < size;
    return {
      ...state,
      pieces: inside ? nextPieces : nextPieces.filter((piece) => piece.id !== step.pieceId)
    };
  }

  function animateStep(state: BoardState, step: AnimationStep, durationMs: number): Promise<BoardState> {
    if (durationMs <= 0) {
      return Promise.resolve(applyAnimationStep(state, step));
    }
    const size = state.size ?? 8;
    const inside = step.to.row >= 0 && step.to.col >= 0 && step.to.row < size && step.to.col < size;
    const start = performance.now();
    const from = { row: step.from.row, col: step.from.col };
    const to = { row: step.to.row, col: step.to.col };

    return new Promise((resolve) => {
      const frame = (now: number) => {
        const elapsed = now - start;
        const t = Math.min(1, elapsed / durationMs);
        const row = from.row + (to.row - from.row) * t;
        const col = from.col + (to.col - from.col) * t;
        const nextPieces = state.pieces.map((piece) => {
          if (piece.id !== step.pieceId) {
            return piece;
          }
          return { ...piece, row, col };
        });
        board.value = {
          ...state,
          pieces: nextPieces
        };
        if (t < 1) {
          requestAnimationFrame(frame);
          return;
        }
        const finalPieces = inside
          ? nextPieces.map((piece) =>
              piece.id === step.pieceId ? { ...piece, row: to.row, col: to.col } : piece
            )
          : nextPieces.filter((piece) => piece.id !== step.pieceId);
        resolve({
          ...state,
          pieces: finalPieces
        });
      };
      requestAnimationFrame(frame);
    });
  }

  function clearGame() {
    game.value = null;
    board.value = null;
    history.value = [];
    localStorage.removeItem(GAME_STORAGE_KEY);
  }

  async function restore() {
    const storedUser = localStorage.getItem(USER_STORAGE_KEY);
    if (storedUser) {
      await guard(async () => {
        user.value = await getUser(storedUser);
      }, 'restoreUser');
      await loadGames();
    }
    const storedGame = localStorage.getItem(GAME_STORAGE_KEY);
    if (storedGame) {
      await loadGame(storedGame, { remember: true });
    } else {
      restoreGameShell();
    }
  }

  const api = {
    loading,
    error,
    user,
    game,
    board,
    history,
    availableGames,
    opponentPreference,
    hasGame,
    ensureUser,
    registerUser,
    loginUser,
    logout,
    clearError,
    refreshUser,
    loadGames,
    startGame,
    loadGame,
    refreshGame,
    submitUserMove,
    triggerOpponentMove,
    clearGame,
    restore
  };

  if (typeof window !== 'undefined') {
    (window as any).__ppoStore = api;
  }

  return api;
}
