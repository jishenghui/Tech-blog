import { defineStore } from 'pinia'
import { ref } from 'vue'

export type ToastType = 'success' | 'error' | 'info'

export interface ToastMessage {
  id: number
  type: ToastType
  text: string
}

export const useToastStore = defineStore('toast', () => {
  const messages = ref<ToastMessage[]>([])
  let nextId = 1

  function push(text: string, type: ToastType = 'info', duration = 4200): void {
    const id = nextId++
    messages.value.push({ id, type, text })
    window.setTimeout(() => {
      messages.value = messages.value.filter((message) => message.id !== id)
    }, duration)
  }

  return {
    messages,
    push,
    success: (text: string) => push(text, 'success'),
    error: (text: string) => push(text, 'error'),
    info: (text: string) => push(text, 'info'),
  }
})
