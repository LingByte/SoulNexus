import { useState } from 'react'
import { 
  Plus, 
  Edit2, 
  Trash2, 
  Eye,
  EyeOff, 
  BookOpen,
  CheckCircle,
  Circle
} from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { cn } from '@/utils/cn.ts'
import Card from '../UI/Card'
import Button from '../UI/Button'
import Input from '../UI/Input'
import Modal from '../UI/Modal'
import Badge from '../UI/Badge'
import EmptyState from '../UI/EmptyState'
import ProgressBar from '../Data/ProgressBar'

interface Chapter {
  id: string
  title: string
  summary: string
  content: string
  wordCount: number
  targetWordCount: number
  status: 'draft' | 'writing' | 'editing' | 'completed'
  order: number
  isVisible: boolean
  tags: string[]
  characters: string[]
  plotPoints: string[]
  createdAt: string
  updatedAt: string
}

interface ChapterOutlineProps {
  chapters: Chapter[]
  onAdd: (chapter: Omit<Chapter, 'id' | 'createdAt' | 'updatedAt'>) => void
  onUpdate: (id: string, chapter: Partial<Chapter>) => void
  onDelete: (id: string) => void
  onReorder: (fromIndex: number, toIndex: number) => void
  onToggleVisibility: (id: string) => void
  onChapterSelect: (chapter: Chapter) => void
  className?: string
}

const ChapterOutline = ({
  chapters,
  onAdd,
  onUpdate,
  onDelete,
  onToggleVisibility,
  onChapterSelect,
  className
}: ChapterOutlineProps) => {
  const [isAddModalOpen, setIsAddModalOpen] = useState(false)
  const [editingChapter, setEditingChapter] = useState<Chapter | null>(null)
  const [viewMode, setViewMode] = useState<'list' | 'timeline' | 'grid'>('list')
  const [filterStatus, setFilterStatus] = useState<string>('all')
  const [showOnlyVisible, setShowOnlyVisible] = useState(false)

  const filteredChapters = chapters
    .filter(chapter => {
      const matchesStatus = filterStatus === 'all' || chapter.status === filterStatus
      const matchesVisible = !showOnlyVisible || chapter.isVisible
      return matchesStatus && matchesVisible
    })
    .sort((a, b) => a.order - b.order)

  const totalWords = chapters.reduce((sum, chapter) => sum + chapter.wordCount, 0)
  const totalTargetWords = chapters.reduce((sum, chapter) => sum + chapter.targetWordCount, 0)
  const completedChapters = chapters.filter(chapter => chapter.status === 'completed').length
  const overallProgress = totalTargetWords > 0 ? (totalWords / totalTargetWords) * 100 : 0

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'draft': return 'gray'
      case 'writing': return 'blue'
      case 'editing': return 'yellow'
      case 'completed': return 'green'
      default: return 'gray'
    }
  }

  const getStatusText = (status: string) => {
    switch (status) {
      case 'draft': return '草稿'
      case 'writing': return '写作中'
      case 'editing': return '编辑中'
      case 'completed': return '已完成'
      default: return '未知'
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'draft': return <Circle className="w-4 h-4" />
      case 'writing': return <BookOpen className="w-4 h-4" />
      case 'editing': return <Edit2 className="w-4 h-4" />
      case 'completed': return <CheckCircle className="w-4 h-4" />
      default: return <Circle className="w-4 h-4" />
    }
  }

  const getProgressColor = (progress: number) => {
    if (progress < 30) return 'error'
    if (progress < 70) return 'warning'
    return 'success'
  }

  return (
    <div className={cn('space-y-6', className)}>
      {/* 统计概览 */}
      <Card className="p-6">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <div className="text-center">
            <div className="text-3xl font-bold text-primary mb-2">
              {chapters.length}
            </div>
            <div className="text-sm text-neutral-600 dark:text-neutral-400">总章节</div>
          </div>
          
          <div className="text-center">
            <div className="text-3xl font-bold text-green-500 mb-2">
              {completedChapters}
            </div>
            <div className="text-sm text-neutral-600 dark:text-neutral-400">已完成</div>
          </div>
          
          <div className="text-center">
            <div className="text-3xl font-bold text-blue-500 mb-2">
              {totalWords.toLocaleString()}
            </div>
            <div className="text-sm text-neutral-600 dark:text-neutral-400">总字数</div>
          </div>
          
          <div className="text-center">
            <div className="text-3xl font-bold text-purple-500 mb-2">
              {Math.round(overallProgress)}%
            </div>
            <div className="text-sm text-neutral-600 dark:text-neutral-400">完成度</div>
          </div>
        </div>
        
        <div className="mt-6">
          <ProgressBar
            value={overallProgress}
            variant={getProgressColor(overallProgress) as any}
            showValue
            label="整体进度"
          />
        </div>
      </Card>

      {/* 控制栏 */}
      <Card className="p-4">
        <div className="flex flex-col md:flex-row gap-4">
          <div className="flex gap-2">
            <Button
              variant={viewMode === 'list' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setViewMode('list')}
            >
              列表
            </Button>
            <Button
              variant={viewMode === 'timeline' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setViewMode('timeline')}
            >
              时间线
            </Button>
            <Button
              variant={viewMode === 'grid' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setViewMode('grid')}
            >
              网格
            </Button>
          </div>
          
          <div className="flex gap-2">
            <select
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
              className="px-3 py-2 border border-neutral-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white"
            >
              <option value="all">所有状态</option>
              <option value="draft">草稿</option>
              <option value="writing">写作中</option>
              <option value="editing">编辑中</option>
              <option value="completed">已完成</option>
            </select>

            <Button
              variant={showOnlyVisible ? 'default' : 'outline'}
              size="sm"
              onClick={() => setShowOnlyVisible(!showOnlyVisible)}
            >
              <Eye className="w-4 h-4 mr-1" />
              可见
            </Button>
          </div>
        </div>
      </Card>

      {/* 章节列表 */}
      {viewMode === 'list' && (
        <div className="space-y-4 max-h-96 overflow-y-auto scrollbar-thin">
          <AnimatePresence>
            {filteredChapters.map((chapter) => (
              <motion.div
                key={chapter.id}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -20 }}
                transition={{ duration: 0.2 }}
              >
                <Card className={cn(
                  'p-4 hover:shadow-lg transition-all cursor-pointer',
                  !chapter.isVisible && 'opacity-50'
                )}>
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center gap-3">
                      <div className="text-2xl font-bold text-neutral-400">
                        {chapter.order}
                      </div>
                      <div>
                        <h3 className="font-semibold text-neutral-900 dark:text-white">
                          {chapter.title}
                        </h3>
                        <div className="flex gap-2 mt-1">
                          <Badge variant={getStatusColor(chapter.status) as any} size="sm">
                            {getStatusIcon(chapter.status)}
                            <span className="ml-1">{getStatusText(chapter.status)}</span>
                          </Badge>
                          <Badge variant="outline" size="sm">
                            {chapter.wordCount.toLocaleString()} / {chapter.targetWordCount.toLocaleString()} 字
                          </Badge>
                        </div>
                      </div>
                    </div>

                    <div className="flex gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => onToggleVisibility(chapter.id)}
                        className={chapter.isVisible ? 'text-blue-500' : 'text-neutral-400'}
                      >
                        {chapter.isVisible ? <Eye className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setEditingChapter(chapter)}
                      >
                        <Edit2 className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => onDelete(chapter.id)}
                        className="text-red-500 hover:text-red-700"
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                  </div>

                  <p className="text-sm text-neutral-600 dark:text-neutral-400 mb-3 line-clamp-2">
                    {chapter.summary}
                  </p>

                  <div className="flex items-center justify-between">
                    <div className="flex gap-4 text-xs text-neutral-500 dark:text-neutral-400">
                      {chapter.characters.length > 0 && (
                        <span>角色: {chapter.characters.join(', ')}</span>
                      )}
                      {chapter.plotPoints.length > 0 && (
                        <span>情节: {chapter.plotPoints.length} 个</span>
                      )}
                    </div>
                    
                    <div className="flex gap-2">
                      <ProgressBar
                        value={(chapter.wordCount / chapter.targetWordCount) * 100}
                        variant={getProgressColor((chapter.wordCount / chapter.targetWordCount) * 100) as any}
                        size="sm"
                        className="w-24"
                      />
                      <Button
                        size="sm"
                        onClick={() => onChapterSelect(chapter)}
                      >
                        编辑
                      </Button>
                    </div>
                  </div>
                </Card>
              </motion.div>
            ))}
          </AnimatePresence>
        </div>
      )}

      {/* 时间线视图 */}
      {viewMode === 'timeline' && (
        <div className="space-y-4">
          {filteredChapters.map((chapter, index) => (
            <div key={chapter.id} className="flex items-start gap-4">
              <div className="flex flex-col items-center">
                <div className={cn(
                  'w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-medium',
                  `bg-${getStatusColor(chapter.status)}-500`
                )}>
                  {chapter.order}
                </div>
                {index < filteredChapters.length - 1 && (
                  <div className="w-0.5 h-16 bg-neutral-200 dark:bg-neutral-700 mt-2" />
                )}
              </div>
              
              <Card className="flex-1 p-4">
                <div className="flex items-start justify-between mb-2">
                  <div>
                    <h3 className="font-semibold text-neutral-900 dark:text-white">
                      {chapter.title}
                    </h3>
                    <div className="flex gap-2 mt-1">
                      <Badge variant={getStatusColor(chapter.status) as any} size="sm">
                        {getStatusText(chapter.status)}
                      </Badge>
                      <Badge variant="outline" size="sm">
                        {chapter.wordCount.toLocaleString()} 字
                      </Badge>
                    </div>
                  </div>
                  
                  <div className="flex gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setEditingChapter(chapter)}
                    >
                      <Edit2 className="w-4 h-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => onChapterSelect(chapter)}
                    >
                      编辑
                    </Button>
                  </div>
                </div>
                
                <p className="text-sm text-neutral-600 dark:text-neutral-400 mb-3">
                  {chapter.summary}
                </p>
                
                <ProgressBar
                  value={(chapter.wordCount / chapter.targetWordCount) * 100}
                  variant={getProgressColor((chapter.wordCount / chapter.targetWordCount) * 100) as any}
                  size="sm"
                />
              </Card>
            </div>
          ))}
        </div>
      )}

      {/* 网格视图 */}
      {viewMode === 'grid' && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredChapters.map((chapter) => (
            <Card key={chapter.id} className="p-4 hover:shadow-lg transition-all">
              <div className="flex items-start justify-between mb-3">
                <div className="text-2xl font-bold text-neutral-400">
                  {chapter.order}
                </div>
                <div className="flex gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onToggleVisibility(chapter.id)}
                    className={chapter.isVisible ? 'text-blue-500' : 'text-neutral-400'}
                  >
                    {chapter.isVisible ? <Eye className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setEditingChapter(chapter)}
                  >
                    <Edit2 className="w-4 h-4" />
                  </Button>
                </div>
              </div>
              
              <h3 className="font-semibold text-neutral-900 dark:text-white mb-2">
                {chapter.title}
              </h3>
              
              <div className="flex gap-2 mb-3">
                <Badge variant={getStatusColor(chapter.status) as any} size="sm">
                  {getStatusText(chapter.status)}
                </Badge>
              </div>
              
              <p className="text-sm text-neutral-600 dark:text-neutral-400 mb-3 line-clamp-2">
                {chapter.summary}
              </p>
              
              <div className="space-y-2">
                <div className="flex justify-between text-xs text-neutral-500 dark:text-neutral-400">
                  <span>{chapter.wordCount.toLocaleString()} 字</span>
                  <span>{Math.round((chapter.wordCount / chapter.targetWordCount) * 100)}%</span>
                </div>
                <ProgressBar
                  value={(chapter.wordCount / chapter.targetWordCount) * 100}
                  variant={getProgressColor((chapter.wordCount / chapter.targetWordCount) * 100) as any}
                  size="sm"
                />
              </div>
              
              <Button
                size="sm"
                className="w-full mt-3"
                onClick={() => onChapterSelect(chapter)}
              >
                编辑章节
              </Button>
            </Card>
          ))}
        </div>
      )}

      {/* 空状态 */}
      {filteredChapters.length === 0 && (
        <EmptyState
          icon={Plus}
          title="没有找到章节"
          description={chapters.length === 0 ? "开始创建你的第一个章节" : "尝试调整筛选条件"}
          action={{
            label: '添加章节',
            onClick: () => setIsAddModalOpen(true)
          }}
        />
      )}

      {/* 添加章节按钮 */}
      <div className="flex justify-center">
        <Button onClick={() => setIsAddModalOpen(true)}>
          <Plus className="w-4 h-4 mr-2" />
          添加章节
        </Button>
      </div>

      {/* 添加/编辑章节模态框 */}
      <Modal
        isOpen={isAddModalOpen || !!editingChapter}
        onClose={() => {
          setIsAddModalOpen(false)
          setEditingChapter(null)
        }}
        title={editingChapter ? '编辑章节' : '添加章节'}
        size="lg"
      >
        <ChapterForm
          chapter={editingChapter}
          onSubmit={(chapter) => {
            if (editingChapter) {
              onUpdate(editingChapter.id, chapter)
            } else {
              onAdd(chapter)
            }
            setIsAddModalOpen(false)
            setEditingChapter(null)
          }}
          onCancel={() => {
            setIsAddModalOpen(false)
            setEditingChapter(null)
          }}
        />
      </Modal>
    </div>
  )
}

// 章节表单组件
const ChapterForm = ({
  chapter,
  onSubmit,
  onCancel
}: {
  chapter?: Chapter | null
  onSubmit: (chapter: Omit<Chapter, 'id' | 'createdAt' | 'updatedAt'>) => void
  onCancel: () => void
}) => {
  const [formData, setFormData] = useState({
    title: chapter?.title || '',
    summary: chapter?.summary || '',
    content: chapter?.content || '',
    wordCount: chapter?.wordCount || 0,
    targetWordCount: chapter?.targetWordCount || 1000,
    status: chapter?.status || 'draft',
    order: chapter?.order || 1,
    isVisible: chapter?.isVisible || true,
    tags: chapter?.tags || [],
    characters: chapter?.characters || [],
    plotPoints: chapter?.plotPoints || []
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit(formData)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Input
          label="章节标题"
          value={formData.title}
          onChange={(e) => setFormData({ ...formData, title: e.target.value })}
          required
        />

        <div>
          <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
            状态
          </label>
          <select
            value={formData.status}
            onChange={(e) => setFormData({ ...formData, status: e.target.value as any })}
            className="w-full px-3 py-2 border border-neutral-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white"
          >
            <option value="draft">草稿</option>
            <option value="writing">写作中</option>
            <option value="editing">编辑中</option>
            <option value="completed">已完成</option>
          </select>
        </div>

        <Input
          label="章节顺序"
          type="number"
          value={formData.order}
          onChange={(e) => setFormData({ ...formData, order: parseInt(e.target.value) })}
          required
        />

        <Input
          label="目标字数"
          type="number"
          value={formData.targetWordCount}
          onChange={(e) => setFormData({ ...formData, targetWordCount: parseInt(e.target.value) })}
          required
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
          章节摘要
        </label>
        <textarea
          value={formData.summary}
          onChange={(e) => setFormData({ ...formData, summary: e.target.value })}
          className="w-full px-3 py-2 border border-neutral-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white"
          rows={3}
          required
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">
          章节内容
        </label>
        <textarea
          value={formData.content}
          onChange={(e) => setFormData({ ...formData, content: e.target.value })}
          className="w-full px-3 py-2 border border-neutral-300 dark:border-neutral-600 rounded-lg bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white"
          rows={6}
        />
      </div>

      <div className="flex gap-4">
        <Button type="submit">
          {chapter ? '更新章节' : '添加章节'}
        </Button>
        <Button type="button" variant="outline" onClick={onCancel}>
          取消
        </Button>
      </div>
    </form>
  )
}

export default ChapterOutline
