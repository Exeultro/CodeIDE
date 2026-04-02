<template>
  <aside class="sidebar side-panel">
    <div class="panel-header">
      <span>EXPLORER</span>
      <div class="panel-actions">
        <!-- Глобальное создание (в корне проекта) -->
        <button class="icon-btn" title="New File" @click="handleStartCreate({ path: '', isDir: false })">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="12" y1="18" x2="12" y2="12"/><line x1="9" y1="15" x2="15" y2="15"/></svg>
        </button>
        <button class="icon-btn" title="New Folder" @click="handleStartCreate({ path: '', isDir: true })">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/><line x1="12" y1="11" x2="12" y2="17"/><line x1="9" y1="14" x2="15" y2="14"/></svg>
        </button>
      </div>
    </div>

    <div class="tree-container">
      <div class="tree-root-name">﹀ ПРОЕКТ</div>
      <ul class="file-tree-root">

        <!-- Инпут для создания файла/папки В КОРНЕ -->
        <li v-if="creatingState.active && creatingState.parentPath === ''" class="tree-item create-item" style="padding-left: 10px;">
          <input
              v-focus
              v-model="creatingState.name"
              class="create-input"
              type="text"
              placeholder="Имя..."
              @keydown.enter.prevent="handleSubmitCreate"
              @keydown.esc="handleCancelCreate"
              @blur="handleCancelCreate"
          />
        </li>

        <!-- Рекурсивный рендер дерева -->
        <FileTreeItem
            v-for="node in nestedTree"
            :key="node.path"
            :node="node"
            :depth="0"
            :creatingState="creatingState"
            @start-create="handleStartCreate"
            @submit-create="handleSubmitCreate"
            @cancel-create="handleCancelCreate"
        />

      </ul>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { reactive, computed } from 'vue';
import { useSessionStore } from '@/store/session.store';
import { api } from '@/services/api';
import FileTreeItem from './FileTreeItem.vue';

const vFocus = { mounted: (el: HTMLInputElement) => el.focus() }

const sessionStore = useSessionStore();

// Состояние создания файла/папки (Единое для всего дерева)
const creatingState = reactive({
  active: false,
  parentPath: '', // Путь папки, внутри которой создаем ('' = корень)
  isDir: false,
  name: ''
});

function handleStartCreate(payload: { path: string, isDir: boolean }) {
  creatingState.active = true;
  creatingState.parentPath = payload.path;
  creatingState.isDir = payload.isDir;
  creatingState.name = '';
}

function handleCancelCreate() {
  creatingState.active = false;
  creatingState.name = '';
}

async function handleSubmitCreate() {
  const rawName = creatingState.name.trim();
  if (!rawName || !sessionStore.currentSession || creatingState.loading) return;

  const fullPath = creatingState.parentPath
      ? `${creatingState.parentPath}/${rawName}`
      : rawName;

  creatingState.loading = true;

  try {
    // 1. Отправляем запрос
    const newFile = await api.createFile(sessionStore.currentSession.id, {
      path: fullPath,
      is_dir: creatingState.isDir
    });

    console.log('[Sidebar] Файл успешно создан:', newFile);

    // 2. СРАЗУ добавляем его в список в сторе (не ждем сокета!)
    const exists = sessionStore.files.some(f => f.path === newFile.path);
    if (!exists) {
      sessionStore.files = [...sessionStore.files, newFile];
    }

    // 3. Закрываем инпут
    handleCancelCreate();

    // 4. Если это файл — открываем его
    if (!newFile.is_dir) {
      await sessionStore.setActiveFile(newFile.path);
    }

  } catch (err: any) {
    console.error('[Sidebar] Ошибка при создании:', err);
    alert(err.message || "Ошибка при создании");
    handleCancelCreate();
  } finally {
    creatingState.loading = false;
  }
}

// АЛГОРИТМ ПРЕОБРАЗОВАНИЯ ПЛОСКОГО СПИСКА В ДЕРЕВО
const nestedTree = computed(() => {
  const root: any[] = [];
  const levelMap: Record<string, any> = { '': root };

  // Важно: берем файлы из стора.
  // Когда в сторе сработает files.value = [...files.value], этот computed пересчитается.
  const sortedFiles = [...sessionStore.files].sort((a, b) => {
    if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
    return a.path.localeCompare(b.path);
  });

  sortedFiles.forEach(file => {
    const parts = file.path.split('/');
    const name = parts.pop()!;
    const parentPath = parts.join('/');

    const node = {
      name, path: file.path, is_dir: file.is_dir,
      children: file.is_dir ? [] : undefined
    };

    if (!levelMap[parentPath]) levelMap[''].push(node);
    else levelMap[parentPath].push(node);

    if (file.is_dir) levelMap[file.path] = node.children;
  });

  return root;
});
</script>

<style scoped>
.sidebar { background-color: var(--bg-color); display: flex; flex-direction: column; border-right: 1px solid var(--border-color); height: 100%; overflow: hidden; }
.panel-header { font-size: 11px; font-weight: 600; color: var(--text-muted); padding: 10px 15px; display: flex; justify-content: space-between; align-items: center; text-transform: uppercase; }
.panel-actions { display: flex; gap: 4px; }
.icon-btn { background: transparent; border: none; outline: none; color: var(--text-muted); cursor: pointer; border-radius: 4px; padding: 4px; display: flex; align-items: center; }
.icon-btn:hover { background-color: var(--surface-color); color: var(--text-main); }
.tree-container { flex: 1; overflow-y: auto; }
.tree-root-name { font-size: 11px; font-weight: bold; padding: 5px 15px; color: var(--text-muted); user-select: none; }
.file-tree-root { list-style: none; padding: 0; margin: 0; }
.create-item { display: flex; align-items: center; padding-right: 10px; }
.create-input { width: 100%; background: var(--surface-color); border: 1px solid var(--primary); color: var(--text-main); font-size: 13px; padding: 2px 6px; outline: none; border-radius: 2px; }
</style>