<template>
  <div class="auth">
    <section class="panel">
      <header class="panel-header">
        <h2>Account</h2>
        <p class="hint">Sign in or create a new user. Only login + password are required.</p>
      </header>

      <div class="tabs" role="tablist" aria-label="Authentication tabs">
        <button
          type="button"
          class="tab"
          :class="{ active: mode === 'login' }"
          role="tab"
          :aria-selected="mode === 'login'"
          @click="mode = 'login'"
        >
          Login
        </button>
        <button
          type="button"
          class="tab"
          :class="{ active: mode === 'register' }"
          role="tab"
          :aria-selected="mode === 'register'"
          @click="mode = 'register'"
        >
          Register
        </button>
      </div>

      <form class="stack" @submit.prevent="submit">
        <input v-model="login" placeholder="Login" autocomplete="username" required />
        <input
          v-model="password"
          type="password"
          placeholder="Password"
          autocomplete="current-password"
          required
        />

        <button type="submit" :disabled="loading || !canSubmit">
          {{ loading ? 'Working…' : mode === 'login' ? 'Login' : 'Register' }}
        </button>
      </form>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useGameStore } from '../viewmodels/useGameViewModel';

const router = useRouter();
const route = useRoute();
const store = useGameStore();

const mode = ref<'login' | 'register'>('login');
const login = ref('');
const password = ref('');
const loading = computed(() => store.loading.value);
const canSubmit = computed(() => login.value.trim().length > 0 && password.value.trim().length > 0);

async function submit() {
  const loginValue = login.value.trim();
  const passwordValue = password.value;
  if (!loginValue || !passwordValue) {
    return;
  }

  if (mode.value === 'register') {
    await store.registerUser(loginValue, passwordValue);
  } else {
    await store.loginUser(loginValue, passwordValue);
  }

  if (store.user.value) {
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/';
    await router.push(redirect);
  }
}
</script>

<style scoped>
.auth {
  max-width: 520px;
}

.panel {
  background: var(--surface-color);
  border-radius: 16px;
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.panel-header h2 {
  margin: 0;
  font-size: 18px;
}

.hint {
  margin: 6px 0 0;
  color: var(--muted-color);
  font-size: 13px;
}

.tabs {
  display: flex;
  gap: 8px;
  padding: 4px;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 14px;
}

.tab {
  flex: 1;
  padding: 10px 12px;
  border-radius: 12px;
  border: none;
  cursor: pointer;
  background: transparent;
  color: var(--muted-color);
  font-weight: 600;
}

.tab.active {
  background: rgba(240, 180, 41, 0.2);
  color: var(--text-color);
}

.stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

input,
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
  font-weight: 700;
  cursor: pointer;
}

button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>

