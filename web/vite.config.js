import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Development uses one browser origin for the page, API, and OAuth callback.
// Set BaseURL to http://127.0.0.1:5173 in the backend development config.
const target = process.env.ADN_API_TARGET || 'http://127.0.0.1:18080'
export default defineConfig({
  base: './',
  plugins: [vue()],
  server: {
    port: 5173,
    strictPort: true,
    proxy: { '/api': { target }, '/auth': { target } },
  },
})
