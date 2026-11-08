<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import { ApiError } from '../lib/api'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const username = ref('')
const password = ref('')
const error = ref('')

async function submit(): Promise<void> {
  error.value = ''
  if (!username.value.trim() || !password.value) {
    error.value = '请输入用户名和密码'
    return
  }
  try {
    await auth.login(username.value.trim(), password.value)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.replace(redirect)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : '登录失败，请稍后重试'
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-glow"></div>
    <div class="login-card">
      <div class="login-brand">
        <span class="brand-mark">j</span>
        <div>
          <p class="eyebrow">BLOG ADMIN</p>
          <h1>登录管理后台</h1>
        </div>
      </div>
      <p class="login-subtitle">单管理员博客后台：文章、目录、资源与固定页内容维护。</p>

      <form class="login-form" @submit.prevent="submit">
        <label class="field">
          <span>用户名</span>
          <input v-model="username" type="text" autocomplete="username" placeholder="admin" autofocus />
        </label>
        <label class="field">
          <span>密码</span>
          <input v-model="password" type="password" autocomplete="current-password" placeholder="请输入密码" />
        </label>
        <p v-if="error" class="form-error">{{ error }}</p>
        <button class="button primary login-submit" type="submit" :disabled="auth.loading">
          <AppIcon name="logout" />
          {{ auth.loading ? '登录中…' : '登录' }}
        </button>
      </form>

      <p class="login-footnote">仅管理员可访问 · refresh token 使用 httpOnly Cookie</p>
    </div>
  </div>
</template>
