<template>
  <div class="history">
    <div class="history-header">
      <h3 class="section-title">Move history</h3>
      <button v-if="showToggle" type="button" class="toggle" @click="expanded = !expanded">
        {{ expanded ? 'Show latest' : `Show all (${moves.length})` }}
      </button>
    </div>
    <table v-if="moves.length" class="history-table">
      <thead>
        <tr>
          <th>#</th>
          <th>Piece</th>
          <th>Start</th>
          <th>Path</th>
          <th>Time</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="move in visibleMoves" :key="move.id">
          <td>{{ move.number }}</td>
          <td>{{ move.pieceId }}</td>
          <td>{{ formatPosition(move.start) }}</td>
          <td>{{ move.path.map(formatPosition).join(' → ') }}</td>
          <td>{{ formatTime(move.createdAt) }}</td>
        </tr>
      </tbody>
    </table>
    <p v-else class="empty">No moves yet</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { Move } from '../types';

const props = defineProps<{ moves: Move[]; initialLimit?: number }>();

const expanded = ref(false);
const limit = computed(() => props.initialLimit ?? 4);
const showToggle = computed(() => props.moves.length > limit.value);
const visibleMoves = computed(() => {
  if (expanded.value) {
    return props.moves;
  }
  return props.moves.slice(-limit.value);
});

function formatPosition({ row, col }: { row: number; col: number }) {
  return `${row},${col}`;
}

function formatTime(value: string) {
  return new Date(value).toLocaleTimeString();
}
</script>

<style scoped>
.history {
  background: var(--surface-color);
  padding: 16px;
  border-radius: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.history-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.section-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.toggle {
  border: none;
  background: rgba(255, 255, 255, 0.08);
  color: var(--muted-color);
  padding: 6px 10px;
  border-radius: 12px;
  font-size: 12px;
  cursor: pointer;
}

.history-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

.history-table th,
.history-table td {
  text-align: left;
  padding: 6px 8px;
}

.history-table tbody tr:nth-child(even) {
  background: rgba(255, 255, 255, 0.04);
}

.empty {
  margin: 0;
  color: var(--muted-color);
}
</style>
