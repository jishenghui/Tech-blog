<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppIcon from '../components/AppIcon.vue'
import { api, ApiError } from '../lib/api'
import type {
  ArticleSummary,
  DirectoryNode,
  ImportJobView,
  Page,
  ProjectSummary,
  RecalcResult,
  TagView,
  TodoView,
} from '../lib/types'
import { useToastStore } from '../stores/toast'

const toast = useToastStore()
const loading = ref(true)
const recalculating = ref(false)
const stats = reactive({
  articles: 0,
  published: 0,
  drafts: 0,
  directories: 0,
  tags: 0,
  projects: 0,
  todos: 0,
  pendingTodos: 0,
  imports: 0,
})
const recentArticles = ref<ArticleSummary[]>([])
const recentImports = ref<ImportJobView[]>([])

function flattenDirectories(nodes: DirectoryNode[]): DirectoryNode[] {
  return nodes.flatMap((node) => [node, ...flattenDirectories(node.children ?? [])])
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const [articles, published, drafts, directories, tags, projects, todos, imports] =
      await Promise.all([
        api.get<Page<ArticleSummary>>('/api/v1/admin/articles?page=1&page_size=5'),
        api.get<Page<ArticleSummary>>('/api/v1/admin/articles?status=published&page=1&page_size=1'),
        api.get<Page<ArticleSummary>>('/api/v1/admin/articles?status=draft&page=1&page_size=1'),
        api.get<DirectoryNode[]>('/api/v1/admin/directories'),
        api.get<TagView[]>('/api/v1/admin/tags'),
        api.get<ProjectSummary[]>('/api/v1/admin/projects'),
        api.get<TodoView[]>('/api/v1/admin/todos'),
        api.get<Page<ImportJobView>>('/api/v1/admin/imports?page=1&page_size=5'),
      ])
    stats.articles = articles.total
    stats.published = published.total
    stats.drafts = drafts.total
    stats.directories = flattenDirectories(directories).length
    stats.tags = tags.length
    stats.projects = projects.length
    stats.todos = todos.length
    stats.pendingTodos = todos.filter((todo) => !todo.done).length
    stats.imports = imports.total
    recentArticles.value = articles.items
    recentImports.value = imports.items
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '加载概览失败')
  } finally {
    loading.value = false
  }
}

async function recalc(): Promise<void> {
  recalculating.value = true
  try {
    const result = await api.post<RecalcResult>('/api/v1/admin/maintenance/recalc')
    toast.success(
      `重算完成：${result.articles} 篇文章、${result.directories} 个目录、${result.tags} 个标签`,
    )
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '重算失败')
  } finally {
    recalculating.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page-stack">
    <section class="page-heading">
      <div>
        <p class="eyebrow">OVERVIEW</p>
        <h2>内容概览</h2>
        <p>文章、目录、标签、固定页与导入任务的整体状态。</p>
      </div>
      <div class="heading-actions">
        <RouterLink class="button primary" to="/articles/new">
          <AppIcon name="plus" /> 新建文章
        </RouterLink>
        <RouterLink class="button" to="/imports">
          <AppIcon name="imports" /> ZIP 导入
        </RouterLink>
      </div>
    </section>

    <div v-if="loading" class="loading-block">加载中…</div>
    <template v-else>
      <section class="stat-grid">
        <RouterLink class="stat-card" to="/articles">
          <span class="stat-icon"><AppIcon name="articles" :size="20" /></span>
          <span class="stat-label">文章</span>
          <strong>{{ stats.articles }}</strong>
          <span class="stat-hint">{{ stats.published }} 已发布 · {{ stats.drafts }} 草稿</span>
        </RouterLink>
        <RouterLink class="stat-card" to="/directories">
          <span class="stat-icon"><AppIcon name="directories" :size="20" /></span>
          <span class="stat-label">目录</span>
          <strong>{{ stats.directories }}</strong>
          <span class="stat-hint">含固定页面与二级目录</span>
        </RouterLink>
        <RouterLink class="stat-card" to="/tags">
          <span class="stat-icon"><AppIcon name="tags" :size="20" /></span>
          <span class="stat-label">标签</span>
          <strong>{{ stats.tags }}</strong>
          <span class="stat-hint">技术栈页按标签聚合</span>
        </RouterLink>
        <RouterLink class="stat-card" to="/projects">
          <span class="stat-icon"><AppIcon name="projects" :size="20" /></span>
          <span class="stat-label">项目</span>
          <strong>{{ stats.projects }}</strong>
          <span class="stat-hint">含调试流程与架构图</span>
        </RouterLink>
        <RouterLink class="stat-card" to="/todos">
          <span class="stat-icon"><AppIcon name="todos" :size="20" /></span>
          <span class="stat-label">待办</span>
          <strong>{{ stats.todos }}</strong>
          <span class="stat-hint">{{ stats.pendingTodos }} 项未完成</span>
        </RouterLink>
        <RouterLink class="stat-card" to="/imports">
          <span class="stat-icon"><AppIcon name="imports" :size="20" /></span>
          <span class="stat-label">导入任务</span>
          <strong>{{ stats.imports }}</strong>
          <span class="stat-hint">ZIP 导入历史</span>
        </RouterLink>
      </section>

      <section class="panel-grid">
        <div class="panel">
          <div class="panel-head">
            <div>
              <p class="eyebrow">RECENT POSTS</p>
              <h3>最近文章</h3>
            </div>
            <RouterLink class="text-link" to="/articles">全部文章 <span>→</span></RouterLink>
          </div>
          <div v-if="recentArticles.length === 0" class="empty-block">还没有文章。</div>
          <ul v-else class="simple-list">
            <li v-for="article in recentArticles" :key="article.id">
              <RouterLink :to="`/articles/${article.id}`">{{ article.title }}</RouterLink>
              <span class="status-pill" :class="article.status">{{ article.status === 'published' ? '已发布' : '草稿' }}</span>
            </li>
          </ul>
        </div>

        <div class="panel">
          <div class="panel-head">
            <div>
              <p class="eyebrow">MAINTENANCE</p>
              <h3>维护</h3>
            </div>
          </div>
          <p class="panel-note">重算文章派生字段、目录/标签计数与资源引用。</p>
          <button class="button" type="button" :disabled="recalculating" @click="recalc">
            <AppIcon name="refresh" />
            {{ recalculating ? '重算中…' : '重算冗余字段' }}
          </button>
          <div class="panel-subsection">
            <p class="eyebrow">RECENT IMPORTS</p>
            <div v-if="recentImports.length === 0" class="empty-block compact">暂无导入任务。</div>
            <ul v-else class="simple-list compact">
              <li v-for="job in recentImports" :key="job.id">
                <RouterLink :to="`/imports?job=${job.id}`">{{ job.original_filename }}</RouterLink>
                <span class="status-pill" :class="job.status">{{ job.status }}</span>
              </li>
            </ul>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>
