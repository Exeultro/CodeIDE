<template>
  <div class="ide-layout" v-if="!sessionStore.isConnecting && !sessionStore.connectionError">
    <!-- 1. ХЕДЕР (IdeTopBar) -->
    <IdeTopBar @go-home="goHome" @open-participants="showParticipants = true" />

    <div class="ide-body">
      <!-- 2. ACTIVITY BAR (Иконки слева) -->
      <IdeActivityBar
          :activeLeft="showLeftPanel ? 'explorer' : 'closed'"
          :activeRight="activeRightPanel"
          :aiCount="sessionStore.aiReviews.length"
          @select="handlePanelSelect"
      />

      <!-- 3. ЛЕВЫЙ САЙДБАР (Файлы) -->
      <div class="ide-sidebar" v-show="showLeftPanel" :style="{ width: leftWidth + 'px' }">
        <IdeSidebar />
      </div>

      <!-- Ресайзер Лево -->
      <div class="resizer resizer-x" v-show="showLeftPanel" @mousedown.prevent="startResizeLeft"></div>

      <!-- 4. ЦЕНТРАЛЬНАЯ ОБЛАСТЬ (ТАБЫ + РЕДАКТОР + ТЕРМИНАЛ) -->
      <div class="ide-main">

        <!-- ============= ТАБЫ (ВЕРНУЛИ!) ============= -->
        <IdeTabs class="tabs-container" />

        <div class="editor-wrapper">
          <CodeEditor
              v-if="sessionStore.activeFilePath"
              :key="sessionStore.activeFilePath"
              :initialContent="sessionStore.fileContent"
              :language="sessionStore.activeLanguage"
          />
          <div v-else class="empty-editor">
            Выберите или создайте файл в Explorer
          </div>
        </div>

        <!-- Ресайзер Терминала -->
        <div class="resizer resizer-y" v-show="showTerminal" @mousedown.prevent="startResizeTerm"></div>

        <IdeTerminal
            v-show="showTerminal"
            class="terminal-container"
            :style="{ height: termHeight + 'px' }"
        />
      </div>

      <!-- Ресайзер Право -->
      <div class="resizer resizer-x" v-show="activeRightPanel !== 'closed'" @mousedown.prevent="startResizeRight"></div>

      <!-- 5. ПРАВЫЙ САЙДБАР (AI / История / Лидерборд) -->
      <div class="ide-sidebar right-sidebar" v-show="activeRightPanel !== 'closed'" :style="{ width: rightWidth + 'px' }">
        <KeepAlive>
          <component :is="rightPanelComponent" />
        </KeepAlive>
      </div>

    </div>

    <!-- Модалка участников -->
    <IdeParticipantsModal v-if="showParticipants" @close="showParticipants = false" />
  </div>

  <!-- В самом низу шаблона -->
  <div v-else class="loader-screen">
    <div class="error-container">
      <!-- Иконка состояния -->
      <div class="error-icon" :class="{ 'is-loading': sessionStore.isConnecting }">
        <svg v-if="sessionStore.isConnecting" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
          <path d="M12 2v4m0 12v4M4.93 4.93l2.83 2.83m8.48 8.48l2.83 2.83M2 12h4m12 0h4M4.93 19.07l2.83-2.83m8.48-8.48l2.83-2.83"/>
        </svg>
        <svg v-else width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
      </div>

      <!-- Текстовый блок -->
      <h2 class="error-title">
        {{ sessionStore.isConnecting ? 'Подключение к сессии' : 'Ошибка соединения' }}
      </h2>

      <p class="error-text">
        {{ sessionStore.isConnecting ? 'Инициализируем рабочее окружение и WebSocket...' : sessionStore.connectionError }}
      </p>

      <!-- Кнопка возврата (появляется только при ошибке) -->
      <button
          v-if="!sessionStore.isConnecting"
          class="btn btn-primary error-btn"
          @click="goHome"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="margin-right: 8px;">
          <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/>
        </svg>
        Вернуться в Dashboard
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useSessionStore } from '@/store/session.store';

import IdeTopBar from '@/components/ide/IdeTopBar.vue';
import IdeActivityBar from '@/components/ide/IdeActivityBar.vue';
import IdeTabs from '@/components/ide/IdeTabs.vue'; // <-- ИМПОРТ ТАБОВ
import CodeEditor from '@/components/editor/CodeEditor.vue';
import IdeTerminal from '@/components/ide/IdeTerminal.vue';
import IdeParticipantsModal from '@/components/ide/IdeParticipantsModal.vue';

// Панели
import IdeSidebar from '@/components/ide/IdeSidebar.vue';
import IdeHistoryPanel from '@/components/ide/IdeHistoryPanel.vue';
import IdeLeaderboardPanel from '@/components/ide/IdeLeaderboardPanel.vue';
import IdeAiPanel from '@/components/ide/IdeAiPanel.vue';

const route = useRoute();
const router = useRouter();
const sessionStore = useSessionStore();

// Состояние
const showLeftPanel = ref(true);
const activeRightPanel = ref('closed');
const showTerminal = ref(true);
const showParticipants = ref(false);

// Размеры
const leftWidth = ref(260);
const rightWidth = ref(300);
const termHeight = ref(200);

const rightPanelComponent = computed(() => {
  switch (activeRightPanel.value) {
    case 'history': return IdeHistoryPanel;
    case 'leaderboard': return IdeLeaderboardPanel;
    case 'ai': return IdeAiPanel;
    default: return null;
  }
});

function handlePanelSelect(panel: string) {
  if (panel === 'explorer') {
    showLeftPanel.value = !showLeftPanel.value;
  } else {
    activeRightPanel.value = (activeRightPanel.value === panel) ? 'closed' : panel;
  }
}

// РЕСАЙЗЕРЫ
function startResizeLeft() {
  const onMouseMove = (e: MouseEvent) => {
    let newWidth = e.clientX - 48;
    if (newWidth < 150) newWidth = 150;
    if (newWidth > 600) newWidth = 600;
    leftWidth.value = newWidth;
  };
  const onMouseUp = () => {
    document.removeEventListener('mousemove', onMouseMove);
    document.removeEventListener('mouseup', onMouseUp);
  };
  document.addEventListener('mousemove', onMouseMove);
  document.addEventListener('mouseup', onMouseUp);
}

function startResizeRight() {
  const onMouseMove = (e: MouseEvent) => {
    let newWidth = document.body.clientWidth - e.clientX;
    if (newWidth < 200) newWidth = 200;
    if (newWidth > 800) newWidth = 800;
    rightWidth.value = newWidth;
  };
  const onMouseUp = () => {
    document.removeEventListener('mousemove', onMouseMove);
    document.removeEventListener('mouseup', onMouseUp);
  };
  document.addEventListener('mousemove', onMouseMove);
  document.addEventListener('mouseup', onMouseUp);
}

function startResizeTerm() {
  const onMouseMove = (e: MouseEvent) => {
    let newHeight = document.body.clientHeight - e.clientY;
    if (newHeight < 60) newHeight = 60;
    if (newHeight > 600) newHeight = 600;
    termHeight.value = newHeight;
  };
  const onMouseUp = () => {
    document.removeEventListener('mousemove', onMouseMove);
    document.removeEventListener('mouseup', onMouseUp);
  };
  document.addEventListener('mousemove', onMouseMove);
  document.addEventListener('mouseup', onMouseUp);
}

onMounted(async () => {
    // Получаем параметры из URL
    const sessionIdFromUrl = route.params.id as string;
    const inviteToken = route.query.invite as string | undefined;
    
    console.log('🔍 SessionView mounted');
    console.log('  Full path:', window.location.pathname);
    console.log('  Route path:', route.path);
    console.log('  Session ID from URL:', sessionIdFromUrl);
    console.log('  Invite token:', inviteToken);
    
    // Если есть invite token, сначала присоединяемся
    if (inviteToken) {
        console.log('📨 Joining by invite token:', inviteToken);
        try {
            const result = await sessionStore.handleInviteToken(inviteToken);
            console.log('✅ Join result:', result);
            
            if (result && result.session_id) {
                // Успешно присоединились, переходим на правильную сессию
                const correctSessionId = result.session_id;
                console.log('🔄 Redirecting to session:', correctSessionId);
                // Убираем invite из URL, но оставляем sessionId
                await router.replace({
                    path: `/session/${correctSessionId}`,
                    query: {} // Убираем invite параметр
                });
                // Подключаемся к сессии
                await sessionStore.joinSession(correctSessionId);
                return;
            }
        } catch (error) {
            console.error('❌ Failed to join by invite:', error);
            alert('Не удалось присоединиться к сессии. Возможно, ссылка устарела.');
            router.push('/');
            return;
        }
    }
    
    // Обычное подключение к сессии (без invite)
    if (sessionIdFromUrl) {
        console.log('🔌 Connecting to session:', sessionIdFromUrl);
        await sessionStore.joinSession(sessionIdFromUrl);
    } else {
        console.error('❌ No session ID provided');
        router.push('/');
    }
});

onUnmounted(() => {
  sessionStore.leaveSession();
});

function goHome() {
  router.push('/');
}
</script>

<style scoped>
.ide-layout { display: flex; flex-direction: column; height: 100vh; background-color: var(--bg-color); color: var(--text-main); overflow: hidden; }
.ide-body { display: flex; flex: 1; overflow: hidden; position: relative; }

/* Сайдбары */
.ide-sidebar {
  background: var(--bg-color); flex-shrink: 0; display: flex; flex-direction: column; overflow: hidden;
}
.right-sidebar { background: var(--surface-color); }

/* Главная область */
.ide-main {
  flex: 1; min-width: 0; display: flex; flex-direction: column; overflow: hidden;
}

/* Контейнер табов (высота фиксированная) */
.tabs-container {
  flex-shrink: 0;
  height: 35px;
  background-color: var(--surface-color);
  border-bottom: 1px solid var(--border-color);
}

.editor-wrapper { flex: 1; position: relative; display: flex; min-height: 0; }
.terminal-container { flex-shrink: 0; background: var(--bg-color); }

/* Полоски ресайзеров */
.resizer {
  background-color: var(--border-color);
  z-index: 10; flex-shrink: 0; transition: background-color 0.2s;
}
.resizer:hover, .resizer:active { background-color: var(--primary); }
.resizer-x { width: 3px; cursor: col-resize; }
.resizer-y { height: 3px; cursor: row-resize; }

.empty-editor { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--text-muted); font-size: 13px; }
.loader-screen { height: 100vh; display: flex; flex-direction: column; align-items: center; justify-content: center; font-size: 14px; }
.error-text { color: #f14c4c; }

.loader-screen {
  height: 100vh;
  width: 100vw;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--bg-color);
  background-image: radial-gradient(circle at center, var(--surface-color) 0%, var(--bg-color) 70%);
}

.error-container {
  text-align: center;
  max-width: 400px;
  padding: 40px;
  background: var(--surface-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  animation: fadeIn 0.4s ease-out;
}

.error-icon {
  margin: 0 auto 24px;
  color: var(--primary);
  display: flex;
  justify-content: center;
}

.error-icon.is-loading {
  animation: rotate 2s linear infinite;
  color: var(--text-muted);
}

.error-title {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 12px;
  color: var(--text-main);
}

.error-text {
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-muted);
  margin-bottom: 32px;
}

.error-btn {
  width: 100%;
  padding: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

@keyframes rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>