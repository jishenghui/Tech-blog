<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import DirectorySelect from '../components/DirectorySelect.vue'
import PaginationBar from '../components/PaginationBar.vue'
import { api, ApiError } from '../lib/api'
import { formatDate, formatFileSize } from '../lib/format'
import type { DirectoryNode, ImportJobView, Page } from '../lib/types'
import { useToastStore } from '../stores/toast'

const toast = useToastStore()
const route = useRoute()

const loading = ref(true)
const uploading = ref(false)
const directories = ref<DirectoryNode[]>([])
const directoryId = ref<number | null>(null)
const publish = ref(false)
const file = ref<File | null>(null)
const fileInput = ref<HTMLInputElement>()
const currentJob = ref<ImportJobView | null>(null)
const history = ref<ImportJobView[]>([])
const historyTotal = ref(0)
const historyPage = ref(1)
const pageSize = 20

function directoryName(id: number): string {
  for (const top of directories.value) {
    for (const child of top.children ?? []) {
      if (child.id === id) {
        return `${top.name} / ${child.name}`
      }
    }
  }
  return `#${id}`
}

async function loadDirectories(): Promise<void> {
  directories.value = await api.get<DirectoryNode[]>('/api/v1/admin/directories')
}

async function loadHistory(): Promise<void> {
  try {
    const result = await api.get<Page<ImportJobView>>(
      `/api/v1/admin/imports?page=${historyPage.value}&page_size=${pageSize}`,
    )
    history.value = result.items
    historyTotal.value = result.total
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '加载导入历史失败')
  }
}

async function loadJob(id: number): Promise<void> {
  try {
    currentJob.value = await api.get<ImportJobView>(`/api/v1/admin/imports/${id}`)
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '加载导入任务失败')
  }
}

function onFileChange(event: Event): void {
  const target = event.target as HTMLInputElement
  file.value = target.files?.[0] ?? null
}

async function upload(): Promise<void> {
  if (!file.value) {
    toast.error('请选择 ZIP 文件')
    return
  }
  if (!directoryId.value) {
    toast.error('请选择目标二级目录')
    return
  }

  uploading.value = true
  currentJob.value = null
  try {
    const formData = new FormData()
    formData.append('file', file.value)
    formData.append('directory_id', String(directoryId.value))
    formData.append('publish', publish.value ? 'true' : 'false')
    const job = await api.upload<ImportJobView>('/api/v1/admin/imports/zip', formData)
    currentJob.value = job
    if (job.status === 'success') {
      toast.success('导入完成')
    } else {
      toast.error(job.error_message || '导入失败')
    }
    if (fileInput.value) {
      fileInput.value.value = ''
    }
    file.value = null
    await loadHistory()
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '上传 ZIP 失败')
  } finally {
    uploading.value = false
  }
}

function changeHistoryPage(page: number): void {
  historyPage.value = page
  loadHistory()
}

onMounted(async () => {
  loading.value = true
  try {
    await Promise.all([loadDirectories(), loadHistory()])
    const jobId = Number(route.query.job)
    if (Number.isFinite(jobId) && jobId > 0) {
      await loadJob(jobId)
    }
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '加载导入页面失败')
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="page-stack">
    <section class="page-heading">
      <div>
        <p class="eyebrow">IMPORT</p>
        <h2>ZIP 导入</h2>
        <p>解压后根层只允许 images/ 与 .md；图片会上传七牛并替换 Markdown 路径。</p>
      </div>
    </section>

    <div v-if="loading" class="loading-block">加载导入页面…</div>
    <template v-else>
      <section class="panel">
        <div class="upload-box">
          <div class="form-grid">
            <label class="field">
              <span>目标二级目录</span>
              <DirectorySelect v-model="directoryId" :directories="directories" />
            </label>
            <label class="field">
              <span>ZIP 文件</span>
              <input ref="fileInput" type="file" accept=".zip,application/zip" @change="onFileChange" />
            </label>
            <label class="checkbox-field span-2">
              <input v-model="publish" type="checkbox" />
              导入后直接发布文章
            </label>
          </div>
          <div class="form-actions" style="justify-content: flex-start; border-top: 0; padding-top: 0;">
            <button class="button primary" type="button" :disabled="uploading" @click="upload">
              <AppIcon name="imports" />
              {{ uploading ? '导入中（同步执行，可能需要几十秒）…' : '开始导入' }}
            </button>
            <span v-if="file" class="field-hint">{{ file.name }} · {{ formatFileSize(file.size) }}</span>
          </div>
        </div>
      </section>

      <section v-if="currentJob" class="panel import-result">
        <div class="panel-head">
          <div>
            <p class="eyebrow">RESULT</p>
            <h3>{{ currentJob.original_filename }}</h3>
          </div>
          <span class="status-pill" :class="currentJob.status">{{ currentJob.status }}</span>
        </div>
        <div class="import-stats">
          <span class="import-stat">Markdown<strong>{{ currentJob.md_count }}</strong></span>
          <span class="import-stat">图片<strong>{{ currentJob.image_count }}</strong></span>
          <span class="import-stat">替换路径<strong>{{ currentJob.replaced_count }}</strong></span>
          <span class="import-stat">解压根<strong>{{ currentJob.extracted_root || '.' }}</strong></span>
        </div>
        <p v-if="currentJob.error_message" class="form-error">{{ currentJob.error_message }}</p>
        <div v-if="currentJob.meta?.failures?.length" class="mini-card">
          <p class="eyebrow">FAILURES</p>
          <ul class="failure-list">
            <li v-for="(failure, index) in currentJob.meta.failures" :key="index">
              {{ failure.file || '全局' }}：{{ failure.reason }}
            </li>
          </ul>
        </div>
        <div v-if="currentJob.meta?.skipped?.length" class="mini-card">
          <p class="eyebrow">SKIPPED</p>
          <ul class="failure-list" style="color: var(--muted);">
            <li v-for="(item, index) in currentJob.meta.skipped" :key="index">{{ item }}</li>
          </ul>
        </div>
      </section>

      <section class="panel">
        <div class="panel-head">
          <div>
            <p class="eyebrow">HISTORY</p>
            <h3>导入历史</h3>
          </div>
        </div>
        <div v-if="history.length === 0" class="empty-block compact">还没有导入任务。</div>
        <div v-else class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>文件</th>
                <th>目录</th>
                <th>状态</th>
                <th>文章 / 图片</th>
                <th>时间</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="job in history" :key="job.id">
                <td class="title-cell">
                  <a href="#" @click.prevent="loadJob(job.id)">{{ job.original_filename }}</a>
                  <span>{{ formatFileSize(job.file_size) }}</span>
                </td>
                <td>{{ directoryName(job.directory_id) }}</td>
                <td><span class="status-pill" :class="job.status">{{ job.status }}</span></td>
                <td>{{ job.md_count }} / {{ job.image_count }}</td>
                <td>{{ formatDate(job.created_at) }}</td>
                <td>
                  <div class="actions-cell">
                    <button class="button small" type="button" @click="loadJob(job.id)">查看</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <PaginationBar
          v-if="historyTotal > pageSize"
          :page="historyPage"
          :total="historyTotal"
          :page-size="pageSize"
          @change="changeHistoryPage"
        />
      </section>
    </template>
  </div>
</template>
