import { Component, ErrorInfo, ReactNode } from 'react'
import { motion } from 'framer-motion'
import { AlertTriangle, CheckCircle2, RefreshCw, RotateCcw, Bug } from 'lucide-react'
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
    // 只记录错误到控制台，不发送请求
    console.error('Error caught by boundary:', error, errorInfo)
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
            'bg-gradient-to-br from-slate-50 via-white to-blue-50 dark:from-slate-900 dark:via-slate-800 dark:to-slate-900',
            this.props.className
          )}
          initial={{ opacity: 0, scale: 0.9 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.3 }}
        >
          <div className="max-w-lg w-full">
            <motion.div
              className="bg-white dark:bg-slate-800 rounded-3xl shadow-2xl border border-slate-200 dark:border-slate-700 p-8 text-center relative overflow-hidden"
              initial={{ y: 20, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              transition={{ delay: 0.1, duration: 0.3 }}
            >
              {/* 背景装饰 */}
              <div className="absolute top-0 left-0 w-full h-2 bg-gradient-to-r from-red-400 via-orange-400 to-yellow-400"></div>
              
              {/* 错误图标 */}
              <motion.div
                className="w-20 h-20 mx-auto mb-6 bg-gradient-to-br from-red-100 to-orange-100 dark:from-red-900/30 dark:to-orange-900/30 rounded-full flex items-center justify-center shadow-lg"
                animate={{ 
                  rotate: [0, -5, 5, -5, 0],
                  scale: [1, 1.05, 1]
                }}
                transition={{ 
                  duration: 2,
                  delay: 0.2,
                  repeat: Infinity,
                  repeatDelay: 3
                }}
              >
                <AlertTriangle className="w-10 h-10 text-red-500 dark:text-red-400" />
              </motion.div>

              {/* 错误标题 */}
              <motion.h1 
                className="text-3xl font-bold text-slate-900 dark:text-white mb-3"
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.3 }}
              >
                哎呀，出错了！
              </motion.h1>
              
              <motion.p 
                className="text-slate-600 dark:text-slate-400 mb-8 text-lg leading-relaxed"
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.4 }}
              >
                应用遇到了一个意外错误，请联系管理员处理。
                <br />
                <span className="text-sm text-slate-500 dark:text-slate-500 mt-2 block">
                  我们已经记录了这个错误，会尽快修复
                </span>
              </motion.p>

              {/* 自动恢复提示 */}
              {this.props.enableRecovery && this.state.retryCount < 3 && (
                <motion.div
                  className="bg-gradient-to-r from-blue-50 to-indigo-50 dark:from-blue-900/20 dark:to-indigo-900/20 border border-blue-200 dark:border-blue-800 rounded-xl p-4 mb-6"
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: 0.3 }}
                >
                  <div className="flex items-center justify-center gap-3 text-blue-800 dark:text-blue-400">
                    <div className="w-5 h-5 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
                    <span className="font-medium">
                      正在尝试自动恢复... ({this.state.retryCount + 1}/3)
                    </span>
                  </div>
                </motion.div>
              )}

              {/* 恢复状态 */}
              {this.state.isRecovering && (
                <motion.div
                  className="bg-gradient-to-r from-green-50 to-emerald-50 dark:from-green-900/20 dark:to-emerald-900/20 border border-green-200 dark:border-green-800 rounded-xl p-4 mb-6"
                  initial={{ opacity: 0, scale: 0.9 }}
                  animate={{ opacity: 1, scale: 1 }}
                >
                  <div className="flex items-center justify-center gap-3 text-green-800 dark:text-green-400">
                    <CheckCircle2 className="w-5 h-5" />
                    <span className="font-medium">
                      应用已恢复，正在重新加载...
                    </span>
                  </div>
                </motion.div>
              )}

              {/* 操作按钮 */}
              <div className="space-y-3">
                {/* 联系管理员提示 */}
                <motion.div
                  className="mt-6 p-4 bg-slate-50 dark:bg-slate-900/50 rounded-xl border border-slate-200 dark:border-slate-700"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.7 }}
                >
                  <p className="text-sm text-slate-600 dark:text-slate-400">
                    如果问题持续存在，请联系系统管理员
                  </p>
                  <div className="mt-2 flex items-center justify-center gap-2">
                    <button
                      onClick={this.handleReportBug}
                      className="inline-flex items-center gap-2 text-xs text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300 transition-colors"
                    >
                      <Bug className="w-3 h-3" />
                      复制错误信息
                    </button>
                  </div>
                </motion.div>
              </div>

              {/* 错误详情（开发环境） */}
              {this.props.showDetails && (import.meta.env?.DEV || import.meta.env?.MODE === 'development') && (
                <motion.details
                  className="mt-6 text-left"
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.8 }}
                >
                  <summary className="cursor-pointer text-sm text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300 font-medium">
                    🔍 查看错误详情 (开发模式)
                  </summary>
                  <div className="mt-3 p-4 bg-slate-100 dark:bg-slate-900 rounded-xl text-xs font-mono text-slate-700 dark:text-slate-300 overflow-auto max-h-60 border border-slate-200 dark:border-slate-700">
                    <div className="mb-3">
                      <strong className="text-red-600 dark:text-red-400">错误信息:</strong><br />
                      <span className="text-red-700 dark:text-red-300">{this.state.error?.message}</span>
                    </div>
                    <div className="mb-3">
                      <strong className="text-orange-600 dark:text-orange-400">堆栈信息:</strong><br />
                      <pre className="whitespace-pre-wrap text-slate-600 dark:text-slate-400 text-xs leading-relaxed">
                        {this.state.error?.stack}
                      </pre>
                    </div>
                    <div>
                      <strong className="text-blue-600 dark:text-blue-400">组件堆栈:</strong><br />
                      <pre className="whitespace-pre-wrap text-slate-600 dark:text-slate-400 text-xs leading-relaxed">
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
