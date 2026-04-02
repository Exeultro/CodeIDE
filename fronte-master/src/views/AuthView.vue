<template>
  <div class="auth-layout">
    <div class="auth-container">
      <div class="brand-logo">
        <span class="logo-icon">&lt;/&gt;</span>
        <h1>Collab <span>IDE</span></h1>
      </div>

      <BaseCard :title="isLogin ? 'Вход в систему' : 'Регистрация'" class="auth-card">
        <form @submit.prevent="handleSubmit">
          <BaseInput
              v-model="form.username"
              label="Имя пользователя"
              placeholder="Например: alice_dev"
              :error="errorMessage"
          />

          <BaseInput
              v-model="form.password"
              type="password"
              label="Пароль"
              placeholder="••••••••"
              class="mt-4"
          />

          <BaseButton class="w-full mt-4 submit-btn" type="submit" :disabled="isLoading">
            {{ isLoading ? 'Загрузка...' : (isLogin ? 'Войти' : 'Зарегистрироваться') }}
          </BaseButton>
        </form>

        <div class="auth-footer mt-4">
          <span class="text-muted">
            {{ isLogin ? 'Нет аккаунта?' : 'Уже есть аккаунт?' }}
          </span>
          <a href="#" @click.prevent="toggleMode">
            {{ isLogin ? 'Создать' : 'Войти' }}
          </a>
        </div>
      </BaseCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '@/store/auth.store';
import BaseCard from '@/base/BaseCard.vue';
import BaseInput from '@/base/BaseInput.vue';
import BaseButton from '@/base/BaseButton.vue';

const router = useRouter();
const authStore = useAuthStore();

const isLogin = ref(true);
const isLoading = ref(false);
const errorMessage = ref('');

const form = reactive({
  username: '',
  password: ''
});

function toggleMode() {
  isLogin.value = !isLogin.value;
  errorMessage.value = '';
}

async function handleSubmit() {
  if (!form.username || !form.password) {
    errorMessage.value = 'Заполните все поля';
    return;
  }

  isLoading.value = true;
  errorMessage.value = '';

  try {
    if (isLogin.value) {
      await authStore.login({ username: form.username, password: form.password });
    } else {
      await authStore.register({ username: form.username, password: form.password });
    }
    router.push('/'); // Переход на дашборд при успехе
  } catch (error: any) {
    errorMessage.value = error.message || 'Ошибка авторизации';
  } finally {
    isLoading.value = false;
  }
}
</script>

<style scoped>
.auth-layout {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: radial-gradient(circle at center, var(--surface-color) 0%, var(--bg-color) 100%);
}

.auth-container {
  width: 100%;
  max-width: 400px;
  padding: 24px;
}

.brand-logo {
  text-align: center;
  margin-bottom: 32px;
}

.brand-logo h1 {
  font-size: 28px;
  margin: 8px 0 0;
}

.brand-logo span {
  color: var(--primary);
}

.logo-icon {
  font-size: 48px;
  color: var(--primary);
  font-family: 'JetBrains Mono', monospace;
  font-weight: bold;
  text-shadow: 0 0 15px var(--primary-light);
}

.auth-card {
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.8), 0 0 0 1px var(--border-color);
}

.w-full { width: 100%; }
.submit-btn { padding: 12px; font-size: 16px; }
.auth-footer { text-align: center; font-size: 14px; }
.auth-footer a { margin-left: 8px; font-weight: 600; }
</style>