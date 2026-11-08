<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Vditor from 'vditor'
import 'vditor/dist/index.css'
import { getAccessToken } from '../lib/session'

const props = withDefaults(
  defineProps<{
    modelValue: string
    height?: number
    placeholder?: string
  }>(),
  {
    height: 620,
    placeholder: '在这里写 Markdown…',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const host = ref<HTMLDivElement>()
let editor: Vditor | null = null

onMounted(() => {
  if (!host.value) {
    return
  }
  editor = new Vditor(host.value, {
    height: props.height,
    mode: 'sv',
    value: props.modelValue,
    cdn: '/vditor',
    theme: 'classic',
    icon: 'ant',
    lang: 'zh_CN',
    placeholder: props.placeholder,
    cache: { enable: false },
    counter: { enable: true, type: 'markdown' },
    preview: {
      theme: { current: 'light' },
      hljs: { style: 'github' },
      math: { engine: 'KaTeX' },
    },
    upload: {
      url: '/api/v1/admin/assets',
      fieldName: 'file',
      accept: 'image/*',
      multiple: false,
      setHeaders: () => ({
        Authorization: `Bearer ${getAccessToken()}`,
      }),
      format: (files: File[], responseText: string) => {
        const fileName = files[0]?.name || 'image'
        try {
          const body = JSON.parse(responseText) as {
            code: number
            message?: string
            data?: { url?: string }
          }
          if (body.code !== 0 || !body.data?.url) {
            return JSON.stringify({
              msg: body.message || '图片上传失败',
              code: body.code || 1,
              data: { errFiles: [fileName], succMap: {} },
            })
          }
          return JSON.stringify({
            msg: '',
            code: 0,
            data: { errFiles: [], succMap: { [fileName]: body.data.url } },
          })
        } catch {
          return JSON.stringify({
            msg: '图片上传失败',
            code: 1,
            data: { errFiles: [fileName], succMap: {} },
          })
        }
      },
    },
    input: (value: string) => {
      emit('update:modelValue', value)
    },
  })
})

watch(
  () => props.modelValue,
  (value) => {
    if (editor && editor.getValue() !== value) {
      editor.setValue(value)
    }
  },
)

onBeforeUnmount(() => {
  editor?.destroy()
  editor = null
})
</script>

<template>
  <div ref="host" class="vditor-host"></div>
</template>
