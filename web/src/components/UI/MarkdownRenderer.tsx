import React from 'react'
import { cn } from '@/utils/cn.ts'

interface MarkdownRendererProps {
  content: string
  className?: string
}

const MarkdownRenderer: React.FC<MarkdownRendererProps> = ({ 
  content, 
  className = '' 
}) => {
  // 简单的Markdown渲染器
  const renderMarkdown = (text: string) => {
    // 处理代码块
    text = text.replace(/```(\w+)?\n([\s\S]*?)```/g, (_match, _lang, code) => {
      return `<pre class="bg-gray-100 dark:bg-gray-800 p-4 rounded-lg overflow-x-auto my-4"><code class="text-sm">${code.trim()}</code></pre>`
    })

    // 处理行内代码
    text = text.replace(/`([^`]+)`/g, '<code class="bg-gray-100 dark:bg-gray-800 px-1 py-0.5 rounded text-sm font-mono">$1</code>')

    // 处理粗体
    text = text.replace(/\*\*(.*?)\*\*/g, '<strong class="font-bold">$1</strong>')

    // 处理斜体
    text = text.replace(/\*(.*?)\*/g, '<em class="italic">$1</em>')

    // 处理链接
    text = text.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" class="text-primary hover:underline" target="_blank" rel="noopener noreferrer">$1</a>')

    // 处理列表
    text = text.replace(/^[\s]*[-*+]\s(.+)$/gm, '<li class="ml-4">$1</li>')
    text = text.replace(/(<li.*<\/li>)/s, '<ul class="list-disc list-inside my-2 space-y-1">$1</ul>')

    // 处理数字列表
    text = text.replace(/^[\s]*\d+\.\s(.+)$/gm, '<li class="ml-4">$1</li>')
    text = text.replace(/(<li.*<\/li>)/s, '<ol class="list-decimal list-inside my-2 space-y-1">$1</ol>')

    // 处理标题
    text = text.replace(/^### (.*$)/gim, '<h3 class="text-lg font-semibold mt-4 mb-2">$1</h3>')
    text = text.replace(/^## (.*$)/gim, '<h2 class="text-xl font-semibold mt-4 mb-2">$1</h2>')
    text = text.replace(/^# (.*$)/gim, '<h1 class="text-2xl font-bold mt-4 mb-2">$1</h1>')

    // 处理换行
    text = text.replace(/\n/g, '<br>')

    return text
  }

  return (
    <div 
      className={cn('prose prose-sm max-w-none', className)}
      dangerouslySetInnerHTML={{ __html: renderMarkdown(content) }}
    />
  )
}

export default MarkdownRenderer
