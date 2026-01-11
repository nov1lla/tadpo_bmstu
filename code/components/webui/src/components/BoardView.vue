<template>
  <div class="board" v-if="board">
    <div class="board-header" :style="axisTopStyle">
      <span class="axis-label axis-label--spacer"></span>
      <span v-for="label in topAxisLabels" :key="`col-${label}`" class="axis-label">
        {{ label }}
      </span>
    </div>
    <div class="board-body">
      <div class="axis-column" :style="axisSideStyle">
        <span v-for="label in sideAxisLabels" :key="`row-${label}`" class="axis-label">
          {{ label }}
        </span>
      </div>
      <div class="board-grid" :style="gridStyle">
        <div
          v-for="cell in cells"
          :key="cell.key"
          class="board-cell"
          :class="{ 'board-cell--dark': cell.dark }"
        ></div>
        <div class="pieces-layer" :style="piecesStyle">
          <div
            v-for="piece in pieces"
            :key="piece.id"
            class="piece"
            :class="{
              'piece--light': piece.color === 'light',
              'piece--dark': piece.color === 'dark',
              'piece--king': piece.kind === 'king'
            }"
            :style="pieceStyle(piece)"
          >
            <span class="piece-id">{{ piece.id }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
  <div v-else class="board board--empty">Board unavailable</div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { BoardState, Piece } from '../types';

const props = defineProps<{ board: BoardState | null }>();
const CELL_SIZE = 40;
const CELL_GAP = 2;
const PIECE_SIZE = 28;

const pieces = computed(() => props.board?.pieces ?? []);

const gridStyle = computed(() => {
  const size = props.board?.size ?? 8;
  return {
    gridTemplateColumns: `repeat(${size}, ${CELL_SIZE}px)`,
    gridTemplateRows: `repeat(${size}, ${CELL_SIZE}px)`
  };
});

const axisSideStyle = computed(() => {
  const size = props.board?.size ?? 8;
  return { gridTemplateRows: `repeat(${size}, ${CELL_SIZE}px)` };
});

const sideAxisLabels = computed(() => {
  const size = props.board?.size ?? 8;
  return Array.from({ length: size }, (_, idx) => idx);
});

const cells = computed(() => {
  const cellsList: Array<{ key: string; row: number; col: number; dark: boolean }> = [];
  const size = props.board?.size ?? 0;
  for (let row = 0; row < size; row += 1) {
    for (let col = 0; col < size; col += 1) {
      const key = `${row}-${col}`;
      cellsList.push({
        key,
        row,
        col,
        dark: (row + col) % 2 === 1
      });
    }
  }
  return cellsList;
});

const axisTopStyle = computed(() => {
  const size = props.board?.size ?? 8;
  return { gridTemplateColumns: `40px repeat(${size}, ${CELL_SIZE}px)` };
});

const topAxisLabels = computed(() => {
  const size = props.board?.size ?? 8;
  return Array.from({ length: size }, (_, idx) => idx);
});

const piecesStyle = computed(() => {
  const size = props.board?.size ?? 8;
  const length = size * CELL_SIZE + (size - 1) * CELL_GAP;
  return {
    width: `${length}px`,
    height: `${length}px`
  };
});

function pieceStyle(piece: Piece) {
  const x = piece.col * (CELL_SIZE + CELL_GAP) + (CELL_SIZE - PIECE_SIZE) / 2;
  const y = piece.row * (CELL_SIZE + CELL_GAP) + (CELL_SIZE - PIECE_SIZE) / 2;
  return {
    width: `${PIECE_SIZE}px`,
    height: `${PIECE_SIZE}px`,
    transform: `translate(${x}px, ${y}px)`
  };
}
</script>

<style scoped>
.board {
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: var(--surface-color);
  padding: 16px;
  border-radius: 16px;
  width: fit-content;
}

.board--empty {
  color: var(--muted-color);
}

.board-header {
  display: grid;
  gap: 2px;
  justify-items: center;
}

.axis-column {
  display: grid;
  gap: 2px;
  justify-items: center;
}

.axis-label {
  color: var(--muted-color);
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
}

.board-body {
  display: flex;
  gap: 8px;
}

.board-grid {
  display: grid;
  gap: 2px;
  position: relative;
  overflow: visible;
}

.board-cell {
  width: 40px;
  height: 40px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.08);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.board-cell--dark {
  background: rgba(0, 0, 0, 0.35);
}

.pieces-layer {
  position: absolute;
  top: 0;
  left: 0;
  pointer-events: none;
}

.piece {
  position: absolute;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 600;
  color: var(--bg-color);
  border: 2px solid transparent;
  transition: transform 0.08s linear;
}

.piece--light {
  background: #f7f5f0;
  border-color: rgba(0, 0, 0, 0.25);
}

.piece--dark {
  background: #2b3a4a;
  border-color: rgba(255, 255, 255, 0.15);
  color: var(--text-color);
}

.piece--king {
  box-shadow: 0 0 0 2px var(--accent-color);
}

.piece-id {
  opacity: 0.6;
}
</style>
