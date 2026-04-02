<template>
  <div class="bottom-panel">
    <div class="panel-tabs">
      <div class="panel-tab active">TERMINAL</div>
      <div class="panel-actions">
        <button class="icon-btn" @click="sessionStore.terminalLines = ['Очищено.']" title="Очистить терминал">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="4.93" y1="4.93" x2="19.07" y2="19.07"/></svg>
        </button>
      </div>
    </div>

    <div class="terminal-content" ref="terminalContentRef" @click="focusInput">
      <div
          v-for="(line, index) in sessionStore.terminalLines"
          :key="index"
          class="terminal-line"
          :class="{ 'is-error': line.includes('ОШИБКА') || line.includes('ERROR') }"
      >
        <pre>{{ line }}</pre>
      </div>

      <div class="terminal-prompt">
        <span class="prompt-arrow">➜</span>
        <span class="prompt-path">/workspace</span>
        <input
            ref="inputRef"
            v-model="localCommand"
            class="terminal-input"
            type="text"
            spellcheck="false"
            @keydown.enter.prevent="handleEnter"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue';
import { useSessionStore } from '@/store/session.store';

const sessionStore = useSessionStore();
const terminalContentRef = ref<HTMLElement | null>(null);
const inputRef = ref<HTMLInputElement | null>(null);

// Локальная переменная для стабильного ввода
const localCommand = ref('');

function handleEnter() {
  const cmd = localCommand.value.trim();
  if (!cmd) return;

  // Передаем команду в стор и выполняем
  sessionStore.terminalCommand = cmd;
  sessionStore.execTerminal();

  // Очищаем инпут
  localCommand.value = '';
}

function focusInput() {
  inputRef.value?.focus();
}

// Автоскролл вниз при появлении новых строк
watch(
    () => sessionStore.terminalLines.length,
    () => {
      nextTick(() => {
        if (terminalContentRef.value) {
          terminalContentRef.value.scrollTop = terminalContentRef.value.scrollHeight;
        }
      });
    }
);
</script>

<style scoped>
.bottom-panel { border-top: 1px solid var(--border-color); background-color: var(--bg-color); display: flex; flex-direction: column; height: 100%; overflow: hidden; }
.panel-tabs { display: flex; height: 35px; border-bottom: 1px solid var(--border-color); }
.panel-tab { padding: 0 15px; display: flex; align-items: center; font-size: 11px; color: var(--text-main); border-bottom: 1px solid var(--primary); }
.panel-actions { margin-left: auto; display: flex; align-items: center; padding-right: 10px; }
.icon-btn { background: transparent; border: none; color: var(--text-muted); cursor: pointer; border-radius: 4px; padding: 4px; outline: none; }
.icon-btn:hover { color: var(--text-main); background: rgba(255,255,255,0.1); }
.terminal-content { flex: 1; padding: 10px 15px; overflow-y: auto; font-family: 'JetBrains Mono', Consolas, monospace; font-size: 13px; cursor: text; }
.terminal-line { margin: 0; color: var(--text-muted); }
.terminal-line pre { margin: 0; white-space: pre-wrap; font-family: inherit; }
.terminal-line.is-error { color: #f14c4c; }
.terminal-prompt { display: flex; align-items: center; gap: 8px; margin-top: 4px; }
.prompt-arrow { color: #4ECDC4; font-weight: bold; }
.prompt-path { color: var(--primary); font-weight: bold; }
.terminal-input { flex: 1; background: transparent; border: none; outline: none; color: var(--text-main); font-family: inherit; font-size: inherit; padding: 0; }
</style>