import { Component, ErrorInfo, ReactNode } from 'react'
import { Result, Button } from '@arco-design/web-react'
import { Bug } from 'lucide-react'
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
        <div
          className={cn(
            'min-h-screen flex items-center justify-center bg-background p-4',
            this.props.className
          )}
        >
          <div className="max-w-lg w-full">
            <Result
              status="error"
              title="哎呀，出错了！"
              subTitle={
                <span>
                  应用遇到了一个意外错误，请联系管理员处理。
                  <span className="block text-sm mt-2" style={{ color: 'var(--color-text-3)' }}>
                    我们已经记录了这个错误，会尽快修复
                  </span>
                </span>
              }
              extra={
                <div className="w-full space-y-4">
                  {this.props.enableRecovery && this.state.retryCount < 3 && (
                    <div
                      className="rounded-lg p-4 text-center text-sm"
                      style={{
                        background: 'var(--color-primary-light-1)',
                        border: '1px solid var(--color-primary-light-3)',
                        color: 'var(--color-text-1)',
                      }}
                    >
                      <span
                        className="inline-block w-4 h-4 rounded-full animate-spin mr-2 align-middle border-2 border-t-transparent"
                        style={{ borderColor: '#5B21B6', borderTopColor: 'transparent' }}
                      />
                      正在尝试自动恢复... ({this.state.retryCount + 1}/3)
                    </div>
                  )}
                  {this.state.isRecovering && (
                    <div
                      className="rounded-lg p-4 text-center text-sm"
                      style={{
                        background: 'var(--color-success-light-1)',
                        border: '1px solid var(--color-success-light-3)',
                      }}
                    >
                      应用已恢复，正在重新加载...
                    </div>
                  )}
                  <div
                    className="rounded-lg p-4 text-center"
                    style={{
                      background: 'var(--color-fill-2)',
                      border: '1px solid var(--color-border)',
                    }}
                  >
                    <p className="text-sm" style={{ color: 'var(--color-text-2)' }}>
                      如果问题持续存在，请联系系统管理员
                    </p>
                    <Button type="text" size="small" className="mt-2" onClick={this.handleReportBug}>
                      <Bug className="w-3 h-3 mr-1 inline" />
                      复制错误信息
                    </Button>
                  </div>
                  {this.props.showDetails && (import.meta.env?.DEV || import.meta.env?.MODE === 'development') && (
                    <details className="mt-2 text-left">
                      <summary className="cursor-pointer text-sm font-medium" style={{ color: 'var(--color-text-3)' }}>
                        查看错误详情 (开发模式)
                      </summary>
                      <div
                        className="mt-3 p-4 rounded-lg text-xs font-mono overflow-auto max-h-60"
                        style={{
                          background: 'var(--color-fill-2)',
                          border: '1px solid var(--color-border)',
                          color: 'var(--color-text-2)',
                        }}
                      >
                        <div className="mb-3">
                          <strong className="text-red-500">错误信息:</strong>
                          <br />
                          {this.state.error?.message}
                        </div>
                        <div className="mb-3">
                          <strong>堆栈信息:</strong>
                          <pre className="whitespace-pre-wrap text-xs leading-relaxed mt-1">
                            {this.state.error?.stack}
                          </pre>
                        </div>
                        <div>
                          <strong>组件堆栈:</strong>
                          <pre className="whitespace-pre-wrap text-xs leading-relaxed mt-1">
                            {this.state.errorInfo?.componentStack}
                          </pre>
                        </div>
                      </div>
                    </details>
                  )}
                </div>
              }
            />
          </div>
        </div>
      )
    }

    return this.props.children
  }
}

export default ErrorBoundary
