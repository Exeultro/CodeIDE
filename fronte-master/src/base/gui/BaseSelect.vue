<template>
  <div class="strict-cascader" ref="cascaderRef">
    <label v-if="label" class="cascader-label">{{ label }}</label>

    <div
        class="cascader-trigger"
        :class="{ 'is-active': isOpen }"
        @click="toggle"
    >
      <span class="trigger-text" :class="{ 'placeholder': !modelValue }">
        {{ displayLabel || placeholder }}
      </span>

      <div class="trigger-actions">
        <svg
            v-if="modelValue"
            class="clear-icon"
            @click.stop="clearSelection"
            viewBox="0 0 24 24"
        >
          <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z" fill="currentColor"/>
        </svg>

        <svg class="chevron" :class="{ 'rotated': isOpen }" viewBox="0 0 24 24">
          <path d="M7 10l5 5 5-5z" fill="currentColor"/>
        </svg>
      </div>
    </div>

    <transition name="menu-fade">
      <div v-if="isOpen" class="cascader-dropdown" @mouseleave="onDropdownLeave">
        <div class="cascader-columns" ref="columnsWrapper">
          <ul
              v-for="(col, colIdx) in columns"
              :key="colIdx"
              class="cascader-col"
              :class="getColGridClass(col)"
          >
            <!-- Кнопка "Выбрать категорию" для родительских элементов -->
            <li
                v-if="colIdx > 0 && hasValue(renderPath[colIdx - 1]?.value)"
                class="cascader-item is-action"
                @mouseenter="onActionHover(colIdx)"
                @click="selectValue(renderPath[colIdx - 1].value)"
            >
              <span class="action-icon">✓</span>
              Выбрать эту категорию
            </li>

            <!-- Обычные элементы списка -->
            <li
                v-for="opt in col"
                :key="opt.value || opt.label"
                class="cascader-item"
                :class="{
                  'is-selected': visualPath[colIdx]?.value === opt.value,
                  'is-unselectable': isUnselectable(opt)
                }"
                @mouseenter="onItemHover(opt, colIdx)"
                @click="onItemClick(opt, colIdx)"
            >
              <span class="item-label">{{ opt.label }}</span>
              <svg v-if="opt.children && opt.children.length" class="arrow-icon" viewBox="0 0 24 24">
                <path d="M10 6l6 6-6 6z" fill="currentColor"/>
              </svg>
            </li>
          </ul>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue';

const props = defineProps({
  modelValue: { type: [String, Number], default: null },
  options: { type: Array, required: true },
  label: { type: String, default: '' },
  placeholder: { type: String, default: 'Выберите категорию...' }
});

const emit = defineEmits(['update:modelValue']);

const cascaderRef = ref(null);
const columnsWrapper = ref(null);
const isOpen = ref(false);

// АРХИТЕКТУРА 2.0: Разделяем визуал и рендер
const visualPath = ref([]); // Обновляется МГНОВЕННО (для красного фона)
const renderPath = ref([]); // Обновляется с ЗАДЕРЖКОЙ (для отрисовки новых колонок)
let hoverTimer = null;

const hasValue = (val) => val !== null && val !== undefined && val !== '';
const isUnselectable = (opt) => !hasValue(opt.value) && (!opt.children || opt.children.length === 0);

const getColGridClass = (col) => {
  const hasHierarchicalItems = col.some(opt => opt.children && opt.children.length > 0);
  if (hasHierarchicalItems) return 'cols-1';
  if (col.length > 36) return 'is-grid cols-4';
  if (col.length > 24) return 'is-grid cols-3';
  if (col.length > 12) return 'is-grid cols-2';
  return 'cols-1';
};

// Колонки строятся строго на базе renderPath (с задержкой)
const columns = computed(() => {
  const cols =[props.options];
  let currentChildren = props.options;

  for (const node of renderPath.value) {
    const found = currentChildren.find(item => item.value === node.value);
    if (found && found.children && found.children.length > 0) {
      cols.push(found.children);
      currentChildren = found.children;
    } else {
      break;
    }
  }
  return cols;
});

// Наведение на элемент
const onItemHover = (opt, colIdx) => {
  if (isUnselectable(opt)) return;

  clearTimeout(hoverTimer);

  // 1. МГНОВЕННО переключаем визуал (красный фон бегает за мышкой без задержек)
  const newVisual = renderPath.value.slice(0, colIdx);
  newVisual.push(opt);
  visualPath.value = newVisual;

  // 2. С ЗАДЕРЖКОЙ перерисовываем следующую колонку (чтобы не промахиваться по диагонали)
  hoverTimer = setTimeout(() => {
    renderPath.value =[...visualPath.value];
    nextTick(() => {
      if (columnsWrapper.value) {
        columnsWrapper.value.scrollLeft = columnsWrapper.value.scrollWidth;
      }
    });
  }, 100);
};

// Наведение на верхнюю кнопку "Выбрать эту категорию"
const onActionHover = (colIdx) => {
  clearTimeout(hoverTimer);
  visualPath.value = renderPath.value.slice(0, colIdx);
};

// Клик по элементу
const onItemClick = (opt, colIdx) => {
  if (isUnselectable(opt)) return;

  if (!opt.children || opt.children.length === 0) {
    selectValue(opt.value);
  } else {
    // При клике на категорию с детьми — открываем ее моментально (без таймера)
    clearTimeout(hoverTimer);
    const newPath = renderPath.value.slice(0, colIdx);
    newPath.push(opt);
    visualPath.value = newPath;
    renderPath.value = newPath;
  }
};

const selectValue = (val) => {
  emit('update:modelValue', val);
  close();
};

const clearSelection = () => {
  emit('update:modelValue', null);
  visualPath.value =[];
  renderPath.value =[];
  close();
};

const toggle = () => {
  if (!isOpen.value) open();
  else close();
};

const open = () => {
  isOpen.value = true;
  resetPathsToValue();
};

const close = () => {
  isOpen.value = false;
  clearTimeout(hoverTimer);
};

// Увод мышки за пределы всего меню
const onDropdownLeave = () => {
  clearTimeout(hoverTimer);
  resetPathsToValue();
};

// Функция сброса: возвращает меню к сохраненному состоянию или в самое начало
const resetPathsToValue = () => {
  if (props.modelValue) {
    const path = findPathToValue(props.options, props.modelValue) || [];
    visualPath.value = [...path];
    renderPath.value =[...path];
  } else {
    visualPath.value = [];
    renderPath.value =[];
  }
};

const findPathToValue = (opts, targetVal, currentPath =[]) => {
  for (const opt of opts) {
    const path = [...currentPath, opt];
    if (opt.value === targetVal) return path;
    if (opt.children) {
      const found = findPathToValue(opt.children, targetVal, path);
      if (found) return found;
    }
  }
  return null;
};

const displayLabel = computed(() => {
  if (!props.modelValue) return '';
  const path = findPathToValue(props.options, props.modelValue);
  return path ? path.map(p => p.label).join(' / ') : '';
});

const handleClickOutside = (event) => {
  if (cascaderRef.value && !cascaderRef.value.contains(event.target)) {
    close();
  }
};

onMounted(() => document.addEventListener('click', handleClickOutside));
onBeforeUnmount(() => document.removeEventListener('click', handleClickOutside));
</script>

<style scoped>
.strict-cascader {
  position: relative;
  width: 100%;
  max-width: 400px;
  font-family: var(--font-family, 'Inter', sans-serif);
  line-height: 1.5;
}

.cascader-label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-muted);
}

.cascader-trigger {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  padding: 8px 12px;
  min-height: 38px;
  font-size: 14px;
  color: var(--text-main);
  background-color: var(--surface-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  cursor: pointer;
  user-select: none;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.cascader-trigger:hover {
  border-color: #c0c0c0;
}

.cascader-trigger.is-active {
  border-color: var(--primary);
  box-shadow: 0 0 0 1px var(--primary);
}

.trigger-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.trigger-text.placeholder {
  color: var(--text-muted);
}

.trigger-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.clear-icon, .chevron {
  width: 18px;
  height: 18px;
  color: var(--text-muted);
}

.clear-icon {
  cursor: pointer;
  transition: color 0.15s ease;
}

.clear-icon:hover {
  color: var(--primary);
}

.chevron {
  transition: transform 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.chevron.rotated {
  transform: rotate(180deg);
  color: var(--primary);
}

.cascader-dropdown {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: 100;
  background-color: var(--surface-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08), 0 1px 4px rgba(0, 0, 0, 0.04);
  overflow: hidden;
  transform-origin: top center;
}

.cascader-columns {
  display: flex;
  max-width: 900px;
  overflow-x: auto;
  overflow-y: hidden;
}

.cascader-col {
  list-style: none;
  margin: 0;
  padding: 0;
  min-width: 220px;
  max-height: 320px;
  overflow-y: auto;
  border-right: 1px solid var(--border-color);
  background: var(--surface-color);
  animation: col-fade-in 0.15s ease-out forwards;
}

.cascader-col:last-child {
  border-right: none;
}

.cascader-col.is-grid { display: grid; align-content: start; }
.cascader-col.cols-1 { width: 220px; display: block; }
.cascader-col.cols-2 { width: 440px; grid-template-columns: repeat(2, 1fr); }
.cascader-col.cols-3 { width: 660px; grid-template-columns: repeat(3, 1fr); }
.cascader-col.cols-4 { width: 880px; grid-template-columns: repeat(4, 1fr); }

.cascader-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 14px;
  font-size: 14px;
  color: var(--text-main);
  background-color: transparent;
  cursor: pointer;
  user-select: none;
  /* Для фона анимация отключена специально для идеальной отзывчивости */
  transition: color 0.1s ease;
}

/* Единственный источник истины для красного фона. Никаких :hover! */
.cascader-item.is-selected {
  background-color: var(--primary-light);
  color: var(--primary);
  font-weight: 500;
}

.cascader-item.is-action {
  position: sticky;
  top: 0;
  background-color: var(--surface-color);
  font-weight: 600;
  color: var(--primary);
  border-bottom: 1px solid var(--border-color);
  z-index: 10;
  grid-column: 1 / -1;
  transition: background-color 0.15s ease, color 0.15s ease;
}

.cascader-item.is-action:hover {
  background-color: var(--primary);
  color: #FFFFFF;
}

.cascader-item.is-unselectable {
  cursor: default;
  color: var(--text-muted);
}

.item-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.action-icon {
  margin-right: 8px;
  font-size: 14px;
}

.arrow-icon {
  width: 16px;
  height: 16px;
  color: var(--text-muted);
  margin-left: 8px;
  flex-shrink: 0;
}

.cascader-item.is-selected .arrow-icon {
  color: var(--primary);
}

.cascader-col::-webkit-scrollbar,
.cascader-columns::-webkit-scrollbar {
  width: 5px;
  height: 5px;
}
.cascader-col::-webkit-scrollbar-track,
.cascader-columns::-webkit-scrollbar-track {
  background: transparent;
}
.cascader-col::-webkit-scrollbar-thumb,
.cascader-columns::-webkit-scrollbar-thumb {
  background: #d0d0d0;
  border-radius: 4px;
}
.cascader-col:hover::-webkit-scrollbar-thumb {
  background: #a0a0a0;
}

.menu-fade-enter-active,
.menu-fade-leave-active {
  transition: opacity 0.15s ease, transform 0.15s cubic-bezier(0.4, 0, 0.2, 1);
}
.menu-fade-enter-from,
.menu-fade-leave-to {
  opacity: 0;
  transform: scaleY(0.96);
}

@keyframes col-fade-in {
  from { opacity: 0; transform: translateX(-2px); }
  to { opacity: 1; transform: translateX(0); }
}
</style>