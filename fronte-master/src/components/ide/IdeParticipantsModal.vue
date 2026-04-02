<template>
  <div class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        УЧАСТНИКИ СЕССИИ
        <button class="close-btn" @click="$emit('close')">&times;</button>
      </div>

      <div class="modal-body">

        <!-- Секция приглашения (оставили одну, чистую) -->
        <div class="invite-section">
          <span class="section-label">Пригласительная ссылка</span>
          <div class="flex-row">
            <input
                type="text"
                readonly
                :value="sessionStore.inviteLink || 'Загрузка...'"
                class="flat-input"
            />
            <button class="btn-primary-sm" @click="copyLink">Копировать</button>
          </div>
        </div>

        <div class="divider-text">СПИСОК И РЕЙТИНГ</div>

        <!-- Список участников -->
        <div class="participants-list">
          <div v-for="user in mergedUsers" :key="user.user_id" class="p-item">
            <!-- Аватар с цветом (можно привязать к ID) -->
            <div class="p-avatar" :style="{ backgroundColor: getUserColor(user.user_id) }">
              {{ user.username.charAt(0).toUpperCase() }}
            </div>

            <div class="p-info">
              <div class="p-name-row">
                <span class="p-name">{{ user.username }}</span>
                <span v-if="user.isOnline" class="status-dot" title="В сети"></span>
                <span v-if="user.isMe" class="me-badge">ВЫ</span>
              </div>
              <span class="p-role">{{ user.isOwner ? 'Владелец' : 'Участник' }}</span>
            </div>

            <div class="p-points">
              <span class="pts-value">{{ user.points }}</span>
              <span class="pts-label">XP</span>
            </div>

            <!-- Кнопка удаления (только для владельца) -->
            <button
                v-if="sessionStore.isOwner && !user.isMe"
                class="kick-btn"
                @click="kickUser(user.user_id, user.username)"
                title="Исключить"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M18 6L6 18M6 6l12 12"/>
              </svg>

            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useSessionStore } from '@/store/session.store';
import { useAuthStore } from '@/store/auth.store';

const emit = defineEmits(['close']);
const sessionStore = useSessionStore();
const authStore = useAuthStore();

onMounted(() => {
  sessionStore.fetchInviteLink();
});

function copyLink() {
  if (sessionStore.inviteLink) {
    navigator.clipboard.writeText(sessionStore.inviteLink);
    alert('Ссылка скопирована!');
  }
}

async function kickUser(id: string, name: string) {
  if (confirm(`Исключить пользователя ${name} из сессии?`)) {
    await sessionStore.kickParticipant(id);
  }
}

// Хелпер для цвета аватарок
function getUserColor(id: string) {
  const colors = ['#D92525', '#4ECDC4', '#45B7D1', '#96CEB4', '#F7B731', '#A55EEA'];
  const index = id.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
  return colors[index % colors.length];
}

// ГЛАВНАЯ ЛОГИКА: Слияние всех данных
const mergedUsers = computed(() => {
  const map = new Map<string, any>();

  const dbArr = sessionStore.dbParticipants || [];
  const onlineArr = sessionStore.participants || [];
  const lbArr = sessionStore.leaderboard || [];

  // 1. Берем всех из БД (кто имеет доступ)
  dbArr.forEach(u => {
    map.set(u.user_id, {
      user_id: u.user_id,
      username: u.username,
      isOnline: false,
      isOwner: sessionStore.currentSession?.owner_id === u.user_id,
      isMe: authStore.userId === u.user_id,
      points: 0
    });
  });

  // 2. Добавляем очки из Лидерборда
  lbArr.forEach(u => {
    if (map.has(u.user_id)) {
      map.get(u.user_id).points = u.points;
    }
  });

  // 3. Отмечаем тех, кто онлайн по сокетам
  onlineArr.forEach(p => {
    if (map.has(p.user_id)) {
      map.get(p.user_id).isOnline = true;
    } else {
      // Если юзер онлайн, но его нет в БД списке (например, зашел только что по ссылке)
      map.set(p.user_id, {
        user_id: p.user_id,
        username: p.username,
        isOnline: true,
        isOwner: false,
        isMe: authStore.userId === p.user_id,
        points: 0
      });
    }
  });

  // Сортировка: Я сверху, потом по очкам
  return Array.from(map.values()).sort((a, b) => {
    if (a.isMe) return -1;
    if (b.isMe) return 1;
    return b.points - a.points;
  });
});
</script>

<style scoped>
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.7); display: flex; align-items: center; justify-content: center; z-index: 2000; }
.modal-content { background: var(--surface-color); border: 1px solid var(--border-color); width: 400px; border-radius: 8px; box-shadow: var(--shadow-lg); overflow: hidden; }
.modal-header { padding: 14px 20px; font-size: 11px; font-weight: 800; color: var(--text-muted); border-bottom: 1px solid var(--border-color); display: flex; justify-content: space-between; align-items: center; letter-spacing: 1px; }
.close-btn { background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 20px; line-height: 1; }
.close-btn:hover { color: var(--primary); }

.modal-body { padding: 20px; }

.section-label { font-size: 10px; font-weight: 700; color: var(--text-muted); text-transform: uppercase; margin-bottom: 8px; display: block; }
.flex-row { display: flex; gap: 10px; }
.flat-input { flex: 1; background: var(--bg-color); border: 1px solid var(--border-color); color: var(--text-main); padding: 8px 12px; font-size: 12px; border-radius: 4px; outline: none; }

.divider-text { font-size: 10px; font-weight: 700; color: var(--text-muted); text-align: center; margin: 24px 0 16px; position: relative; }
.divider-text::before, .divider-text::after { content: ''; position: absolute; top: 50%; width: 30%; height: 1px; background: var(--border-color); }
.divider-text::before { left: 0; } .divider-text::after { right: 0; }

.participants-list { display: flex; flex-direction: column; gap: 8px; max-height: 300px; overflow-y: auto; }
.p-item { display: flex; align-items: center; gap: 12px; padding: 10px; background: rgba(255,255,255,0.02); border-radius: 6px; border: 1px solid transparent; transition: 0.2s; }
.p-item:hover { border-color: var(--border-color); background: rgba(255,255,255,0.04); }

.p-avatar { width: 36px; height: 36px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-weight: 700; color: #fff; font-size: 14px; flex-shrink: 0; }

.p-info { flex: 1; display: flex; flex-direction: column; gap: 2px; }
.p-name-row { display: flex; align-items: center; gap: 8px; }
.p-name { font-size: 14px; font-weight: 600; color: var(--text-main); }
.status-dot { width: 7px; height: 7px; background: #4ECDC4; border-radius: 50%; box-shadow: 0 0 5px #4ECDC4; }
.me-badge { font-size: 9px; background: var(--primary); color: #fff; padding: 1px 4px; border-radius: 3px; font-weight: 800; }
.p-role { font-size: 11px; color: var(--text-muted); }

.p-points { text-align: right; margin-right: 10px; }
.pts-value { display: block; font-size: 14px; font-weight: 800; color: #F7B731; }
.pts-label { font-size: 9px; color: var(--text-muted); font-weight: 700; }

.kick-btn { background: none; border: none; color: var(--text-muted); font-size: 18px; cursor: pointer; padding: 0 5px; }
.kick-btn:hover { color: var(--primary); }

.btn-primary-sm { background: var(--primary); border: none; color: #fff; padding: 0 15px; font-size: 12px; font-weight: 600; cursor: pointer; border-radius: 4px; }
.btn-primary-sm:hover { background: var(--primary-hover); }
</style>