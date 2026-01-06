import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api/emails/stream': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        ws: true,
        timeout: 0,
        proxyTimeout: 0
      },
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  }
})
