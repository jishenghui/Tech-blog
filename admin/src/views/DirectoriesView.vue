<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppIcon from '../components/AppIcon.vue'
import PageModal from '../components/PageModal.vue'
import { api, ApiError } from '../lib/api'
import type { ArticleSummary, DirectoryNode, Page } from '../lib/types'
import { useToastStore } from '../stores/toast'

const toast = useToastStore()
const loading = ref(true)
const tree = ref<DirectoryNode[]>([])
const articles = ref<ArticleSummary[]>([])
const draggedDirectoryId = ref<number | null>(null)
const draggedArticleId = ref<number | null>(null)

const modal = reactive({
  open: false,
  mode: 'top' as 'top' | 'child' | 'edit',
  id: 0,
  parentId: null as number | null,
  title: '',
  name: '',
  slug: '',
  visible: true,
  description: '',
  fixed: false,
})

const articlesByDirectory = computed(() => {
  const map = new Map<number, ArticleSummary[]>()
  for (const article of articles.value) {
    if (article.directory_id) {
      const list = map.get(article.directory_id) ?? []
      list.push(article)
      map.set(article.directory_id, list)
    }
  }
  return map
})

async function loadArticles(): Promise<void> {
  const first = await api.get<Page<ArticleSummary>>('/api/v1/admin/articles?page=1&page_size=100')
  const pages = Math.ceil(first.total / 100)
  const rest = await Promise.all(
    Array.from({ length: Math.max(0, pages - 1) }, (_, index) =>
      api.get<Page<ArticleSummary>>(`/api/v1/admin/articles?page=${index + 2}&page_size=100`),
    ),
  )
  articles.value = first.items.concat(...rest.map((page) => page.items))
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const [directoryTree] = await Promise.all([
      api.get<DirectoryNode[]>('/api/v1/admin/directories'),
      loadArticles(),
    ])
    tree.value = directoryTree
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '加载目录失败')
  } finally {
    loading.value = false
  }
}

function openCreateTop(): void {
  modal.open = true
  modal.mode = 'top'
  modal.title = '新建一级目录'
  modal.id = 0
  modal.parentId = null
  modal.name = ''
  modal.slug = ''
  modal.visible = true
  modal.description = ''
  modal.fixed = false
}

function openCreateChild(parent: DirectoryNode): void {
  modal.open = true
  modal.mode = 'child'
  modal.title = `在「${parent.name}」下新建二级目录`
  modal.id = 0
  modal.parentId = parent.id
  modal.name = ''
  modal.slug = ''
  modal.visible = true
  modal.description = ''
  modal.fixed = false
}

function openEdit(directory: DirectoryNode): void {
  modal.open = true
  modal.mode = 'edit'
  modal.title = `编辑「${directory.name}」`
  modal.id = directory.id
  modal.parentId = directory.parent_id
  modal.name = directory.name
  modal.slug = directory.slug
  modal.visible = directory.visible
  modal.description = directory.description
  modal.fixed = directory.is_fixed
}

async function saveDirectory(): Promise<void> {
  const name = modal.name.trim()
  const slug = modal.slug.trim()
  if (!name || (!modal.fixed && !slug)) {
    toast.error('名称和 slug 不能为空')
    return
  }
  try {
    if (modal.mode === 'edit') {
      const payload: Record<string, unknown> = {
        visible: modal.visible,
        description: modal.description,
      }
      if (!modal.fixed) {
        payload.name = name
        payload.slug = slug
      }
      await api.put(`/api/v1/admin/directories/${modal.id}`, payload)
      toast.success('目录已更新')
    } else {
      const payload: Record<string, unknown> = { name, slug }
      if (modal.mode === 'child' && modal.parentId) {
        payload.parent_id = modal.parentId
      }
      await api.post('/api/v1/admin/directories', payload)
      toast.success('目录已创建')
    }
    modal.open = false
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '保存目录失败')
  }
}

async function dissolve(directory: DirectoryNode): Promise<void> {
  const message =
    directory.parent_id === null
      ? `解散一级目录「${directory.name}」？其二级目录会归入「其他」。`
      : `解散二级目录「${directory.name}」？其文章会归入「其他/其他」。`
  if (!window.confirm(message)) {
    return
  }
  try {
    await api.post(`/api/v1/admin/directories/${directory.id}/dissolve`)
    toast.success('目录已解散')
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '解散失败')
  }
}

async function reorderTop(index: number, delta: number): Promise<void> {
  const target = index + delta
  if (target < 0 || target >= tree.value.length) {
    return
  }
  const ordered = [...tree.value]
  ;[ordered[index], ordered[target]] = [ordered[target], ordered[index]]
  try {
    await api.post('/api/v1/admin/directories/reorder', {
      items: ordered.map((item, order) => ({ id: item.id, parent_id: null, sort_order: order })),
    })
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '排序失败')
  }
}

async function reorderChild(top: DirectoryNode, index: number, delta: number): Promise<void> {
  const children = top.children ?? []
  const target = index + delta
  if (target < 0 || target >= children.length) {
    return
  }
  const ordered = [...children]
  ;[ordered[index], ordered[target]] = [ordered[target], ordered[index]]
  try {
    await api.post('/api/v1/admin/directories/reorder', {
      items: ordered.map((item, order) => ({
        id: item.id,
        parent_id: top.id,
        sort_order: order,
      })),
    })
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '排序失败')
  }
}

function onDirectoryDragStart(event: DragEvent, directory: DirectoryNode): void {
  draggedDirectoryId.value = directory.id
  draggedArticleId.value = null
  event.dataTransfer?.setData('text/plain', `directory:${directory.id}`)
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
}

function onArticleDragStart(event: DragEvent, article: ArticleSummary): void {
  draggedArticleId.value = article.id
  draggedDirectoryId.value = null
  event.dataTransfer?.setData('text/plain', `article:${article.id}`)
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
}

function clearDrag(): void {
  draggedDirectoryId.value = null
  draggedArticleId.value = null
}

async function dropOnTop(top: DirectoryNode): Promise<void> {
  const directoryId = draggedDirectoryId.value
  clearDrag()
  if (!directoryId || top.kind !== 'container') {
    return
  }
  try {
    await api.post(`/api/v1/admin/directories/${directoryId}/move`, { parent_id: top.id })
    toast.success('目录已移动')
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '移动目录失败')
  }
}

async function dropOnChild(child: DirectoryNode, top: DirectoryNode): Promise<void> {
  const directoryId = draggedDirectoryId.value
  const articleId = draggedArticleId.value
  clearDrag()

  // 目录拖到组内任意位置：视为移动到该一级目录下。
  if (directoryId) {
    if (top.kind !== 'container') {
      return
    }
    try {
      await api.post(`/api/v1/admin/directories/${directoryId}/move`, { parent_id: top.id })
      toast.success('目录已移动')
      await load()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : '移动目录失败')
    }
    return
  }

  if (!articleId) {
    return
  }
  try {
    await api.post(`/api/v1/admin/articles/${articleId}/move`, { directory_id: child.id })
    toast.success('文章已移动')
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '移动文章失败')
  }
}

onMounted(load)
</script>

<template>
  <div class="page-stack">
    <section class="page-heading">
      <div>
        <p class="eyebrow">STRUCTURE</p>
        <h2>目录</h2>
        <p>固定页面不能改名/解散；二级目录可拖到其他一级目录，文章可拖到其他二级目录。</p>
      </div>
      <div class="heading-actions">
        <button class="button primary" type="button" @click="openCreateTop">
          <AppIcon name="plus" /> 新建一级目录
        </button>
      </div>
    </section>

    <div v-if="loading" class="loading-block">加载目录…</div>
    <div v-else-if="tree.length === 0" class="empty-block">还没有目录。</div>
    <div v-else class="directory-list">
      <section
        v-for="(top, topIndex) in tree"
        :key="top.id"
        class="directory-group"
        @dragover.prevent
        @drop.prevent="dropOnTop(top)"
      >
        <header class="directory-group-head">
          <span class="status-pill" :class="top.kind">{{ top.kind }}</span>
          <div>
            <div class="directory-name">
              {{ top.name }}
              <span v-if="top.is_fixed" class="directory-slug">固定</span>
            </div>
            <div class="directory-slug">/{{ top.slug }}/ · {{ top.article_count }} 已发布 · {{ top.draft_count }} 草稿</div>
          </div>
          <div class="directory-actions">
            <button class="button-icon" type="button" title="上移" :disabled="topIndex === 0" @click="reorderTop(topIndex, -1)">
              <AppIcon name="up" :size="14" />
            </button>
            <button class="button-icon" type="button" title="下移" :disabled="topIndex === tree.length - 1" @click="reorderTop(topIndex, 1)">
              <AppIcon name="down" :size="14" />
            </button>
            <button v-if="top.kind === 'container'" class="button small" type="button" @click="openCreateChild(top)">
              <AppIcon name="plus" :size="13" /> 二级目录
            </button>
            <button class="button-icon" type="button" title="编辑" @click="openEdit(top)">
              <AppIcon name="edit" :size="14" />
            </button>
            <button v-if="!top.is_fixed" class="button-icon danger" type="button" title="解散" @click="dissolve(top)">
              <AppIcon name="trash" :size="14" />
            </button>
          </div>
        </header>

        <div v-if="top.kind === 'page'" class="directory-child" style="color: var(--muted); font-size: 12px;">
          固定页面，文章之外的内容由对应管理页维护。
        </div>
        <div v-else-if="(top.children ?? []).length === 0" class="directory-child" style="color: var(--muted); font-size: 12px;">
          暂无二级目录。
        </div>
        <div
          v-for="(child, childIndex) in (top.children ?? [])"
          :key="child.id"
          class="directory-child"
          draggable="true"
          @dragstart="onDirectoryDragStart($event, child)"
          @dragend="clearDrag"
          @dragover.prevent
          @drop.stop.prevent="dropOnChild(child, top)"
        >
          <span class="directory-drag" title="拖到其他一级目录">
            <AppIcon name="drag" :size="15" />
          </span>
          <div>
            <div class="directory-name">{{ child.name }}</div>
            <div class="directory-slug">/{{ top.slug }}/{{ child.slug }}/ · {{ child.article_count }} 已发布 · {{ child.draft_count }} 草稿</div>
          </div>
          <div class="directory-actions">
            <button class="button-icon" type="button" title="上移" :disabled="childIndex === 0" @click="reorderChild(top, childIndex, -1)">
              <AppIcon name="up" :size="14" />
            </button>
            <button class="button-icon" type="button" title="下移" :disabled="childIndex === (top.children?.length ?? 1) - 1" @click="reorderChild(top, childIndex, 1)">
              <AppIcon name="down" :size="14" />
            </button>
            <button class="button-icon" type="button" title="编辑" @click="openEdit(child)">
              <AppIcon name="edit" :size="14" />
            </button>
            <button class="button-icon danger" type="button" title="解散" @click="dissolve(child)">
              <AppIcon name="trash" :size="14" />
            </button>
          </div>
          <div
            v-if="(articlesByDirectory.get(child.id) ?? []).length > 0"
            class="article-chips"
          >
            <RouterLink
              v-for="article in (articlesByDirectory.get(child.id) ?? [])"
              :key="article.id"
              class="article-chip"
              :class="article.status"
              :to="`/articles/${article.id}`"
              draggable="true"
              @dragstart.stop="onArticleDragStart($event, article)"
              @dragend="clearDrag"
            >
              {{ article.title }}
            </RouterLink>
          </div>
        </div>
      </section>
    </div>

    <PageModal :open="modal.open" :title="modal.title" @close="modal.open = false">
      <div class="form-grid">
        <label class="field">
          <span>名称</span>
          <input v-model="modal.name" type="text" :disabled="modal.fixed" placeholder="例如：学习" />
        </label>
        <label class="field">
          <span>Slug</span>
          <input v-model="modal.slug" type="text" :disabled="modal.fixed" placeholder="例如：study" />
        </label>
        <label class="field span-2">
          <span>描述</span>
          <textarea v-model="modal.description" placeholder="可选，用于目录页说明" />
        </label>
        <label class="checkbox-field span-2">
          <input v-model="modal.visible" type="checkbox" />
          在导航中显示
        </label>
        <p v-if="modal.fixed" class="field-hint span-2">固定目录不可改名、改 slug 或解散。</p>
      </div>
      <template #actions>
        <button class="button" type="button" @click="modal.open = false">取消</button>
        <button class="button primary" type="button" @click="saveDirectory">保存</button>
      </template>
    </PageModal>
  </div>
</template>
