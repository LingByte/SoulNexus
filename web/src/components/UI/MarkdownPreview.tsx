import React from 'react'
import MarkdownPreview from '@uiw/react-markdown-preview'

interface MarkdownPreviewProps {
  content: string
  className?: string
}

const MarkdownPreviewComponent: React.FC<MarkdownPreviewProps> = ({ 
  content, 
  className = '' 
}) => {
  return (
    <div className={`markdown-preview ${className}`}>
      <MarkdownPreview
        source={content}
        style={{
          backgroundColor: 'transparent',
          color: 'inherit',
          fontSize: 'inherit',
          lineHeight: 'inherit',
        }}
        wrapperElement={{
          'data-color-mode': 'light'
        }}
        rehypeRewrite={(node, _index, parent) => {
          // 自定义样式重写
          if (node.type === 'element' && node.tagName === 'pre') {
            node.properties = {
              ...node.properties,
              className: 'bg-gray-50 dark:bg-neutral-600 rounded-lg p-3 overflow-x-auto'
            }
          }
          // @ts-ignore
          if (node.type === 'element' && node.tagName === 'code' && !parent?.tagName) {
            node.properties = {
              ...node.properties,
              className: 'bg-gray-100 dark:bg-neutral-600 px-1 py-0.5 rounded text-sm'
            }
          }
          if (node.type === 'element' && node.tagName === 'blockquote') {
            node.properties = {
              ...node.properties,
              className: 'border-l-4 border-purple-300 dark:border-purple-600 pl-4 italic'
            }
          }
        }}
      />
    </div>
  )
}

export default MarkdownPreviewComponent
