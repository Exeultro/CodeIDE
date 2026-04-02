<template>
  <div class="editor-container" ref="editorRef"></div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue';
import * as monaco from 'monaco-editor';
import { useSessionStore } from '@/store/session.store';
import { useSettingsStore } from '@/store/settings.store';
import { useAuthStore } from '@/store/auth.store';
import { yjsService } from '@/services/yjs.provider';

const props = defineProps<{
  initialContent: string;
  language: string;
}>();

const sessionStore = useSessionStore();
const settingsStore = useSettingsStore();
const authStore = useAuthStore();

const editorRef = ref<HTMLElement | null>(null);
let editorInstance: monaco.editor.IStandaloneCodeEditor | null = null;

onMounted(() => {
  if (!editorRef.value) return;

  // 1. Создаем редактор Monaco
  editorInstance = monaco.editor.create(editorRef.value, {
    value: props.initialContent,
    language: props.language,
    theme: settingsStore.theme,
    fontSize: settingsStore.fontSize,
    lineHeight: settingsStore.lineHeight,
    automaticLayout: true,
    minimap: { enabled: false },
    fixedOverflowWidgets: true, // Чтобы курсоры коллег не обрезались
  });

  // 2. Инициализируем Yjs СЕРВИС
  if (sessionStore.currentSession && sessionStore.activeFilePath) {
    yjsService.setup({
      sessionId: sessionStore.currentSession.id,
      filePath: sessionStore.activeFilePath,
      userId: authStore.userId || 'anon',
      username: authStore.username || 'Anonymous',
      editorInstance: editorInstance,
      initialContent: props.initialContent,
      // Убедись, что этот порт (1234) открыт на сервере!
      serverUrl: 'ws://localhost:1234'
    });
  }

  // 3. Следим за изменениями текста (для обычного сохранения в БД)
  editorInstance.onDidChangeModelContent((event) => {
    if (!event.isFlush) {
      sessionStore.updateEditorContent(editorInstance?.getValue() || '');
    }
  });
});

// Следим за темой
watch(() => settingsStore.theme, (newTheme) => {
  monaco.editor.setTheme(newTheme);
});

onBeforeUnmount(() => {
  // Уничтожаем Yjs и редактор при закрытии вкладки/файла
  yjsService.destroy();
  if (editorInstance) {
    editorInstance.dispose();
  }
});
</script>

<style scoped>
.editor-container {
  width: 100%;
  height: 100%;
}

/* Стили для курсоров коллег (y-monaco их создает автоматически) */
:deep(.yRemoteSelection) {
  background-color: rgba(217, 37, 37, 0.3);
}
:deep(.yRemoteSelectionHead) {
  position: absolute;
  border-left: 2px solid var(--primary);
  border-top: 2px solid var(--primary);
  border-bottom: 2px solid var(--primary);
  height: 100%;
  box-sizing: border-box;
}
:deep(.yRemoteSelectionHead::after) {
  content: attr(data-username);
  position: absolute;
  top: -14px;
  left: -2px;
  background: var(--primary);
  color: white;
  font-size: 10px;
  padding: 0 4px;
  border-radius: 2px;
  white-space: nowrap;
  font-family: sans-serif;
}
</style>