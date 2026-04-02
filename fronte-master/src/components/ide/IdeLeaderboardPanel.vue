<template>
  <aside class="sidebar secondary-panel">
    <div class="panel-header lb-header">
      <div class="lb-title">

        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 15l-3 3-3-3m6 0V3m0 12l3 3 3-3"/></svg>
        <span>ЛИДЕРБОРД</span>
      </div>
      <button class="icon-btn" @click="showSettings = !showSettings" title="Настройки профиля">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="3"></circle>
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
        </svg>
      </button>
    </div>

    <div class="panel-content">
      <!-- Настройки инкогнито (выдвигаются) -->
      <div v-if="showSettings && sessionStore.userProfile" class="profile-settings">
        <label class="toggle-row">
          <span>Режим инкогнито</span>
          <input type="checkbox" v-model="profileDraft.incognito" />
        </label>
        <div class="input-row" v-if="profileDraft.incognito">
          <input type="text" v-model="profileDraft.nickname" placeholder="Тайный псевдоним" />
        </div>
        <button class="btn-primary-sm" @click="saveProfile">Сохранить</button>
      </div>

      <!-- Список лидеров -->
      <div class="lb-list">
        <div v-for="(user, idx) in sessionStore.leaderboard" :key="user.user_id" class="lb-item">
          <div class="lb-rank">#{{ idx + 1 }}</div>
          <div class="lb-avatar" :class="{'incognito': user.incognito}">
            {{ user.display_name.charAt(0).toUpperCase() }}
          </div>
          <div class="lb-info">
            <span class="lb-name">{{ user.display_name }}</span>
            <span class="lb-pts">{{ user.points }} XP</span>
          </div>
        </div>

        <div v-if="sessionStore.leaderboard.length === 0" class="empty-state">
          Пишите код, чтобы заработать очки!
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue';
import { useSessionStore } from '@/store/session.store';

const sessionStore = useSessionStore();
const showSettings = ref(false);

const profileDraft = reactive({
  incognito: false,
  nickname: ''
});

// Синхронизируем черновик со store
watch(() => sessionStore.userProfile, (newVal) => {
  if (newVal) {
    profileDraft.incognito = newVal.incognito;
    profileDraft.nickname = newVal.nickname;
  }
}, { immediate: true });

function saveProfile() {
  sessionStore.updateProfile(profileDraft.incognito, profileDraft.nickname);
  showSettings.value = false;
}
</script>

<style scoped>
.sidebar { border-left: 1px solid var(--border-color); display: flex; flex-direction: column; background: var(--bg-color); height: 100%; }
.lb-header { height: 40px; padding: 0 12px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid var(--border-color); }
.lb-title { font-size: 11px; font-weight: 600; color: var(--primary); display: flex; align-items: center; gap: 6px; }
.panel-content { flex: 1; overflow-y: auto; }

/* Настройки */
.profile-settings { padding: 12px; background: var(--surface-color); border-bottom: 1px solid var(--border-color); display: flex; flex-direction: column; gap: 10px; font-size: 12px; }
.toggle-row { display: flex; justify-content: space-between; align-items: center; cursor: pointer; }
.input-row input { width: 100%; background: #000; border: 1px solid var(--border-color); color: #fff; padding: 6px; font-size: 12px; border-radius: 4px; }
.btn-primary-sm { background: var(--primary); color: #fff; border: none; padding: 6px; border-radius: 4px; cursor: pointer; }

/* Элементы списка */
.lb-list { padding: 12px; display: flex; flex-direction: column; gap: 10px; }
.lb-item { display: flex; align-items: center; gap: 10px; background: var(--surface-color); padding: 8px 12px; border-radius: 6px; border: 1px solid rgba(255,255,255,0.05); }
.lb-rank { font-weight: bold; color: var(--text-muted); font-size: 11px; width: 20px; }
.lb-avatar { width: 28px; height: 28px; border-radius: 50%; background: var(--primary); display: flex; align-items: center; justify-content: center; font-weight: bold; color: #fff; font-size: 12px; }
.lb-avatar.incognito { background: #333; border: 1px dashed var(--text-muted); color: var(--text-muted); }
.lb-info { display: flex; flex-direction: column; }
.lb-name { font-size: 13px; font-weight: 500; color: var(--text-main); }
.lb-pts { font-size: 11px; color: #f59f00; font-weight: bold; }
.empty-state { text-align: center; color: var(--text-muted); font-size: 12px; margin-top: 20px; }
.icon-btn { background: none; border: none; color: var(--text-muted); cursor: pointer; }
.icon-btn:hover { color: var(--text-main); }
</style>