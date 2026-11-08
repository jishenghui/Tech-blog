<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppIcon from '../components/AppIcon.vue'
import AssetUploadButton from '../components/AssetUploadButton.vue'
import VditorEditor from '../components/VditorEditor.vue'
import { api, ApiError } from '../lib/api'
import type { AdminProfileView, AssetView, ProfileInput } from '../lib/types'
import { useToastStore } from '../stores/toast'

type TabKey = 'basic' | 'resume' | 'awards' | 'experiences' | 'featured'

const toast = useToastStore()
const loading = ref(true)
const saving = ref(false)
const activeTab = ref<TabKey>('basic')
const contactRows = ref<{ key: string; value: string }[]>([])

const form = reactive<ProfileInput>({
  headline: '',
  avatar_url: '',
  resume_md: '',
  awards: [],
  experiences: [],
  featured_projects: [],
  contacts: {},
})

function uniqueId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2, 6)}`
}

function fillFromProfile(profile: AdminProfileView): void {
  form.headline = profile.headline
  form.avatar_url = profile.avatar_url
  form.resume_md = profile.resume_md
  form.awards = (profile.awards ?? []).map((award) => ({ ...award }))
  form.experiences = (profile.experiences ?? []).map((experience) => ({ ...experience }))
  form.featured_projects = (profile.featured_projects ?? []).map((project) => ({ ...project }))
  form.contacts = { ...(profile.contacts ?? {}) }
  contactRows.value = Object.entries(form.contacts).map(([key, value]) => ({
    key,
    value: String(value),
  }))
  if (contactRows.value.length === 0) {
    contactRows.value = [{ key: 'email', value: '' }]
  }
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const profile = await api.get<AdminProfileView>('/api/v1/admin/profile')
    fillFromProfile(profile)
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '加载首页资料失败')
  } finally {
    loading.value = false
  }
}

function moveItem<T>(list: T[], index: number, delta: number): void {
  const target = index + delta
  if (target < 0 || target >= list.length) {
    return
  }
  const [item] = list.splice(index, 1)
  list.splice(target, 0, item)
}

function addAward(): void {
  form.awards.push({
    id: uniqueId('award'),
    title: '',
    image_url: '',
    date: '',
    description: '',
  })
}

function addExperience(): void {
  form.experiences.push({ company: '', role: '', period: '', description: '' })
}

function addFeatured(): void {
  form.featured_projects.push({ name: '', summary: '', url: '', cover_url: '' })
}

function addContact(): void {
  contactRows.value.push({ key: '', value: '' })
}

function onAwardImage(index: number, asset: AssetView): void {
  form.awards[index].image_url = asset.url
}

function onFeaturedCover(index: number, asset: AssetView): void {
  form.featured_projects[index].cover_url = asset.url
}

async function save(): Promise<void> {
  if (form.awards.some((award) => !award.title.trim())) {
    toast.error('奖项标题不能为空')
    return
  }
  if (form.experiences.some((experience) => !experience.company.trim() || !experience.role.trim())) {
    toast.error('经历的公司和职位不能为空')
    return
  }
  if (form.featured_projects.some((project) => !project.name.trim())) {
    toast.error('推荐项目名称不能为空')
    return
  }

  const contacts: Record<string, string> = {}
  for (const row of contactRows.value) {
    const key = row.key.trim()
    if (key) {
      contacts[key] = row.value
    }
  }
  form.contacts = contacts

  saving.value = true
  try {
    const profile = await api.put<AdminProfileView>('/api/v1/admin/profile', { ...form })
    fillFromProfile(profile)
    toast.success('首页资料已保存')
  } catch (err) {
    toast.error(err instanceof ApiError ? err.message : '保存首页资料失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page-stack">
    <section class="editor-toolbar">
      <div>
        <p class="eyebrow" style="margin-bottom: 2px;">PROFILE</p>
        <h2 style="margin: 0; font-size: 22px; letter-spacing: -0.03em;">首页资料</h2>
      </div>
      <span class="spacer"></span>
      <button class="button primary" type="button" :disabled="saving || loading" @click="save">
        <AppIcon name="save" /> {{ saving ? '保存中…' : '保存首页资料' }}
      </button>
    </section>

    <div v-if="loading" class="loading-block">加载首页资料…</div>
    <template v-else>
      <div class="tabs">
        <button class="tab" :class="{ active: activeTab === 'basic' }" type="button" @click="activeTab = 'basic'">基础</button>
        <button class="tab" :class="{ active: activeTab === 'resume' }" type="button" @click="activeTab = 'resume'">简历</button>
        <button class="tab" :class="{ active: activeTab === 'awards' }" type="button" @click="activeTab = 'awards'">奖项 ({{ form.awards.length }})</button>
        <button class="tab" :class="{ active: activeTab === 'experiences' }" type="button" @click="activeTab = 'experiences'">经历 ({{ form.experiences.length }})</button>
        <button class="tab" :class="{ active: activeTab === 'featured' }" type="button" @click="activeTab = 'featured'">推荐项目 ({{ form.featured_projects.length }})</button>
      </div>

      <section v-if="activeTab === 'basic'" class="panel">
        <div class="form-grid">
          <label class="field span-2">
            <span>一句话简介</span>
            <input v-model="form.headline" type="text" placeholder="后端开发 / 技术博客作者" />
          </label>
          <div class="field span-2">
            <span>头像</span>
            <input v-model="form.avatar_url" type="text" placeholder="https://…" />
            <div style="display: flex; gap: 8px; align-items: center; margin-top: 6px;">
              <AssetUploadButton label="上传头像" @uploaded="(asset) => (form.avatar_url = asset.url)" />
              <img v-if="form.avatar_url" :src="form.avatar_url" alt="头像" style="width: 64px; height: 64px; border-radius: 50%; object-fit: cover; border: 1px solid var(--border);" />
            </div>
          </div>
        </div>

        <div class="repeater-head" style="margin-top: 22px;">
          <div>
            <p class="eyebrow">CONTACTS</p>
            <h3>联系方式</h3>
          </div>
          <button class="button small" type="button" @click="addContact">
            <AppIcon name="plus" :size="13" /> 添加
          </button>
        </div>
        <div class="repeater-list">
          <div v-for="(row, index) in contactRows" :key="index" class="contact-row">
            <input v-model="row.key" style="padding: 9px 11px; border: 1px solid var(--border); border-radius: 8px;" placeholder="email" />
            <input v-model="row.value" style="padding: 9px 11px; border: 1px solid var(--border); border-radius: 8px;" placeholder="me@example.com" />
            <button class="button-icon danger" type="button" title="删除" @click="contactRows.splice(index, 1)">
              <AppIcon name="trash" :size="13" />
            </button>
          </div>
        </div>
      </section>

      <section v-else-if="activeTab === 'resume'" class="editor-main">
        <VditorEditor v-model="form.resume_md" :height="620" placeholder="首页简历 Markdown…" />
      </section>

      <section v-else-if="activeTab === 'awards'" class="panel">
        <div class="repeater-head">
          <div>
            <p class="eyebrow">AWARDS</p>
            <h3>奖项与荣誉</h3>
          </div>
          <button class="button small" type="button" @click="addAward">
            <AppIcon name="plus" :size="13" /> 添加奖项
          </button>
        </div>
        <div v-if="form.awards.length === 0" class="empty-block compact">还没有奖项。</div>
        <div v-else class="repeater-list">
          <article v-for="(award, index) in form.awards" :key="award.id" class="mini-card">
            <div class="step-card-head">
              <strong>奖项 {{ index + 1 }}</strong>
              <span style="flex: 1;"></span>
              <button class="button-icon" type="button" title="上移" :disabled="index === 0" @click="moveItem(form.awards, index, -1)">
                <AppIcon name="up" :size="13" />
              </button>
              <button class="button-icon" type="button" title="下移" :disabled="index === form.awards.length - 1" @click="moveItem(form.awards, index, 1)">
                <AppIcon name="down" :size="13" />
              </button>
              <button class="button-icon danger" type="button" title="删除" @click="form.awards.splice(index, 1)">
                <AppIcon name="trash" :size="13" />
              </button>
            </div>
            <div class="step-fields">
              <label class="field">
                <span>标题</span>
                <input v-model="award.title" type="text" />
              </label>
              <label class="field">
                <span>日期</span>
                <input v-model="award.date" type="text" placeholder="2024" />
              </label>
              <label class="field span-2">
                <span>说明</span>
                <textarea v-model="award.description" />
              </label>
              <label class="field span-2">
                <span>奖状图片</span>
                <input v-model="award.image_url" type="text" placeholder="https://…" />
                <div style="margin-top: 6px;">
                  <AssetUploadButton label="上传奖状" @uploaded="(asset) => onAwardImage(index, asset)" />
                </div>
                <img v-if="award.image_url" class="cover-preview" :src="award.image_url" alt="奖状预览" />
              </label>
            </div>
          </article>
        </div>
      </section>

      <section v-else-if="activeTab === 'experiences'" class="panel">
        <div class="repeater-head">
          <div>
            <p class="eyebrow">EXPERIENCE</p>
            <h3>工作与实习经历</h3>
          </div>
          <button class="button small" type="button" @click="addExperience">
            <AppIcon name="plus" :size="13" /> 添加经历
          </button>
        </div>
        <div v-if="form.experiences.length === 0" class="empty-block compact">还没有经历。</div>
        <div v-else class="repeater-list">
          <article v-for="(experience, index) in form.experiences" :key="index" class="mini-card">
            <div class="step-card-head">
              <strong>经历 {{ index + 1 }}</strong>
              <span style="flex: 1;"></span>
              <button class="button-icon" type="button" title="上移" :disabled="index === 0" @click="moveItem(form.experiences, index, -1)">
                <AppIcon name="up" :size="13" />
              </button>
              <button class="button-icon" type="button" title="下移" :disabled="index === form.experiences.length - 1" @click="moveItem(form.experiences, index, 1)">
                <AppIcon name="down" :size="13" />
              </button>
              <button class="button-icon danger" type="button" title="删除" @click="form.experiences.splice(index, 1)">
                <AppIcon name="trash" :size="13" />
              </button>
            </div>
            <div class="step-fields">
              <label class="field">
                <span>公司</span>
                <input v-model="experience.company" type="text" />
              </label>
              <label class="field">
                <span>职位</span>
                <input v-model="experience.role" type="text" />
              </label>
              <label class="field">
                <span>时间</span>
                <input v-model="experience.period" type="text" placeholder="2024.07 - 至今" />
              </label>
              <label class="field span-2">
                <span>描述</span>
                <textarea v-model="experience.description" />
              </label>
            </div>
          </article>
        </div>
      </section>

      <section v-else class="panel">
        <div class="repeater-head">
          <div>
            <p class="eyebrow">FEATURED</p>
            <h3>推荐项目</h3>
          </div>
          <button class="button small" type="button" @click="addFeatured">
            <AppIcon name="plus" :size="13" /> 添加项目
          </button>
        </div>
        <div v-if="form.featured_projects.length === 0" class="empty-block compact">还没有推荐项目。</div>
        <div v-else class="repeater-list">
          <article v-for="(project, index) in form.featured_projects" :key="index" class="mini-card">
            <div class="step-card-head">
              <strong>项目 {{ index + 1 }}</strong>
              <span style="flex: 1;"></span>
              <button class="button-icon" type="button" title="上移" :disabled="index === 0" @click="moveItem(form.featured_projects, index, -1)">
                <AppIcon name="up" :size="13" />
              </button>
              <button class="button-icon" type="button" title="下移" :disabled="index === form.featured_projects.length - 1" @click="moveItem(form.featured_projects, index, 1)">
                <AppIcon name="down" :size="13" />
              </button>
              <button class="button-icon danger" type="button" title="删除" @click="form.featured_projects.splice(index, 1)">
                <AppIcon name="trash" :size="13" />
              </button>
            </div>
            <div class="step-fields">
              <label class="field">
                <span>名称</span>
                <input v-model="project.name" type="text" />
              </label>
              <label class="field">
                <span>链接</span>
                <input v-model="project.url" type="text" placeholder="/projects/…/" />
              </label>
              <label class="field span-2">
                <span>摘要</span>
                <textarea v-model="project.summary" />
              </label>
              <label class="field span-2">
                <span>封面</span>
                <input v-model="project.cover_url" type="text" placeholder="https://…" />
                <div style="margin-top: 6px;">
                  <AssetUploadButton label="上传封面" @uploaded="(asset) => onFeaturedCover(index, asset)" />
                </div>
              </label>
            </div>
          </article>
        </div>
      </section>
    </template>
  </div>
</template>
