import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { api } from '@/services/api';
import { wsClient } from '@/services/ws.client';
import { useAuthStore } from './auth.store';
import { yjsService } from '@/services/yjs.provider';
import type {
  AiReview,
  FileNode,
  Session,
  WsParticipant,
  SessionEvent,
  HistoryVersion,
  LeaderboardEntry,
  UserProfile,
  DbParticipant
} from '@/services/types';
import router from '@/router';

function detectLanguage(path: string | null, fallback?: string | null) {
  const lower = (path || '').toLowerCase();
  if (lower.endsWith('.ts')) return 'typescript';
  if (lower.endsWith('.js')) return 'javascript';
  if (lower.endsWith('.py')) return 'python';
  if (lower.endsWith('.go')) return 'go';
  if (lower.endsWith('.rs')) return 'rust';
  if (lower.endsWith('.html')) return 'html';
  if (lower.endsWith('.css')) return 'css';
  if (lower.endsWith('.json')) return 'json';
  if (fallback === 'typescript') return 'typescript';
  if (fallback === 'javascript') return 'javascript';
  if (fallback === 'python') return 'python';
  if (fallback === 'go') return 'go';
  if (fallback === 'rust') return 'rust';
  return 'plaintext';
}

export const useSessionStore = defineStore('session', () => {
  const authStore = useAuthStore();

  const aiHint = ref<string | null>(null);
  const isHintLoading = ref(false);
  // Основные состояния IDE
  const openedTabs = ref<string[]>([]);
  const currentSession = ref<Session | null>(null);
  const files = ref<FileNode[]>([]);
  const participants = ref<WsParticipant[]>([]);
  const aiReviews = ref<AiReview[]>([]);

  // Состояния текущего файла
  const activeFilePath = ref<string | null>(null);
  const activeFileVersion = ref<number>(0);
  const fileContent = ref('');
  const dirty = ref(false);
  const fileCache = ref<Map<string, { content: string; version: number }>>(new Map());

  // Дополнительные данные (Лидерборд, История, События, Участники БД)
  const events = ref<SessionEvent[]>([]);
  const history = ref<HistoryVersion[]>([]);
  const leaderboard = ref<LeaderboardEntry[]>([]);
  const userProfile = ref<UserProfile | null>(null);
  const dbParticipants = ref<DbParticipant[]>([]);
  const inviteLink = ref<string>('');

  // UI Состояния
  const isConnecting = ref(false);
  const connectionError = ref<string | null>(null);
  const isFileLoading = ref(false);
  const isSaving = ref(false);
  const saveError = ref<string | null>(null);

  // Терминал
  const terminalLines = ref<string[]>(['Терминал подключен.']);
  const terminalCommand = ref('');

  let autoSaveInterval: ReturnType<typeof setInterval> | null = null;
  const wsUnsubs: Array<() => void> = [];

  // Вычисляемые свойства
  const fileTree = computed(() => [...files.value].sort((a, b) => a.path.localeCompare(b.path)));
  const activeLanguage = computed(() => detectLanguage(activeFilePath.value, currentSession.value?.language));
  const primaryFilePath = computed(() => currentSession.value?.file_name || files.value.find((file) => !file.is_dir)?.path || null);

  const isOwner = computed(() => {
    return currentSession.value?.owner_id === authStore.userId;
  });

  function clearWsListeners() {
    while (wsUnsubs.length) {
      const off = wsUnsubs.pop();
      off?.();
    }
  }

  function appendTerminal(text: string) {
    if (!text) return;
    terminalLines.value.push(text);
  }

  // --- Загрузка дополнительных данных в фоне ---
  async function fetchExtraData() {
    if (!currentSession.value) return;
    try {
      const sessionId = currentSession.value.id;
      const [eventsData, historyData, leaderboardData, profileData, participantsData] = await Promise.all([
        api.getSessionEvents(sessionId).catch(() => []),
        api.getSessionHistory(sessionId).catch(() => []),
        api.getLeaderboard(sessionId).catch(() => []),
        api.getSessionProfile(sessionId).catch(() => null),
        api.getParticipants(sessionId).catch(() => [])
      ]);
      events.value = eventsData;
      history.value = historyData;
      leaderboard.value = leaderboardData;
      dbParticipants.value = participantsData;
      if (profileData) userProfile.value = profileData;
    } catch (e) {
      console.warn('[Стор] Не удалось загрузить доп. данные:', e);
    }
  }

  // --- Подключение к сессии ---
  async function joinSession(sessionId: string) {
    leaveSession();
    isConnecting.value = true;
    connectionError.value = null;

    try {
      const [sessionData, sessionFiles, reviewsData] = await Promise.all([
        api.getSession(sessionId),
        api.getFiles(sessionId),
        api.getAiReviews(sessionId).catch(() => []),
      ]);

      currentSession.value = sessionData;
      files.value = sessionFiles;
      aiReviews.value = reviewsData.filter((review) => !review.resolved);

      setupWsListeners();
      wsClient.connect(sessionId, authStore.userId || 'anonymous', authStore.username || 'Anonymous');
      startAutoSave();

      const initialFile = sessionData.file_name || sessionFiles.find((file) => !file.is_dir)?.path || null;
      if (initialFile) {
        await setActiveFile(initialFile);
      }

      // Загружаем лидерборд, историю, профиль и участников
      void fetchExtraData();

    } catch (error: any) {
      console.error('[Session] Error joining session:', error);
      connectionError.value = error.message || 'Сессия не найдена или сервер недоступен.';
    } finally {
      isConnecting.value = false;
    }
  }

  function leaveSession() {
    stopAutoSave();
    clearWsListeners();
    wsClient.disconnect();
    currentSession.value = null;
    files.value = [];
    participants.value = [];
    aiReviews.value = [];
    events.value = [];
    history.value = [];
    leaderboard.value = [];
    userProfile.value = null;
    dbParticipants.value = [];
    inviteLink.value = '';
    activeFilePath.value = null;
    activeFileVersion.value = 0;
    fileContent.value = '';
    terminalLines.value = ['Терминал подключен.'];
    terminalCommand.value = '';
    saveError.value = null;
    dirty.value = false;
    openedTabs.value = [];
    fileCache.value.clear();
  }

  // --- Управление файлами ---
  async function setActiveFile(path: string) {
    if (!currentSession.value) return;

    if (!openedTabs.value.includes(path)) {
      openedTabs.value.push(path);
    }

    if (activeFilePath.value === path) return;

    if (activeFilePath.value) {
      fileCache.value.set(activeFilePath.value, {
        content: fileContent.value,
        version: activeFileVersion.value
      });

      if (dirty.value) {
        void saveActiveFile(true);
      }
    }

    isFileLoading.value = true;

    try {
      const cached = fileCache.value.get(path);

      if (cached) {
        fileContent.value = cached.content;
        activeFileVersion.value = cached.version;
        activeFilePath.value = path;
        isFileLoading.value = false;
        dirty.value = false;

        api.getFileContent(currentSession.value.id, path).then(data => {
          fileCache.value.set(path, { content: data.content, version: data.version });
        });
      } else {
        const data = await api.getFileContent(currentSession.value.id, path);
        fileContent.value = data.content || '';
        activeFileVersion.value = Number(data.version || 0);

        activeFilePath.value = path;
        isFileLoading.value = false;
        dirty.value = false;

        fileCache.value.set(path, { content: fileContent.value, version: activeFileVersion.value });
      }
    } catch (error) {
      console.error('[Стор] Ошибка при смене файла:', error);
      isFileLoading.value = false;
    }
  }

  function closeTab(path: string) {
    const index = openedTabs.value.indexOf(path);
    if (index === -1) return;

    openedTabs.value.splice(index, 1);

    if (activeFilePath.value === path) {
      if (openedTabs.value.length > 0) {
        const nextTab = openedTabs.value[Math.max(0, index - 1)];
        setActiveFile(nextTab);
      } else {
        activeFilePath.value = null;
        fileContent.value = '';
        activeFileVersion.value = 0;
        dirty.value = false;
      }
    }
  }

  function updateEditorContent(content: string) {
    fileContent.value = content;
    dirty.value = true;
  }

  async function saveActiveFile(force = false) {
    if (!currentSession.value || !activeFilePath.value) return;
    if (isSaving.value && !force) return; // Не спамим, если запрос уже летит
    if (!dirty.value && !force) return;

    isSaving.value = true;
    try {
      const response = await api.updateFileContent(currentSession.value.id, activeFilePath.value, {
        content: fileContent.value,
        base_version: activeFileVersion.value,
      });

      activeFileVersion.value = response.new_version;
      dirty.value = false;

      // Обновляем версию в общем списке файлов
      const f = files.value.find(f => f.path === activeFilePath.value);
      if (f) f.version = response.new_version;

    } catch (error: any) {
      const isConflict = error?.code === 409 || String(error.message).includes('409');

    if (isConflict) {
        const data = await api.getFileContent(currentSession.value.id, activeFilePath.value);
        activeFileVersion.value = data.version;
        fileContent.value = data.content;
        dirty.value = false;
        console.log('[Store] Версия выправлена на:', data.version);
    } else {
        saveError.value = error.message;
      }
    } finally {
      isSaving.value = false;
    }
  }

  async function restoreVersion(version: number) {
    if (!currentSession.value || !activeFilePath.value) return;

    const path = activeFilePath.value;
    isSaving.value = true;

    try {
      // 1. Вызываем откат. Бэкенд возвращает { content, version }
      const response = await api.restoreSessionVersion(currentSession.value.id, version);

      console.log('[Store] Восстановление успешно, контент получен:', response);

      // 2. Обновляем локальный стейт напрямую из ответа
      fileContent.value = response.content || '';
      activeFileVersion.value = response.version;
      dirty.value = false;

      // 3. Обновляем Yjs (отправляем новый текст всем коллегам)
      yjsService.applyValue(response.content);

      // 4. Синхронизируем кэш
      fileCache.value.set(path, {
        content: response.content || '',
        version: response.version
      });

      appendTerminal(`\n[СИСТЕМА] Файл ${path} откатан к версии ${version}. Новая версия: v${response.version}`);

      // Обновляем историю в боковой панели (там появится новая точка)
      void fetchExtraData();

    } catch (error: any) {
      appendTerminal(`\n[ОШИБКА ОТКАТА] ${error.message}`);
    } finally {
      isSaving.value = false;
    }
  }

  // --- Управление профилем и участниками ---
  async function updateProfile(incognito: boolean, nickname: string) {
    if (!currentSession.value) return;
    try {
      const updated = await api.updateSessionProfile(currentSession.value.id, { incognito, nickname });
      userProfile.value = updated;
      leaderboard.value = await api.getLeaderboard(currentSession.value.id);
    } catch (error) {
      console.error('[Стор] Ошибка обновления профиля:', error);
    }
  }

  async function fetchInviteLink() {
    if (!currentSession.value) return;
    try {
      const res = await api.getInviteLink(currentSession.value.id);
      inviteLink.value = res.invite_link;
    } catch (error) {
      console.error('[Стор] Ошибка получения ссылки:', error);
    }
  }

  async function inviteUser(username: string) {
    if (!currentSession.value) return;
    await api.inviteUser(currentSession.value.id, username);
    void fetchExtraData();
  }

  async function kickParticipant(userId: string) {
    if (!currentSession.value) return;
    await api.removeParticipant(currentSession.value.id, userId);
    void fetchExtraData();
  }

async function handleInviteToken(token: string) {
    console.log('[Session Store] Joining with token:', token);
    try {
        const result = await api.joinByLink(token);
        console.log('[Session Store] Join result:', result);
        
        if (result && result.joined && result.session_id) {
            // Добавляем пользователя в participants на бэкенде
            console.log('[Session Store] Successfully joined session:', result.session_id);
            return result;
        }
        throw new Error('Failed to join session: invalid response');
    } catch (error: any) {
        console.error('[Session Store] Error joining by link:', error);
        console.error('[Session Store] Error details:', error.message);
        throw error;
    }
}

  // --- Выполнение кода и терминал ---
  async function runCode() {
    if (!currentSession.value) return;

    if (!wsClient.isOpen()) {
      appendTerminal(`\n[ОШИБКА] Основной WebSocket отключен. Обновите страницу.`);
      return;
    }

    appendTerminal(`\n$ Запуск кода...`);
    wsClient.send({ type: 'run_code' });
  }

  async function execTerminal() {
    const command = terminalCommand.value.trim();
    if (!command) return;

    appendTerminal(`\n$ ${command}`);

    if (!wsClient.isOpen()) {
      appendTerminal(`[ОШИБКА] Нет связи с сервером терминала (WebSocket отключен). Обновите страницу.`);
      return;
    }

    wsClient.send({ type: 'terminal_exec', payload: { command } });
    terminalCommand.value = ''; // Сбрасываем внутри стора на всякий случай
  }

  // --- AI ---
  function dismissReview(reviewId: string) {
    aiReviews.value = aiReviews.value.filter((r) => r.id !== reviewId);
  }

  async function applyReview(reviewId: string) {
    if (!currentSession.value || !activeFilePath.value) return;
    const path = activeFilePath.value;
    isSaving.value = true;
    try {
      await api.applyAiReview(currentSession.value.id, reviewId);
      const freshData = await api.getFileContent(currentSession.value.id, path);

      fileContent.value = freshData.content;
      activeFileVersion.value = freshData.version;
      dirty.value = false;

      import('@/services/yjs.provider').then(m => m.yjsService.applyValue(freshData.content));
      fileCache.value.set(path, { content: freshData.content, version: freshData.version });

      dismissReview(reviewId);
      void fetchExtraData();
    } catch (error: any) {
      appendTerminal(`\n[AI ОШИБКА] Не удалось применить: ${error.message}`);
    } finally {
      isSaving.value = false;
    }
  }

  // ОБНОВЛЕННАЯ ФУНКЦИЯ (Теперь сохраняет в переменную)
  async function fetchHint() {
    if (!currentSession.value) return;
    isHintLoading.value = true;
    aiHint.value = null; // Очищаем старую подсказку
    try {
      const res = await api.getAiHint(currentSession.value.id);
      aiHint.value = res.hint;
    } catch (error: any) {
      aiHint.value = `Ошибка получения подсказки: ${error.message}`;
    } finally {
      isHintLoading.value = false;
    }
  }



  // --- Настройка WebSocket ---
  function setupWsListeners() {
    clearWsListeners();

    wsUnsubs.push(wsClient.on('participants', (p) => {
      participants.value = Array.isArray(p) ? p : [];
    }));

    wsUnsubs.push(wsClient.on('user_removed', (payload) => {
      console.log('[WS] Пользователь удален:', payload);

      if (payload.user_id === authStore.userId) {
        // ЕСЛИ УДАЛИЛИ МЕНЯ
        alert('Вас исключили из этой сессии.');
        leaveSession(); // Очищаем сокеты и стейт
        router.push('/'); // Уходим на главную
      } else {
        // ЕСЛИ УДАЛИЛИ ДРУГОГО
        participants.value = participants.value.filter(p => p.user_id !== payload.user_id);
        dbParticipants.value = dbParticipants.value.filter(p => p.user_id !== payload.user_id);
        appendTerminal(`\n[СИСТЕМА] Пользователь ${payload.username} исключён из сессии.`);
      }
    }));

    // 2. Когда кого-то пригласили (чтобы список в модалке обновился у всех)
    wsUnsubs.push(wsClient.on('user_invited', (payload) => {
      const exists = dbParticipants.value.some(p => p.user_id === payload.user_id);
      if (!exists) {
        dbParticipants.value = [...dbParticipants.value, {
          user_id: payload.user_id,
          username: payload.username,
          joined_at: new Date().toISOString()
        }];
      }
    }));

    wsUnsubs.push(wsClient.on('file_created', (payload) => {
      console.log("file create: ", payload);
      if (!payload || !payload.path) return;

      // Добавляем только если такого пути еще нет в списке
      const exists = files.value.some(f => f.path === payload.path);
      if (!exists) {
        files.value = [...files.value, payload];
      }
    }));

    wsUnsubs.push(wsClient.on('file_updated', (payload) => {
      const index = files.value.findIndex(f => f.path === payload.path);
      if (index !== -1) {
        files.value[index] = { ...files.value[index], ...payload };
        files.value = [...files.value];
      }
    }));

    wsUnsubs.push(wsClient.on('file_deleted', (file) => {
      files.value = files.value.filter((item) => item.path !== file.path);
      if (activeFilePath.value === file.path) {
        activeFilePath.value = null;
      }
      void fetchExtraData();
    }));

    wsUnsubs.push(wsClient.on('code_output', (payload) => {
      if (payload?.output) appendTerminal(payload.output);
      if (payload?.error && payload?.error_msg) appendTerminal(`[ОШИБКА КОДА] ${payload.error_msg}`);
    }));

    wsUnsubs.push(wsClient.on('terminal_output', (payload) => {
      if (payload?.output) appendTerminal(payload.output);
      if (payload?.error && payload?.error_msg) appendTerminal(`[ОШИБКА ТЕРМИНАЛА] ${payload.error_msg}`);
    }));

    wsUnsubs.push(wsClient.on('ai_suggestion', async (suggestion) => {
      if (suggestion?.message) appendTerminal(`\n[AI] ${suggestion.message}`);
      if (currentSession.value) {
        try {
          const freshReviews = await api.getAiReviews(currentSession.value.id);
          aiReviews.value = freshReviews.filter((r) => !r.resolved);
        } catch (e) {}
      }
    }));

    wsUnsubs.push(wsClient.on('error', (payload) => {
      appendTerminal(`\n[СЕРВЕР ОШИБКА] ${payload?.message || 'Неизвестная ошибка'}`);
    }));
  }

  function startAutoSave() {
    stopAutoSave();
    autoSaveInterval = setInterval(() => {
      void saveActiveFile();
    }, 5000);
  }

  function stopAutoSave() {
    if (autoSaveInterval) {
      clearInterval(autoSaveInterval);
      autoSaveInterval = null;
    }
  }

  // Обязательно экспортируем всё, что нужно в компонентах
  return {
    currentSession,
    files,
    fileTree,
    participants,
    dbParticipants,
    inviteLink,
    aiReviews,
    activeFilePath,
    activeFileVersion,
    activeLanguage,
    fileContent,
    events,
    history,
    leaderboard,
    userProfile,
    isConnecting,
    connectionError,
    isFileLoading,
    isSaving,
    saveError,
    terminalLines,
    terminalCommand,
    dirty,
    openedTabs,
    isOwner,
    joinSession,
    leaveSession,
    setActiveFile,
    updateEditorContent,
    saveActiveFile,
    restoreVersion,
    updateProfile,
    runCode,
    execTerminal,
    dismissReview,
    applyReview,
    closeTab,
    fetchHint,
    fetchInviteLink,
    inviteUser,
    kickParticipant,
    handleInviteToken,
    aiHint,
    isHintLoading
  };
});