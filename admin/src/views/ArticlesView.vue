<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppIcon from '../components/AppIcon.vue'
import DirectorySelect from '../components/DirectorySelect.vue'
import PaginationBar from '../components/PaginationBar.vue'
import { api, ApiError } from '../lib/api'
import { formatDate } from '../lib/format'
import type { ArticleSummary, DirectoryNode, Page } from '../lib/types'
import { useToastStore } from '../stores/toast'

const toast = useToastStore()
const loading = ref(true)
const articles = ref<ArticleSummary[]>([])
const total = ref(0)
const directories = ref<DirectoryNode[]>([])
const filters = reactive({
  directoryId: null as number | null,
  status: '',
  q: '',
  page: 1,
  pageSize: 20,
})

async function loadDirectories(): Promise<void> {
  try {
    directories.value = await api.get<DirectoryNode[]>('/api/v1/admin/directories')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '加载目录失败')
  }
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page: String(filters.page),
      page_size: String(filters.pageSize),
    })
    if (filters.directoryId) {
      params.set('directory_id', String(filters.directoryId))
    }
    if (filters.status) {
      params.set('status', filters.status)
    }
    if (filters.q.trim()) {
      params.set('q', filters.q.trim())
    }
    const result = await api.get<Page<ArticleSummary>>(`/api/v1/admin/articles?${params.toString()}`)
    articles.value = result.items
    total.value = result.total
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '加载文章失败')
  } finally {
    loading.value = false
  }
}

function applyFilters(): void {
  filters.page = 1
  load()
}

function changePage(page: number): void {
  filters.page = page
  load()
}

async function togglePublish(article: ArticleSummary): Promise<void> {
  try {
    if (article.status === 'published') {
      await api.post(`/api/v1/admin/articles/${article.id}/unpublish`)
      toast.success('已取消发布')
    } else {
      await api.post(`/api/v1/admin/articles/${article.id}/publish`)
      toast.success('已发布')
    }
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '操作失败')
  }
}

async function removeArticle(article: ArticleSummary): Promise<void> {
  if (!window.confirm(`确定删除文章「${article.title}」吗？此操作会软删除文章。`)) {
    return
  }
  try {
    await api.del(`/api/v1/admin/articles/${article.id}`)
    toast.success('文章已删除')
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除失败')
  }
}

onMounted(() => {
  loadDirectories()
  load()
})
</script>

<template>
  <div class="page-stack">
    <section class="page-heading">
      <div>
        <p class="eyebrow">POSTS</p>
        <h2>文章</h2>
        <p>草稿、发布、目录与标签管理。发布后触发前台重建（M6 接入 Actions）。</p>
      </div>
      <div class="heading-actions">
        <RouterLink class="button primary" to="/articles/new">
          <AppIcon name="plus" /> 新建文章
        </RouterLink>
      </div>
    </section>

    <section class="panel">
      <div class="table-toolbar">
        <div class="field search-field">
          <span>搜索</span>
          <input v-model="filters.q" type="search" placeholder="标题或正文关键词，回车筛选" @keydown.enter="applyFilters" />
        </div>
        <div class="field">
          <span>目录</span>
          <DirectorySelect
            :directories="directories"
            :model-value="filters.directoryId"
            @update:model-value="(value) => { filters.directoryId = value; applyFilters() }"
          />
        </div>
        <div class="field">
          <span>状态</span>
          <select v-model="filters.status" @change="applyFilters">
            <option value="">全部</option>
            <option value="published">已发布</option>
            <option value="draft">草稿</option>
          </select>
        </div>
        <button class="button" type="button" @click="applyFilters">
          <AppIcon name="search" /> 筛选
        </button>
      </div>

      <div v-if="loading" class="loading-block">加载中…</div>
      <div v-else-if="articles.length === 0" class="empty-block">没有匹配的文章。</div>
      <div v-else class="table-wrap">
        <table class="data-table">
          <thead>
            <tr>
              <th>文章</th>
              <th>目录</th>
              <th>状态</th>
              <th>标签</th>
              <th>更新时间</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="article in articles" :key="article.id">
              <td class="title-cell">
                <RouterLink :to="`/articles/${article.id}`">{{ article.title }}</RouterLink>
                <span>/posts/{{ article.slug }}/</span>
              </td>
              <td>{{ article.directory?.name || '—' }}</td>
              <td>
                <span class="status-pill" :class="article.status">
                  {{ article.status === 'published' ? '已发布' : '草稿' }}
                </span>
              </td>
              <td>
                <div class="chip-list">
                  <span v-for="tag in article.tags" :key="tag.id" class="chip">{{ tag.name }}</span>
                </div>
              </td>
              <td>{{ formatDate(article.updated_at) }}</td>
              <td>
                <div class="actions-cell">
                  <RouterLink class="button-icon" :to="`/articles/${article.id}`" aria-label="编辑">
                    <AppIcon name="edit" :size="15" />
                  </RouterLink>
                  <button class="button small" type="button" @click="togglePublish(article)">
                    {{ article.status === 'published' ? '取消发布' : '发布' }}
                  </button>
                  <button class="button-icon danger" type="button" aria-label="删除" @click="removeArticle(article)">
                    <AppIcon name="trash" :size="15" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <PaginationBar
        v-if="!loading && total > 0"
        :page="filters.page"
        :total="total"
        :page-size="filters.pageSize"
        @change="changePage"
      />
    </section>
  </div>
</template>
