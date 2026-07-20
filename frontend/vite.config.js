import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    proxy: {
      '/manager': 'http://localhost:8887'
    }
  },
  build: {
    outDir: 'dist'
  }
})
