import { createRouter, createWebHistory } from 'vue-router';

const MainMenuPage = () => import('./views/MainMenuPage.vue');
const GamePlayPage = () => import('./views/GamePlayPage.vue');
const GameHistoryPage = () => import('./views/GameHistoryPage.vue');
const AuthPage = () => import('./views/AuthPage.vue');

const USER_STORAGE_KEY = 'ppo:user-id';

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: MainMenuPage },
    { path: '/auth', name: 'auth', component: AuthPage },
    { path: '/game', name: 'game', component: GamePlayPage },
    { path: '/history', name: 'history', component: GameHistoryPage }
  ]
});

router.beforeEach((to) => {
  if (to.name === 'auth') {
    return true;
  }
  if (typeof window === 'undefined') {
    return true;
  }
  const hasUser = !!localStorage.getItem(USER_STORAGE_KEY);
  if (!hasUser) {
    return { name: 'auth', query: { redirect: to.fullPath } };
  }
  return true;
});
