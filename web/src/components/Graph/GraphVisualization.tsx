import React, { useEffect, useRef, useState, useMemo } from 'react'
import { Network } from 'vis-network'
import type { GraphNode, GraphEdge, GraphStats } from '@/api/assistant'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'
import { Loader2, Network as NetworkIcon, Users, MessageCircle, Tag, BookOpen, Filter, RotateCcw, ZoomIn, ZoomOut } from 'lucide-react'
import Button from '@/components/UI/Button'

interface GraphVisualizationProps {
  nodes: GraphNode[]
  edges: GraphEdge[]
  stats: GraphStats
  loading?: boolean
  className?: string
}

const GraphVisualization: React.FC<GraphVisualizationProps> = ({
  nodes,
  edges,
  stats,
  loading = false,
  className = ''
}) => {
  const networkRef = useRef<HTMLDivElement>(null)
  const networkInstanceRef = useRef<Network | null>(null)
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null)
  const [filteredNodeTypes, setFilteredNodeTypes] = useState<Set<string>>(new Set())
  const [showControls, setShowControls] = useState(true)

  // 节点类型颜色映射
  const nodeColors: Record<string, { background: string; border: string }> = {
    Assistant: { background: '#8b5cf6', border: '#6d28d9' },
    User: { background: '#3b82f6', border: '#2563eb' },
    Conversation: { background: '#10b981', border: '#059669' },
    Topic: { background: '#f59e0b', border: '#d97706' },
    Intent: { background: '#ef4444', border: '#dc2626' },
    Knowledge: { background: '#06b6d4', border: '#0891b2' },
  }

  // 关系类型颜色映射
  const edgeColors: Record<string, string> = {
    WITH_ASSISTANT: '#8b5cf6',
    HAS_CONVERSATION: '#10b981',
    DISCUSSES: '#f59e0b',
    HAS_INTENT: '#ef4444',
    HAS_KNOWLEDGE: '#06b6d4',
    LIKES: '#ec4899',
    RELATED_TO: '#6366f1',
  }

  // 获取所有节点类型
  const nodeTypes = useMemo(() => {
    const types = new Set(nodes.map(node => node.type))
    return Array.from(types).sort()
  }, [nodes])

  // 过滤节点和边
  const filteredData = useMemo(() => {
    if (filteredNodeTypes.size === 0) {
      return { nodes, edges }
    }

    const filteredNodes = nodes.filter(node => !filteredNodeTypes.has(node.type))
    const filteredNodeIds = new Set(filteredNodes.map(n => n.id))
    const filteredEdges = edges.filter(
      edge => filteredNodeIds.has(edge.source) && filteredNodeIds.has(edge.target)
    )

    return { nodes: filteredNodes, edges: filteredEdges }
  }, [nodes, edges, filteredNodeTypes])

  // 切换节点类型过滤
  const toggleNodeTypeFilter = (type: string) => {
    setFilteredNodeTypes(prev => {
      const newSet = new Set(prev)
      if (newSet.has(type)) {
        newSet.delete(type)
      } else {
        newSet.add(type)
      }
      return newSet
    })
  }

  // 重置视图
  const resetView = () => {
    if (networkInstanceRef.current) {
      networkInstanceRef.current.fit({
        animation: {
          duration: 500,
          easingFunction: 'easeInOutQuad'
        }
      })
    }
  }

  // 缩放控制
  const zoomIn = () => {
    if (networkInstanceRef.current) {
      const scale = networkInstanceRef.current.getScale()
      networkInstanceRef.current.moveTo({
        scale: scale * 1.2,
        animation: true
      })
    }
  }

  const zoomOut = () => {
    if (networkInstanceRef.current) {
      const scale = networkInstanceRef.current.getScale()
      networkInstanceRef.current.moveTo({
        scale: scale * 0.8,
        animation: true
      })
    }
  }

  useEffect(() => {
    if (!networkRef.current || loading || filteredData.nodes.length === 0) return

    // 转换节点数据 - 优化标签显示
    const visNodes = filteredData.nodes.map(node => {
      const color = nodeColors[node.type] || { background: '#6b7280', border: '#4b5563' }
      
      // 优化标签：如果标签太长，截断并添加省略号
      let label = node.label || node.type
      if (label.length > 15) {
        label = label.substring(0, 15) + '...'
      }

      // 根据节点类型设置不同大小
      let size = 20
      if (node.type === 'Assistant') {
        size = 35
      } else if (node.type === 'User') {
        size = 28
      } else if (node.type === 'Conversation') {
        size = 24
      } else if (node.type === 'Topic' || node.type === 'Intent') {
        size = 22
      } else if (node.type === 'Knowledge') {
        size = 20
      }

      return {
        id: node.id,
        label: label,
        color: {
          background: color.background,
          border: color.border,
          highlight: {
            background: color.background,
            border: '#ffffff',
          },
        },
        shape: node.type === 'Assistant' ? 'diamond' : node.type === 'User' ? 'box' : 'dot',
        size: size,
        font: {
          size: node.type === 'Assistant' ? 16 : node.type === 'User' ? 14 : 12,
          color: '#000000',
          face: 'Arial',
          ...(node.type === 'Assistant' || node.type === 'User' ? { bold: 'bold' as const } : {}),
        },
        title: `${node.type}: ${node.label}\n${JSON.stringify(node.props, null, 2)}`,
        // 添加固定位置（可选，让布局更稳定）
        fixed: false,
        // 添加质量属性，让重要节点更稳定
        mass: node.type === 'Assistant' ? 5 : node.type === 'User' ? 3 : 1,
      }
    })

    // 转换边数据 - 优化显示
    const visEdges = filteredData.edges.map(edge => {
      const color = edgeColors[edge.type] || '#6b7280'
      
      // 简化边标签，避免显示过长
      let label = edge.type
      if (label.length > 12) {
        label = label.replace(/_/g, ' ').substring(0, 12) + '...'
      }

      return {
        id: edge.id,
        from: edge.source,
        to: edge.target,
        label: '', // 默认不显示标签，减少视觉混乱
        color: {
          color: color,
          highlight: color,
          hover: color,
        },
        width: edge.type === 'WITH_ASSISTANT' ? 3 : 2,
        arrows: {
          to: {
            enabled: true,
            scaleFactor: 1.2,
            type: 'arrow',
          },
        },
        title: `${edge.type}\n${JSON.stringify(edge.props, null, 2)}`,
        // 优化边的平滑度
        smooth: {
          enabled: true,
          type: 'continuous',
          roundness: 0.3,
        },
        // 添加选择高亮
        selectionWidth: 3,
      }
    })

    // 创建网络图
    const data = {
      nodes: visNodes,
      edges: visEdges,
    }

    const options = {
      nodes: {
        borderWidth: 2,
        shadow: {
          enabled: true,
          color: 'rgba(0,0,0,0.2)',
          size: 5,
          x: 2,
          y: 2,
        },
        font: {
          size: 14,
          face: 'Arial',
          color: '#000000',
        },
        // 添加节点间距
        margin: {
          top: 10,
          right: 10,
          bottom: 10,
          left: 10,
        },
      },
      edges: {
        width: 2,
        shadow: {
          enabled: true,
          color: 'rgba(0,0,0,0.1)',
          size: 3,
        },
        smooth: {
          enabled: true,
          type: 'continuous',
          roundness: 0.3,
        },
        // 减少边的标签显示
        font: {
          size: 10,
          align: 'middle',
          color: '#000000',
        },
      },
      physics: {
        enabled: true,
        stabilization: {
          enabled: true,
          iterations: 300, // 增加迭代次数，让布局更稳定
          fit: true,
        },
        // 使用改进的力导向算法
        barnesHut: {
          gravitationalConstant: -4000, // 增加斥力，让节点分散更开
          centralGravity: 0.1, // 减少中心引力
          springLength: 300, // 增加理想弹簧长度
          springConstant: 0.02, // 减少弹簧常数
          damping: 0.15, // 增加阻尼，让布局更快稳定
          avoidOverlap: 1, // 避免重叠
        },
        // 添加最大速度限制
        maxVelocity: 50,
        // 添加最小速度阈值
        minVelocity: 0.1,
      },
      interaction: {
        hover: true,
        tooltipDelay: 200,
        zoomView: true,
        dragView: true,
        // 添加选择功能
        selectConnectedEdges: true,
        // 添加导航按钮
        navigationButtons: false, // 我们自定义控制按钮
      },
      // 添加布局配置
      layout: {
        improvedLayout: true,
        hierarchical: {
          enabled: false, // 使用力导向布局
        },
      },
    }

    const network = new Network(networkRef.current, data, options)

    // 节点点击事件
    network.on('click', (params) => {
      if (params.nodes.length > 0) {
        const nodeId = params.nodes[0]
        const node = nodes.find(n => n.id === nodeId)
        setSelectedNode(node || null)
      } else {
        setSelectedNode(null)
      }
    })

    networkInstanceRef.current = network

    // 布局稳定后自动适应视图
    setTimeout(() => {
      network.fit({
        animation: {
          duration: 1000,
          easingFunction: 'easeInOutQuad'
        }
      })
    }, 1500)

    networkInstanceRef.current = network

    return () => {
      if (networkInstanceRef.current) {
        networkInstanceRef.current.destroy()
        networkInstanceRef.current = null
      }
    }
  }, [filteredData.nodes, filteredData.edges, loading])

  if (loading) {
    return (
      <Card className={className}>
        <CardContent className="flex items-center justify-center h-96">
          <div className="flex flex-col items-center gap-4">
            <Loader2 className="w-8 h-8 animate-spin text-primary" />
            <p className="text-gray-600 dark:text-gray-400">加载图数据中...</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (nodes.length === 0) {
    return (
      <Card className={className}>
        <CardContent className="flex items-center justify-center h-96">
          <div className="flex flex-col items-center gap-4 text-center">
            <NetworkIcon className="w-16 h-16 text-gray-400" />
            <p className="text-gray-600 dark:text-gray-400">暂无图数据</p>
            <p className="text-sm text-gray-500 dark:text-gray-500">
              该助手尚未启用图记忆功能或还没有对话记录
            </p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className={className}>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <NetworkIcon className="w-5 h-5" />
            知识图谱可视化
          </CardTitle>
        </CardHeader>
        <CardContent>
          {/* 统计信息 */}
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-6">
            <div className="flex flex-col items-center p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <NetworkIcon className="w-5 h-5 text-primary mb-1" />
              <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {stats.totalNodes}
              </div>
              <div className="text-xs text-gray-600 dark:text-gray-400">节点</div>
            </div>
            <div className="flex flex-col items-center p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <NetworkIcon className="w-5 h-5 text-blue-500 mb-1" />
              <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {stats.totalEdges}
              </div>
              <div className="text-xs text-gray-600 dark:text-gray-400">关系</div>
            </div>
            <div className="flex flex-col items-center p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <Users className="w-5 h-5 text-blue-500 mb-1" />
              <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {stats.usersCount}
              </div>
              <div className="text-xs text-gray-600 dark:text-gray-400">用户</div>
            </div>
            <div className="flex flex-col items-center p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <MessageCircle className="w-5 h-5 text-green-500 mb-1" />
              <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {stats.conversationsCount}
              </div>
              <div className="text-xs text-gray-600 dark:text-gray-400">对话</div>
            </div>
            <div className="flex flex-col items-center p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <Tag className="w-5 h-5 text-yellow-500 mb-1" />
              <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {stats.topicsCount}
              </div>
              <div className="text-xs text-gray-600 dark:text-gray-400">主题</div>
            </div>
            <div className="flex flex-col items-center p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <BookOpen className="w-5 h-5 text-cyan-500 mb-1" />
              <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                {stats.knowledgeCount}
              </div>
              <div className="text-xs text-gray-600 dark:text-gray-400">知识</div>
            </div>
          </div>

          {/* 控制面板 */}
          {showControls && (
            <div className="mb-4 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
              <div className="flex items-center justify-between mb-4">
                <h3 className="font-semibold text-gray-900 dark:text-gray-100 flex items-center gap-2">
                  <Filter className="w-4 h-4" />
                  节点类型过滤
                </h3>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setShowControls(!showControls)}
                >
                  收起
                </Button>
              </div>
              
              <div className="flex flex-wrap gap-2 mb-4">
                {nodeTypes.map(type => {
                  const isFiltered = filteredNodeTypes.has(type)
                  const color = nodeColors[type] || { background: '#6b7280' }
                  return (
                    <button
                      key={type}
                      onClick={() => toggleNodeTypeFilter(type)}
                      className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-all ${
                        isFiltered
                          ? 'bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400 line-through'
                          : 'text-white shadow-sm hover:opacity-80'
                      }`}
                      style={{
                        backgroundColor: isFiltered ? undefined : color.background,
                      }}
                    >
                      {type}
                    </button>
                  )
                })}
              </div>

              {/* 视图控制 */}
              <div className="flex items-center gap-2 pt-3 border-t border-gray-200 dark:border-gray-700">
                <span className="text-sm text-gray-600 dark:text-gray-400 mr-2">视图控制:</span>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={resetView}
                  leftIcon={<RotateCcw className="w-4 h-4" />}
                >
                  重置视图
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={zoomIn}
                  leftIcon={<ZoomIn className="w-4 h-4" />}
                >
                  放大
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={zoomOut}
                  leftIcon={<ZoomOut className="w-4 h-4" />}
                >
                  缩小
                </Button>
              </div>
            </div>
          )}

          {!showControls && (
            <div className="mb-4">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowControls(true)}
                leftIcon={<Filter className="w-4 h-4" />}
              >
                显示控制面板
              </Button>
            </div>
          )}

          {/* 图可视化 */}
          <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden relative">
            <div ref={networkRef} className="w-full h-[700px] bg-white dark:bg-gray-900" />
            
            {/* 显示过滤状态 */}
            {filteredNodeTypes.size > 0 && (
              <div className="absolute top-4 right-4 px-3 py-1.5 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 rounded-lg text-sm">
                已隐藏 {filteredNodeTypes.size} 种节点类型
              </div>
            )}
          </div>

          {/* 节点详情 */}
          {selectedNode && (
            <div className="mt-4 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
              <h3 className="font-semibold text-gray-900 dark:text-gray-100 mb-2">
                节点详情: {selectedNode.label}
              </h3>
              <div className="space-y-1 text-sm">
                <div>
                  <span className="text-gray-600 dark:text-gray-400">类型:</span>{' '}
                  <span className="text-gray-900 dark:text-gray-100">{selectedNode.type}</span>
                </div>
                {Object.entries(selectedNode.props).map(([key, value]) => (
                  <div key={key}>
                    <span className="text-gray-600 dark:text-gray-400">{key}:</span>{' '}
                    <span className="text-gray-900 dark:text-gray-100">
                      {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* 图例 */}
          <div className="mt-4 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
            <h3 className="font-semibold text-gray-900 dark:text-gray-100 mb-3">图例</h3>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              {Object.entries(nodeColors).map(([type, color]) => (
                <div key={type} className="flex items-center gap-2">
                  <div
                    className="w-4 h-4 rounded-full"
                    style={{ backgroundColor: color.background }}
                  />
                  <span className="text-sm text-gray-700 dark:text-gray-300">{type}</span>
                </div>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

export default GraphVisualization

