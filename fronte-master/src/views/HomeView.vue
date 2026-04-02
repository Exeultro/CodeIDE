<template>
  <div class="home-layout container">
    <header class="home-header">
      <div class="brand">CodeSync <span style="color: var(--primary)">AI</span></div>
      <div class="user-menu">
        <span class="text-muted">Привет, <b>{{ authStore.username }}</b></span>
        <BaseButton variant="outline" @click="logout" style="margin-left: 16px;">Выйти</BaseButton>
      </div>
    </header>

    <main class="dashboard-grid">
      <section class="create-section">
        <BaseCard title="Создать новую сессию">
          <form @submit.prevent="createSession">
            <BaseInput v-model="newSession.name" label="Название проекта" placeholder="Например: Хакатон AI" />
            <BaseInput v-model="newSession.file_name" label="Главный файл" placeholder="main.py" class="mt-4" />
            <div class="mt-4">
              <BaseSelect v-model="newSession.language" label="Язык программирования" :options="langOptions" />
            </div>
            <BaseButton class="mt-4 w-full" type="submit" :disabled="!isFormValid || isCreating">
              {{ isCreating ? 'Создание...' : 'Запустить среду' }}
            </BaseButton>
            <p v-if="errorMessage" class="error-text">{{ errorMessage }}</p>
          </form>
        </BaseCard>
      </section>

      <section class="sessions-section">
        <h2 class="mb-4">Мои сессии</h2>
        <div v-if="sessions.length === 0" class="empty-state">
          У вас пока нет активных сессий. Создайте первую слева.
        </div>
        <div v-else class="sessions-list">
          <BaseCard v-for="session in sessions" :key="session.id" class="session-card">
            <div class="session-info">
              <h3>{{ session.name }}</h3>
              <p class="text-muted">Язык: {{ session.language }} | Файл: {{ session.file_name }}</p>
            </div>
            <BaseButton @click="goToSession(session.id)">Войти</BaseButton>
          </BaseCard>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/store/auth.store';
import { api } from '@/services/api';
import type { Session } from '@/services/types';

import BaseCard from '@/base/BaseCard.vue';
import BaseInput from '@/base/BaseInput.vue';
import BaseButton from '@/base/BaseButton.vue';
import BaseSelect from '@/base/gui/BaseSelect.vue';

const router = useRouter();
const authStore = useAuthStore();

const sessions = ref<Session[]>([]);
const isCreating = ref(false);
const errorMessage = ref('');

const newSession = ref({
  name: '',
  file_name: 'main.py',
  language: 'python',
});

const langOptions = [
  { label: 'Python', value: 'python' },
  { label: 'JavaScript', value: 'javascript' },
  { label: 'TypeScript', value: 'typescript' },
  { label: 'Go', value: 'go' },
  { label: 'Rust', value: 'rust' },
];

const isFormValid = computed(() => !!(newSession.value.name && newSession.value.file_name && newSession.value.language));

async function loadSessions() {
  try {
    sessions.value = await api.getMySessions();
  } catch (error: any) {
    errorMessage.value = error.message || 'Ошибка загрузки сессий';
  }
}

onMounted(async () => {
  await loadSessions();
});

async function createSession() {
  if (!isFormValid.value) return;
  isCreating.value = true;
  errorMessage.value = '';
  try {
    const created = await api.createSession({
      name: newSession.value.name,
      file_name: newSession.value.file_name,
      language: newSession.value.language,
    });
    router.push(`/session/${created.id}`);
  } catch (error: any) {
    errorMessage.value = error.message || 'Ошибка создания сессии';
  } finally {
    isCreating.value = false;
  }
}

function goToSession(id: string) {
  router.push(`/session/${id}`);
}

function logout() {
  authStore.logout();
  router.push('/auth');
}
</script>

<style scoped>
.home-layout { padding-top: 32px; padding-bottom: 64px; overflow: auto; height: 100vh; box-sizing: border-box; }
.home-header { display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid var(--border-color); padding-bottom: 24px; margin-bottom: 32px; }
.brand { font-size: 24px; font-weight: bold; font-family: 'JetBrains Mono', monospace; }
.dashboard-grid { display: grid; grid-template-columns: 350px 1fr; gap: 32px; align-items: start; }
.mb-4 { margin-bottom: 16px; }
.w-full { width: 100%; }
.sessions-list { display: flex; flex-direction: column; gap: 16px; }
.session-card { display: flex; justify-content: space-between; align-items: center; padding: 16px 24px; border-left: 4px solid var(--primary); transition: transform 0.2s; }
.session-card:hover { transform: translateY(-2px); box-shadow: var(--shadow-md); border-color: var(--primary-hover); }
.session-info h3 { margin: 0 0 4px 0; font-size: 18px; }
.session-info p { margin: 0; font-size: 13px; font-family: 'JetBrains Mono', monospace; }
.empty-state { padding: 48px; text-align: center; color: var(--text-muted); border: 1px dashed var(--border-color); border-radius: var(--radius-md); }
.error-text { color: var(--primary); margin-top: 12px; font-size: 14px; }
@media (max-width: 768px) { .dashboard-grid { grid-template-columns: 1fr; } }
</style>
