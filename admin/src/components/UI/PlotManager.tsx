import { useState } from 'react'
import { Plus, Edit2, Trash2, Calendar, Star, StarOff, Eye, EyeOff } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { cn } from '@/utils/cn'
import Card from '../UI/Card'
import Button from '../UI/Button'
import Input from '../UI/Input'
import Modal from '../UI/Modal'
import Badge from '../UI/Badge'
import EmptyState from '../UI/EmptyState'
import { useDebounce } from '@/hooks/useDebounce'

interface PlotPoint {
  id: string
  title: string
  description: string
  type: 'exposition' | 'rising_action' | 'climax' | 'falling_action' | 'resolution' | 'custom'
  chapter?: string
  characters: string[]
  location?: string
  time?: string
  isImportant: boolean
  isVisible: boolean
  order: number
  tags: string[]
  createdAt: string
  updatedAt: string
}

interface PlotManagerProps {
  plotPoints: PlotPoint[]
  onAdd: (plotPoint: Omit<PlotPoint, 'id' | 'createdAt' | 'updatedAt'>) => void
  onUpdate: (id: string, plotPoint: Partial<PlotPoint>) => void
  onDelete: (id: string) => void
  onReorder: (fromIndex: number, toIndex: number) => void
  onToggleVisibility: (id: string) => void
  onToggleImportance: (id: string) => void
  className?: string
}

const PlotManager = ({
  plotPoints,
  onAdd,
  onUpdate,
  onDelete,
  onToggleVisibility,
  onToggleImportance,
  className
}: PlotManagerProps) => {
  const [isAddModalOpen, setIsAddModalOpen] = useState(false)
  const [editingPlotPoint, setEditingPlotPoint] = useState<PlotPoint | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [filterType, setFilterType] = useState<string>('all')
  const [showOnlyImportant, setShowOnlyImportant] = useState(false)
  const [showOnlyVisible, setShowOnlyVisible] = useState(false)
  const [viewMode, setViewMode] = useState<'list' | 'timeline'>('list')

  const debouncedSearchTerm = useDebounce(searchTerm, 300)

  const filteredPlotPoints = plotPoints
    .filter(plotPoint => {
      const matchesSearch = plotPoint.title.toLowerCase().includes(debouncedSearchTerm.toLowerCase()) ||
                           plotPoint.description.toLowerCase().includes(debouncedSearchTerm.toLowerCase())
      const matchesType = filterType === 'all' || plotPoint.type === filterType
      const matchesImportant = !showOnlyImportant || plotPoint.isImportant
      const matchesVisible = !showOnlyVisible || plotPoint.isVisible

      return matchesSearch && matchesType && matchesImportant && matchesVisible
    })
    .sort((a, b) => a.order - b.order)

  const getTypeColor = (type: string) => {
    switch (type) {
      case 'exposition': return 'blue'
      case 'rising_action': return 'green'
      case 'climax': return 'red'
      case 'falling_action': return 'orange'
      case 'resolution': return 'purple'
      case 'custom': return 'gray'
      default: return 'gray'
    }
  }

  const getTypeText = (type: string) => {
    switch (type) {
      case 'exposition': return '开端'
      case 'rising_action': return '发展'
      case 'climax': return '高潮'
      case 'falling_action': return '回落'
      case 'resolution': return '结局'
      case 'custom': return '自定义'
      default: return '未知'
    }
  }

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'exposition': return '📖'
      case 'rising_action': return '📈'
      case 'climax': return '⚡'
      case 'falling_action': return '📉'
      case 'resolution': return '🎯'
      case 'custom': return '📝'
      default: return '📄'
    }
  }

  return (
    <div className={cn('space-y-6', className)}>
      {/* 搜索和筛选 */}
      <Card className="p-4">
        <div className="flex flex-col md:flex-row gap-4">
          <div className="flex-1">
            <Input
              placeholder="搜索情节..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full"
            />
          </div>
          
          <div className="flex gap-2">
            <select
              value={filterType}
              onChange={(e) => setFilterType(e.target.value)}
              className="px-3 py-2 border border-neutral-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white"
            >
              <option value="all">所有类型</option>
              <option value="exposition">开端</option>
              <option value="rising_action">发展</option>
              <option value="climax">高潮</option>
              <option value="falling_action">回落</option>
              <option value="resolution">结局</option>
              <option value="custom">自定义</option>
            </select>

            <Button
              variant={showOnlyImportant ? 'default' : 'outline'}
              size="sm"
              onClick={() => setShowOnlyImportant(!showOnlyImportant)}
            >
              <Star className="w-4 h-4 mr-1" />
              重要
            </Button>

            <Button
              variant={showOnlyVisible ? 'default' : 'outline'}
              size="sm"
              onClick={() => setShowOnlyVisible(!showOnlyVisible)}
            >
              <Eye className="w-4 h-4 mr-1" />
              可见
            </Button>

            <Button
              variant={viewMode === 'timeline' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setViewMode(viewMode === 'list' ? 'timeline' : 'list')}
            >
              <Calendar className="w-4 h-4 mr-1" />
              {viewMode === 'list' ? '时间线' : '列表'}
            </Button>
          </div>
        </div>
      </Card>

      {/* 情节列表 */}
      {viewMode === 'list' ? (
        <div className="space-y-4 max-h-96 overflow-y-auto scrollbar-thin">
          <AnimatePresence>
            {filteredPlotPoints.map((plotPoint) => (
              <motion.div
                key={plotPoint.id}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -20 }}
                transition={{ duration: 0.2 }}
              >
                <Card className={cn(
                  'p-4 hover:shadow-lg transition-all',
                  !plotPoint.isVisible && 'opacity-50'
                )}>
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center gap-3">
                      <div className="text-2xl">
                        {getTypeIcon(plotPoint.type)}
                      </div>
                      <div>
                        <h3 className="font-semibold text-neutral-900 dark:text-white">
                          {plotPoint.title}
                        </h3>
                        <div className="flex gap-2 mt-1">
                          <Badge variant={getTypeColor(plotPoint.type) as any} size="sm">
                            {getTypeText(plotPoint.type)}
                          </Badge>
                          {plotPoint.chapter && (
                            <Badge variant="outline" size="sm">
                              第 {plotPoint.chapter} 章
                            </Badge>
                          )}
                        </div>
                      </div>
                    </div>

                    <div className="flex gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => onToggleImportance(plotPoint.id)}
                        className={plotPoint.isImportant ? 'text-yellow-500' : 'text-neutral-400'}
                      >
                        {plotPoint.isImportant ? <Star className="w-4 h-4" /> : <StarOff className="w-4 h-4" />}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => onToggleVisibility(plotPoint.id)}
                        className={plotPoint.isVisible ? 'text-blue-500' : 'text-neutral-400'}
                      >
                        {plotPoint.isVisible ? <Eye className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setEditingPlotPoint(plotPoint)}
                      >
                        <Edit2 className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => onDelete(plotPoint.id)}
                        className="text-red-500 hover:text-red-700"
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                  </div>

                  <p className="text-sm text-neutral-600 dark:text-neutral-400 mb-3 line-clamp-2">
                    {plotPoint.description}
                  </p>

                  <div className="flex flex-wrap gap-2 text-xs text-neutral-500 dark:text-neutral-400">
                    {plotPoint.characters.length > 0 && (
                      <span>角色: {plotPoint.characters.join(', ')}</span>
                    )}
                    {plotPoint.location && (
                      <span>地点: {plotPoint.location}</span>
                    )}
                    {plotPoint.time && (
                      <span>时间: {plotPoint.time}</span>
                    )}
                  </div>

                  {plotPoint.tags.length > 0 && (
                    <div className="flex flex-wrap gap-1 mt-2">
                      {plotPoint.tags.map((tag, tagIndex) => (
                        <Badge key={tagIndex} variant="outline" size="sm">
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  )}
                </Card>
              </motion.div>
            ))}
          </AnimatePresence>
        </div>
      ) : (
        <div className="space-y-4">
          {filteredPlotPoints.map((plotPoint, index) => (
            <div key={plotPoint.id} className="flex items-start gap-4">
              <div className="flex flex-col items-center">
                <div className={cn(
                  'w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-medium',
                  `bg-${getTypeColor(plotPoint.type)}-500`
                )}>
                  {index + 1}
                </div>
                {index < filteredPlotPoints.length - 1 && (
                  <div className="w-0.5 h-8 bg-neutral-200 dark:bg-neutral-700 mt-2" />
                )}
              </div>
              
              <Card className="flex-1 p-4">
                <div className="flex items-start justify-between mb-2">
                  <div>
                    <h3 className="font-semibold text-neutral-900 dark:text-white">
                      {plotPoint.title}
                    </h3>
                    <div className="flex gap-2 mt-1">
                      <Badge variant={getTypeColor(plotPoint.type) as any} size="sm">
                        {getTypeText(plotPoint.type)}
                      </Badge>
                      {plotPoint.chapter && (
                        <Badge variant="outline" size="sm">
                          第 {plotPoint.chapter} 章
                        </Badge>
                      )}
                    </div>
                  </div>
                  
                  <div className="flex gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => onToggleImportance(plotPoint.id)}
                      className={plotPoint.isImportant ? 'text-yellow-500' : 'text-neutral-400'}
                    >
                      {plotPoint.isImportant ? <Star className="w-4 h-4" /> : <StarOff className="w-4 h-4" />}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setEditingPlotPoint(plotPoint)}
                    >
                      <Edit2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
                
                <p className="text-sm text-neutral-600 dark:text-neutral-400">
                  {plotPoint.description}
                </p>
              </Card>
            </div>
          ))}
        </div>
      )}

      {/* 空状态 */}
      {filteredPlotPoints.length === 0 && (
        <EmptyState
          icon={Plus}
          title="没有找到情节"
          description={plotPoints.length === 0 ? "开始创建你的第一个情节点" : "尝试调整筛选条件"}
          action={{
            label: '添加情节',
            onClick: () => setIsAddModalOpen(true)
          }}
        />
      )}

      {/* 添加情节按钮 */}
      <div className="flex justify-center">
        <Button onClick={() => setIsAddModalOpen(true)}>
          <Plus className="w-4 h-4 mr-2" />
          添加情节
        </Button>
      </div>

      {/* 添加/编辑情节模态框 */}
      <Modal
        isOpen={isAddModalOpen || !!editingPlotPoint}
        onClose={() => {
          setIsAddModalOpen(false)
          setEditingPlotPoint(null)
        }}
        title={editingPlotPoint ? '编辑情节' : '添加情节'}
        size="lg"
      >
        <PlotPointForm
          plotPoint={editingPlotPoint}
          onSubmit={(plotPoint) => {
            if (editingPlotPoint) {
              onUpdate(editingPlotPoint.id, plotPoint)
            } else {
              onAdd(plotPoint)
            }
            setIsAddModalOpen(false)
            setEditingPlotPoint(null)
          }}
          onCancel={() => {
            setIsAddModalOpen(false)
            setEditingPlotPoint(null)
          }}
        />
      </Modal>
    </div>
  )
}

// 情节表单组件
const PlotPointForm = ({
  plotPoint,
  onSubmit,
  onCancel
}: {
  plotPoint?: PlotPoint | null
  onSubmit: (plotPoint: Omit<PlotPoint, 'id' | 'createdAt' | 'updatedAt'>) => void
  onCancel: () => void
}) => {
  const [formData, setFormData] = useState({
    title: plotPoint?.title || '',
    description: plotPoint?.description || '',
    type: plotPoint?.type || 'custom',
    chapter: plotPoint?.chapter || '',
    characters: plotPoint?.characters || [],
    location: plotPoint?.location || '',
    time: plotPoint?.time || '',
    isImportant: plotPoint?.isImportant || false,
    isVisible: plotPoint?.isVisible || true,
    order: plotPoint?.order || 0,
    tags: plotPoint?.tags || []
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit(formData)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Input
          label="情节标题"
          value={formData.title}
          onChange={(e) => setFormData({ ...formData, title: e.target.value })}
          required
        />

        <div>
          <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
            情节类型
          </label>
          <select
            value={formData.type}
            onChange={(e) => setFormData({ ...formData, type: e.target.value as any })}
            className="w-full px-3 py-2 border border-neutral-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white"
          >
            <option value="exposition">开端</option>
            <option value="rising_action">发展</option>
            <option value="climax">高潮</option>
            <option value="falling_action">回落</option>
            <option value="resolution">结局</option>
            <option value="custom">自定义</option>
          </select>
        </div>

        <Input
          label="所属章节"
          value={formData.chapter}
          onChange={(e) => setFormData({ ...formData, chapter: e.target.value })}
        />

        <Input
          label="地点"
          value={formData.location}
          onChange={(e) => setFormData({ ...formData, location: e.target.value })}
        />

        <Input
          label="时间"
          value={formData.time}
          onChange={(e) => setFormData({ ...formData, time: e.target.value })}
        />

        <Input
          label="角色 (用逗号分隔)"
          value={formData.characters.join(', ')}
          onChange={(e) => setFormData({ 
            ...formData, 
            characters: e.target.value.split(',').map(c => c.trim()).filter(c => c)
          })}
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
          情节描述
        </label>
        <textarea
          value={formData.description}
          onChange={(e) => setFormData({ ...formData, description: e.target.value })}
          className="w-full px-3 py-2 border border-neutral-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white"
          rows={4}
          required
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
          标签 (用逗号分隔)
        </label>
        <Input
          value={formData.tags.join(', ')}
          onChange={(e) => setFormData({ 
            ...formData, 
            tags: e.target.value.split(',').map(t => t.trim()).filter(t => t)
          })}
        />
      </div>

      <div className="flex gap-4">
        <Button type="submit">
          {plotPoint ? '更新情节' : '添加情节'}
        </Button>
        <Button type="button" variant="outline" onClick={onCancel}>
          取消
        </Button>
      </div>
    </form>
  )
}

export default PlotManager
