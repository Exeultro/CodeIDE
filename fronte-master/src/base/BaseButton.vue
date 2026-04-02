<template>
  <button
      class="btn"
      :class="[`btn-${variant}`, { 'is-disabled': disabled }]"
      :disabled="disabled"
      @click="$emit('click', $event)"
  >
    <slot />
  </button>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'primary' | 'outline' | 'flat';
  disabled?: boolean;
}>(), {
  variant: 'primary',
  disabled: false
});

defineEmits(['click']);
</script>

<style scoped>
.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  pointer-events: none;
}
</style>