<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppIcon from '../components/AppIcon.vue'
import PageModal from '../components/PageModal.vue'
import { api, ApiError } from '../lib/api'
import type { TagView } from '../lib/types'
import { useToastStore } from '../stores/toast'

const toast = useToastStore()
const loading = ref(true)
const tags = ref<TagView[]>([])

const modal = reactive({
  open: false,
  mode: 'create' as 'create' | 'edit',
  id: 0,
  name: '',
  slug: '',
  description: '',
})

async function load(): Promise<void> {
  loading.value = true
  try {
    tags.value = await api.get<TagView[]>('/api/v1/admin/tags')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '加载标签失败')
  } finally {
    loading.value = false
  }
}

function openCreate(): void {
  modal.open = true
  modal.mode = 'create'
  modal.id = 0
  modal.name = ''
  modal.slug = ''
  modal.description = ''
}

function openEdit(tag: TagView): void {
  modal.open = true
  modal.mode = 'edit'
  modal.id = tag.id
  modal.name = tag.name
  modal.slug = tag.slug
  modal.description = tag.description ?? ''
}

async function save(): Promise<void> {
  if (!modal.name.trim()) {
    toast.error('标签名称不能为空')
    return
  }
  try {
    if (modal.mode === 'create') {
      await api.post<TagView>('/api/v1/admin/tags', {
        name: modal.name.trim(),
        slug: modal.slug.trim(),
        description: modal.description,
      })
      toast.success('标签已创建')
    } else {
      await api.put<TagView>(`/api/v1/admin/tags/${modal.id}`, {
        name: modal.name.trim(),
        slug: modal.slug.trim(),
        description: modal.description,
      })
      toast.success('标签已更新')
    }
    modal.open = false
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '保存标签失败')
  }
}

async function remove(tag: TagView): Promise<void> {
  if (!window.confirm(`删除标签「${tag.name}」？关联文章会失去该标签。`)) {
    return
  }
  try {
    await api.del(`/api/v1/admin/tags/${tag.id}`)
    toast.success('标签已删除')
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除标签失败')
  }
}

onMounted(load)
</script>

<template>
  <div class="page-stack">
    <section class="page-heading">
      <div>
        <p class="eyebrow">TAGS</p>
        <h2>标签</h2>
        <p>标签用于技术栈页聚合。文章发布/取消发布时计数会同步更新。</p>
      </div>
      <div class="heading-actions">
        <button class="button primary" type="button" @click="openCreate">
          <AppIcon name="plus" /> 新建标签
        </button>
      </div>
    </section>

    <div v-if="loading" class="loading-block">加载标签…</div>
    <div v-else-if="tags.length === 0" class="empty-block">还没有标签。</div>
    <div v-else class="card-grid">
      <article v-for="tag in tags" :key="tag.id" class="entity-card">
        <div class="entity-card-head">
          <div>
            <h3>{{ tag.name }}</h3>
            <span class="directory-slug">#{{ tag.slug }}</span>
          </div>
          <span class="status-pill">{{ tag.article_count }} 篇</span>
        </div>
        <p v-if="tag.description">{{ tag.description }}</p>
        <div class="card-actions">
          <button class="button small" type="button" @click="openEdit(tag)">
            <AppIcon name="edit" :size="13" /> 编辑
          </button>
          <button class="button small danger" type="button" @click="remove(tag)">
            <AppIcon name="trash" :size="13" /> 删除
          </button>
        </div>
      </article>
    </div>

    <PageModal :open="modal.open" :title="modal.mode === 'create' ? '新建标签' : '编辑标签'" @close="modal.open = false">
      <div class="form-grid">
        <label class="field">
          <span>名称</span>
          <input v-model="modal.name" type="text" placeholder="例如：Go语言" />
        </label>
        <label class="field">
          <span>Slug</span>
          <input v-model="modal.slug" type="text" placeholder="留空自动生成" />
        </label>
        <label class="field span-2">
          <span>描述</span>
          <textarea v-model="modal.description" placeholder="可选" />
        </label>
      </div>
      <template #actions>
        <button class="button" type="button" @click="modal.open = false">取消</button>
        <button class="button primary" type="button" @click="save">保存</button>
      </template>
    </PageModal>
  </div>
</template>
