import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { api } from '@/services/api';
import type { AuthRequest } from '@/services/types';

export const useAuthStore = defineStore('auth', () => {
    // Состояние
    const token = ref<string | null>(localStorage.getItem('token') || null);
    const userId = ref<string | null>(localStorage.getItem('user_id') || null);
    const username = ref<string | null>(localStorage.getItem('username') || null);

    // Геттеры
    const isAuthenticated = computed(() => !!token.value);

    // Действия (Actions)
    function setAuthData(newToken: string, newUserId: string, newUsername: string) {
        token.value = newToken;
        userId.value = newUserId;
        username.value = newUsername;

        localStorage.setItem('token', newToken);
        localStorage.setItem('user_id', newUserId);
        localStorage.setItem('username', newUsername);

        api.setToken(newToken); // Передаем токен в API клиент
    }

    async function login(credentials: AuthRequest) {
        const response = await api.login(credentials);
        setAuthData(response.token, response.user_id, response.username);
    }

    async function register(credentials: AuthRequest) {
        const response = await api.register(credentials);
        setAuthData(response.token, response.user_id, response.username);
    }

    function logout() {
        token.value = null;
        userId.value = null;
        username.value = null;

        localStorage.removeItem('token');
        localStorage.removeItem('user_id');
        localStorage.removeItem('username');

        api.clearToken();
    }

    // Инициализация при старте приложения (вызывается в main.ts)
    function init() {
        if (token.value) {
            api.setToken(token.value);
        }
    }

    return {
        token,
        userId,
        username,
        isAuthenticated,
        login,
        register,
        logout,
        init
    };
});