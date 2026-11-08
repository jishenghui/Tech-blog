<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppIcon from '../components/AppIcon.vue'
import { api, ApiError } from '../lib/api'
import { formatDate } from '../lib/format'
import type { ProjectSummary } from '../lib/types'
import { useToastStore } from '../stores/toast'

const toast = useToastStore()
const loading = ref(true)
const projects = ref<ProjectSummary[]>([])

async function load(): Promise<void> {
  loading.value = true
  try {
    projects.value = await api.get<ProjectSummary[]>('/api/v1/admin/projects')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '加载项目失败')
  } finally {
    loading.value = false
  }
}

async function reorder(index: number, delta: number): Promise<void> {
  const target = index + delta
  if (target < 0 || target >= projects.value.length) {
    return
  }
  const ordered = [...projects.value]
  ;[ordered[index], ordered[target]] = [ordered[target], ordered[index]]
  try {
    await api.post('/api/v1/admin/projects/reorder', {
      items: ordered.map((project, order) => ({ id: project.id, sort_order: order })),
    })
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '排序失败')
  }
}

async function remove(project: ProjectSummary): Promise<void> {
  if (!window.confirm(`删除项目「${project.title}」？`)) {
    return
  }
  try {
    await api.del(`/api/v1/admin/projects/${project.id}`)
    toast.success('项目已删除')
    await load()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除项目失败')
  }
}

onMounted(load)
</script>

<template>
  <div class="page-stack">
    <section class="page-heading">
      <div>
        <p class="eyebrow">PROJECTS</p>
        <h2>项目</h2>
        <p>项目页左侧列表顺序、调试流程、架构图和技术栈都在这里维护。</p>
      </div>
      <div class="heading-actions">
        <RouterLink class="button primary" to="/projects/new">
          <AppIcon name="plus" /> 新建项目
        </RouterLink>
      </div>
    </section>

    <div v-if="loading" class="loading-block">加载项目…</div>
    <div v-else-if="projects.length === 0" class="empty-block">还没有项目。</div>
    <div v-else class="card-grid">
      <article v-for="(project, index) in projects" :key="project.id" class="entity-card">
        <div class="entity-card-head">
          <div>
            <h3>{{ project.title }}</h3>
            <span class="directory-slug">/projects/{{ project.slug }}/</span>
          </div>
          <span class="status-pill" :class="project.status">
            {{ project.status === 'published' ? '已发布' : '草稿' }}
          </span>
        </div>
        <p>{{ project.summary || '暂无摘要' }}</p>
        <div class="chip-list">
          <span v-for="tech in project.tech_stack" :key="tech" class="chip">{{ tech }}</span>
        </div>
        <span class="field-hint">更新于 {{ formatDate(project.updated_at) }}</span>
        <div class="card-actions">
          <RouterLink class="button small" :to="`/projects/${project.id}`">
            <AppIcon name="edit" :size="13" /> 编辑
          </RouterLink>
          <button class="button-icon" type="button" title="上移" :disabled="index === 0" @click="reorder(index, -1)">
            <AppIcon name="up" :size="14" />
          </button>
          <button class="button-icon" type="button" title="下移" :disabled="index === projects.length - 1" @click="reorder(index, 1)">
            <AppIcon name="down" :size="14" />
          </button>
          <button class="button-icon danger" type="button" title="删除" @click="remove(project)">
            <AppIcon name="trash" :size="14" />
          </button>
        </div>
      </article>
    </div>
  </div>
</template>
