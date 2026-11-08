<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from './AppIcon.vue'
import PageModal from './PageModal.vue'
import { api, ApiError } from '../lib/api'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const toast = useToastStore()
const sidebarOpen = ref(false)

const passwordOpen = ref(false)
const passwordSaving = ref(false)
const passwordError = ref('')
const passwordForm = reactive({
  old_password: '',
  new_password: '',
  confirm: '',
})

const navItems = [
  { to: '/', label: '概览', icon: 'dashboard' },
  { to: '/articles', label: '文章', icon: 'articles' },
  { to: '/directories', label: '目录', icon: 'directories' },
  { to: '/tags', label: '标签', icon: 'tags' },
  { to: '/projects', label: '项目', icon: 'projects' },
  { to: '/profile', label: '首页资料', icon: 'profile' },
  { to: '/todos', label: '待办', icon: 'todos' },
  { to: '/imports', label: 'ZIP 导入', icon: 'imports' },
]

const pageTitle = computed(() => (route.meta.title as string) || '管理后台')
const userLabel = computed(() => auth.user?.nickname || auth.user?.username || '管理员')
const initial = computed(() => userLabel.value.slice(0, 1).toUpperCase())

async function logout(): Promise<void> {
  await auth.logout()
  router.push({ name: 'login' })
}

function openPassword(): void {
  passwordError.value = ''
  passwordForm.old_password = ''
  passwordForm.new_password = ''
  passwordForm.confirm = ''
  passwordOpen.value = true
}

function closePassword(): void {
  if (passwordSaving.value) {
    return
  }
  passwordOpen.value = false
}

async function submitPassword(): Promise<void> {
  passwordError.value = ''
  if (!passwordForm.old_password) {
    passwordError.value = '请输入旧密码'
    return
  }
  if (passwordForm.new_password.length < 8) {
    passwordError.value = '新密码至少 8 个字符'
    return
  }
  if (passwordForm.new_password !== passwordForm.confirm) {
    passwordError.value = '两次输入的新密码不一致'
    return
  }
  if (passwordForm.new_password === passwordForm.old_password) {
    passwordError.value = '新密码不能与旧密码相同'
    return
  }
  passwordSaving.value = true
  try {
    await api.put('/api/v1/auth/password', {
      old_password: passwordForm.old_password,
      new_password: passwordForm.new_password,
    })
    passwordOpen.value = false
    toast.success('密码已更新')
  } catch (err) {
    passwordError.value = err instanceof ApiError ? err.message : '修改密码失败'
  } finally {
    passwordSaving.value = false
  }
}
</script>

<template>
  <div class="admin-shell" :class="{ 'sidebar-open': sidebarOpen }">
    <aside class="sidebar">
      <RouterLink class="sidebar-brand" to="/" @click="sidebarOpen = false">
        jishenghui<span>.admin</span>
      </RouterLink>

      <nav class="sidebar-nav" aria-label="后台导航">
        <RouterLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="nav-link"
          @click="sidebarOpen = false"
        >
          <AppIcon :name="item.icon" />
          <span>{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="sidebar-footer">
        <div class="sidebar-user">
          <span class="avatar small">{{ initial }}</span>
          <div>
            <strong>{{ userLabel }}</strong>
            <span>管理员</span>
          </div>
        </div>
        <button class="nav-link" type="button" @click="openPassword">
          <AppIcon name="lock" />
          <span>修改密码</span>
        </button>
        <button class="nav-link logout" type="button" @click="logout">
          <AppIcon name="logout" />
          <span>退出登录</span>
        </button>
      </div>
    </aside>

    <button class="sidebar-overlay" type="button" aria-label="关闭菜单" @click="sidebarOpen = false"></button>

    <section class="admin-main">
      <header class="topbar">
        <button class="icon-button menu-button" type="button" aria-label="打开菜单" @click="sidebarOpen = true">
          <AppIcon name="menu" />
        </button>
        <div class="topbar-title">
          <p class="eyebrow">BLOG ADMIN</p>
          <h1>{{ pageTitle }}</h1>
        </div>
        <div class="topbar-user">
          <span class="avatar">{{ initial }}</span>
          <span>{{ userLabel }}</span>
        </div>
      </header>

      <main class="admin-content">
        <RouterView />
      </main>
    </section>

    <PageModal :open="passwordOpen" title="修改密码" @close="closePassword">
      <form class="form-grid" @submit.prevent="submitPassword">
        <label class="field span-2">
          <span>旧密码</span>
          <input
            v-model="passwordForm.old_password"
            type="password"
            autocomplete="current-password"
            :disabled="passwordSaving"
          />
        </label>
        <label class="field span-2">
          <span>新密码</span>
          <input
            v-model="passwordForm.new_password"
            type="password"
            autocomplete="new-password"
            :disabled="passwordSaving"
          />
          <small class="field-hint">至少 8 个字符，不能与旧密码相同。</small>
        </label>
        <label class="field span-2">
          <span>确认新密码</span>
          <input
            v-model="passwordForm.confirm"
            type="password"
            autocomplete="new-password"
            :disabled="passwordSaving"
          />
        </label>
        <p v-if="passwordError" class="form-error span-2">{{ passwordError }}</p>
      </form>
      <template #actions>
        <button class="button ghost" type="button" :disabled="passwordSaving" @click="closePassword">取消</button>
        <button class="button primary" type="button" :disabled="passwordSaving" @click="submitPassword">
          {{ passwordSaving ? '保存中…' : '保存' }}
        </button>
      </template>
    </PageModal>
  </div>
</template>
