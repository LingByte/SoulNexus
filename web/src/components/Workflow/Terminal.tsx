import React, { useEffect, useRef } from 'react'
import { Terminal as TerminalIcon, X, Copy, Trash2 } from 'lucide-react'
import Button from '@/components/UI/Button'
import { motion, AnimatePresence } from 'framer-motion'

export interface TerminalLog {
  timestamp: string
  level: 'info' | 'success' | 'warning' | 'error' | 'debug'
  message: string
  nodeId?: string
  nodeName?: string
}

interface TerminalProps {
  logs: TerminalLog[]
  isVisible: boolean
  onClose: () => void
  onClear: () => void
}

const Terminal: React.FC<TerminalProps> = ({ logs, isVisible, onClose, onClear }) => {
  const terminalRef = useRef<HTMLDivElement>(null)
  const endRef = useRef<HTMLDivElement>(null)

  // Auto scroll to bottom when new logs arrive
  useEffect(() => {
    if (endRef.current) {
      endRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [logs])

  const getLogColor = (level: TerminalLog['level']) => {
    switch (level) {
      case 'info':
        return 'text-blue-400'
      case 'success':
        return 'text-green-400'
      case 'warning':
        return 'text-yellow-400'
      case 'error':
        return 'text-red-400'
      case 'debug':
        return 'text-gray-400'
      default:
        return 'text-gray-300'
    }
  }

  const getLogPrefix = (level: TerminalLog['level']) => {
    switch (level) {
      case 'info':
        return '[INFO]'
      case 'success':
        return '[SUCCESS]'
      case 'warning':
        return '[WARN]'
      case 'error':
        return '[ERROR]'
      case 'debug':
        return '[DEBUG]'
      default:
        return '[LOG]'
    }
  }

  const copyToClipboard = async () => {
    const text = logs.map(log => 
      `[${log.timestamp}] ${getLogPrefix(log.level)} ${log.nodeName ? `[${log.nodeName}]` : ''} ${log.message}`
    ).join('\n')
    
    try {
      await navigator.clipboard.writeText(text)
      // You can add a toast notification here if needed
    } catch (err) {
      console.error('Failed to copy to clipboard:', err)
    }
  }

  return (
    <AnimatePresence>
      {isVisible && (
        <motion.div
          initial={{ y: '100%' }}
          animate={{ y: 0 }}
          exit={{ y: '100%' }}
          transition={{ type: 'spring', damping: 25, stiffness: 200 }}
          className="fixed bottom-0 left-0 right-0 z-50 bg-gray-900 border-t border-gray-700 shadow-2xl"
          style={{ height: '40vh', maxHeight: '500px' }}
        >
          {/* Terminal Header */}
          <div className="flex items-center justify-between px-4 py-2 bg-gray-800 border-b border-gray-700">
            <div className="flex items-center gap-2">
              <TerminalIcon className="w-4 h-4 text-green-400" />
              <span className="text-sm font-medium text-gray-200">工作流执行终端</span>
              <span className="text-xs text-gray-400">({logs.length} 条日志)</span>
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="xs"
                onClick={copyToClipboard}
                className="text-gray-400 hover:text-gray-200"
              >
                <Copy className="w-4 h-4" />
              </Button>
              <Button
                variant="ghost"
                size="xs"
                onClick={onClear}
                className="text-gray-400 hover:text-gray-200"
              >
                <Trash2 className="w-4 h-4" />
              </Button>
              <Button
                variant="ghost"
                size="xs"
                onClick={onClose}
                className="text-gray-400 hover:text-gray-200"
              >
                <X className="w-4 h-4" />
              </Button>
            </div>
          </div>

          {/* Terminal Content */}
          <div
            ref={terminalRef}
            className="h-full overflow-y-auto p-4 font-mono text-sm"
            style={{ backgroundColor: '#1e1e1e' }}
          >
            {logs.length === 0 ? (
              <div className="text-gray-500 text-center py-8">
                等待工作流执行...
              </div>
            ) : (
              <div className="space-y-1">
                {logs.map((log, index) => (
                  <div
                    key={index}
                    className="flex items-start gap-2 hover:bg-gray-800/50 px-2 py-1 rounded"
                  >
                    <span className="text-gray-500 text-xs whitespace-nowrap">
                      {log.timestamp}
                    </span>
                    <span className={`${getLogColor(log.level)} whitespace-nowrap`}>
                      {getLogPrefix(log.level)}
                    </span>
                    {log.nodeName && (
                      <span className="text-purple-400 whitespace-nowrap">
                        [{log.nodeName}]
                      </span>
                    )}
                    <span className="text-gray-300 flex-1 break-words">
                      {log.message}
                    </span>
                  </div>
                ))}
                <div ref={endRef} />
              </div>
            )}
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}

export default Terminal

