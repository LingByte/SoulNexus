import { useState, useCallback, useRef } from 'react'
import { DragType, DragDirection } from '../components/UI/DragDrop'

// 拖拽状态接口
interface DragState {
  isDragging: boolean
  isDragOver: boolean
  dragData: any
  dragPosition: { x: number; y: number } | null
  dragOffset: { x: number; y: number }
}

// 拖拽配置接口
interface DragConfig {
  type: DragType
  direction?: DragDirection
  snapToGrid?: boolean
  gridSize?: number
  constrainToParent?: boolean
  onDragStart?: (data: any) => void
  onDragEnd?: (data: any) => void
  onDragMove?: (position: { x: number; y: number }, data: any) => void
  onDrop?: (data: any) => void
}

// 文件拖拽配置接口
interface FileDragConfig extends DragConfig {
  type: DragType.FILE
  accept?: string
  multiple?: boolean
  maxSize?: number
  maxFiles?: number
  onFileDrop: (files: File[]) => void
}

// 元素拖拽配置接口
interface ElementDragConfig extends DragConfig {
  type: DragType.ELEMENT
  onElementDrop: (dragData: any, dropData: any) => void
}

// 混合拖拽配置接口
interface BothDragConfig extends DragConfig {
  type: DragType.BOTH
  onFileDrop: (files: File[]) => void
  onElementDrop: (dragData: any, dropData: any) => void
  accept?: string
  multiple?: boolean
  maxSize?: number
  maxFiles?: number
}

// 联合类型
type UseDragDropConfig = FileDragConfig | ElementDragConfig | BothDragConfig

// 拖拽钩子返回值接口
interface UseDragDropReturn {
  dragState: DragState
  dragProps: {
    draggable?: boolean
    onDragStart: (e: React.DragEvent) => void
    onDragEnd: (e: React.DragEvent) => void
    onDragOver: (e: React.DragEvent) => void
    onDragEnter: (e: React.DragEvent) => void
    onDragLeave: (e: React.DragEvent) => void
    onDrop: (e: React.DragEvent) => void
  }
  dropProps: {
    onDragOver: (e: React.DragEvent) => void
    onDragEnter: (e: React.DragEvent) => void
    onDragLeave: (e: React.DragEvent) => void
    onDrop: (e: React.DragEvent) => void
  }
  setDragData: (data: any) => void
  clearDragData: () => void
}

/**
 * 拖拽钩子
 * 提供拖拽功能的完整状态管理和事件处理
 */
export const useDragDrop = (config: UseDragDropConfig): UseDragDropReturn => {
  const [dragState, setDragState] = useState<DragState>({
    isDragging: false,
    isDragOver: false,
    dragData: null,
    dragPosition: null,
    dragOffset: { x: 0, y: 0 }
  })

  const dragCounterRef = useRef(0)

  // 文件验证函数
  const validateFile = useCallback((file: File, maxSize?: number) => {
    if (maxSize && file.size > maxSize) {
      throw new Error(`文件 ${file.name} 超过最大大小限制 (${Math.round(maxSize / 1024 / 1024)}MB)`)
    }
    return true
  }, [])

  // 处理拖拽开始
  const handleDragStart = useCallback((e: React.DragEvent) => {
    const rect = e.currentTarget.getBoundingClientRect()
    
    setDragState(prev => ({
      ...prev,
      isDragging: true,
      dragData: prev.dragData,
      dragOffset: {
        x: e.clientX - rect.left,
        y: e.clientY - rect.top
      }
    }))

    // 设置拖拽数据
    if (dragState.dragData !== null) {
      e.dataTransfer.setData('application/json', JSON.stringify(dragState.dragData))
    }

    // 设置拖拽效果
    if (config.type === DragType.FILE || config.type === DragType.BOTH) {
      e.dataTransfer.effectAllowed = 'copy'
    } else {
      e.dataTransfer.effectAllowed = 'move'
    }

    config.onDragStart?.(dragState.dragData)
  }, [config, dragState.dragData])

  // 处理拖拽结束
  const handleDragEnd = useCallback((_: React.DragEvent) => {
    setDragState(prev => ({
      ...prev,
      isDragging: false,
      dragPosition: null
    }))

    config.onDragEnd?.(dragState.dragData)
  }, [config, dragState.dragData])

  // 处理拖拽悬停
  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()

    setDragState(prev => ({
      ...prev,
      dragPosition: { x: e.clientX, y: e.clientY }
    }))

    // 设置拖拽效果
    if (config.type === DragType.FILE || config.type === DragType.BOTH) {
      e.dataTransfer.dropEffect = 'copy'
    } else {
      e.dataTransfer.dropEffect = 'move'
    }

    config.onDragMove?.(dragState.dragPosition || { x: 0, y: 0 }, dragState.dragData)
  }, [config, dragState.dragPosition, dragState.dragData])

  // 处理拖拽进入
  const handleDragEnter = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()

    dragCounterRef.current += 1
    setDragState(prev => ({
      ...prev,
      isDragOver: true
    }))
  }, [])

  // 处理拖拽离开
  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()

    dragCounterRef.current -= 1
    if (dragCounterRef.current <= 0) {
      setDragState(prev => ({
        ...prev,
        isDragOver: false
      }))
    }
  }, [])

  // 处理文件拖拽
  const handleFileDrop = useCallback((e: React.DragEvent) => {
    const files = Array.from(e.dataTransfer.files)
    
    if (config.type === DragType.FILE || config.type === DragType.BOTH) {
      const fileConfig = config as FileDragConfig | BothDragConfig
      
      // 检查文件数量限制
      if (!fileConfig.multiple && files.length > 1) {
        console.warn('只能上传一个文件')
        return
      }

      if (fileConfig.maxFiles && files.length > fileConfig.maxFiles) {
        console.warn(`最多只能上传 ${fileConfig.maxFiles} 个文件`)
        return
      }

      try {
        // 验证文件
        files.forEach(file => validateFile(file, fileConfig.maxSize))
        fileConfig.onFileDrop(files)
      } catch (error) {
        console.error('文件验证失败:', error)
      }
    }
  }, [config, validateFile])

  // 处理元素拖拽
  const handleElementDrop = useCallback((e: React.DragEvent) => {
    if (config.type === DragType.ELEMENT || config.type === DragType.BOTH) {
      const elementConfig = config as ElementDragConfig | BothDragConfig
      
      try {
        const dragData = JSON.parse(e.dataTransfer.getData('application/json'))
        const dropData = dragState.dragData
        
        elementConfig.onElementDrop(dragData, dropData)
      } catch (error) {
        console.warn('解析拖拽数据失败:', error)
      }
    }
  }, [config, dragState.dragData])

  // 处理拖拽放置
  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()

    setDragState(prev => ({
      ...prev,
      isDragOver: false
    }))

    dragCounterRef.current = 0

    // 根据拖拽类型处理
    if (e.dataTransfer.files.length > 0) {
      handleFileDrop(e)
    } else {
      handleElementDrop(e)
    }

    config.onDrop?.(dragState.dragData)
  }, [config, handleFileDrop, handleElementDrop, dragState.dragData])

  // 设置拖拽数据
  const setDragData = useCallback((data: any) => {
    setDragState(prev => ({
      ...prev,
      dragData: data
    }))
  }, [])

  // 清除拖拽数据
  const clearDragData = useCallback(() => {
    setDragState(prev => ({
      ...prev,
      dragData: null
    }))
  }, [])

  // 拖拽属性
  const dragProps = {
    draggable: config.type === DragType.ELEMENT || config.type === DragType.BOTH,
    onDragStart: handleDragStart,
    onDragEnd: handleDragEnd,
    onDragOver: handleDragOver,
    onDragEnter: handleDragEnter,
    onDragLeave: handleDragLeave,
    onDrop: handleDrop
  }

  // 放置属性
  const dropProps = {
    onDragOver: handleDragOver,
    onDragEnter: handleDragEnter,
    onDragLeave: handleDragLeave,
    onDrop: handleDrop
  }

  return {
    dragState,
    dragProps,
    dropProps,
    setDragData,
    clearDragData
  }
}

export default useDragDrop
export type { UseDragDropConfig, UseDragDropReturn, DragState }
