import type { Plugin } from 'vite'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import { readdirSync } from 'fs'
import { join, normalize } from 'path'

const VIRTUAL_MONSTER_LIST = 'virtual:infinite-map-monster-files'
const RESOLVED_VIRTUAL_MONSTER_LIST = '\0' + VIRTUAL_MONSTER_LIST

/** Scan public/map/monster/*.png for Infinite Map (FrameRonin parity). */
function infiniteMapMonsterScanPlugin(): Plugin {
  const scan = (): string[] => {
    const dir = join(process.cwd(), 'public', 'map', 'monster')
    try {
      return readdirSync(dir)
        .filter((name) => /\.png$/i.test(name) && !name.startsWith('.'))
        .sort((a, b) => a.localeCompare(b, undefined, { sensitivity: 'base' }))
    } catch {
      return []
    }
  }
  const isUnderMonsterDir = (filePath: string) =>
    normalize(filePath).replace(/\\/g, '/').includes('/public/map/monster/')

  return {
    name: 'infinite-map-monster-scan',
    resolveId(id) {
      if (id === VIRTUAL_MONSTER_LIST) return RESOLVED_VIRTUAL_MONSTER_LIST
    },
    load(id) {
      if (id !== RESOLVED_VIRTUAL_MONSTER_LIST) return null
      return `export const MONSTER_PUBLIC_FILENAMES = ${JSON.stringify(scan())}`
    },
    configureServer(server) {
      const monsterDir = join(process.cwd(), 'public', 'map', 'monster')
      const invalidate = () => {
        const mod = server.moduleGraph.getModuleById(RESOLVED_VIRTUAL_MONSTER_LIST)
        if (mod) server.moduleGraph.invalidateModule(mod)
      }
      server.watcher.add(monsterDir)
      for (const ev of ['add', 'unlink'] as const) {
        server.watcher.on(ev, (filePath) => {
          if (isUnderMonsterDir(filePath)) invalidate()
        })
      }
    },
  }
}

export default defineConfig({
  plugins: [infiniteMapMonsterScanPlugin(), react()],
  base: '/',
  resolve: {
    dedupe: ['react', 'react-dom', 'scheduler'],
    alias: {
      '@': path.resolve(__dirname, './src'),
      react: path.resolve(__dirname, 'node_modules/react'),
      'react-dom': path.resolve(__dirname, 'node_modules/react-dom'),
      'react/jsx-runtime': path.resolve(__dirname, 'node_modules/react/jsx-runtime.js'),
      'react/jsx-dev-runtime': path.resolve(__dirname, 'node_modules/react/jsx-dev-runtime.js'),
    },
  },
  server: {
    port: 3000,
    open: true,
    host: true,
    hmr: {
      port: 3001,
    },
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:9003',
        changeOrigin: true,
        ws: true,
        configure: (proxy) => {
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
    sourcemap: false,
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
          antd: ['antd', '@ant-design/icons'],
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
      'react/jsx-runtime',
      '@arco-design/web-react',
      'antd',
      '@ant-design/icons',
      'react-router-dom',
      'zustand',
      'echarts',
      'gifenc',
      'gifuct-js',
    ],
  },
})
