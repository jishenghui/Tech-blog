<script setup lang="ts">
import { ref } from 'vue'
import AppIcon from './AppIcon.vue'
import { api, ApiError } from '../lib/api'
import type { AssetView } from '../lib/types'
import { useToastStore } from '../stores/toast'

const props = defineProps<{
  label?: string
  accept?: string
}>()

const emit = defineEmits<{
  uploaded: [asset: AssetView]
}>()

const toast = useToastStore()
const input = ref<HTMLInputElement>()
const uploading = ref(false)

function pick(): void {
  input.value?.click()
}

async function onChange(event: Event): Promise<void> {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  target.value = ''
  if (!file) {
    return
  }
  uploading.value = true
  try {
    const form = new FormData()
    form.append('file', file)
    const asset = await api.upload<AssetView>('/api/v1/admin/assets', form)
    emit('uploaded', asset)
    toast.success('上传成功')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '上传失败')
  } finally {
    uploading.value = false
  }
}
</script>

<template>
  <span>
    <input
      ref="input"
      type="file"
      :accept="accept || 'image/*'"
      hidden
      @change="onChange"
    />
    <button class="button small" type="button" :disabled="uploading" @click="pick">
      <AppIcon name="image" :size="14" />
      {{ uploading ? '上传中…' : (label || '上传图片') }}
    </button>
  </span>
</template>
