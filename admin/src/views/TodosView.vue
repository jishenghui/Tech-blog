<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppIcon from '../components/AppIcon.vue'
import PageModal from '../components/PageModal.vue'
import { api, ApiError } from '../lib/api'
import { formatDate, fromDateInput, toDateInput } from '../lib/format'
import type { TodoInput, TodoView } from '../lib/types'
import { useToastStore } from '../stores/toast'

const toast = useToastStore()
const loading = ref(true)
const todos = ref<TodoView[]>([])
const newContent = ref('')
const newDueDate = ref('')
const adding = ref(false)

const modal = reactive({
  open: false,
  id: 0,
  content: '',
  done: false,
  dueDate: '',
})

async function load(): Promise<void> {
  loading.value = true
  try {
    todos.value = await api.get<TodoView[]>('/api/v1/admin/todos')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '加载待办失败')
  } finally {
    loading.value = false
  }
}

async function addTodo(): Promise<void> {
  const content = newContent.value.trim()
  if (!content) {
    toast.error('待办内容不能为空')
    return
  }
  adding.value = true
  try {
    await api.post<TodoView>('/api/v1/admin/todos', {
      content,
      done: false,
      due_date: fromDateInput(newDueDate.value),
    })
    newContent.value = ''
    newDueDate.value = ''
    toast.success('待办已添加')
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '添加待办失败')
  } finally {
    adding.value = false
  }
}

async function toggleDone(todo: TodoView): Promise<void> {
  const payload: TodoInput = {
    content: todo.content,
    done: !todo.done,
    due_date: todo.due_date,
  }
  try {
    await api.put(`/api/v1/admin/todos/${todo.id}`, payload)
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '更新待办失败')
  }
}

function openEdit(todo: TodoView): void {
  modal.open = true
  modal.id = todo.id
  modal.content = todo.content
  modal.done = todo.done
  modal.dueDate = toDateInput(todo.due_date)
}

async function saveEdit(): Promise<void> {
  if (!modal.content.trim()) {
    toast.error('待办内容不能为空')
    return
  }
  try {
    await api.put(`/api/v1/admin/todos/${modal.id}`, {
      content: modal.content.trim(),
      done: modal.done,
      due_date: fromDateInput(modal.dueDate),
    })
    modal.open = false
    toast.success('待办已更新')
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '更新待办失败')
  }
}

async function remove(todo: TodoView): Promise<void> {
  if (!window.confirm(`删除待办「${todo.content}」？`)) {
    return
  }
  try {
    await api.del(`/api/v1/admin/todos/${todo.id}`)
    toast.success('待办已删除')
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除待办失败')
  }
}

async function reorder(index: number, delta: number): Promise<void> {
  const target = index + delta
  if (target < 0 || target >= todos.value.length) {
    return
  }
  const ordered = [...todos.value]
  ;[ordered[index], ordered[target]] = [ordered[target], ordered[index]]
  try {
    await api.post('/api/v1/admin/todos/reorder', {
      items: ordered.map((todo, order) => ({ id: todo.id, sort_order: order })),
    })
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '排序失败')
  }
}

onMounted(load)
</script>

<template>
  <div class="page-stack">
    <section class="page-heading">
      <div>
        <p class="eyebrow">TODOS</p>
        <h2>待办</h2>
        <p>公开待办页会展示这里的清单；完成后仍保留记录。</p>
      </div>
    </section>

    <section class="panel">
      <div class="todo-add">
        <label class="field">
          <span>新待办</span>
          <input v-model="newContent" type="text" placeholder="准备做什么？" @keydown.enter="addTodo" />
        </label>
        <label class="field" style="max-width: 200px;">
          <span>截止日期</span>
          <input v-model="newDueDate" type="date" />
        </label>
        <button class="button primary" type="button" :disabled="adding" @click="addTodo">
          <AppIcon name="plus" /> {{ adding ? '添加中…' : '添加' }}
        </button>
      </div>

      <div v-if="loading" class="loading-block">加载待办…</div>
      <div v-else-if="todos.length === 0" class="empty-block">还没有待办。</div>
      <div v-else class="todo-rows">
        <article v-for="(todo, index) in todos" :key="todo.id" class="todo-row" :class="{ done: todo.done }">
          <input type="checkbox" :checked="todo.done" @change="toggleDone(todo)" />
          <div class="todo-main">
            <div class="todo-text">{{ todo.content }}</div>
            <div v-if="todo.due_date" class="todo-due">截止：{{ formatDate(todo.due_date) }}</div>
          </div>
          <div class="todo-actions">
            <button class="button-icon" type="button" title="上移" :disabled="index === 0" @click="reorder(index, -1)">
              <AppIcon name="up" :size="14" />
            </button>
            <button class="button-icon" type="button" title="下移" :disabled="index === todos.length - 1" @click="reorder(index, 1)">
              <AppIcon name="down" :size="14" />
            </button>
            <button class="button-icon" type="button" title="编辑" @click="openEdit(todo)">
              <AppIcon name="edit" :size="14" />
            </button>
            <button class="button-icon danger" type="button" title="删除" @click="remove(todo)">
              <AppIcon name="trash" :size="14" />
            </button>
          </div>
        </article>
      </div>
    </section>

    <PageModal :open="modal.open" title="编辑待办" @close="modal.open = false">
      <div class="form-grid">
        <label class="field span-2">
          <span>内容</span>
          <textarea v-model="modal.content" />
        </label>
        <label class="field">
          <span>截止日期</span>
          <input v-model="modal.dueDate" type="date" />
        </label>
        <label class="checkbox-field">
          <input v-model="modal.done" type="checkbox" />
          已完成
        </label>
      </div>
      <template #actions>
        <button class="button" type="button" @click="modal.open = false">取消</button>
        <button class="button primary" type="button" @click="saveEdit">保存</button>
      </template>
    </PageModal>
  </div>
</template>
