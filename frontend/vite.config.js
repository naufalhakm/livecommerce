import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 3000,
    allowedHosts: [
      'livecomerce.laidtechnology.tech',
      'localhost',
      '127.0.0.1'
    ],
    hmr: {
      port: 3000,
      host: process.env.NODE_ENV === 'production' ? 'livecomerce.laidtechnology.tech' : 'localhost'
    }
  },
  build: {
    outDir: 'dist',
    sourcemap: false
  }
})