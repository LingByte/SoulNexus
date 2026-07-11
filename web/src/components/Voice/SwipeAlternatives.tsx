import React, { useState, useEffect, useCallback } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { ChevronLeft, ChevronRight, GitBranch, RefreshCw } from 'lucide-react'
import { cn } from '@/utils/cn'
import type { MessageAlternative } from '@/api/chat'
import {
  regenerateMessage,
  branchFromMessage,
} from '@/api/chat'

export interface SwipeAlternativesProps {
  parentMessageId: string
  currentAlternative: MessageAlternative | null
  alternatives: MessageAlternative[]
  onAlternativeChange: (alt: MessageAlternative) => void
  onRegenerateStart: () => void
  onRegenerateComplete: (result: { id: string; content: string; branchIndex: number }) => void
  onBranch: (branchSessionId: string, title: string) => void
  disabled?: boolean
  className?: string
}

const SwipeAlternatives: React.FC<SwipeAlternativesProps> = ({
  parentMessageId,
  currentAlternative,
  alternatives: initialAlternatives,
  onAlternativeChange,
  onRegenerateStart,
  onRegenerateComplete,
  onBranch,
  disabled = false,
  className,
}) => {
  const [alternatives, setAlternatives] = useState<MessageAlternative[]>(initialAlternatives || [])
  const [currentIndex, setCurrentIndex] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Sync with parent alternatives
  useEffect(() => {
    setAlternatives(initialAlternatives || [])
  }, [initialAlternatives])

  // Sync current index
  useEffect(() => {
    if (currentAlternative && alternatives.length > 0) {
      const idx = alternatives.findIndex(a => a.id === currentAlternative.id)
      if (idx >= 0) setCurrentIndex(idx)
    }
  }, [currentAlternative, alternatives])

  const total = alternatives.length
  const hasMultiple = total > 1

  const handlePrev = useCallback(() => {
    if (currentIndex > 0) {
      const newIdx = currentIndex - 1
      setCurrentIndex(newIdx)
      onAlternativeChange(alternatives[newIdx])
    }
  }, [currentIndex, alternatives, onAlternativeChange])

  const handleNext = useCallback(() => {
    if (currentIndex < total - 1) {
      const newIdx = currentIndex + 1
      setCurrentIndex(newIdx)
      onAlternativeChange(alternatives[newIdx])
    }
  }, [currentIndex, total, alternatives, onAlternativeChange])

  const handleRegenerate = useCallback(async () => {
    if (loading || disabled) return
    setLoading(true)
    setError(null)
    onRegenerateStart()

    try {
      const res = await regenerateMessage(parentMessageId)
      if (res.code === 200 && res.data) {
        const newAlt: MessageAlternative = {
          id: res.data.id,
          content: res.data.content,
          branchIndex: res.data.branchIndex,
          isActiveBranch: res.data.isActiveBranch,
          tokenCount: res.data.tokenCount,
          model: res.data.model,
          provider: res.data.provider,
          createdAt: res.data.createdAt,
        }

        setAlternatives(prev => {
          // deactivate all, add new one as active
          const updated = prev.map(a => ({ ...a, isActiveBranch: false }))
          updated.push(newAlt)
          return updated
        })
        setCurrentIndex(alternatives.length) // the new one is last
        onRegenerateComplete({
          id: res.data.id,
          content: res.data.content,
          branchIndex: res.data.branchIndex,
        })
      }
    } catch (err: any) {
      setError(err?.message || 'Regenerate failed')
    } finally {
      setLoading(false)
    }
  }, [parentMessageId, loading, disabled, alternatives.length, onRegenerateStart, onRegenerateComplete])

  const handleBranch = useCallback(async () => {
    if (loading || disabled) return
    setLoading(true)
    try {
      const res = await branchFromMessage(parentMessageId)
      if (res.code === 200 && res.data) {
        onBranch(res.data.branchSessionId, res.data.title)
      }
    } catch (err: any) {
      setError(err?.message || 'Branch failed')
    } finally {
      setLoading(false)
    }
  }, [parentMessageId, loading, disabled, onBranch])

  return (
    <div className={cn('flex flex-col gap-1', className)}>
      {/* Navigation bar */}
      <div className="flex items-center justify-between gap-1 px-1">
        <div className="flex items-center gap-1">
          {hasMultiple && (
            <>
              <button
                type="button"
                onClick={handlePrev}
                disabled={currentIndex === 0 || disabled}
                className={cn(
                  'p-0.5 rounded transition-colors',
                  currentIndex === 0
                    ? 'text-gray-300 dark:text-neutral-600 cursor-not-allowed'
                    : 'text-gray-500 hover:text-gray-700 hover:bg-gray-200 dark:text-gray-400 dark:hover:text-gray-200 dark:hover:bg-neutral-700'
                )}
                title="上一个回复"
              >
                <ChevronLeft className="w-3.5 h-3.5" />
              </button>

              <span className="text-[11px] tabular-nums text-gray-400 dark:text-gray-500 select-none min-w-[2rem] text-center">
                {currentIndex + 1}/{total}
              </span>

              <button
                type="button"
                onClick={handleNext}
                disabled={currentIndex >= total - 1 || disabled}
                className={cn(
                  'p-0.5 rounded transition-colors',
                  currentIndex >= total - 1
                    ? 'text-gray-300 dark:text-neutral-600 cursor-not-allowed'
                    : 'text-gray-500 hover:text-gray-700 hover:bg-gray-200 dark:text-gray-400 dark:hover:text-gray-200 dark:hover:bg-neutral-700'
                )}
                title="下一个回复"
              >
                <ChevronRight className="w-3.5 h-3.5" />
              </button>
            </>
          )}
          {!hasMultiple && total === 1 && (
            <span className="text-[11px] tabular-nums text-gray-400 dark:text-gray-500 select-none">1/1</span>
          )}
        </div>

        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={handleRegenerate}
            disabled={loading || disabled}
            className={cn(
              'p-0.5 rounded transition-colors',
              loading
                ? 'text-blue-400 animate-spin'
                : 'text-gray-400 hover:text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/30'
            )}
            title="重新生成"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>

          <button
            type="button"
            onClick={handleBranch}
            disabled={loading || disabled}
            className="p-0.5 rounded transition-colors text-gray-400 hover:text-purple-500 hover:bg-purple-50 dark:hover:bg-purple-900/30"
            title="从此消息创建分支"
          >
            <GitBranch className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {/* Error message */}
      <AnimatePresence>
        {error && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            className="overflow-hidden"
          >
            <div className="text-[11px] text-red-500 px-1">{error}</div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

export default SwipeAlternatives
