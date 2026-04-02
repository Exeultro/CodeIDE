<template>
  <header class="title-bar">
    <div class="tb-left">
      <button class="icon-btn" @click="emit('go-home')" title="На главную">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 18l-6-6 6-6"/></svg>
      </button>
      <div class="window-title">
        {{ sessionStore.currentSession?.name || 'Загрузка...' }}
        <span class="tb-subtitle" v-if="sessionStore.activeFilePath">— {{ sessionStore.activeFilePath }}</span>
      </div>
    </div>

    <div class="tb-center" @click="$emit('open-participants')" style="cursor: pointer;" title="Посмотреть участников">
      <div class="avatar-stack" @click="$emit('open-participants')" style="cursor: pointer;">
        <div v-for="p in sessionStore.participants.slice(0, 3)" :key="p.user_id" class="avatar">
          {{ p.username.charAt(0).toUpperCase() }}
        </div>
        <div v-if="sessionStore.participants.length > 3" class="avatar more">+{{ sessionStore.participants.length - 3 }}</div>
      </div>

      <!-- КНОПКА ПРИГЛАШЕНИЯ -->
      <button class="invite-trigger-btn" @click="$emit('open-participants')" title="Пригласить участников">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><line x1="20" y1="8" x2="20" y2="14"/><line x1="23" y1="11" x2="17" y2="11"/>
        </svg>
        <span>Invite</span>
      </button>
      <button class="flat-action-btn leave-btn" @click="handleLeave" title="Выйти из сессии">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/>
        </svg>
        <span>Leave</span>
      </button>

    </div>

    <div class="tb-right flex-gap">
      <button class="flat-action-btn" @click="sessionStore.saveActiveFile(true)" :disabled="!sessionStore.activeFilePath || sessionStore.isSaving" title="Сохранить">
        <span v-if="sessionStore.dirty" class="dirty-dot" style="position: absolute; top: 4px; right: 4px;"></span>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>
      </button>
      <div class="divider-v"></div>
      <button class="flat-action-btn play-btn" @click="sessionStore.runCode" title="Run Code">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="5 3 19 12 5 21 5 3"/></svg>
        <span>Run</span>
      </button>
      <div class="divider-v"></div>
      <IdeSettingsDropdown />
    </div>
  </header>
</template>

<script setup lang="ts">
import { useSessionStore } from '@/store/session.store';
import IdeSettingsDropdown from "@/components/ide/IdeSettingsDropdown.vue";
import { useRouter } from 'vue-router';
const router = useRouter();

function handleLeave() {
  if (confirm('Вы уверены, что хотите покинуть сессию?')) {
    sessionStore.leaveSession();
    router.push('/');
  }
}

const sessionStore = useSessionStore();
// const emit = defineEmits(['go-home']);
const emit = defineEmits(['go-home', 'open-participants']);
</script>

<style scoped>
.title-bar {
  display: flex; justify-content: space-between; align-items: center;
  height: 40px; padding: 0 12px; font-size: 13px;
  background-color: var(--surface-color); border-bottom: 1px solid var(--border-color);
  user-select: none;
}
.tb-left, .tb-center, .tb-right { display: flex; align-items: center; gap: 10px; }
.tb-center { flex: 1; justify-content: center; }
.window-title { display: flex; gap: 6px; align-items: center; color: var(--text-main); }
.tb-subtitle { color: var(--text-muted); }

/* СТИЛИ КНОПОК БЕЗ БЕЛОГО ФОНА */
button { font-family: inherit; }
.icon-btn {
  background: transparent; border: none; color: var(--text-muted);
  cursor: pointer; border-radius: 4px; padding: 4px;
  display: flex; align-items: center; justify-content: center; outline: none;
}
.icon-btn:hover { background-color: var(--bg-color); color: var(--text-main); }

.flat-action-btn {
  background: transparent; border: none; color: var(--text-muted);
  display: flex; align-items: center; gap: 6px; padding: 6px 10px;
  border-radius: var(--radius-sm); cursor: pointer; font-size: 13px;
  transition: all 0.15s ease; position: relative; outline: none;
}
.flat-action-btn:hover:not(:disabled) { background: var(--bg-color); color: var(--text-main); }
.flat-action-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.play-btn { color: var(--text-main); }
.play-btn:hover { color: var(--primary); background: var(--primary-light); }

.invite-trigger-btn {
  background: var(--bg-color); border: 1px solid var(--border-color); color: var(--text-muted);
  display: flex; align-items: center; gap: 6px; padding: 4px 10px; border-radius: 12px;
  cursor: pointer; font-size: 11px; margin-left: 10px; transition: 0.2s;
}
.invite-trigger-btn:hover { border-color: var(--primary); color: var(--text-main); }

.divider-v { width: 1px; height: 16px; background-color: var(--border-color); }
.avatar-stack { display: flex; align-items: center; }
.avatar { width: 24px; height: 24px; border-radius: 50%; background: var(--primary); color: #fff; display: flex; align-items: center; justify-content: center; font-size: 11px; font-weight: bold; margin-left: -6px; border: 2px solid var(--surface-color); position: relative; z-index: 1; }
.avatar:first-child { margin-left: 0; }
.dirty-dot { display: inline-block; width: 8px; height: 8px; background-color: var(--primary); border-radius: 50%; box-shadow: 0 0 4px var(--primary); }
</style>