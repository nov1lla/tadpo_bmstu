<template>
  <div class="app">
    <header class="app-header">
      <div class="title-block">
        <h1>My Checkers</h1>
      </div>
      <nav class="nav">
        <RouterLink to="/" class="nav-link" active-class="nav-link--active">Menu</RouterLink>
        <RouterLink to="/game" class="nav-link" active-class="nav-link--active">Play</RouterLink>
        <RouterLink to="/history" class="nav-link" active-class="nav-link--active">History</RouterLink>
      </nav>
      <button class="refresh" @click="refreshAll" :disabled="loading">
        Refresh state
      </button>
    </header>

    <main class="app-main">
      <RouterView />
    </main>

    <transition name="toast">
      <div v-if="toast.visible" class="toast" role="alert">
        <span class="toast-message">{{ toast.message }}</span>
        <button class="toast-close" type="button" @click="dismissToast">×</button>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, watch } from 'vue';
import { RouterLink, RouterView } from 'vue-router';
import { useGameStore } from './viewmodels/useGameViewModel';

const store = useGameStore();
const loading = computed(() => store.loading.value);
const error = computed(() => store.error.value);
const toast = reactive({ visible: false, message: '' });
let toastTimer: ReturnType<typeof setTimeout> | null = null;

onMounted(() => {
  store.restore();
});

async function refreshAll() {
  await store.refreshUser();
  await store.refreshGame();
}

function dismissToast() {
  toast.visible = false;
  toast.message = '';
  if (toastTimer) {
    clearTimeout(toastTimer);
    toastTimer = null;
  }
  store.clearError();
}

watch(error, (value) => {
  if (!value) {
    return;
  }
  toast.message = value;
  toast.visible = true;
  if (toastTimer) {
    clearTimeout(toastTimer);
  }
  toastTimer = setTimeout(() => {
    dismissToast();
  }, 4000);
});
</script>

<style scoped>
.app {
  min-height: 100vh;
  padding: 24px 32px 48px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.app-header {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 24px;
  align-items: center;
}

.title-block h1 {
  margin: 0;
  font-size: 28px;
  letter-spacing: 0.02em;
}


.nav {
  display: inline-flex;
  gap: 12px;
  justify-content: center;
}

.nav-link {
  padding: 8px 16px;
  border-radius: 20px;
  text-decoration: none;
  color: var(--muted-color);
  background: rgba(255, 255, 255, 0.04);
  transition: background 0.2s ease, color 0.2s ease;
}

.nav-link--active,
.nav-link:hover {
  color: var(--text-color);
  background: rgba(240, 180, 41, 0.2);
}

.refresh {
  padding: 8px 16px;
  border-radius: 12px;
  border: none;
  background: rgba(240, 180, 41, 0.2);
  color: var(--accent-color);
  font-weight: 600;
}

.refresh:disabled {
  opacity: 0.6;
}

.app-main {
  flex: 1;
}

.toast {
  position: fixed;
  right: 24px;
  bottom: 24px;
  background: rgba(20, 20, 20, 0.95);
  color: #ffe3d9;
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid rgba(255, 140, 120, 0.4);
  display: flex;
  gap: 12px;
  align-items: center;
  max-width: 360px;
  box-shadow: 0 12px 24px rgba(0, 0, 0, 0.25);
}

.toast-message {
  font-size: 13px;
  line-height: 1.3;
}

.toast-close {
  border: none;
  background: transparent;
  color: #ffb4a3;
  font-size: 18px;
  cursor: pointer;
}

.toast-enter-active,
.toast-leave-active {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
