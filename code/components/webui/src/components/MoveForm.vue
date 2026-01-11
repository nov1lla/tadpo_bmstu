<template>
  <form class="move-form" @submit.prevent="handleSubmit">
    <h3 class="section-title">New Move</h3>
    <div class="fields">
      <div class="field-group" v-if="pieceOptions.length">
        <label>Piece</label>
        <select v-model="selectedPieceId" @change="applySelectedPiece">
          <option v-for="piece in pieceOptions" :key="piece.id" :value="piece.id">
            {{ piece.id }} · ({{ piece.row }},{{ piece.col }})
          </option>
        </select>
      </div>
      <div class="field-group">
        <label>Start position</label>
        <div class="field-row">
          <input type="number" min="0" max="7" v-model.number="form.start.row" placeholder="row" />
          <input type="number" min="0" max="7" v-model.number="form.start.col" placeholder="col" />
        </div>
      </div>
      <div class="field-group">
        <label>Path</label>
        <div class="steps">
          <div class="field-row" v-for="(step, index) in form.steps" :key="index">
            <input type="number" min="0" max="7" v-model.number="step.row" placeholder="row" />
            <input type="number" min="0" max="7" v-model.number="step.col" placeholder="col" />
            <button type="button" class="icon-button" @click="removeStep(index)" :disabled="form.steps.length === 1">
              &minus;
            </button>
          </div>
          <button type="button" class="icon-button" @click="addStep">+</button>
        </div>
      </div>
      <div class="field-row toggles">
        <label class="checkbox">
          <input type="checkbox" v-model="form.autoOpponent" />
          <span>Ask opponent immediately</span>
        </label>
        <select v-model="form.difficulty">
          <option value="easy">easy</option>
          <option value="medium">medium</option>
          <option value="hard">hard</option>
        </select>
      </div>
    </div>
    <button type="submit" class="submit" :disabled="disabled">
      {{ disabled ? 'Working…' : 'Submit move' }}
    </button>
  </form>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import type { Difficulty, PositionInput } from '../services/api';

interface PieceOption {
  id: string;
  row: number;
  col: number;
}

const props = defineProps<{
  disabled?: boolean;
  defaultDifficulty?: Difficulty;
  autoOpponent?: boolean;
  pieces?: PieceOption[];
}>();

const emit = defineEmits<{
  (e: 'submit', payload: { start: PositionInput; steps: PositionInput[]; auto: boolean; difficulty: Difficulty }): void;
}>();

const form = reactive({
  start: { row: 0, col: 0 },
  steps: [{ row: 0, col: 0 }],
  difficulty: props.defaultDifficulty ?? 'medium',
  autoOpponent: props.autoOpponent ?? true
});

const pieceOptions = computed(() => props.pieces ?? []);
const selectedPieceId = ref('');

function addStep() {
  form.steps.push({ row: 0, col: 0 });
}

function removeStep(index: number) {
  if (form.steps.length === 1) {
    return;
  }
  form.steps.splice(index, 1);
}

function applySelectedPiece() {
  const piece = pieceOptions.value.find((item) => item.id === selectedPieceId.value);
  if (!piece) {
    return;
  }
  form.start.row = piece.row;
  form.start.col = piece.col;
}

function handleSubmit() {
  if (!Number.isInteger(form.start.row) || !Number.isInteger(form.start.col)) {
    return;
  }
  const sanitizedSteps = form.steps
    .filter((step) => Number.isInteger(step.row) && Number.isInteger(step.col))
    .map((step) => ({ row: Number(step.row), col: Number(step.col) }));

  emit('submit', {
    start: { row: Number(form.start.row), col: Number(form.start.col) },
    steps: sanitizedSteps,
    auto: form.autoOpponent,
    difficulty: form.difficulty
  });
}

watch(
  () => pieceOptions.value,
  (options) => {
    if (!options.length) {
      selectedPieceId.value = '';
      return;
    }
    if (!selectedPieceId.value || !options.find((option) => option.id === selectedPieceId.value)) {
      selectedPieceId.value = options[0].id;
      applySelectedPiece();
    }
  },
  { immediate: true }
);

onMounted(() => {
  if (pieceOptions.value.length > 0) {
    selectedPieceId.value = pieceOptions.value[0].id;
    applySelectedPiece();
  }
});
</script>

<style scoped>
.move-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: var(--surface-color);
  padding: 16px;
  border-radius: 16px;
  min-width: 260px;
}

.section-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.fields {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field-row {
  display: flex;
  gap: 8px;
  align-items: center;
}

.field-row input,
.field-row select {
  width: 100%;
  padding: 8px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  color: var(--text-color);
}

.steps {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.icon-button {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: none;
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-color);
  cursor: pointer;
}

.icon-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.checkbox {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--muted-color);
  font-size: 14px;
}

.toggles {
  justify-content: space-between;
}

.submit {
  padding: 10px 16px;
  background: var(--accent-color);
  border: none;
  border-radius: 12px;
  color: #1a1a1a;
  font-weight: 600;
  cursor: pointer;
}

.submit:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
