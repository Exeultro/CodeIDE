<template>
  <li>
    <!-- Сам элемент (папка или файл) -->
    <div
        class="tree-item"
        :class="{ 'active': sessionStore.activeFilePath === node.path && !node.is_dir }"
        :style="{ paddingLeft: `${depth * 12 + 10}px` }"
        @click.stop="handleClick"
        @mouseenter="isHovered = true"
        @mouseleave="isHovered = false"
    >
      <div class="item-main">
        <svg v-if="node.is_dir && isOpen" class="file-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
        <svg v-else-if="node.is_dir && !isOpen" class="file-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
        <svg v-else class="file-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><polyline points="13 2 13 9 20 9"/></svg>

        <span class="file-name">{{ node.name }}</span>
        <span v-if="sessionStore.activeFilePath === node.path && sessionStore.dirty" class="dirty-dot ml-2"></span>
      </div>

      <!-- Иконки создания (появляются только при ховере на папку) -->
      <div v-show="isHovered && node.is_dir" class="item-actions">
        <button class="action-btn" title="New File" @click.stop="$emit('start-create', { path: node.path, isDir: false })">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="12" y1="18" x2="12" y2="12"/><line x1="9" y1="15" x2="15" y2="15"/></svg>
        </button>
        <button class="action-btn" title="New Folder" @click.stop="$emit('start-create', { path: node.path, isDir: true })">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/><line x1="12" y1="11" x2="12" y2="17"/><line x1="9" y1="14" x2="15" y2="14"/></svg>
        </button>
      </div>
    </div>

    <!-- Дети (если папка открыта) -->
    <ul v-if="node.is_dir && isOpen" class="file-tree-children">
      <!-- Инпут создания, если создание инициировано ИМЕННО в этой папке -->
      <li v-if="creatingState.active && creatingState.parentPath === node.path" class="tree-item create-item" :style="{ paddingLeft: `${(depth + 1) * 12 + 10}px` }">
        <input
            v-focus
            v-model="creatingState.name"
            class="create-input"
            type="text"
            placeholder="Имя..."
            @keydown.enter.prevent="$emit('submit-create')"
            @keydown.esc="$emit('cancel-create')"
            @blur="$emit('cancel-create')"
        />
      </li>

      <FileTreeItem
          v-for="child in node.children"
          :key="child.path"
          :node="child"
          :depth="depth + 1"
          :creatingState="creatingState"
          @start-create="$emit('start-create', $event)"
          @submit-create="$emit('submit-create')"
          @cancel-create="$emit('cancel-create')"
      />
    </ul>
  </li>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useSessionStore } from '@/store/session.store';

// Директива для автофокуса на инпуте
const vFocus = {
  mounted: (el: HTMLInputElement) => el.focus()
}

const props = defineProps<{
  node: any;
  depth: number;
  creatingState: { active: boolean; parentPath: string; isDir: boolean; name: string };
}>();

const emit = defineEmits(['start-create', 'submit-create', 'cancel-create']);
const sessionStore = useSessionStore();
const isOpen = ref(true);
const isHovered = ref(false);

function handleClick() {
  if (props.node.is_dir) {
    isOpen.value = !isOpen.value;
  } else {
    sessionStore.setActiveFile(props.node.path);
  }
}
</script>

<style scoped>
.tree-item {
  display: flex; justify-content: space-between; align-items: center;
  padding: 4px 10px; cursor: pointer; font-size: 13px;
  color: var(--text-muted); user-select: none;
}
.tree-item:hover { background-color: var(--surface-color); color: var(--text-main); }
.tree-item.active { background-color: var(--primary-light); color: var(--text-main); }

.item-main { display: flex; align-items: center; gap: 6px; }
.file-icon { color: inherit; opacity: 0.8; flex-shrink: 0; }
.tree-item.active .file-icon { color: var(--primary); opacity: 1; }
.dirty-dot { display: inline-block; width: 6px; height: 6px; background-color: var(--primary); border-radius: 50%; }

.item-actions { display: flex; gap: 2px; }
.action-btn {
  background: transparent; border: none; color: var(--text-muted);
  cursor: pointer; border-radius: 4px; padding: 4px;
  display: flex; align-items: center; justify-content: center; outline: none;
}
.action-btn:hover { background-color: var(--bg-color); color: var(--text-main); }

.file-tree-children { list-style: none; padding: 0; margin: 0; }

/* Инпут внутри папки */
.create-item { padding-right: 10px; background: transparent !important; }
.create-input {
  width: 100%; background: var(--surface-color); border: 1px solid var(--primary);
  color: var(--text-main); font-size: 13px; padding: 2px 6px; outline: none; border-radius: 2px;
}
</style>