<template>
  <div class="play" v-if="game">
    <section class="play-main">
      <BoardView :board="board" />
      <div class="play-controls">
        <MoveForm
          :disabled="loading"
          :defaultDifficulty="opponentPreference.difficulty"
          :autoOpponent="opponentPreference.auto"
          :pieces="playerPieces"
          @submit="submitMove"
          @requestOpponent="requestOpponentMove"
        />
        <button class="danger" @click="resign" :disabled="loading">
          Resign game
        </button>
      </div>
    </section>

    <HistoryTable :moves="history" />
  </div>
  <div v-else class="empty">
    <p>No active game. Start one from the main menu.</p>
    <RouterLink to="/" class="nav-button">Back to menu</RouterLink>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { RouterLink } from 'vue-router';
import BoardView from '../components/BoardView.vue';
import MoveForm from '../components/MoveForm.vue';
import HistoryTable from '../components/HistoryTable.vue';
import { useGameStore } from '../viewmodels/useGameViewModel';
import type { Difficulty } from '../services/api';

const store = useGameStore();

const loading = computed(() => store.loading.value);
const opponentPreference = store.opponentPreference;
const game = computed(() => store.game.value ?? null);
const board = computed(() => store.board.value ?? null);
const history = computed(() => store.history.value);

const playerPieces = computed(() => {
  const currentGame = game.value;
  const currentBoard = board.value;
  if (!currentGame || !currentBoard) {
    return [];
  }
  const color = currentGame.playerColor;
  return (currentBoard.pieces ?? [])
    .filter((piece) => piece.color === color)
    .map((piece) => ({
      id: piece.id,
      row: Math.round(piece.row),
      col: Math.round(piece.col),
      color: piece.color
    }))
    .sort((a, b) => (a.row === b.row ? a.col - b.col : a.row - b.row));
});

async function submitMove(payload: { start: { row: number; col: number }; steps: { row: number; col: number }[]; auto: boolean; difficulty: Difficulty }) {
  await store.submitUserMove(payload.start, payload.steps, {
    auto: payload.auto,
    difficulty: payload.difficulty
  });
}

async function requestOpponentMove(payload: { difficulty: Difficulty }) {
  await store.triggerOpponentMove(payload.difficulty);
}

function resign() {
  store.clearGame();
}

onMounted(() => {
  store.refreshGame();
});
</script>

<style scoped>
.play {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.play-main {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
}

.play-controls {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

button.secondary,
button.danger {
  padding: 10px 12px;
  border-radius: 12px;
  border: none;
  font-weight: 600;
  cursor: pointer;
}

button.secondary {
  background: rgba(240, 180, 41, 0.18);
  color: var(--accent-color);
}

button.danger {
  background: rgba(208, 77, 57, 0.2);
  color: #ffb4a3;
}

button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.empty {
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: flex-start;
  background: var(--surface-color);
  padding: 24px;
  border-radius: 16px;
}

.nav-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 10px 12px;
  border-radius: 12px;
  background: rgba(240, 180, 41, 0.18);
  color: var(--accent-color);
  text-decoration: none;
  font-weight: 600;
}
</style>
