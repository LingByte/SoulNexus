// Service Worker 清理工具
export const cleanupServiceWorkers = async () => {
  if ('serviceWorker' in navigator) {
    try {
      // 获取所有注册的 Service Workers
      const registrations = await navigator.serviceWorker.getRegistrations()
      
      // 注销所有 Service Workers
      await Promise.all(
        registrations.map(registration => registration.unregister())
      )
      
      console.log('All Service Workers unregistered')
    } catch (error) {
      console.warn('Failed to cleanup Service Workers:', error)
    }
  }
}

// 开发环境自动清理
if (import.meta.env.DEV) {
  // 页面加载时清理
  window.addEventListener('load', cleanupServiceWorkers)
  
  // 页面卸载时也清理
  window.addEventListener('beforeunload', cleanupServiceWorkers)
}
