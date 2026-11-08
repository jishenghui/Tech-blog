<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import AssetUploadButton from '../components/AssetUploadButton.vue'
import DirectorySelect from '../components/DirectorySelect.vue'
import TagPicker from '../components/TagPicker.vue'
import VditorEditor from '../components/VditorEditor.vue'
import { api, ApiError } from '../lib/api'
import type { AdminArticleDetail, AssetView, DirectoryNode, TagView } from '../lib/types'
import { useToastStore } from '../stores/toast'

const route = useRoute()
const router = useRouter()
const toast = useToastStore()

const isNew = computed(() => route.name === 'article-new')
const articleId = ref<number | null>(null)
const status = ref<'draft' | 'published'>('draft')
const loading = ref(false)
const saving = ref(false)
const directories = ref<DirectoryNode[]>([])
const tags = ref<TagView[]>([])

const form = reactive({
  title: '',
  slug: '',
  summary: '',
  directory_id: null as number | null,
  cover_url: '',
  is_pinned: false,
  tag_ids: [] as number[],
  content_md: '',
})

let originalDirectoryId: number | null = null

function resetForNew(): void {
  const queryDirectory = Number(route.query.directory_id)
  form.title = ''
  form.slug = ''
  form.summary = ''
  form.directory_id = Number.isFinite(queryDirectory) && queryDirectory > 0 ? queryDirectory : null
  form.cover_url = ''
  form.is_pinned = false
  form.tag_ids = []
  form.content_md = ''
  articleId.value = null
  originalDirectoryId = null
  status.value = 'draft'
}

function fillFromDetail(detail: AdminArticleDetail): void {
  form.title = detail.title
  form.slug = detail.slug
  form.summary = detail.summary
  form.directory_id = detail.directory_id
  form.cover_url = detail.cover_url
  form.is_pinned = detail.is_pinned
  form.tag_ids = detail.tag_ids ?? []
  form.content_md = detail.content_md
  articleId.value = detail.id
  originalDirectoryId = detail.directory_id
  status.value = detail.status === 'published' ? 'published' : 'draft'
}

async function loadMeta(): Promise<void> {
  try {
    const [directoryTree, tagList] = await Promise.all([
      api.get<DirectoryNode[]>('/api/v1/admin/directories'),
      api.get<TagView[]>('/api/v1/admin/tags'),
    ])
    directories.value = directoryTree
    tags.value = tagList
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '加载基础数据失败')
  }
}

async function loadArticle(): Promise<void> {
  if (isNew.value) {
    resetForNew()
    return
  }
  const id = Number(route.params.id)
  if (!Number.isFinite(id) || id <= 0) {
    toast.error('无效的文章 ID')
    return
  }
  loading.value = true
  try {
    const detail = await api.get<AdminArticleDetail>(`/api/v1/admin/articles/${id}`)
    fillFromDetail(detail)
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '加载文章失败')
  } finally {
    loading.value = false
  }
}

async function save(silent = false): Promise<boolean> {
  if (!form.title.trim()) {
    toast.error('文章标题不能为空')
    return false
  }
  if (!form.directory_id) {
    toast.error('请选择二级目录')
    return false
  }

  saving.value = true
  try {
    if (!articleId.value) {
      const created = await api.post<AdminArticleDetail>('/api/v1/admin/articles', {
        directory_id: form.directory_id,
        title: form.title.trim(),
        slug: form.slug.trim(),
        summary: form.summary,
        content_md: form.content_md,
        cover_url: form.cover_url,
        is_pinned: form.is_pinned,
        tag_ids: form.tag_ids,
      })
      fillFromDetail(created)
      await router.replace({ name: 'article-edit', params: { id: created.id } })
      if (!silent) {
        toast.success('已保存草稿')
      }
    } else {
      const updated = await api.put<AdminArticleDetail>(`/api/v1/admin/articles/${articleId.value}`, {
        title: form.title.trim(),
        slug: form.slug.trim() || undefined,
        summary: form.summary,
        content_md: form.content_md,
        cover_url: form.cover_url,
        is_pinned: form.is_pinned,
        tag_ids: form.tag_ids,
      })
      if (form.directory_id !== originalDirectoryId) {
        await api.post(`/api/v1/admin/articles/${articleId.value}/move`, {
          directory_id: form.directory_id,
        })
        originalDirectoryId = form.directory_id
      }
      form.slug = updated.slug
      status.value = updated.status === 'published' ? 'published' : 'draft'
      if (!silent) {
        toast.success('已保存')
      }
    }
    return true
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '保存失败')
    return false
  } finally {
    saving.value = false
  }
}

async function publish(): Promise<void> {
  if (!(await save(true)) || !articleId.value) {
    return
  }
  try {
    await api.post(`/api/v1/admin/articles/${articleId.value}/publish`)
    status.value = 'published'
    toast.success('文章已发布')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '发布失败')
  }
}

async function unpublish(): Promise<void> {
  if (!articleId.value) {
    return
  }
  try {
    await api.post(`/api/v1/admin/articles/${articleId.value}/unpublish`)
    status.value = 'draft'
    toast.success('已取消发布')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '取消发布失败')
  }
}

async function remove(): Promise<void> {
  if (!articleId.value || !window.confirm(`确定删除文章「${form.title}」吗？`)) {
    return
  }
  try {
    await api.del(`/api/v1/admin/articles/${articleId.value}`)
    toast.success('文章已删除')
    router.push('/articles')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除失败')
  }
}

function onCoverUploaded(asset: AssetView): void {
  form.cover_url = asset.url
}

watch(
  () => route.params.id,
  (value) => {
    if (route.name !== 'article-edit') {
      return
    }
    const id = Number(value)
    if (Number.isFinite(id) && id > 0 && id !== articleId.value) {
      loadArticle()
    }
  },
)

watch(
  () => route.name,
  (name, previous) => {
    if (name === 'article-new' && previous !== 'article-new') {
      resetForNew()
    }
  },
)

onMounted(async () => {
  await loadMeta()
  await loadArticle()
})
</script>

<template>
  <div class="page-stack">
    <section class="editor-toolbar">
      <RouterLink class="button ghost" to="/articles">
        <AppIcon name="back" /> 返回文章
      </RouterLink>
      <span class="status-pill" :class="status">{{ status === 'published' ? '已发布' : '草稿' }}</span>
      <span class="spacer"></span>
      <button v-if="status === 'published'" class="button" type="button" :disabled="saving" @click="unpublish">
        取消发布
      </button>
      <button class="button" type="button" :disabled="saving" @click="save()">
        <AppIcon name="save" /> {{ saving ? '保存中…' : '保存草稿' }}
      </button>
      <button class="button primary" type="button" :disabled="saving" @click="publish">
        <AppIcon name="check" /> 发布
      </button>
    </section>

    <div v-if="loading" class="loading-block">加载文章…</div>
    <div v-else class="editor-grid">
      <section class="editor-main">
        <div class="field" style="margin-bottom: 14px;">
          <span>标题</span>
          <input v-model="form.title" class="input-title" type="text" placeholder="文章标题" />
        </div>
        <VditorEditor v-model="form.content_md" :height="640" />
      </section>

      <aside class="editor-side">
        <div class="panel">
          <div class="panel-head">
            <div>
              <p class="eyebrow">PUBLISHING</p>
              <h3>发布设置</h3>
            </div>
          </div>
          <div class="field">
            <span>二级目录</span>
            <DirectorySelect v-model="form.directory_id" :directories="directories" />
            <span v-if="directories.length === 0" class="field-hint">
              还没有目录，先去 <RouterLink to="/directories">目录管理</RouterLink> 创建。
            </span>
          </div>
          <div class="field" style="margin-top: 12px;">
            <span>Slug</span>
            <input v-model="form.slug" type="text" placeholder="留空自动生成" />
            <span class="field-hint">稳定 URL：/posts/{{ form.slug || '…' }}/</span>
          </div>
          <label class="checkbox-field" style="margin-top: 12px;">
            <input v-model="form.is_pinned" type="checkbox" />
            置顶
          </label>
          <div class="form-actions" style="margin-top: 14px;">
            <button class="button danger small" type="button" :disabled="saving" @click="remove">
              <AppIcon name="trash" :size="14" /> 删除
            </button>
          </div>
        </div>

        <div class="panel">
          <div class="panel-head">
            <div>
              <p class="eyebrow">TAGS</p>
              <h3>标签</h3>
            </div>
          </div>
          <TagPicker
            v-model="form.tag_ids"
            :tags="tags"
            allow-create
            @created="(tag) => tags.push(tag)"
          />
        </div>

        <div class="panel">
          <div class="panel-head">
            <div>
              <p class="eyebrow">COVER</p>
              <h3>封面</h3>
            </div>
          </div>
          <div class="field">
            <span>封面 URL</span>
            <input v-model="form.cover_url" type="text" placeholder="https://…" />
          </div>
          <div style="margin-top: 8px;">
            <AssetUploadButton label="上传封面" @uploaded="onCoverUploaded" />
          </div>
          <img v-if="form.cover_url" class="cover-preview" :src="form.cover_url" alt="封面预览" />
        </div>

        <div class="panel">
          <div class="panel-head">
            <div>
              <p class="eyebrow">SUMMARY</p>
              <h3>摘要</h3>
            </div>
          </div>
          <div class="field">
            <span>列表摘要</span>
            <textarea v-model="form.summary" placeholder="留空自动截取正文前 120 字" />
          </div>
        </div>
      </aside>
    </div>
  </div>
</template>
