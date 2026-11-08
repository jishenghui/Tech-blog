<script setup lang="ts">
import { ref } from 'vue'
import AppIcon from './AppIcon.vue'
import { api, ApiError } from '../lib/api'
import type { TagView } from '../lib/types'
import { useToastStore } from '../stores/toast'

const props = defineProps<{
  tags: TagView[]
  modelValue: number[]
  allowCreate?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: number[]]
  created: [tag: TagView]
}>()

const toast = useToastStore()
const newName = ref('')
const creating = ref(false)

function toggle(id: number): void {
  const selected = new Set(props.modelValue)
  if (selected.has(id)) {
    selected.delete(id)
  } else {
    selected.add(id)
  }
  emit('update:modelValue', [...selected])
}

async function createTag(): Promise<void> {
  const name = newName.value.trim()
  if (!name) {
    return
  }
  creating.value = true
  try {
    const tag = await api.post<TagView>('/api/v1/admin/tags', { name, slug: '' })
    emit('created', tag)
    emit('update:modelValue', [...props.modelValue, tag.id])
    newName.value = ''
    toast.success(`已创建标签「${tag.name}」`)
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '创建标签失败')
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <div class="tag-picker">
    <span v-if="tags.length === 0" class="tag-picker-empty">还没有标签。</span>
    <button
      v-for="tag in tags"
      :key="tag.id"
      type="button"
      class="chip"
      :class="{ selected: modelValue.includes(tag.id) }"
      @click="toggle(tag.id)"
    >
      {{ tag.name }}
      <span v-if="modelValue.includes(tag.id)">✓</span>
    </button>
    <template v-if="allowCreate">
      <span class="tag-picker-empty">|</span>
      <input
        v-model="newName"
        style="width: 110px; border: 0; outline: none; background: transparent; color: var(--text); font-size: 11px;"
        placeholder="新标签名"
        @keydown.enter.prevent="createTag"
      />
      <button class="chip" type="button" :disabled="creating" @click="createTag">
        <AppIcon name="plus" :size="12" />
        {{ creating ? '创建中' : '新建' }}
      </button>
    </template>
  </div>
</template>
