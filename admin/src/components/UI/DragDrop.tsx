import React, { useState, useRef, useCallback } from 'react'
import { cn } from '@/utils/cn.ts'
import { showAlert } from '@/utils/notification.ts'

// 拖拽类型枚举
export enum DragType {
  FILE = 'file',
  ELEMENT = 'element',
  BOTH = 'both'
}

// 拖拽方向枚举
export enum DragDirection {
  HORIZONTAL = 'horizontal',
  VERTICAL = 'vertical',
  BOTH = 'both'
}

// 基础拖拽属性接口
interface BaseDragProps {
  children: React.ReactNode
  className?: string
  disabled?: boolean
  onDragStart?: (e: React.DragEvent) => void
  onDragEnd?: (e: React.DragEvent) => void
  onDragOver?: (e: React.DragEvent) => void
  onDragEnter?: (e: React.DragEvent) => void
  onDragLeave?: (e: React.DragEvent) => void
  onDrop?: (e: React.DragEvent) => void
}

// 文件拖拽属性接口
interface FileDragProps extends BaseDragProps {
  type: DragType.FILE
  onFileDrop: (files: File[]) => void
  accept?: string
  multiple?: boolean
  maxSize?: number
  maxFiles?: number
  showOverlay?: boolean
  overlayText?: string
  overlayIcon?: React.ReactNode
}

// 元素拖拽属性接口
interface ElementDragProps extends BaseDragProps {
  type: DragType.ELEMENT
  draggable?: boolean
  dragData?: any
  onElementDrop?: (dragData: any, dropData: any) => void
  direction?: DragDirection
  snapToGrid?: boolean
  gridSize?: number
  constrainToParent?: boolean
}

// 混合拖拽属性接口
interface BothDragProps extends BaseDragProps {
  type: DragType.BOTH
  onFileDrop: (files: File[]) => void
  onElementDrop?: (dragData: any, dropData: any) => void
  accept?: string
  multiple?: boolean
  maxSize?: number
  maxFiles?: number
  draggable?: boolean
  dragData?: any
  direction?: DragDirection
  snapToGrid?: boolean
  gridSize?: number
  constrainToParent?: boolean
  showOverlay?: boolean
  overlayText?: string
  overlayIcon?: React.ReactNode
}

// 联合类型
type DragDropProps = FileDragProps | ElementDragProps | BothDragProps

// 拖拽状态接口
interface DragState {
  isDragOver: boolean
  isDragging: boolean
  dragCounter: number
  dragPosition: { x: number; y: number } | null
  dragOffset?: { x: number; y: number }
}

const DragDrop: React.FC<DragDropProps> = (props) => {
  const {
    children,
    className,
    disabled = false,
    onDragStart,
    onDragEnd,
    onDragOver,
    onDragEnter,
    onDragLeave,
    onDrop,
    type
  } = props

  const [dragState, setDragState] = useState<DragState>({
    isDragOver: false,
    isDragging: false,
    dragCounter: 0,
    dragPosition: null
  })

  const dropRef = useRef<HTMLDivElement>(null)

  // 文件验证函数
  const validateFile = useCallback((file: File, maxSize?: number) => {
    if (maxSize && file.size > maxSize) {
      throw new Error(`文件 ${file.name} 超过最大大小限制 (${Math.round(maxSize / 1024 / 1024)}MB)`)
    }
    return true
  }, [])

  // 处理拖拽开始
  const handleDragStart = useCallback((e: React.DragEvent) => {
    if (disabled) {
      e.preventDefault()
      return
    }

    const rect = e.currentTarget.getBoundingClientRect()
    const offsetX = e.clientX - rect.left
    const offsetY = e.clientY - rect.top

    setDragState(prev => ({
      ...prev,
      isDragging: true,
      dragOffset: { x: offsetX, y: offsetY }
    }))

    // 设置拖拽数据
    if ('dragData' in props && props.dragData !== undefined) {
      e.dataTransfer.setData('application/json', JSON.stringify(props.dragData))
    }

    // 设置拖拽效果
    if ('direction' in props) {
      e.dataTransfer.effectAllowed = 'move'
    }

    // 设置拖拽图像偏移
    if (e.dataTransfer.setDragImage) {
      const dragImage = e.currentTarget.cloneNode(true) as HTMLElement
      dragImage.style.opacity = '0.8'
      dragImage.style.transform = 'rotate(2deg)'
      dragImage.style.pointerEvents = 'none'
      document.body.appendChild(dragImage)
      e.dataTransfer.setDragImage(dragImage, offsetX, offsetY)

      // 清理拖拽图像
      setTimeout(() => {
        if (document.body.contains(dragImage)) {
          document.body.removeChild(dragImage)
        }
      }, 0)
    }

    onDragStart?.(e)
  }, [disabled, props, onDragStart])

  // 处理拖拽结束
  const handleDragEnd = useCallback((e: React.DragEvent) => {
    setDragState(prev => ({
      ...prev,
      isDragging: false,
      dragPosition: null
    }))
    onDragEnd?.(e)
  }, [onDragEnd])

  // 处理拖拽进入
  const handleDragEnter = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (disabled) return

    setDragState(prev => ({
      ...prev,
      dragCounter: prev.dragCounter + 1,
      isDragOver: true
    }))

    onDragEnter?.(e)
  }, [disabled, onDragEnter])

  // 处理拖拽离开
  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (disabled) return

    setDragState(prev => {
      const newCounter = prev.dragCounter - 1
      return {
        ...prev,
        dragCounter: newCounter,
        isDragOver: newCounter <= 0 ? false : prev.isDragOver
      }
    })

    onDragLeave?.(e)
  }, [disabled, onDragLeave])

  // 处理拖拽悬停
  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (disabled) return

    // 更新拖拽位置
    setDragState(prev => ({
      ...prev,
      dragPosition: { x: e.clientX, y: e.clientY }
    }))

    // 设置拖拽效果
    if (type === DragType.FILE || type === DragType.BOTH) {
      e.dataTransfer.dropEffect = 'copy'
    } else {
      e.dataTransfer.dropEffect = 'move'
    }

    onDragOver?.(e)
  }, [disabled, type, onDragOver])

  // 处理文件拖拽
  const handleFileDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (disabled) return

    const files = Array.from(e.dataTransfer.files)

    if (type === DragType.FILE || type === DragType.BOTH) {
      const fileProps = props as FileDragProps | BothDragProps

      // 检查文件数量限制
      if (!fileProps.multiple && files.length > 1) {
        showAlert('只能上传一个文件', 'warning', '文件限制')
        return
      }

      if (fileProps.maxFiles && files.length > fileProps.maxFiles) {
        showAlert(`最多只能上传 ${fileProps.maxFiles} 个文件`, 'warning', '文件限制')
        return
      }

      try {
        // 验证文件
        files.forEach(file => validateFile(file, fileProps.maxSize))
        fileProps.onFileDrop(files)
      } catch (error) {
        showAlert(error instanceof Error ? error.message : '文件验证失败', 'error', '验证失败')
      }
    }
  }, [disabled, type, props, validateFile])

  // 处理元素拖拽
  const handleElementDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (disabled) return

    if (type === DragType.ELEMENT || type === DragType.BOTH) {
      const elementProps = props as ElementDragProps | BothDragProps

      try {
        const dragData = JSON.parse(e.dataTransfer.getData('application/json'))
        const dropData = elementProps.dragData

        elementProps.onElementDrop?.(dragData, dropData)
      } catch (error) {
        console.warn('解析拖拽数据失败:', error)
      }
    }
  }, [disabled, type, props])

  // 处理拖拽放置
  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (disabled) return

    setDragState(prev => ({
      ...prev,
      isDragOver: false,
      dragCounter: 0,
      dragPosition: null
    }))

    // 根据拖拽类型处理
    if (e.dataTransfer.files.length > 0) {
      handleFileDrop(e)
    } else {
      handleElementDrop(e)
    }

    onDrop?.(e)
  }, [disabled, handleFileDrop, handleElementDrop, onDrop])

  // 获取覆盖层内容
  const getOverlayContent = () => {
    if (type === DragType.FILE || type === DragType.BOTH) {
      const fileProps = props as FileDragProps | BothDragProps
      if (!fileProps.showOverlay) return null

      return (
          <div className="absolute inset-0 bg-primary/20 backdrop-blur-sm border-2 border-dashed border-primary rounded-lg flex items-center justify-center z-50">
            <div className="text-center bg-white/90 dark:bg-neutral-800/90 px-6 py-4 rounded-lg shadow-lg">
              {fileProps.overlayIcon && (
                  <div className="mb-3">
                    {fileProps.overlayIcon}
                  </div>
              )}
              <div className="text-primary font-semibold text-lg mb-2">
                {fileProps.overlayText || '释放文件到此处'}
              </div>
              <div className="text-sm text-primary/80 font-medium">
                支持 {fileProps.accept || '*/*'} 格式
              </div>
            </div>
          </div>
      )
    }
    return null
  }

  // 获取拖拽属性
  const getDragProps = () => {
    const baseProps = {
      ref: dropRef,
      className: cn(
          'relative',
          dragState.isDragOver && !disabled && 'ring-2 ring-primary ring-opacity-50',
          className
      ),
      onDragEnter: handleDragEnter,
      onDragLeave: handleDragLeave,
      onDragOver: handleDragOver,
      onDrop: handleDrop
    }

    if (type === DragType.ELEMENT || type === DragType.BOTH) {
      const elementProps = props as ElementDragProps | BothDragProps
      return {
        ...baseProps,
        draggable: elementProps.draggable !== false,
        onDragStart: handleDragStart,
        onDragEnd: handleDragEnd
      }
    }

    return baseProps
  }

  return (
      <div {...getDragProps()}>
        {children}
        {getOverlayContent()}
      </div>
  )
}

export default DragDrop
export type { DragDropProps, FileDragProps, ElementDragProps, BothDragProps }
