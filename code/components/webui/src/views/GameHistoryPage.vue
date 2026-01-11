<template>
  <div class="history" v-if="user">
    <aside class="history-sidebar">
      <h3>Games</h3>
      <ul>
        <li v-for="gameItem in games" :key="gameItem.id">
          <button
            :class="['list-button', { active: gameItem.id === game?.id }]"
            @click="openGame(gameItem.id)"
            :disabled="loading"
          >
            <span class="id">{{ gameItem.id }}</span>
            <span class="meta">{{ formatStatus(gameItem.status) }} · {{ formatDate(gameItem.startedAt) }}</span>
          </button>
        </li>
      </ul>
    </aside>

    <section class="history-content" v-if="game">
      <div class="summary">
        <StatsPanel :user="user" :game="game" />
        <RouterLink to="/game" class="nav-button">Open in play view</RouterLink>
      </div>
      <BoardView :board="board" />
      <HistoryTable :moves="history" />
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
import { computed, onMounted } from 'vue';
import { RouterLink } from 'vue-router';
import BoardView from '../components/BoardView.vue';
import HistoryTable from '../components/HistoryTable.vue';
import StatsPanel from '../components/StatsPanel.vue';
import { useGameStore } from '../viewmodels/useGameViewModel';

const store = useGameStore();

const loading = computed(() => store.loading.value);
const user = computed(() => store.user.value ?? null);
const game = computed(() => store.game.value ?? null);
const board = computed(() => store.board.value ?? null);
const history = computed(() => store.history.value);
const games = computed(() => store.availableGames.value);

onMounted(() => {
  store.loadGames();
});

async function openGame(id: string) {
  await store.loadGame(id);
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

.history-empty {
  background: var(--surface-color);
  padding: 24px;
  border-radius: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
