import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import { setSessionExpiredHandler } from './lib/session'
import router from './router'
import { useAuthStore } from './stores/auth'
import './styles/main.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

setSessionExpiredHandler(() => {
  const auth = useAuthStore(pinia)
  auth.reset()
  if (router.currentRoute.value.name !== 'login') {
    router.push({ name: 'login', query: { redirect: router.currentRoute.value.fullPath } })
  }
})

router.isReady().then(() => {
  app.mount('#app')
})
