import vue from '@astrojs/vue'
import { defineConfig } from 'astro/config'

export default defineConfig({
  site: process.env.SITE_URL || 'https://www.example.com',
  trailingSlash: 'always',
  integrations: [vue()],
  build: {
    format: 'directory',
  },
  server: {
    host: true,
    port: 4321,
  },
})
