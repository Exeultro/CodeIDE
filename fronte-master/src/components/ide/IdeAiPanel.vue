<!-- Отрезок кода для IdeAiPanel.vue -->
<template>
  <aside class="sidebar secondary-panel ai-panel">
    <!-- ... хедер ... -->

    <div class="ai-hint-section">
      <button
          class="btn-primary-sm hint-btn"
          @click="sessionStore.fetchHint"
          :disabled="sessionStore.isHintLoading"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/>
        </svg>
        <span>{{ sessionStore.isHintLoading ? 'ИИ анализирует...' : 'Спросить совет у ИИ' }}</span>
      </button>

      <!-- Блок с ответом (с добавленной кнопкой закрыть) -->
      <div v-if="sessionStore.aiHint || sessionStore.isHintLoading" class="hint-result">
        <div class="hint-result-header">
          <span>ОТВЕТ НЕЙРОСЕТИ</span>
          <button class="close-hint" @click="sessionStore.aiHint = null">×</button>
        </div>

        <div v-if="sessionStore.isHintLoading" class="pulsing-loader">
          Генерация ответа... (это может занять время)
        </div>

        <!-- Используем v-html если хочешь рендерить markdown,
             но пока оставим просто текст для стабильности -->
        <pre v-else class="hint-text">{{ sessionStore.aiHint }}</pre>
      </div>
    </div>

    <!-- ... далее ai-feed ... -->
  </aside>
</template>

<style scoped>
/* Добавь/обнови эти стили */
.ai-hint-section {
  padding: 12px;
  background: var(--surface-color);
  border-bottom: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  gap: 12px;
  z-index: 10; /* Чтобы было поверх всего */
}

.hint-result {
  background: var(--bg-color);
  border: 1px solid var(--primary); /* Выделим красным бордером для заметности */
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  max-height: 400px;
  overflow: hidden;
}

.hint-result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 10px;
  background: var(--primary-light);
  font-size: 9px;
  font-weight: 800;
  color: var(--primary);
  letter-spacing: 0.5px;
}

.close-hint {
  background: none; border: none; color: var(--primary);
  cursor: pointer; font-size: 16px; font-weight: bold;
}

.hint-text {
  margin: 0;
  padding: 12px;
  white-space: pre-wrap;
  font-family: var(--font-family);
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-main);
  overflow-y: auto;
}

.pulsing-loader {
  padding: 20px;
  text-align: center;
  font-size: 12px;
  color: var(--text-muted);
  animation: pulse 1.5s infinite;
}

@keyframes pulse { 0% { opacity: 0.5; } 50% { opacity: 1; } 100% { opacity: 0.5; } }
</style>
<script setup lang="ts">
import { ref } from 'vue';
import { useSessionStore } from '@/store/session.store';

const sessionStore = useSessionStore();
const applyingId = ref<string | null>(null);

async function handleApply(id: string) {
  applyingId.value = id;
  try {
    await sessionStore.applyReview(id);
  } finally {
    applyingId.value = null;
  }
}
</script>

<style scoped>
.ai-panel { border-left: 1px solid var(--border-color); display: flex; flex-direction: column; background-color: var(--bg-color); height: 100%; overflow: hidden; }
.ai-header { height: 40px; border-bottom: 1px solid var(--border-color); background-color: var(--surface-color); padding: 0 12px; display: flex; align-items: center; justify-content: space-between; flex-shrink: 0; }
.ai-title { display: flex; align-items: center; gap: 8px; color: var(--primary); font-size: 11px; font-weight: 600; }
.sparkles-icon { color: var(--primary); }
.badge { background: var(--primary); color: #fff; padding: 2px 8px; border-radius: 12px; font-size: 11px; font-weight: bold; }

/* Секция подсказок */
.ai-hint-section { padding: 12px; border-bottom: 1px solid var(--border-color); background: var(--surface-color); display: flex; flex-direction: column; gap: 10px; flex-shrink: 0; }
.hint-btn { width: 100%; display: flex; justify-content: center; align-items: center; gap: 8px; font-size: 12px; padding: 8px; border-radius: 6px; cursor: pointer; border: none; background: rgba(255, 255, 255, 0.05); color: var(--text-main); transition: 0.2s; }
.hint-btn:hover:not(:disabled) { background: var(--primary-light); color: var(--primary); }
.hint-btn:disabled { opacity: 0.5; cursor: wait; }

.hint-result { background: var(--bg-color); border: 1px solid var(--border-color); border-radius: 6px; padding: 10px; font-size: 12px; max-height: 300px; overflow-y: auto; }
.pulsing-loader { color: var(--primary); font-weight: 500; animation: pulse 1.5s infinite; text-align: center; }
.hint-text { margin: 0; white-space: pre-wrap; font-family: 'Inter', sans-serif; color: var(--text-main); line-height: 1.5; }

.ai-feed { padding: 12px; overflow-y: auto; flex: 1; display: flex; flex-direction: column; gap: 16px; }
.empty-state-text { text-align: center; color: var(--text-muted); font-size: 13px; margin-top: 40px; display: flex; flex-direction: column; align-items: center; gap: 12px; }

/* Карточки */
.ai-card { background: var(--surface-color); border: 1px solid var(--border-color); border-radius: 8px; overflow: hidden; flex-shrink: 0; }
.ai-card-header { padding: 8px 12px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid rgba(255,255,255,0.05); }
.ai-location { font-size: 11px; color: var(--text-muted); font-family: 'JetBrains Mono', Consolas, monospace; }
.ai-type { font-size: 10px; text-transform: uppercase; background: rgba(255,255,255,0.1); padding: 2px 6px; border-radius: 4px; color: var(--text-muted); }
.ai-msg { padding: 12px; margin: 0; font-size: 13px; line-height: 1.5; color: var(--text-main); }

.code-diff { background: #0A0A0A; font-family: 'JetBrains Mono', Consolas, monospace; font-size: 12px; border-top: 1px solid var(--border-color); border-bottom: 1px solid var(--border-color); }
.diff-line { display: flex; width: 100%; }
.diff-sign { width: 24px; flex-shrink: 0; text-align: center; user-select: none; padding: 4px 0; font-weight: bold; }
.diff-code { margin: 0; padding: 4px 8px 4px 0; white-space: pre-wrap; word-break: break-all; flex: 1; }
.diff-line.removed { background-color: rgba(248, 81, 73, 0.15); }
.diff-line.removed .diff-sign, .diff-line.removed .diff-code { color: #ffa198; }
.diff-line.added { background-color: rgba(46, 160, 67, 0.15); }
.diff-line.added .diff-sign, .diff-line.added .diff-code { color: #7ee787; }

.ai-actions { display: flex; justify-content: flex-end; gap: 8px; padding: 10px 12px; background: var(--surface-color); }
.btn-ghost { background: transparent; color: var(--text-muted); border: none; padding: 6px 12px; border-radius: 4px; font-size: 12px; cursor: pointer; transition: 0.2s; }
.btn-ghost:hover { background: rgba(255,255,255,0.05); color: var(--text-main); }
.btn-primary-sm { background: var(--primary); color: #fff; border: none; padding: 6px 12px; border-radius: 4px; font-size: 12px; cursor: pointer; transition: 0.2s; }
.btn-primary-sm:hover:not(:disabled) { background: var(--primary-hover); }
.btn-primary-sm:disabled { opacity: 0.6; cursor: not-allowed; }

@keyframes pulse { 0% { opacity: 0.6; } 50% { opacity: 1; } 100% { opacity: 0.6; } }
</style>