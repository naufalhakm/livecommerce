import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 3000,
    allowedHosts: [
      'localhost',
      '127.0.0.1',
      'livecomerce.laidtechnology.tech',
      '.laidtechnology.tech'
    ]
  },
  define: {
    global: 'globalThis',
    'process.env': {}
  },
  optimizeDeps: {
    include: ['simple-peer']
  },
  build: {
    target: 'es2015',
    rollupOptions: {
      output: {
        manualChunks: undefined
      }
    }
  }
})
