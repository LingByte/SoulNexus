import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  base: '/',
  resolve: {
    dedupe: ['react', 'react-dom'],
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    open: true,
    host: true, // 允许外部访问
    hmr: {
      port: 3001, // 使用不同的端口用于HMR
    },
    // 开发态：浏览器打同域 /api，由 Vite 反代到本地后端（避免 CORS）
    // 配合 VITE_API_BASE_URL=/api；WebSocket（/api/ws、voice-session 等）一并转发
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:9003',
        changeOrigin: true,
        ws: true,
        configure: (proxy) => {
          // http-proxy 默认对 ECONNREFUSED 回 500；网关语义应是 502 Bad Gateway
          proxy.on('error', (err, _req, res) => {
            console.error('[vite proxy /api]', err.message)
            if (res && 'writeHead' in res && typeof res.writeHead === 'function' && !res.headersSent) {
              res.writeHead(502, { 'Content-Type': 'application/json; charset=utf-8' })
              res.end(
                JSON.stringify({
                  code: 502,
                  msg: 'Bad Gateway: backend unreachable (is the API server running on :9003?)',
                  data: null,
                }),
              )
            }
          })
        },
      },
      '/uploads': {
        target: 'http://127.0.0.1:9003',
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on('error', (err, _req, res) => {
            console.error('[vite proxy /uploads]', err.message)
            if (res && 'writeHead' in res && typeof res.writeHead === 'function' && !res.headersSent) {
              res.writeHead(502, { 'Content-Type': 'application/json; charset=utf-8' })
              res.end(
                JSON.stringify({
                  code: 502,
                  msg: 'Bad Gateway: backend unreachable',
                  data: null,
                }),
              )
            }
          })
        },
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: false, // 生产环境关闭sourcemap提升性能
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true,
        drop_debugger: true,
      },
    },
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom'],
          arco: ['@arco-design/web-react'],
          router: ['react-router-dom'],
          utils: ['zustand', 'clsx', 'tailwind-merge'],
        },
      },
    },
    reportCompressedSize: true,
    chunkSizeWarningLimit: 1000,
  },
  optimizeDeps: {
    include: [
      'react',
      'react-dom',
      '@arco-design/web-react',
      'react-router-dom',
      'zustand',
      'echarts',
    ],
  },
})
