<template>
  <div v-if="message" class="base-alert card" :class="type">
    <div class="alert-content">
      <span class="alert-icon">{{ type === 'error' ? '⚠️' : 'ℹ️' }}</span>
      <span class="alert-text">{{ message }}</span>
    </div>
    <button v-if="closable" class="close-btn" @click="$emit('close')">&times;</button>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  message: string;
  type?: 'error' | 'info' | 'success';
  closable?: boolean;
}>(), {
  type: 'error',
  closable: true
});

defineEmits(['close']);
</script>

<style scoped>
.base-alert {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  margin-bottom: 16px;
  border-left: 4px solid var(--border-color);
}

.alert-content {
  display: flex;
  align-items: center;
  gap: 12px;
}

.alert-icon {
  font-size: 18px;
}

.alert-text {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-main);
}

.error {
  background-color: rgba(217, 37, 37, 0.1);
  border-left-color: var(--primary);
}

.error .alert-text {
  color: #ff6b6b;
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: 20px;
  cursor: pointer;
  padding: 0 4px;
  transition: color 0.2s;
}

.close-btn:hover {
  color: var(--text-main);
}
</style>