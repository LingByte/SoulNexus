import { useEffect } from 'react'

interface DevErrorHandlerProps {
  onError?: (error: Error) => void
}

const DevErrorHandler = ({ onError }: DevErrorHandlerProps) => {
  useEffect(() => {
    // 处理未捕获的错误
    const handleError = (event: ErrorEvent) => {
      console.error('Uncaught error:', event.error)
      onError?.(event.error)
    }

    // 处理未处理的Promise拒绝
    const handleUnhandledRejection = (event: PromiseRejectionEvent) => {
      console.error('Unhandled promise rejection:', event.reason)
      onError?.(new Error(event.reason))
    }

    // 处理资源加载错误
    const handleResourceError = (event: Event) => {
      const target = event.target as HTMLElement
      if (target.tagName === 'IMG') {
        console.warn('Image load error:', (target as HTMLImageElement).src)
      }
    }

    window.addEventListener('error', handleError)
    window.addEventListener('unhandledrejection', handleUnhandledRejection)
    window.addEventListener('error', handleResourceError, true)

    return () => {
      window.removeEventListener('error', handleError)
      window.removeEventListener('unhandledrejection', handleUnhandledRejection)
      window.removeEventListener('error', handleResourceError, true)
    }
  }, [onError])

  // 开发环境下的连接状态指示器
  if (import.meta.env.MODE !== 'development') return null
}

export default DevErrorHandler
