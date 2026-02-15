<template>
  <div class="history" v-if="user">
    <aside class="history-sidebar">
      <h3>Games</h3>
      <ul>
        <li v-for="gameItem in games" :key="gameItem.id">
          <button
            :class="['list-button', { active: gameItem.id === selectedGame?.id }]"
            @click="openGame(gameItem.id)"
            :disabled="loading"
          >
            <span class="id">{{ formatGameLabel(gameItem) }}</span>
            <span class="meta">{{ formatStatus(gameItem.status) }} · {{ formatDate(gameItem.startedAt) }}</span>
          </button>
        </li>
      </ul>
    </aside>

    <section class="history-content" v-if="selectedGame">
      <div class="summary">
        <GamePanel :game="selectedGame">
          <button class="primary" @click="resumeGame" :disabled="loading">Resume game</button>
          <button class="danger" @click="deleteGame" :disabled="loading">Delete game</button>
        </GamePanel>
      </div>
      <div class="board-wide">
        <BoardView :board="selectedBoard" />
      </div>
      <HistoryTable :moves="selectedHistory" />
    </section>
    <section v-else class="history-empty">
      <p>Select a game to view its history.</p>
    </section>
  </div>
  <div v-else class="history-empty">
    <p>Create a player first, then return here to view games.</p>
    <RouterLink to="/" class="nav-button">Back to menu</RouterLink>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { RouterLink, useRouter } from 'vue-router';
import BoardView from '../components/BoardView.vue';
import HistoryTable from '../components/HistoryTable.vue';
import GamePanel from '../components/GamePanel.vue';
import { useGameStore } from '../viewmodels/useGameViewModel';
import { getGameState } from '../services/api';
import type { BoardState, Game, Move } from '../types';

const store = useGameStore();
const router = useRouter();

const loading = computed(() => store.loading.value);
const user = computed(() => store.user.value ?? null);
const games = computed(() => store.availableGames.value);

const selectedGame = ref<Game | null>(null);
const selectedBoard = ref<BoardState | null>(null);
const selectedHistory = ref<Move[]>([]);

onMounted(() => {
  store.loadGames();
});

async function openGame(id: string) {
  const state = await getGameState(id);
  selectedGame.value = state.game;
  selectedBoard.value = state.board;
  selectedHistory.value = state.moves;
}

async function resumeGame() {
  if (!selectedGame.value) {
    return;
  }
  await store.loadGame(selectedGame.value.id, { remember: true });
  await router.push('/game');
}

async function deleteGame() {
  if (!selectedGame.value) {
    return;
  }
  const id = selectedGame.value.id;
  await store.deleteGame(id);
  if (selectedGame.value?.id === id) {
    selectedGame.value = null;
    selectedBoard.value = null;
    selectedHistory.value = [];
  }
}

function formatStatus(status: string | undefined | null) {
  if (!status) {
    return 'unknown';
  }
  return status.replace(/_/g, ' ');
}

function formatDate(iso: string | undefined | null) {
  if (!iso) {
    return '—';
  }
  const parsed = new Date(iso);
  if (Number.isNaN(parsed.getTime())) {
    return '—';
  }
  return parsed.toLocaleString();
}

function formatGameLabel(gameItem: Game) {
  if (gameItem.number !== undefined && gameItem.number !== null) {
    return `#${gameItem.number}`;
  }
  if (gameItem.startedAt) {
    const parsed = new Date(gameItem.startedAt);
    if (!Number.isNaN(parsed.getTime())) {
      return parsed.toLocaleString();
    }
  }
  return gameItem.id;
}
</script>

<style scoped>
.history {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 24px;
}

.history-sidebar {
  background: var(--surface-color);
  padding: 16px;
  border-radius: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.history-sidebar ul {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.list-button {
  width: 100%;
  padding: 10px 12px;
  border-radius: 12px;
  border: none;
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-color);
  text-align: left;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.list-button.active {
  background: rgba(240, 180, 41, 0.18);
  color: var(--accent-color);
}

.list-button .id {
  font-weight: 600;
}

.list-button .meta {
  font-size: 12px;
  color: var(--muted-color);
}

.history-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.summary {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  width: 100%;
}

.summary :deep(.block) {
  width: 100%;
}

.primary,
.danger {
  padding: 10px 12px;
  border-radius: 12px;
  border: none;
  font-weight: 600;
  cursor: pointer;
}

.primary {
  background: var(--accent-color);
  color: #1a1a1a;
}

.danger {
  background: rgba(208, 77, 57, 0.2);
  color: #ffb4a3;
}

.primary:disabled,
.danger:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.history-empty {
  background: var(--surface-color);
  padding: 24px;
  border-radius: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.board-wide {
  width: 100%;
}

.board-wide :deep(.board) {
  width: 100%;
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
