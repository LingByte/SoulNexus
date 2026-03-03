import { useCallback, RefObject } from 'react'
import type { WorkflowNode } from '../types/workflow'

interface UseCanvasControlProps {
  nodes: WorkflowNode[]
  canvasRef: RefObject<HTMLDivElement>
  canvasOffset: { x: number; y: number }
  canvasScale: number
  setCanvasOffset: (offset: { x: number; y: number }) => void
  setCanvasScale: (scale: number) => void
}

/**
 * 画布控制 hooks
 * 包含缩放、平移、居中等画布操作
 */
export const useCanvasControl = ({
  nodes,
  canvasRef,
  canvasOffset,
  canvasScale,
  setCanvasOffset,
  setCanvasScale
}: UseCanvasControlProps) => {
  // 重置画布视图
  const resetCanvasView = useCallback(() => {
    setCanvasOffset({ x: 100000, y: 100000 })
    setCanvasScale(1)
  }, [setCanvasOffset, setCanvasScale])

  // 放大
  const zoomIn = useCallback(() => {
    setCanvasScale(prev => Math.min(prev * 1.2, 3))
  }, [setCanvasScale])

  // 缩小
  const zoomOut = useCallback(() => {
    setCanvasScale(prev => Math.max(prev / 1.2, 0.3))
  }, [setCanvasScale])

  // 居中显示所有节点
  const centerOnNodes = useCallback(() => {
    if (nodes.length === 0) return
    
    const bounds = nodes.reduce((acc, node) => {
      return {
        minX: Math.min(acc.minX, node.position.x),
        minY: Math.min(acc.minY, node.position.y),
        maxX: Math.max(acc.maxX, node.position.x + 180),
        maxY: Math.max(acc.maxY, node.position.y + 50)
      }
    }, { minX: Infinity, minY: Infinity, maxX: -Infinity, maxY: -Infinity })
    
    const centerX = (bounds.minX + bounds.maxX) / 2
    const centerY = (bounds.minY + bounds.maxY) / 2
    
    const rect = canvasRef.current?.getBoundingClientRect()
    if (rect) {
      const viewportCenterX = rect.width / 2
      const viewportCenterY = rect.height / 2
      setCanvasOffset({ 
        x: viewportCenterX + 100000 - centerX * canvasScale, 
        y: viewportCenterY + 100000 - centerY * canvasScale 
      })
    }
  }, [nodes, canvasScale, canvasRef, setCanvasOffset])

  return {
    resetCanvasView,
    zoomIn,
    zoomOut,
    centerOnNodes
  }
}
