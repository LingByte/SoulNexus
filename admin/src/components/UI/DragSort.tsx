import React, { useState, useRef, useCallback } from 'react'
import { cn } from '@/utils/cn.ts'

// 拖拽排序项接口
interface DragSortItem {
  id: string | number
  data: any
  disabled?: boolean
}

// 拖拽排序属性接口
interface DragSortProps {
  items: DragSortItem[]
  onSort: (newItems: DragSortItem[]) => void
  children: (item: DragSortItem, index: number, isDragging: boolean) => React.ReactNode
  className?: string
  itemClassName?: string
  disabled?: boolean
  direction?: 'horizontal' | 'vertical'
  gap?: number
  placeholder?: React.ReactNode
  placeholderClassName?: string
  animation?: boolean
  animationDuration?: number
}

// 拖拽状态接口
interface DragState {
  draggedIndex: number | null
  draggedOverIndex: number | null
  draggedItem: DragSortItem | null
  isDragging: boolean
  dragOffset: { x: number; y: number }
}

const DragSort: React.FC<DragSortProps> = ({
  items,
  onSort,
  children,
  className,
  itemClassName,
  disabled = false,
  direction = 'vertical',
  gap = 8,
  placeholder,
  placeholderClassName,
  animation = true,
  animationDuration = 200
}) => {
  const [dragState, setDragState] = useState<DragState>({
    draggedIndex: null,
    draggedOverIndex: null,
    draggedItem: null,
    isDragging: false,
    dragOffset: { x: 0, y: 0 }
  })

  const containerRef = useRef<HTMLDivElement>(null)
  const itemRefs = useRef<(HTMLDivElement | null)[]>([])

  // 获取项目位置
  const getItemPosition = useCallback((index: number) => {
    const item = itemRefs.current[index]
    if (!item) return null

    const rect = item.getBoundingClientRect()
    return {
      top: rect.top,
      left: rect.left,
      width: rect.width,
      height: rect.height
    }
  }, [])

  // 获取拖拽目标索引
  const getDragTargetIndex = useCallback((clientX: number, clientY: number) => {
    if (!containerRef.current) return null

    const containerRect = containerRef.current.getBoundingClientRect()
    const relativeX = clientX - containerRect.left
    const relativeY = clientY - containerRect.top

    for (let i = 0; i < items.length; i++) {
      const position = getItemPosition(i)
      if (!position) continue

      const relativeItemTop = position.top - containerRect.top
      const relativeItemLeft = position.left - containerRect.left

      if (direction === 'vertical') {
        if (relativeY >= relativeItemTop && relativeY <= relativeItemTop + position.height) {
          return relativeX < relativeItemLeft + position.width / 2 ? i : i + 1
        }
      } else {
        if (relativeX >= relativeItemLeft && relativeX <= relativeItemLeft + position.width) {
          return relativeY < relativeItemTop + position.height / 2 ? i : i + 1
        }
      }
    }

    return items.length
  }, [items.length, direction, getItemPosition])

  // 处理拖拽开始
  const handleDragStart = useCallback((e: React.DragEvent, index: number) => {
    if (disabled || items[index].disabled) {
      e.preventDefault()
      return
    }

    const item = items[index]
    const rect = e.currentTarget.getBoundingClientRect()
    const offsetX = e.clientX - rect.left
    const offsetY = e.clientY - rect.top
    
    setDragState({
      draggedIndex: index,
      draggedOverIndex: null,
      draggedItem: item,
      isDragging: true,
      dragOffset: {
        x: offsetX,
        y: offsetY
      }
    })

    // 设置拖拽数据
    e.dataTransfer.setData('application/json', JSON.stringify({ index, item }))
    e.dataTransfer.effectAllowed = 'move'

    // 设置拖拽图像
    if (e.dataTransfer.setDragImage) {
      const dragImage = e.currentTarget.cloneNode(true) as HTMLElement
      dragImage.style.opacity = '0.8'
      dragImage.style.transform = 'rotate(2deg)'
      dragImage.style.pointerEvents = 'none'
      dragImage.style.boxShadow = '0 10px 25px rgba(0, 0, 0, 0.2)'
      document.body.appendChild(dragImage)
      e.dataTransfer.setDragImage(dragImage, offsetX, offsetY)
      
      // 清理拖拽图像
      setTimeout(() => {
        if (document.body.contains(dragImage)) {
          document.body.removeChild(dragImage)
        }
      }, 0)
    }
  }, [disabled, items])

  // 处理拖拽结束
  const handleDragEnd = useCallback(() => {
    setDragState({
      draggedIndex: null,
      draggedOverIndex: null,
      draggedItem: null,
      isDragging: false,
      dragOffset: { x: 0, y: 0 }
    })
  }, [])

  // 处理拖拽悬停
  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (disabled) return

    const targetIndex = getDragTargetIndex(e.clientX, e.clientY)
    if (targetIndex !== null && targetIndex !== dragState.draggedOverIndex) {
      setDragState(prev => ({
        ...prev,
        draggedOverIndex: targetIndex
      }))
    }

    e.dataTransfer.dropEffect = 'move'
  }, [disabled, getDragTargetIndex, dragState.draggedOverIndex])

  // 处理拖拽离开
  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    
    // 检查是否真的离开了容器
    if (!containerRef.current?.contains(e.relatedTarget as Node)) {
      setDragState(prev => ({
        ...prev,
        draggedOverIndex: null
      }))
    }
  }, [])

  // 处理拖拽放置
  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (disabled) return

    try {
      const dragData = JSON.parse(e.dataTransfer.getData('application/json'))
      const { index: fromIndex } = dragData
      const toIndex = dragState.draggedOverIndex

      if (fromIndex !== toIndex && toIndex !== null) {
        const newItems = [...items]
        const [draggedItem] = newItems.splice(fromIndex, 1)
        newItems.splice(toIndex, 0, draggedItem)
        onSort(newItems)
      }
    } catch (error) {
      console.warn('解析拖拽数据失败:', error)
    }

    setDragState({
      draggedIndex: null,
      draggedOverIndex: null,
      draggedItem: null,
      isDragging: false,
      dragOffset: { x: 0, y: 0 }
    })
  }, [disabled, items, onSort, dragState.draggedOverIndex])

  // 渲染占位符
  const renderPlaceholder = () => {
    if (dragState.draggedOverIndex === null || dragState.draggedIndex === null) return null

    const placeholderContent = placeholder || (
      <div className="h-12 bg-primary/20 border-2 border-dashed border-primary rounded-lg flex items-center justify-center">
        <span className="text-primary text-sm">放置位置</span>
      </div>
    )

    return (
      <div
        className={cn(
          'transition-all duration-200',
          placeholderClassName
        )}
        style={{
          order: dragState.draggedOverIndex,
          marginBottom: direction === 'vertical' ? `${gap}px` : 0,
          marginRight: direction === 'horizontal' ? `${gap}px` : 0
        }}
      >
        {placeholderContent}
      </div>
    )
  }

  // 获取项目样式
  const getItemStyle = (index: number) => {
    const isDragging = dragState.draggedIndex === index
    const isPlaceholder = dragState.draggedOverIndex === index && dragState.draggedIndex !== null

    return {
      order: isPlaceholder ? dragState.draggedOverIndex ?? index : index,
      opacity: isDragging ? 0.3 : 1,
      transform: isDragging ? 'rotate(2deg) scale(1.05)' : 'none',
      transition: animation ? `all ${animationDuration}ms ease` : 'none',
      marginBottom: direction === 'vertical' ? `${gap}px` : 0,
      marginRight: direction === 'horizontal' ? `${gap}px` : 0,
      zIndex: isDragging ? 1000 : 'auto',
      boxShadow: isDragging ? '0 10px 25px rgba(0, 0, 0, 0.2)' : 'none'
    }
  }

  return (
    <div
      ref={containerRef}
      className={cn(
        'relative',
        direction === 'horizontal' ? 'flex flex-wrap' : 'flex flex-col',
        className
      )}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      {items.map((item, index) => (
        <div
          key={item.id}
          ref={el => itemRefs.current[index] = el}
          className={cn(
            'cursor-move select-none',
            item.disabled && 'opacity-50 cursor-not-allowed',
            itemClassName
          )}
          style={getItemStyle(index)}
          draggable={!disabled && !item.disabled}
          onDragStart={(e) => handleDragStart(e, index)}
          onDragEnd={handleDragEnd}
        >
          {children(item, index, dragState.draggedIndex === index)}
        </div>
      ))}
      {renderPlaceholder()}
    </div>
  )
}

export default DragSort
export type { DragSortProps, DragSortItem }
