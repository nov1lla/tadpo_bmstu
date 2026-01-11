<template>
  <div class="history">
    <h3 class="section-title">Move history</h3>
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
        <tr v-for="move in moves" :key="move.id">
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
import type { Move } from '../types';

defineProps<{ moves: Move[] }>();

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

.section-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
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
