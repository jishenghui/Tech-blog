<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  page: number
  total: number
  pageSize: number
}>()

const emit = defineEmits<{
  change: [page: number]
}>()

const pages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
const visiblePages = computed(() =>
  Array.from({ length: pages.value }, (_, index) => index + 1).filter(
    (value) => value === 1 || value === pages.value || Math.abs(value - props.page) <= 2,
  ),
)

function hasGap(index: number): boolean {
  return index > 0 && visiblePages.value[index] - visiblePages.value[index - 1] > 1
}
</script>

<template>
  <div class="table-footer">
    <span>共 {{ total }} 条 · 第 {{ page }} / {{ pages }} 页</span>
    <div class="pager">
      <button type="button" :disabled="page <= 1" @click="emit('change', page - 1)">←</button>
      <template v-for="(value, index) in visiblePages" :key="value">
        <span v-if="hasGap(index)">…</span>
        <button
          type="button"
          :class="{ current: value === page }"
          @click="value !== page && emit('change', value)"
        >
          {{ value }}
        </button>
      </template>
      <button type="button" :disabled="page >= pages" @click="emit('change', page + 1)">→</button>
    </div>
  </div>
</template>
