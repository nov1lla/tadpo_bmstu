<template>
  <div class="block">
    <h3 class="title">Game</h3>
    <template v-if="game">
      <p class="stat"><span class="label">Game</span><span>{{ gameLabel }}</span></p>
      <p class="stat"><span class="label">Status</span><span>{{ game.status }}</span></p>
      <p class="stat"><span class="label">You play</span><span>{{ game.playerColor }}</span></p>
      <p class="stat"><span class="label">First move</span><span>{{ game.isPlayerFirst ? 'yes' : 'no' }}</span></p>
    </template>
    <p v-else class="placeholder">Start a match to see details</p>
    <div v-if="hasActions" class="actions">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, useSlots } from 'vue';
import type { Game } from '../types';

const props = defineProps<{ game: Game | null }>();
const slots = useSlots();
const hasActions = computed(() => !!slots.default);

const gameLabel = computed(() => {
  if (!props.game) {
    return '—';
  }
  if (props.game.number !== undefined && props.game.number !== null) {
    return `#${props.game.number}`;
  }
  if (props.game.startedAt) {
    const parsed = new Date(props.game.startedAt);
    if (!Number.isNaN(parsed.getTime())) {
      return parsed.toLocaleString();
    }
  }
  return props.game.id;
});
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

.actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
</style>
