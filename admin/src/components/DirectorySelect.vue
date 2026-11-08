<script setup lang="ts">
import { computed } from 'vue'
import type { DirectoryNode } from '../lib/types'

const props = defineProps<{
  directories: DirectoryNode[]
  modelValue: number | null
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const options = computed(() =>
  props.directories.flatMap((top) =>
    (top.children ?? []).map((child) => ({
      id: child.id,
      label: `${top.name} / ${child.name}`,
    })),
  ),
)

function onChange(event: Event): void {
  const value = (event.target as HTMLSelectElement).value
  emit('update:modelValue', value ? Number(value) : null)
}
</script>

<template>
  <select :value="modelValue ?? ''" :disabled="disabled" @change="onChange">
    <option value="">选择二级目录</option>
    <option v-for="option in options" :key="option.id" :value="option.id">
      {{ option.label }}
    </option>
  </select>
</template>
