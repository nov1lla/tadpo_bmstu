<template>
  <div class="menu">
    <section class="panel">
      <h3>New game</h3>
      <form class="stack" @submit.prevent="startMatch">
        <div class="field">
          <label>Piece color</label>
          <select v-model="playerColor">
            <option value="light">light (bottom)</option>
            <option value="dark">dark (top)</option>
          </select>
        </div>
        <label class="checkbox">
          <input type="checkbox" v-model="isPlayerFirst" />
          <span>Player moves first</span>
        </label>
        <button type="submit" :disabled="loading || !user">
          {{ loading ? 'Preparing…' : 'New game' }}
        </button>
      </form>
    </section>

    <section class="panel">
      <PlayerPanel :user="user" />
    </section>

    <section class="panel">
      <GamePanel :game="game" />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import PlayerPanel from '../components/PlayerPanel.vue';
import GamePanel from '../components/GamePanel.vue';
import { useGameStore } from '../viewmodels/useGameViewModel';

const store = useGameStore();
const loading = computed(() => store.loading.value);
const user = computed(() => store.user.value ?? null);
const game = computed(() => store.game.value ?? null);

const playerColor = ref<'light' | 'dark'>('light');
const isPlayerFirst = ref(true);

async function startMatch() {
  const current = store.user.value;
  if (!current) {
    return;
  }
  await store.startGame({
    userId: current.id,
    playerColor: playerColor.value,
    isPlayerFirst: isPlayerFirst.value
  });
}
</script>

<style scoped>
.menu {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 16px;
}

.panel {
  background: var(--surface-color);
  border-radius: 16px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

h3 {
  margin: 0;
  font-size: 16px;
}

.stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 14px;
}

input,
select,
button {
  padding: 10px 12px;
  border-radius: 12px;
  border: none;
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-color);
}

button {
  background: var(--accent-color);
  color: #1a1a1a;
  font-weight: 600;
  cursor: pointer;
}

button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.checkbox {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--muted-color);
  font-size: 14px;
}

.hint {
  margin: 0;
  font-size: 12px;
  color: var(--muted-color);
}
</style>
