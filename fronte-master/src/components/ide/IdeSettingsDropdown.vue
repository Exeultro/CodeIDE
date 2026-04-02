<template>
  <div class="settings-wrapper" ref="menuRef">
    <button class="icon-btn" :class="{ active: isOpen }" @click="isOpen = !isOpen" title="Настройки IDE">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
      </svg>
    </button>

    <transition name="fade">
      <div v-if="isOpen" class="settings-menu">
        <div class="menu-header">НАСТРОЙКИ IDE</div>

        <div class="setting-row">
          <label>Тема</label>
          <select v-model="settings.theme" class="flat-input">
            <option value="vs-dark">Dark (VS)</option>
            <option value="light">Light</option>
            <option value="hc-black">High Contrast</option>
          </select>
        </div>

        <div class="setting-row">
          <label>Размер шрифта</label>
          <input type="number" v-model.number="settings.fontSize" class="flat-input" min="10" max="32" />
        </div>

        <div class="setting-row">
          <label>Высота строки</label>
          <input type="number" step="0.1" v-model.number="settings.lineHeight" class="flat-input" min="1" max="3" />
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue';
import { useSettingsStore } from '@/store/settings.store';

const settings = useSettingsStore();
const isOpen = ref(false);
const menuRef = ref<HTMLElement | null>(null);

const closeHandler = (e: MouseEvent) => {
  if (menuRef.value && !menuRef.value.contains(e.target as Node)) {
    isOpen.value = false;
  }
};

onMounted(() => document.addEventListener('click', closeHandler));
onBeforeUnmount(() => document.removeEventListener('click', closeHandler));
</script>

<style scoped>
.settings-wrapper { position: relative; }
.icon-btn { background: transparent; border: none; color: var(--text-muted); cursor: pointer; padding: 4px; display: flex; border-radius: 4px; outline: none; }
.icon-btn:hover, .icon-btn.active { color: var(--text-main); background: var(--bg-color); }

.settings-menu {
  position: absolute; top: calc(100% + 8px); right: 0; z-index: 1000;
  width: 220px; background: var(--surface-color);
  border: 1px solid var(--border-color); border-radius: 4px;
  box-shadow: var(--shadow-md); padding: 8px;
}
.menu-header { font-size: 10px; font-weight: bold; color: var(--text-muted); margin-bottom: 10px; padding: 0 4px; }

.setting-row { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.setting-row label { font-size: 12px; color: var(--text-main); }
.flat-input {
  background: var(--bg-color); border: 1px solid var(--border-color);
  color: var(--text-main); font-size: 12px; padding: 4px 6px;
  width: 100px; border-radius: 2px; outline: none; font-family: inherit;
}
.flat-input:focus { border-color: var(--primary); }

.fade-enter-active, .fade-leave-active { transition: opacity 0.15s, transform 0.15s; }
.fade-enter-from, .fade-leave-to { opacity: 0; transform: translateY(-5px); }
</style>