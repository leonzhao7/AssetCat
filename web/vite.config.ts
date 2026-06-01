import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: '0.0.0.0',
    port: 6173,
    proxy: {
      '/assets': 'http://127.0.0.1:9080',
      '/summary': 'http://127.0.0.1:9080',
      '/healthz': 'http://127.0.0.1:9080',
    },
  },
})
