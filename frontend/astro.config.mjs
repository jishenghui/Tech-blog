import { defineConfig } from 'astro/config'
import vue from '@astrojs/vue'

export default defineConfig({
  integrations: [vue()],
  build: {
    format: 'directory',
  },
  server: {
    host: true,
    port: 4321,
  },
})
