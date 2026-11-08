<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import AssetUploadButton from '../components/AssetUploadButton.vue'
import ProjectArchitecturePreview from '../components/ProjectArchitecturePreview.vue'
import VditorEditor from '../components/VditorEditor.vue'
import { api, ApiError } from '../lib/api'
import { fromDateInput, toDateInput } from '../lib/format'
import type {
  AdminProjectDetail,
  AssetView,
  DebugStep,
  ProjectInput,
} from '../lib/types'
import { useToastStore } from '../stores/toast'

type TabKey = 'basic' | 'content' | 'steps' | 'architecture'

const route = useRoute()
const router = useRouter()
const toast = useToastStore()

const isNew = computed(() => route.name === 'project-new')
const projectId = ref<number | null>(null)
const loading = ref(false)
const saving = ref(false)
const activeTab = ref<TabKey>('basic')
const techStackText = ref('')
const sortOrderText = ref('')
const startedDate = ref('')
const endedDate = ref('')

const form = reactive<ProjectInput>({
  title: '',
  slug: '',
  summary: '',
  content_md: '',
  tech_stack: [],
  repo_url: '',
  demo_url: '',
  cover_url: '',
  sort_order: null,
  status: 'published',
  started_at: null,
  ended_at: null,
  debug_steps: [],
  architecture: { nodes: [], edges: [] },
})

const techStackPreview = computed(() =>
  techStackText.value
    .split(/[,，]/)
    .map((item) => item.trim())
    .filter(Boolean),
)

function resetForNew(): void {
  projectId.value = null
  form.title = ''
  form.slug = ''
  form.summary = ''
  form.content_md = ''
  form.tech_stack = []
  form.repo_url = ''
  form.demo_url = ''
  form.cover_url = ''
  form.sort_order = null
  form.status = 'published'
  form.started_at = null
  form.ended_at = null
  form.debug_steps = []
  form.architecture = { nodes: [], edges: [] }
  techStackText.value = ''
  sortOrderText.value = ''
  startedDate.value = ''
  endedDate.value = ''
  activeTab.value = 'basic'
}

function fillFromDetail(detail: AdminProjectDetail): void {
  projectId.value = detail.id
  form.title = detail.title
  form.slug = detail.slug
  form.summary = detail.summary
  form.content_md = detail.content_md
  form.tech_stack = [...detail.tech_stack]
  form.repo_url = detail.repo_url
  form.demo_url = detail.demo_url
  form.cover_url = detail.cover_url
  form.sort_order = detail.sort_order
  form.status = detail.status || 'published'
  form.started_at = detail.started_at
  form.ended_at = detail.ended_at
  form.debug_steps = (detail.debug_steps ?? []).map((step) => ({ ...step }))
  form.architecture = {
    nodes: (detail.architecture?.nodes ?? []).map((node) => ({ ...node })),
    edges: (detail.architecture?.edges ?? []).map((edge) => ({ ...edge })),
  }
  techStackText.value = detail.tech_stack.join(', ')
  sortOrderText.value = String(detail.sort_order ?? '')
  startedDate.value = toDateInput(detail.started_at)
  endedDate.value = toDateInput(detail.ended_at)
}

async function loadProject(): Promise<void> {
  if (isNew.value) {
    resetForNew()
    return
  }
  const id = Number(route.params.id)
  if (!Number.isFinite(id) || id <= 0) {
    toast.error('无效的项目 ID')
    return
  }
  loading.value = true
  try {
    const detail = await api.get<AdminProjectDetail>(`/api/v1/admin/projects/${id}`)
    fillFromDetail(detail)
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '加载项目失败')
  } finally {
    loading.value = false
  }
}

function uniqueId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2, 6)}`
}

function addStep(): void {
  const step: DebugStep = {
    id: uniqueId('step'),
    title: '',
    description: '',
    image_url: '',
    video_url: '',
    video_poster: '',
    code: '',
    code_lang: 'go',
    article_slug: '',
    tag_slug: '',
  }
  form.debug_steps.push(step)
}

function removeStep(index: number): void {
  form.debug_steps.splice(index, 1)
}

function moveStep(index: number, delta: number): void {
  const target = index + delta
  if (target < 0 || target >= form.debug_steps.length) {
    return
  }
  const [step] = form.debug_steps.splice(index, 1)
  form.debug_steps.splice(target, 0, step)
}

function onStepImage(index: number, asset: AssetView): void {
  form.debug_steps[index].image_url = asset.url
}

function onStepPoster(index: number, asset: AssetView): void {
  form.debug_steps[index].video_poster = asset.url
}

function addNode(): void {
  form.architecture.nodes.push({
    id: uniqueId('node'),
    label: '',
    kind: 'infra',
    x: 80,
    y: 80,
    description: '',
    tag_slug: '',
    search_query: '',
  })
}

function removeNode(index: number): void {
  const node = form.architecture.nodes[index]
  if (!node) {
    return
  }
  form.architecture.nodes.splice(index, 1)
  form.architecture.edges = form.architecture.edges.filter(
    (edge) => edge.from !== node.id && edge.to !== node.id,
  )
}

function addEdge(): void {
  const first = form.architecture.nodes[0]?.id ?? ''
  const second = form.architecture.nodes[1]?.id ?? first
  form.architecture.edges.push({ from: first, to: second })
}

function removeEdge(index: number): void {
  form.architecture.edges.splice(index, 1)
}

async function save(): Promise<void> {
  if (!form.title.trim()) {
    toast.error('项目标题不能为空')
    return
  }
  if (form.debug_steps.some((step) => !step.title.trim())) {
    toast.error('调试步骤的标题不能为空')
    return
  }
  const sortOrder = sortOrderText.value.trim() === '' ? null : Number(sortOrderText.value)
  if (sortOrder !== null && !Number.isFinite(sortOrder)) {
    toast.error('排序必须是数字')
    return
  }

  form.tech_stack = [...techStackPreview.value]
  form.sort_order = sortOrder
  form.started_at = fromDateInput(startedDate.value)
  form.ended_at = fromDateInput(endedDate.value)

  saving.value = true
  try {
    let detail: AdminProjectDetail
    if (!projectId.value) {
      detail = await api.post<AdminProjectDetail>('/api/v1/admin/projects', {
        ...form,
        slug: form.slug.trim(),
        title: form.title.trim(),
      })
      fillFromDetail(detail)
      await router.replace({ name: 'project-edit', params: { id: detail.id } })
      toast.success('项目已创建')
    } else {
      detail = await api.put<AdminProjectDetail>(`/api/v1/admin/projects/${projectId.value}`, {
        ...form,
        slug: form.slug.trim(),
        title: form.title.trim(),
      })
      fillFromDetail(detail)
      toast.success('项目已保存')
    }
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '保存项目失败')
  } finally {
    saving.value = false
  }
}

async function removeProject(): Promise<void> {
  if (!projectId.value || !window.confirm(`删除项目「${form.title}」？`)) {
    return
  }
  try {
    await api.del(`/api/v1/admin/projects/${projectId.value}`)
    toast.success('项目已删除')
    router.push('/projects')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '删除项目失败')
  }
}

watch(
  () => route.params.id,
  (value) => {
    if (route.name !== 'project-edit') {
      return
    }
    const id = Number(value)
    if (Number.isFinite(id) && id > 0 && id !== projectId.value) {
      loadProject()
    }
  },
)

watch(
  () => route.name,
  (name, previous) => {
    if (name === 'project-new' && previous !== 'project-new') {
      resetForNew()
    }
  },
)

onMounted(loadProject)
</script>

<template>
  <div class="page-stack">
    <section class="editor-toolbar">
      <RouterLink class="button ghost" to="/projects">
        <AppIcon name="back" /> 返回项目
      </RouterLink>
      <span class="status-pill" :class="form.status">
        {{ form.status === 'published' ? '已发布' : '草稿' }}
      </span>
      <span class="spacer"></span>
      <button v-if="projectId" class="button danger" type="button" :disabled="saving" @click="removeProject">
        <AppIcon name="trash" :size="14" /> 删除
      </button>
      <button class="button primary" type="button" :disabled="saving" @click="save">
        <AppIcon name="save" /> {{ saving ? '保存中…' : '保存项目' }}
      </button>
    </section>

    <div v-if="loading" class="loading-block">加载项目…</div>
    <template v-else>
      <div class="tabs">
        <button class="tab" :class="{ active: activeTab === 'basic' }" type="button" @click="activeTab = 'basic'">基本信息</button>
        <button class="tab" :class="{ active: activeTab === 'content' }" type="button" @click="activeTab = 'content'">正文</button>
        <button class="tab" :class="{ active: activeTab === 'steps' }" type="button" @click="activeTab = 'steps'">
          调试流程 ({{ form.debug_steps.length }})
        </button>
        <button class="tab" :class="{ active: activeTab === 'architecture' }" type="button" @click="activeTab = 'architecture'">
          架构图 ({{ form.architecture.nodes.length }})
        </button>
      </div>

      <section v-if="activeTab === 'basic'" class="panel">
        <div class="form-grid">
          <label class="field">
            <span>标题</span>
            <input v-model="form.title" type="text" placeholder="项目标题" />
          </label>
          <label class="field">
            <span>Slug</span>
            <input v-model="form.slug" type="text" placeholder="/projects/{{ form.slug || 'slug' }}/" />
          </label>
          <label class="field span-2">
            <span>摘要</span>
            <textarea v-model="form.summary" placeholder="项目列表摘要，留空自动截取正文" />
          </label>
          <label class="field span-2">
            <span>技术栈（逗号分隔）</span>
            <input v-model="techStackText" type="text" placeholder="Go, Astro, SQLite" />
            <div v-if="techStackPreview.length" class="chip-list" style="margin-top: 6px;">
              <span v-for="tech in techStackPreview" :key="tech" class="chip">{{ tech }}</span>
            </div>
          </label>
          <label class="field">
            <span>代码仓库</span>
            <input v-model="form.repo_url" type="text" placeholder="https://github.com/…" />
          </label>
          <label class="field">
            <span>在线演示</span>
            <input v-model="form.demo_url" type="text" placeholder="https://…" />
          </label>
          <label class="field span-2">
            <span>封面 URL</span>
            <input v-model="form.cover_url" type="text" placeholder="https://…" />
            <div style="margin-top: 6px;">
              <AssetUploadButton label="上传封面" @uploaded="(asset) => (form.cover_url = asset.url)" />
            </div>
            <img v-if="form.cover_url" class="cover-preview" :src="form.cover_url" alt="封面预览" />
          </label>
          <label class="field">
            <span>状态</span>
            <select v-model="form.status">
              <option value="published">已发布</option>
              <option value="draft">草稿</option>
            </select>
          </label>
          <label class="field">
            <span>左侧排序</span>
            <input v-model="sortOrderText" type="number" placeholder="留空自动追加" />
          </label>
          <label class="field">
            <span>开始时间</span>
            <input v-model="startedDate" type="date" />
          </label>
          <label class="field">
            <span>结束时间</span>
            <input v-model="endedDate" type="date" />
          </label>
        </div>
      </section>

      <section v-else-if="activeTab === 'content'" class="editor-main">
        <VditorEditor v-model="form.content_md" :height="580" placeholder="项目 Markdown 正文…" />
      </section>

      <section v-else-if="activeTab === 'steps'" class="panel">
        <div class="repeater-head">
          <div>
            <p class="eyebrow">DEBUG STEPS</p>
            <h3>调试与排查流程</h3>
          </div>
          <button class="button small" type="button" @click="addStep">
            <AppIcon name="plus" :size="13" /> 添加步骤
          </button>
        </div>
        <div v-if="form.debug_steps.length === 0" class="empty-block compact">还没有调试步骤。</div>
        <div v-else class="step-list">
          <article v-for="(step, index) in form.debug_steps" :key="step.id" class="step-card">
            <div class="step-card-head">
              <strong>步骤 {{ index + 1 }}</strong>
              <span class="directory-slug">{{ step.id }}</span>
              <span class="spacer" style="flex: 1;"></span>
              <button class="button-icon" type="button" title="上移" :disabled="index === 0" @click="moveStep(index, -1)">
                <AppIcon name="up" :size="13" />
              </button>
              <button class="button-icon" type="button" title="下移" :disabled="index === form.debug_steps.length - 1" @click="moveStep(index, 1)">
                <AppIcon name="down" :size="13" />
              </button>
              <button class="button-icon danger" type="button" title="删除" @click="removeStep(index)">
                <AppIcon name="trash" :size="13" />
              </button>
            </div>
            <div class="step-fields">
              <label class="field">
                <span>标题</span>
                <input v-model="step.title" type="text" placeholder="例如：复现 500" />
              </label>
              <label class="field">
                <span>代码语言</span>
                <input v-model="step.code_lang" type="text" placeholder="go" />
              </label>
              <label class="field span-2">
                <span>说明</span>
                <textarea v-model="step.description" placeholder="这一步的排查说明" />
              </label>
              <label class="field">
                <span>图片 URL</span>
                <input v-model="step.image_url" type="text" placeholder="https://…" />
                <AssetUploadButton label="上传图片" @uploaded="(asset) => onStepImage(index, asset)" />
              </label>
              <label class="field">
                <span>视频 URL</span>
                <input v-model="step.video_url" type="text" placeholder="七牛视频 URL（可选）" />
              </label>
              <label class="field">
                <span>视频封面 URL</span>
                <input v-model="step.video_poster" type="text" placeholder="https://…" />
                <AssetUploadButton label="上传封面" @uploaded="(asset) => onStepPoster(index, asset)" />
              </label>
              <label class="field">
                <span>关联文章 slug</span>
                <input v-model="step.article_slug" type="text" placeholder="theme-article" />
              </label>
              <label class="field">
                <span>关联标签 slug</span>
                <input v-model="step.tag_slug" type="text" placeholder="golang" />
              </label>
              <label class="field span-2">
                <span>代码</span>
                <textarea v-model="step.code" class="code-input" placeholder="展示在右侧面板的代码" />
              </label>
            </div>
          </article>
        </div>
      </section>

      <section v-else class="architecture-editor-grid">
        <div class="panel">
          <div class="repeater-head">
            <div>
              <p class="eyebrow">NODES</p>
              <h3>架构节点</h3>
            </div>
            <button class="button small" type="button" @click="addNode">
              <AppIcon name="plus" :size="13" /> 添加节点
            </button>
          </div>
          <div v-if="form.architecture.nodes.length === 0" class="empty-block compact">还没有节点。</div>
          <div v-else class="node-list">
            <article v-for="(node, index) in form.architecture.nodes" :key="node.id" class="node-card">
              <div class="node-card-head">
                <strong>节点 {{ index + 1 }}</strong>
                <span class="directory-slug">{{ node.id }}</span>
                <span style="flex: 1;"></span>
                <button class="button-icon danger" type="button" title="删除" @click="removeNode(index)">
                  <AppIcon name="trash" :size="13" />
                </button>
              </div>
              <div class="step-fields">
                <label class="field">
                  <span>ID</span>
                  <input v-model="node.id" type="text" />
                </label>
                <label class="field">
                  <span>名称</span>
                  <input v-model="node.label" type="text" />
                </label>
                <label class="field">
                  <span>类型</span>
                  <select v-model="node.kind">
                    <option value="frontend">frontend</option>
                    <option value="backend">backend</option>
                    <option value="data">data</option>
                    <option value="infra">infra</option>
                    <option value="admin">admin</option>
                  </select>
                </label>
                <label class="field">
                  <span>X / Y</span>
                  <div style="display: flex; gap: 6px;">
                    <input v-model.number="node.x" type="number" />
                    <input v-model.number="node.y" type="number" />
                  </div>
                </label>
                <label class="field span-2">
                  <span>说明</span>
                  <textarea v-model="node.description" placeholder="该组件在本项目中的作用" />
                </label>
                <label class="field">
                  <span>标签 slug</span>
                  <input v-model="node.tag_slug" type="text" />
                </label>
                <label class="field">
                  <span>搜索词</span>
                  <input v-model="node.search_query" type="text" />
                </label>
              </div>
            </article>
          </div>

          <div class="repeater-head" style="margin-top: 22px;">
            <div>
              <p class="eyebrow">EDGES</p>
              <h3>连线</h3>
            </div>
            <button class="button small" type="button" :disabled="form.architecture.nodes.length < 2" @click="addEdge">
              <AppIcon name="plus" :size="13" /> 添加连线
            </button>
          </div>
          <div v-if="form.architecture.edges.length === 0" class="empty-block compact">还没有连线。</div>
          <div v-else class="edge-list">
            <article v-for="(edge, index) in form.architecture.edges" :key="index" class="edge-card">
              <div class="edge-card-head">
                <strong>连线 {{ index + 1 }}</strong>
                <span style="flex: 1;"></span>
                <button class="button-icon danger" type="button" title="删除" @click="removeEdge(index)">
                  <AppIcon name="trash" :size="13" />
                </button>
              </div>
              <div style="display: flex; gap: 8px; align-items: center;">
                <select v-model="edge.from" style="flex: 1;">
                  <option value="">起点</option>
                  <option v-for="node in form.architecture.nodes" :key="node.id" :value="node.id">{{ node.label || node.id }}</option>
                </select>
                <span>→</span>
                <select v-model="edge.to" style="flex: 1;">
                  <option value="">终点</option>
                  <option v-for="node in form.architecture.nodes" :key="node.id" :value="node.id">{{ node.label || node.id }}</option>
                </select>
              </div>
            </article>
          </div>
        </div>

        <aside>
          <ProjectArchitecturePreview :architecture="form.architecture" />
          <p class="field-hint" style="margin-top: 8px;">
            预览使用 SVG 坐标（viewBox 800×470），节点 x/y 可手动调整。
          </p>
        </aside>
      </section>
    </template>
  </div>
</template>
