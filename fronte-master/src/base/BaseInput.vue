<template>
  <div class="form-group">
    <label v-if="label" class="label">{{ label }}</label>

    <textarea
        v-if="type === 'textarea'"
        class="textarea"
        :class="{ 'is-error': error }"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        @input="onInput"
    ></textarea>

    <input
        v-else
        class="input"
        :class="{ 'is-error': error }"
        :type="type"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        @input="onInput"
    />

    <span v-if="error" class="text-muted" style="color: var(--primary); margin-top: 4px; display: block;">
      {{ typeof error === 'string' ? error : 'Ошибка заполнения' }}
    </span>
  </div>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  modelValue: string | number;
  label?: string;
  type?: 'text' | 'password' | 'email' | 'number' | 'textarea';
  placeholder?: string;
  disabled?: boolean;
  error?: boolean | string;
}>(), {
  type: 'text',
  placeholder: '',
  disabled: false,
  error: false
});

const emit = defineEmits(['update:modelValue']);

function onInput(event: Event) {
  const target = event.target as HTMLInputElement | HTMLTextAreaElement;
  emit('update:modelValue', target.value);
}
</script>