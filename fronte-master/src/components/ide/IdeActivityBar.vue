<template>
  <div class="activity-bar">
    <div class="ab-top">
      <!-- Управление левой панелью -->
      <button class="ab-icon" :class="{ active: activeLeft === 'explorer' }" @click="$emit('select', 'explorer')" title="Explorer (Файлы)">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
      </button>

      <div class="ab-divider"></div>

      <!-- Управление правой панелью -->
      <button class="ab-icon" :class="{ active: activeRight === 'history' }" @click="$emit('select', 'history')" title="History & Events">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
      </button>

      <button class="ab-icon" :class="{ active: activeRight === 'leaderboard' }" @click="$emit('select', 'leaderboard')" title="Leaderboard">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 15l-3 3-3-3m6 0V3m0 12l3 3 3-3"/></svg>
      </button>

      <button class="ab-icon ai-btn" :class="{ active: activeRight === 'ai' }" @click="$emit('select', 'ai')" title="AI Review">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
        <div class="badge" v-if="aiCount > 0">{{ aiCount }}</div>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ activeLeft: string; activeRight: string; aiCount: number }>();
defineEmits(['select']);
</script>

<style scoped>
.activity-bar {
  width: 48px; background: var(--surface-color); border-right: 1px solid var(--border-color);
  display: flex; flex-direction: column; align-items: center; padding-top: 10px; flex-shrink: 0; z-index: 20;
}
.ab-icon {
  width: 48px; height: 48px; background: transparent; border: none; color: var(--text-muted);
  cursor: pointer; display: flex; align-items: center; justify-content: center; position: relative;
  border-left: 2px solid transparent; transition: all 0.1s; outline: none;
}
.ab-icon:hover { color: var(--text-main); }
.ab-icon.active { color: var(--text-main); border-left-color: var(--primary); }
.ab-divider { width: 24px; height: 1px; background: var(--border-color); margin: 8px 0; }
.badge {
  position: absolute; bottom: 8px; right: 8px; background: var(--primary); color: #fff;
  font-size: 9px; font-weight: bold; width: 14px; height: 14px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
}
</style>