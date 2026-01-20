import { Component, ErrorInfo, ReactNode } from 'react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/cn.ts'
import { showAlert } from '@/utils/notification.ts'

interface Props {
  children: ReactNode
  fallback?: ReactNode
  onError?: (error: Error, errorInfo: ErrorInfo) => void
  className?: string
  showDetails?: boolean
  enableRecovery?: boolean
}

interface State {
  hasError: boolean
  error: Error | null
  errorInfo: ErrorInfo | null
  retryCount: number
  isRecovering: boolean
}

class ErrorBoundary extends Component<Props, State> {
  private retryTimeoutId: number | null = null

  constructor(props: Props) {
    super(props)
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
      retryCount: 0,
      isRecovering: false
    }
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return {
      hasError: true,
      error
    }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.setState({
      error,
      errorInfo
    })

    // 调用错误处理回调
    this.props.onError?.(error, errorInfo)

    // 发送错误报告到监控服务
    this.reportError(error, errorInfo)

    // 自动恢复机制
    if (this.props.enableRecovery && this.state.retryCount < 3) {
      this.scheduleRecovery()
    }
  }

  private reportError = (error: Error, errorInfo: ErrorInfo) => {
    // 这里可以集成错误监控服务，如 Sentry
    console.error('Error caught by boundary:', error, errorInfo)
    
    // 示例：发送到错误监控服务
    if (typeof window !== 'undefined' && 'navigator' in window) {
      // 收集错误上下文
      const errorContext = {
        message: error.message,
        stack: error.stack,
        componentStack: errorInfo.componentStack,
        timestamp: new Date().toISOString(),
        userAgent: navigator.userAgent,
        url: window.location.href,
        retryCount: this.state.retryCount
      }

      // 发送到监控服务（示例）
      this.sendToMonitoringService(errorContext)
    }
  }

  private sendToMonitoringService = (errorContext: any) => {
    // 示例实现，实际项目中应该发送到真实的监控服务
    try {
      // 使用 fetch 发送错误信息
      fetch('/api/errors', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(errorContext)
      }).catch(() => {
        // 如果发送失败，可以存储到本地存储
        this.storeErrorLocally(errorContext)
      })
    } catch (e) {
      this.storeErrorLocally(errorContext)
    }
  }

  private storeErrorLocally = (errorContext: any) => {
    try {
      const errors = JSON.parse(localStorage.getItem('errorLogs') || '[]')
      errors.push(errorContext)
      // 只保留最近10个错误
      if (errors.length > 10) {
        errors.splice(0, errors.length - 10)
      }
      localStorage.setItem('errorLogs', JSON.stringify(errors))
    } catch (e) {
      console.warn('Failed to store error locally:', e)
    }
  }

  private scheduleRecovery = () => {
    if (this.retryTimeoutId) {
      window.clearTimeout(this.retryTimeoutId)
    }

    this.retryTimeoutId = window.setTimeout(() => {
      this.setState(prevState => ({
        hasError: false,
        error: null,
        errorInfo: null,
        retryCount: prevState.retryCount + 1,
        isRecovering: true
      }))

      // 恢复后重置状态
      setTimeout(() => {
        this.setState({ isRecovering: false })
      }, 5000)
    }, Math.pow(2, this.state.retryCount) * 1000) // 指数退避
  }

  private handleRetry = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
      retryCount: 0,
      isRecovering: false
    })
  }

  private handleReload = () => {
    window.location.reload()
  }

  private handleReportBug = () => {
    const errorDetails = {
      error: this.state.error?.message,
      stack: this.state.error?.stack,
      componentStack: this.state.errorInfo?.componentStack,
      timestamp: new Date().toISOString(),
      url: window.location.href
    }

    // 创建错误报告
    const report = `错误报告\n\n` +
      `错误信息: ${errorDetails.error}\n` +
      `时间: ${errorDetails.timestamp}\n` +
      `页面: ${errorDetails.url}\n\n` +
      `堆栈信息:\n${errorDetails.stack}\n\n` +
      `组件堆栈:\n${errorDetails.componentStack}`

    // 复制到剪贴板
    if (navigator.clipboard) {
      navigator.clipboard.writeText(report).then(() => {
        showAlert('错误报告已复制到剪贴板', 'success', '复制成功')
      })
    } else {
      // 降级方案
      const textArea = document.createElement('textarea')
      textArea.value = report
      document.body.appendChild(textArea)
      textArea.select()
      document.execCommand('copy')
      document.body.removeChild(textArea)
      showAlert('错误报告已复制到剪贴板', 'success', '复制成功')
    }
  }

  componentWillUnmount() {
    if (this.retryTimeoutId) {
      window.clearTimeout(this.retryTimeoutId)
    }
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback
      }

      return (
        <motion.div
          className={cn(
            'min-h-screen flex items-center justify-center p-4',
            'bg-gradient-to-br from-red-50 via-white to-orange-50',
            this.props.className
          )}
          initial={{ opacity: 0, scale: 0.9 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3 }}
        >
          <div className="max-w-md w-full">
            <motion.div
              className="bg-white rounded-2xl shadow-2xl p-8 text-center"
              initial={{ y: 20, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              transition={{ delay: 0.1, duration: 0.3 }}
            >
              {/* 错误图标 */}
              <motion.div
                className="w-16 h-16 mx-auto mb-6 bg-red-100 rounded-full flex items-center justify-center"
                animate={{ rotate: [0, -10, 10, -10, 0] }}
                transition={{ duration: 0.5, delay: 0.2 }}
              >
                <span className="text-3xl">⚠️</span>
              </motion.div>

              {/* 错误标题 */}
              <h1 className="text-2xl font-bold text-gray-900 mb-2">
                哎呀，出错了！
              </h1>
              <p className="text-gray-600 mb-6">
                应用遇到了一个意外错误，我们正在努力修复它。
              </p>

              {/* 自动恢复提示 */}
              {this.props.enableRecovery && this.state.retryCount < 3 && (
                <motion.div
                  className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6"
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: 0.3 }}
                >
                  <div className="flex items-center justify-center gap-2 text-blue-800">
                    <div className="w-4 h-4 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
                    <span className="text-sm font-medium">
                      正在尝试自动恢复... ({this.state.retryCount + 1}/3)
                    </span>
                  </div>
                </motion.div>
              )}

              {/* 恢复状态 */}
              {this.state.isRecovering && (
                <motion.div
                  className="bg-green-50 border border-green-200 rounded-lg p-4 mb-6"
                  initial={{ opacity: 0, scale: 0.9 }}
                  animate={{ opacity: 1, scale: 1 }}
                >
                  <div className="flex items-center justify-center gap-2 text-green-800">
                    <span className="text-lg">✅</span>
                    <span className="text-sm font-medium">
                      应用已恢复，正在重新加载...
                    </span>
                  </div>
                </motion.div>
              )}

              {/* 操作按钮 */}
              <div className="space-y-3">
                <motion.button
                  onClick={this.handleRetry}
                  className="w-full bg-blue-600 text-white font-semibold py-3 px-6 rounded-lg hover:bg-blue-700 transition-colors"
                  whileHover={{ scale: 1.02 }}
                  whileTap={{ scale: 0.98 }}
                >
                  重试
                </motion.button>

                <div className="flex gap-3">
                  <motion.button
                    onClick={this.handleReload}
                    className="flex-1 bg-gray-100 text-gray-700 font-medium py-2 px-4 rounded-lg hover:bg-gray-200 transition-colors"
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                  >
                    刷新页面
                  </motion.button>

                  <motion.button
                    onClick={this.handleReportBug}
                    className="flex-1 bg-orange-100 text-orange-700 font-medium py-2 px-4 rounded-lg hover:bg-orange-200 transition-colors"
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                  >
                    报告问题
                  </motion.button>
                </div>
              </div>

              {/* 错误详情（开发环境） */}
              {this.props.showDetails && (import.meta.env?.DEV || import.meta.env?.MODE === 'development') && (
                <motion.details
                  className="mt-6 text-left"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.5 }}
                >
                  <summary className="cursor-pointer text-sm text-gray-500 hover:text-gray-700">
                    查看错误详情
                  </summary>
                  <div className="mt-2 p-4 bg-gray-100 rounded-lg text-xs font-mono text-gray-700 overflow-auto max-h-40">
                    <div className="mb-2">
                      <strong>错误信息:</strong><br />
                      {this.state.error?.message}
                    </div>
                    <div className="mb-2">
                      <strong>堆栈信息:</strong><br />
                      <pre className="whitespace-pre-wrap">
                        {this.state.error?.stack}
                      </pre>
                    </div>
                    <div>
                      <strong>组件堆栈:</strong><br />
                      <pre className="whitespace-pre-wrap">
                        {this.state.errorInfo?.componentStack}
                      </pre>
                    </div>
                  </div>
                </motion.details>
              )}
            </motion.div>
          </div>
        </motion.div>
      )
    }

    return this.props.children
  }
}

export default ErrorBoundary
