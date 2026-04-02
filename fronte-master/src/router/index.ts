import { createRouter, createWebHistory } from 'vue-router'; // Используем createWebHistory вместо createWebHashHistory
import { useAuthStore } from '@/store/auth.store';

const router = createRouter({
  history: createWebHistory('/web/'), // 👈 Важно! Базовый путь /web/
  routes: [
    {
      path: '/auth',
      name: 'Auth',
      component: () => import('@/views/AuthView.vue'),
      meta: { requiresGuest: true },
    },
    {
      path: '/',
      name: 'Dashboard',
      component: () => import('@/views/HomeView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/session/:id',
      name: 'Session',
      component: () => import('@/views/SessionView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
});

router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore();
  const isAuthenticated = authStore.isAuthenticated;

  if (to.meta.requiresAuth && !isAuthenticated) {
    next({ name: 'Auth' });
  } else if (to.meta.requiresGuest && isAuthenticated) {
    next({ name: 'Dashboard' });
  } else {
    next();
  }
});

export default router;