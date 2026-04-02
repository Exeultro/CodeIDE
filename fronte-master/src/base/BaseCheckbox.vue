<template>
  <div class="form-group">
    <label class="checkbox-wrapper">
      <input
          type="checkbox"
          class="checkbox-input"
          :checked="modelValue"
          :disabled="disabled"
          @change="onChange"
      />
      <span class="checkbox-box"></span>
      <span v-if="label" class="checkbox-text">{{ label }}</span>
    </label>
  </div>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  modelValue: boolean;
  label?: string;
  disabled?: boolean;
}>(), {
  disabled: false
});

const emit = defineEmits(['update:modelValue']);

function onChange(event: Event) {
  const target = event.target as HTMLInputElement;
  emit('update:modelValue', target.checked);
}
</script>