<template>
  <aside class="sidebar secondary-panel">
    <div class="panel-header">
      <div class="tabs">
        <span :class="{ active: tab === 'history' }" @click="tab = 'history'">ИСТОРИЯ</span>
        <span :class="{ active: tab === 'events' }" @click="tab = 'events'">СОБЫТИЯ</span>
      </div>
    </div>

    <div class="panel-content">
      <!-- Вкладка История -->
      <div v-if="tab === 'history'" class="list-container">
        <div v-if="sessionStore.history.length === 0" class="empty-state">Нет истории версий.</div>

        <div v-for="item in sessionStore.history" :key="item.version" class="history-card">
          <div class="hc-header">
            <span class="hc-version">v{{ item.version }}</span>
            <span class="hc-date">{{ new Date(item.created_at).toLocaleString() }}</span>
          </div>
          <pre class="hc-preview">{{ item.preview }}</pre>
          <div class="hc-actions">
            <button class="btn-ghost" @click="restore(item.version)">Откатить к v{{ item.version }}</button>
          </div>
        </div>
      </div>

      <!-- Вкладка События -->
      <div v-if="tab === 'events'" class="list-container">
        <div v-if="sessionStore.events.length === 0" class="empty-state">Нет событий.</div>

        <div v-for="ev in sessionStore.events" :key="ev.id" class="event-item">
          <div class="ev-dot" :class="ev.event_type"></div>
          <div class="ev-info">
            <span class="ev-desc">{{ getEventDescription(ev) }}</span>
            <span class="ev-time">{{ new Date(ev.created_at).toLocaleTimeString() }}</span>
          </div>
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useSessionStore } from '@/store/session.store';

const sessionStore = useSessionStore();
const tab = ref<'history' | 'events'>('history');

function restore(version: number) {
  if(confirm(`Откатить текущий файл к версии ${version}? Все несохраненные изменения будут потеряны.`)) {
    sessionStore.restoreVersion(version);
  }
}

function getEventDescription(ev: any) {
  const name = ev.details?.username || 'Кто-то';
  switch (ev.event_type) {
    case 'join': return `${name} присоединился`;
    case 'leave': return `${name} вышел`;
    case 'save': return `${name} сохранил файл (v${ev.details?.version})`;
    case 'file_created': return `Создан файл`;
    default: return ev.event_type;
  }
}
</script>

<style scoped>
.sidebar { border-left: 1px solid var(--border-color); display: flex; flex-direction: column; background: var(--bg-color); height: 100%; }
.panel-header { padding: 0; border-bottom: 1px solid var(--border-color); }
.tabs { display: flex; height: 40px; }
.tabs span { flex: 1; display: flex; align-items: center; justify-content: center; font-size: 11px; font-weight: 600; color: var(--text-muted); cursor: pointer; border-bottom: 2px solid transparent; }
.tabs span.active { color: var(--primary); border-bottom-color: var(--primary); }
.panel-content { flex: 1; overflow-y: auto; padding: 12px; }
.empty-state { text-align: center; font-size: 12px; color: var(--text-muted); margin-top: 20px; }

/* Карточка истории */
.history-card { background: var(--surface-color); border: 1px solid var(--border-color); border-radius: 6px; margin-bottom: 12px; overflow: hidden; }
.hc-header { display: flex; justify-content: space-between; padding: 8px 12px; background: rgba(0,0,0,0.2); font-size: 11px; }
.hc-version { font-weight: bold; color: var(--primary); }
.hc-date { color: var(--text-muted); }
.hc-preview { margin: 0; padding: 12px; font-size: 11px; font-family: monospace; color: var(--text-muted); background: #000; border-top: 1px solid var(--border-color); border-bottom: 1px solid var(--border-color); max-height: 80px; overflow: hidden; text-overflow: ellipsis; }
.hc-actions { padding: 6px 12px; text-align: right; }
.btn-ghost { background: transparent; color: var(--text-main); border: none; font-size: 11px; cursor: pointer; opacity: 0.8; }
.btn-ghost:hover { opacity: 1; color: var(--primary); }

/* Лента событий */
.event-item { display: flex; align-items: flex-start; gap: 10px; margin-bottom: 12px; font-size: 12px; }
.ev-dot { width: 8px; height: 8px; border-radius: 50%; margin-top: 4px; background: gray; }
.ev-dot.join { background: #4ECDC4; }
.ev-dot.leave { background: #FF6B6B; }
.ev-dot.save { background: #45B7D1; }
.ev-info { display: flex; flex-direction: column; gap: 2px; }
.ev-time { font-size: 10px; color: var(--text-muted); }
</style>