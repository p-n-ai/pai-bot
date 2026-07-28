import { URL, fileURLToPath } from 'node:url'
import { tanstackRouter } from '@tanstack/router-plugin/vite'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [
    tanstackRouter({
      target: 'react',
      autoCodeSplitting: true,
    }),
    tailwindcss(),
    react(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      '/api/embed': {
        target: process.env.VITE_API_URL ?? 'http://127.0.0.1:8080',
        changeOrigin: false,
        xfwd: true,
      },
      '/api': {
        target: process.env.VITE_API_URL ?? 'http://127.0.0.1:8080',
        changeOrigin: true,
        xfwd: true,
      },
      '/health/api': {
        target: process.env.VITE_API_URL ?? 'http://127.0.0.1:8080',
        changeOrigin: true,
        xfwd: true,
      },
      '/health/status': {
        target: process.env.VITE_API_URL ?? 'http://127.0.0.1:8080',
        changeOrigin: true,
        xfwd: true,
      },
      '/embed': {
        target: process.env.VITE_API_URL ?? 'http://127.0.0.1:8080',
        changeOrigin: false,
        xfwd: true,
      },
      '/ws': {
        target: process.env.VITE_API_URL ?? 'http://127.0.0.1:8080',
        changeOrigin: false,
        xfwd: true,
        ws: true,
      },
    },
  },
})
