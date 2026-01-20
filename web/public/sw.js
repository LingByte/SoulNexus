// Service Worker for PWA functionality
const CACHE_NAME = 'writewiz-v1.0.0'
const STATIC_CACHE = 'static-v1.0.0'
const DYNAMIC_CACHE = 'dynamic-v1.0.0'

// 需要缓存的静态资源
const STATIC_ASSETS = [
  '/',
  '/index.html',
  '/manifest.json',
  // 添加其他关键静态资源
]

// 需要缓存的API路径
const API_CACHE_PATTERNS = [
  /\/api\/auth/,
  /\/api\/projects/,
  /\/api\/novels/,
  /\/api\/chapters/
]

// 安装事件
self.addEventListener('install', (event) => {
  console.log('Service Worker installing...')
  
  event.waitUntil(
    caches.open(STATIC_CACHE)
      .then((cache) => {
        console.log('Caching static assets...')
        return cache.addAll(STATIC_ASSETS)
      })
      .then(() => {
        console.log('Static assets cached successfully')
        return self.skipWaiting()
      })
      .catch((error) => {
        console.error('Failed to cache static assets:', error)
      })
  )
})

// 激活事件
self.addEventListener('activate', (event) => {
  console.log('Service Worker activating...')
  
  event.waitUntil(
    caches.keys()
      .then((cacheNames) => {
        return Promise.all(
          cacheNames.map((cacheName) => {
            if (cacheName !== STATIC_CACHE && cacheName !== DYNAMIC_CACHE) {
              console.log('Deleting old cache:', cacheName)
              return caches.delete(cacheName)
            }
          })
        )
      })
      .then(() => {
        console.log('Service Worker activated')
        return self.clients.claim()
      })
  )
})

// 拦截网络请求
self.addEventListener('fetch', (event) => {
  const { request } = event
  const url = new URL(request.url)

  // 跳过开发环境的资源请求
  if (url.hostname === 'localhost' || url.hostname === '127.0.0.1') {
    // 跳过Vite开发服务器的资源
    if (url.pathname.startsWith('/@') || 
        url.pathname.startsWith('/src/') ||
        url.pathname.startsWith('/node_modules/') ||
        url.pathname.includes('?t=') ||
        url.pathname.includes('@vite') ||
        url.pathname.includes('@react-refresh')) {
      return
    }
  }

  // 只处理GET请求
  if (request.method !== 'GET') {
    return
  }

  // 处理静态资源
  if (STATIC_ASSETS.includes(url.pathname)) {
    event.respondWith(
      caches.match(request)
        .then((response) => {
          if (response) {
            return response
          }
          return fetch(request)
            .then((response) => {
              if (response.status === 200) {
                const responseClone = response.clone()
                caches.open(STATIC_CACHE)
                  .then((cache) => {
                    cache.put(request, responseClone)
                  })
              }
              return response
            })
        })
    )
    return
  }

  // 处理API请求
  if (API_CACHE_PATTERNS.some(pattern => pattern.test(url.pathname))) {
    event.respondWith(
      caches.match(request)
        .then((response) => {
          if (response) {
            // 返回缓存的数据，同时更新缓存
            fetch(request)
              .then((fetchResponse) => {
                if (fetchResponse.status === 200) {
                  const responseClone = fetchResponse.clone()
                  caches.open(DYNAMIC_CACHE)
                    .then((cache) => {
                      cache.put(request, responseClone)
                    })
                }
              })
            return response
          }

          // 如果缓存中没有，则从网络获取
          return fetch(request)
            .then((response) => {
              if (response.status === 200) {
                const responseClone = response.clone()
                caches.open(DYNAMIC_CACHE)
                  .then((cache) => {
                    cache.put(request, responseClone)
                  })
              }
              return response
            })
            .catch(() => {
              // 网络错误时，尝试返回缓存中的旧数据
              return caches.match(request)
            })
        })
    )
    return
  }

  // 处理图片资源
  if (request.destination === 'image') {
    event.respondWith(
      caches.match(request)
        .then((response) => {
          if (response) {
            return response
          }
          return fetch(request)
            .then((response) => {
              if (response.status === 200) {
                const responseClone = response.clone()
                caches.open(DYNAMIC_CACHE)
                  .then((cache) => {
                    cache.put(request, responseClone)
                  })
              }
              return response
            })
        })
    )
    return
  }

  // 其他请求直接通过网络获取
  event.respondWith(fetch(request))
})

// 处理后台同步
self.addEventListener('sync', (event) => {
  if (event.tag === 'background-sync') {
    event.waitUntil(
      // 执行后台同步任务
      doBackgroundSync()
    )
  }
})

// 处理推送通知
self.addEventListener('push', (event) => {
  if (event.data) {
    const data = event.data.json()
    const options = {
      body: data.body,
      icon: 'https://cetide-1325039295.cos.ap-chengdu.myqcloud.com/folder/icon-192x192.ico',
      badge: '/badge-72x72.png',
      vibrate: [100, 50, 100],
      data: {
        dateOfArrival: Date.now(),
        primaryKey: data.primaryKey
      },
      actions: [
        {
          action: 'explore',
          title: '查看详情',
          icon: 'https://cetide-1325039295.cos.ap-chengdu.myqcloud.com/folder/icon-192x192.ico'
        },
        {
          action: 'close',
          title: '关闭',
          icon: 'https://cetide-1325039295.cos.ap-chengdu.myqcloud.com/folder/icon-192x192.ico'
        }
      ]
    }

    event.waitUntil(
      self.registration.showNotification(data.title, options)
    )
  }
})

// 处理通知点击
self.addEventListener('notificationclick', (event) => {
  event.notification.close()

  if (event.action === 'explore') {
    event.waitUntil(
      clients.openWindow('/')
    )
  }
})

// 后台同步函数
async function doBackgroundSync() {
  try {
    // 实现后台同步逻辑
    console.log('Performing background sync...')
  } catch (error) {
    console.error('Background sync failed:', error)
  }
}

// 清理过期缓存
async function cleanOldCaches() {
  const cacheNames = await caches.keys()
  const validCaches = [STATIC_CACHE, DYNAMIC_CACHE]
  
  const deletePromises = cacheNames
    .filter(cacheName => !validCaches.includes(cacheName))
    .map(cacheName => caches.delete(cacheName))
  
  await Promise.all(deletePromises)
}

// 定期清理缓存
setInterval(cleanOldCaches, 24 * 60 * 60 * 1000) // 每24小时清理一次
