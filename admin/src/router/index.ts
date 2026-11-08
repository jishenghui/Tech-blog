import { createRouter, createWebHistory } from 'vue-router'
import AppLayout from '../components/AppLayout.vue'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
      meta: { public: true, title: '登录' },
    },
    {
      path: '/',
      component: AppLayout,
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('../views/DashboardView.vue'),
          meta: { title: '概览' },
        },
        {
          path: 'articles',
          name: 'articles',
          component: () => import('../views/ArticlesView.vue'),
          meta: { title: '文章' },
        },
        {
          path: 'articles/new',
          name: 'article-new',
          component: () => import('../views/ArticleEditorView.vue'),
          meta: { title: '新建文章' },
        },
        {
          path: 'articles/:id',
          name: 'article-edit',
          component: () => import('../views/ArticleEditorView.vue'),
          meta: { title: '编辑文章' },
        },
        {
          path: 'directories',
          name: 'directories',
          component: () => import('../views/DirectoriesView.vue'),
          meta: { title: '目录' },
        },
        {
          path: 'tags',
          name: 'tags',
          component: () => import('../views/TagsView.vue'),
          meta: { title: '标签' },
        },
        {
          path: 'projects',
          name: 'projects',
          component: () => import('../views/ProjectsView.vue'),
          meta: { title: '项目' },
        },
        {
          path: 'projects/new',
          name: 'project-new',
          component: () => import('../views/ProjectEditorView.vue'),
          meta: { title: '新建项目' },
        },
        {
          path: 'projects/:id',
          name: 'project-edit',
          component: () => import('../views/ProjectEditorView.vue'),
          meta: { title: '编辑项目' },
        },
        {
          path: 'profile',
          name: 'profile',
          component: () => import('../views/ProfileView.vue'),
          meta: { title: '首页资料' },
        },
        {
          path: 'todos',
          name: 'todos',
          component: () => import('../views/TodosView.vue'),
          meta: { title: '待办' },
        },
        {
          path: 'imports',
          name: 'imports',
          component: () => import('../views/ImportsView.vue'),
          meta: { title: 'ZIP 导入' },
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('../views/NotFoundView.vue'),
      meta: { public: true, title: '页面不存在' },
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  // 公开页（登录/404）不主动刷新会话，避免未登录时产生无意义的 401。
  if (!to.meta.public && !auth.initialized) {
    await auth.bootstrap()
  }
  if (to.meta.public) {
    if (to.name === 'login' && auth.user) {
      return { name: 'dashboard' }
    }
    return true
  }
  if (!auth.user) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  return true
})

router.afterEach((to) => {
  document.title = `${to.meta.title || '管理后台'} · Blog Admin`
})

export default router
