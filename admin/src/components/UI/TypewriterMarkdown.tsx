import React, { useState } from 'react'
import { Typewriter } from '../UX/MicroInteractions'
import MarkdownRenderer from './MarkdownRenderer'
import { cn } from '@/utils/cn.ts'

interface TypewriterMarkdownProps {
  content: string
  speed?: number
  className?: string
  showMarkdownAfterTypewriter?: boolean
  onComplete?: () => void
}

const TypewriterMarkdown: React.FC<TypewriterMarkdownProps> = ({
  content,
  speed = 30,
  className = '',
  showMarkdownAfterTypewriter = true,
  onComplete
}) => {
  const [showMarkdown, setShowMarkdown] = useState(false)
  const [typewriterComplete, setTypewriterComplete] = useState(false)

  const handleTypewriterComplete = () => {
    setTypewriterComplete(true)
    if (showMarkdownAfterTypewriter) {
      // 延迟显示Markdown，让用户有时间阅读Typewriter效果
      setTimeout(() => {
        setShowMarkdown(true)
        onComplete?.()
      }, 1000)
    } else {
      onComplete?.()
    }
  }

  // 检测内容是否包含Markdown语法
  const hasMarkdownSyntax = (text: string): boolean => {
    const markdownPatterns = [
      /```[\s\S]*?```/, // 代码块
      /`[^`]+`/, // 行内代码
      /\*\*[^*]+\*\*/, // 粗体
      /\*[^*]+\*/, // 斜体
      /\[([^\]]+)\]\(([^)]+)\)/, // 链接
      /^#{1,6}\s/m, // 标题
      /^[\s]*[-*+]\s/m, // 列表
      /^[\s]*\d+\.\s/m // 数字列表
    ]
    
    return markdownPatterns.some(pattern => pattern.test(text))
  }

  const shouldShowMarkdown = hasMarkdownSyntax(content) && showMarkdown

  return (
    <div className={cn('relative', className)}>
      {/* Typewriter效果 */}
      <div className={cn(
        'transition-opacity duration-500',
        shouldShowMarkdown ? 'opacity-0 absolute' : 'opacity-100'
      )}>
        <Typewriter 
          text={content} 
          speed={speed}
          className="whitespace-pre-wrap"
          onComplete={handleTypewriterComplete}
        />
      </div>

      {/* Markdown渲染 */}
      {shouldShowMarkdown && (
        <div className="animate-fade-in">
          <MarkdownRenderer 
            content={content} 
            className="prose-sm max-w-none"
          />
        </div>
      )}

      {/* 切换按钮 */}
      {typewriterComplete && hasMarkdownSyntax(content) && !showMarkdown && (
        <div className="mt-2 flex justify-end">
          <button
            onClick={() => setShowMarkdown(true)}
            className="text-xs text-primary hover:text-primary/80 underline"
          >
            查看格式化内容
          </button>
        </div>
      )}

      {/* 返回按钮 */}
      {showMarkdown && (
        <div className="mt-2 flex justify-end">
          <button
            onClick={() => setShowMarkdown(false)}
            className="text-xs text-gray-500 hover:text-gray-700 underline"
          >
            返回打字效果
          </button>
        </div>
      )}
    </div>
  )
}

export default TypewriterMarkdown
