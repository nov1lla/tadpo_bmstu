<template>
  <div class="block">
    <h3 class="title">Player</h3>
    <template v-if="user">
      <p class="stat"><span class="label">Name</span><span>{{ user.name }}</span></p>
      <p class="stat"><span class="label">Rating</span><span>{{ user.rating }}</span></p>
      <p class="stat"><span class="label">Win streak</span><span>{{ user.winStreak }}</span></p>
      <p class="stat"><span class="label">Last game</span><span>{{ user.lastGameId ?? '—' }}</span></p>
    </template>
    <p v-else class="placeholder">Create a player to begin</p>
  </div>
</template>

<script setup lang="ts">
import type { User } from '../types';

defineProps<{ user: User | null }>();
</script>

<style scoped>
.block {
  background: var(--surface-color);
  border-radius: 16px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.stat {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  margin: 0;
  font-size: 14px;
}

.label {
  color: var(--muted-color);
}

.placeholder {
  margin: 0;
  color: var(--muted-color);
  font-size: 14px;
}
</style>
