import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { 
  Search, 
  Download, 
  Star, 
  Package, 
  Code, 
  Zap, 
  Bell, 
  Wrench,
  Grid3X3,
  List,
  ExternalLink,
  Calendar,
  Tag,
  Settings,
  Workflow,
  Eye,
  Edit,
  Trash2,
  Upload,
  Save,
  ChevronLeft,
  ChevronRight
} from 'lucide-react'

import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Card from '@/components/UI/Card'
import Badge from '@/components/UI/Badge'
import Modal from '@/components/UI/Modal'
import ConfirmDialog from '@/components/UI/ConfirmDialog'
import { showAlert } from '@/utils/notification'
import { workflowPluginService, WorkflowPlugin, WorkflowPluginCategory } from '@/api/workflowPlugin'
import { workflowService } from '@/api/workflow'
import { useAuthStore } from '@/stores/authStore'

// 分类图标映射
const categoryIcons = {
  data_processing: Grid3X3,
  api_integration: Code,
  ai_service: Zap,
  notification: Bell,
  utility: Wrench,
  business: Package,
  custom: Settings
}

// 分类颜色映射
const categoryColors = {
  data_processing: 'green',
  api_integration: 'blue', 
  ai_service: 'purple',
  notification: 'orange',
  utility: 'gray',
  business: 'indigo',
  custom: 'pink'
}

const NodePluginMarket: React.FC = () => {
  const { isAuthenticated, user } = useAuthStore()
  const [plugins, setPlugins] = useState<WorkflowPlugin[]>([])
  const [loading, setLoading] = useState(true)
  const [searchKeyword, setSearchKeyword] = useState('')
  const [selectedCategory, setSelectedCategory] = useState<WorkflowPluginCategory | ''>('')
  const [selectedStatus, setSelectedStatus] = useState<'published' | 'draft' | ''>('')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [selectedPlugin, setSelectedPlugin] = useState<WorkflowPlugin | null>(null)
  const [showPluginDetail, setShowPluginDetail] = useState(false)
  const [showCreatePlugin, setShowCreatePlugin] = useState(false)
  const [installedPlugins, setInstalledPlugins] = useState<Set<number>>(new Set())
  const [showMyPlugins, setShowMyPlugins] = useState(false)

  // 确认删除对话框状态
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [pluginToDelete, setPluginToDelete] = useState<WorkflowPlugin | null>(null)
  const [deleteLoading, setDeleteLoading] = useState(false)

  // 编辑插件状态
  const [showEditPlugin, setShowEditPlugin] = useState(false)
  const [editingPlugin, setEditingPlugin] = useState<WorkflowPlugin | null>(null)
  const [editFormData, setEditFormData] = useState({
    displayName: '',
    description: '',
    category: 'utility' as WorkflowPluginCategory,
    icon: '',
    color: '#6366f1',
    tags: '',
    author: '',
    homepage: '',
    repository: '',
    license: 'MIT',
    version: '',
    changeLog: '',
    inputSchema: {
      parameters: [] as any[]
    },
    outputSchema: {
      parameters: [] as any[]
    }
  })
  const [editLoading, setEditLoading] = useState(false)

  // 分页状态
  const [currentPage, setCurrentPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [pageSize] = useState(20)

  // 加载插件列表
  const loadPlugins = async () => {
    setLoading(true)
    try {
      let params: any = {
        category: selectedCategory || undefined,
        status: selectedStatus || undefined,
        keyword: searchKeyword || undefined,
        page: currentPage,
        pageSize
      }

      // 如果显示我的插件，添加用户ID过滤
      if (showMyPlugins && user) {
        params.userId = user.id
      }

      const response = await workflowPluginService.listWorkflowPlugins(params)
      
      if (response.code === 200 && response.data) {
        setPlugins(response.data.plugins)
        setTotalPages(Math.ceil(response.data.total / pageSize))
      } else {
        console.error('加载插件失败:', response.msg)
        showAlert(response.msg || '无法加载插件列表', 'error', '加载失败')
      }
    } catch (error: any) {
      console.error('加载插件失败:', error)
      showAlert(error.message || '网络错误，请稍后重试', 'error', '加载失败')
    } finally {
      setLoading(false)
    }
  }

  // 加载已安装插件
  const loadInstalledPlugins = async () => {
    if (!isAuthenticated) {
      return
    }
    
    try {
      const response = await workflowPluginService.listInstalledWorkflowPlugins()
      if (response.data) {
        const installedIds = new Set(response.data.map((item: any) => item.pluginId))
        setInstalledPlugins(installedIds)
      }
    } catch (error: any) {
      console.error('加载已安装插件失败:', error)
      showAlert('无法加载已安装插件列表', 'error', '加载失败')
    }
  }

  useEffect(() => {
    setCurrentPage(1) // 重置页码
    loadPlugins()
  }, [selectedCategory, selectedStatus, searchKeyword, showMyPlugins])

  useEffect(() => {
    loadPlugins()
  }, [currentPage])

  useEffect(() => {
    loadInstalledPlugins()
  }, [isAuthenticated]) // 依赖认证状态

  // 安装插件
  const handleInstallPlugin = async (plugin: WorkflowPlugin) => {
    if (!isAuthenticated) {
      showAlert('请先登录后再安装插件', 'warning', '请先登录')
      return
    }
    
    try {
      const response = await workflowPluginService.installWorkflowPlugin(plugin.id)
      if (response.data) {
        setInstalledPlugins(prev => new Set([...prev, plugin.id]))
        showAlert(`插件 "${plugin.displayName}" 已成功安装`, 'success', '安装成功')
      }
    } catch (error: any) {
      console.error('安装插件失败:', error)
      showAlert(error.message || '安装插件时发生错误，请稍后重试', 'error', '安装失败')
    }
  }

  // 删除插件
  const handleDeletePlugin = async (plugin: WorkflowPlugin) => {
    setPluginToDelete(plugin)
    setShowDeleteConfirm(true)
  }

  // 确认删除插件
  const confirmDeletePlugin = async () => {
    if (!pluginToDelete) return
    
    setDeleteLoading(true)
    try {
      const response = await workflowPluginService.deleteWorkflowPlugin(pluginToDelete.id)
      if (response.data) {
        showAlert(`插件 "${pluginToDelete.displayName}" 已成功删除`, 'success', '删除成功')
        // 重新加载插件列表
        loadPlugins()
        setShowDeleteConfirm(false)
        setPluginToDelete(null)
      }
    } catch (error: any) {
      console.error('删除插件失败:', error)
      showAlert(error.message || '删除插件时发生错误，请稍后重试', 'error', '删除失败')
    } finally {
      setDeleteLoading(false)
    }
  }

  // 取消删除
  const cancelDelete = () => {
    setShowDeleteConfirm(false)
    setPluginToDelete(null)
    setDeleteLoading(false)
  }

  // 编辑插件
  const handleEditPlugin = async (plugin: WorkflowPlugin) => {
    try {
      const response = await workflowPluginService.getWorkflowPlugin(plugin.id)
      if (response.data) {
        const p = response.data
        setEditingPlugin(p)
        setEditFormData({
          displayName: p.displayName || '',
          description: p.description || '',
          category: p.category || 'utility',
          icon: p.icon || '',
          color: p.color || '#6366f1',
          tags: (p.tags || []).join(', '),
          author: p.author || '',
          homepage: p.homepage || '',
          repository: p.repository || '',
          license: p.license || 'MIT',
          version: p.version || '1.0.0',
          changeLog: '',
          inputSchema: p.inputSchema || { parameters: [] },
          outputSchema: p.outputSchema || { parameters: [] }
        })
        setShowEditPlugin(true)
      }
    } catch (error: any) {
      console.error('获取插件详情失败:', error)
      showAlert('无法获取插件详情，请稍后重试', 'error', '加载失败')
    }
  }

  // 提交编辑
  const handleSubmitEdit = async () => {
    if (!editingPlugin) return
    
    setEditLoading(true)
    try {
      const updateData = {
        displayName: editFormData.displayName,
        description: editFormData.description,
        category: editFormData.category,
        icon: editFormData.icon,
        color: editFormData.color,
        tags: editFormData.tags.split(',').map((tag: string) => tag.trim()).filter(Boolean),
        author: editFormData.author,
        homepage: editFormData.homepage,
        repository: editFormData.repository,
        license: editFormData.license,
        version: editFormData.version,
        changeLog: editFormData.changeLog,
        inputSchema: editFormData.inputSchema,
        outputSchema: editFormData.outputSchema
      }

      const response = await workflowPluginService.updateWorkflowPlugin(editingPlugin.id, updateData)
      if (response.data) {
        showAlert('插件已更新成功', 'success', '更新成功')
        setShowEditPlugin(false)
        setEditingPlugin(null)
        loadPlugins()
      } else {
        showAlert(response.msg || '更新失败', 'error', '更新失败')
      }
    } catch (error: any) {
      console.error('更新插件失败:', error)
      showAlert(error.message || '更新插件时发生错误，请稍后重试', 'error', '更新失败')
    } finally {
      setEditLoading(false)
    }
  }

  // 查看插件详情
  const handleViewPlugin = async (plugin: WorkflowPlugin) => {
    try {
      const response = await workflowPluginService.getWorkflowPlugin(plugin.id)
      if (response.data) {
        setSelectedPlugin(response.data)
        setShowPluginDetail(true)
      }
    } catch (error: any) {
      console.error('获取插件详情失败:', error)
      showAlert('无法获取插件详情，请稍后重试', 'error', '加载失败')
    }
  }

  // 渲染插件卡片
  const renderPluginCard = (plugin: WorkflowPlugin) => {
    const CategoryIcon = categoryIcons[plugin.category] || Package
    const isInstalled = installedPlugins.has(plugin.id)
    
    return (
      <motion.div
        key={plugin.id}
        layout
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -20 }}
        whileHover={{ y: -4 }}
        className="group"
      >
        <Card className="h-full flex flex-col p-6 hover:shadow-xl transition-all duration-300 border-2 hover:border-blue-200 dark:hover:border-blue-800">
          {/* 插件头部 */}
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3 flex-1 min-w-0">
              <div 
                className="p-3 rounded-xl shadow-md flex-shrink-0"
                style={{ 
                  backgroundColor: plugin.color || '#6366f1',
                  color: 'white'
                }}
              >
                <CategoryIcon className="w-6 h-6" />
              </div>
              <div className="flex-1 min-w-0">
                <h3 className="font-bold text-lg text-gray-900 dark:text-white group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors truncate">
                  {plugin.displayName}
                </h3>
                <p className="text-sm text-gray-500 dark:text-gray-400 truncate">
                  by {plugin.author || '未知作者'}
                </p>
              </div>
            </div>
            <Badge 
              variant={categoryColors[plugin.category] as any}
              className="text-xs flex-shrink-0 ml-2"
            >
              {plugin.category}
            </Badge>
          </div>

          {/* 插件描述 */}
          <div className="flex-1 mb-4">
            <p className="text-gray-600 dark:text-gray-300 text-sm line-clamp-3">
              {plugin.description || '暂无描述'}
            </p>
          </div>

          {/* 标签 */}
          {plugin.tags && plugin.tags.length > 0 && (
            <div className="flex flex-wrap gap-1 mb-4">
              {plugin.tags.slice(0, 3).map((tag, index) => (
                <span 
                  key={index}
                  className="px-2 py-1 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 text-xs rounded-md"
                >
                  {tag}
                </span>
              ))}
              {plugin.tags.length > 3 && (
                <span className="px-2 py-1 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 text-xs rounded-md">
                  +{plugin.tags.length - 3}
                </span>
              )}
            </div>
          )}

          {/* 统计信息和版本 */}
          <div className="flex items-center justify-between text-sm text-gray-500 dark:text-gray-400 mb-4">
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-1">
                <Download className="w-4 h-4 flex-shrink-0" />
                <span>{plugin.downloadCount || 0}</span>
              </div>
              <div className="flex items-center gap-1">
                <Star className="w-4 h-4 flex-shrink-0" />
                <span>{(plugin.rating || 0).toFixed(1)}</span>
              </div>
            </div>
            <span className="text-xs flex-shrink-0">v{plugin.version || '1.0.0'}</span>
          </div>

          {/* 操作按钮 */}
          <div className="flex gap-2 mt-auto">
            <button
              onClick={() => handleViewPlugin(plugin)}
              className="flex-1 inline-flex items-center justify-center gap-1 px-3 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
            >
              <Eye className="w-3 h-3" />
              查看
            </button>
            
            {/* 如果是当前用户的插件，显示编辑和删除按钮 */}
            {showMyPlugins && user && plugin.userId === user.id ? (
              <>
                <button
                  onClick={() => handleEditPlugin(plugin)}
                  className="flex-1 inline-flex items-center justify-center gap-1 px-3 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                >
                  <Edit className="w-3 h-3" />
                  编辑
                </button>
                <button
                  onClick={() => handleDeletePlugin(plugin)}
                  className="flex-1 inline-flex items-center justify-center gap-1 px-3 py-2 text-sm font-medium text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 hover:bg-red-100 dark:hover:bg-red-900/30 rounded transition-colors"
                >
                  <Trash2 className="w-3 h-3" />
                  删除
                </button>
              </>
            ) : (
              <button
                onClick={() => !isInstalled && handleInstallPlugin(plugin)}
                disabled={isInstalled}
                className={`flex-1 inline-flex items-center justify-center gap-1 px-3 py-2 text-sm font-medium rounded transition-colors ${
                  isInstalled
                    ? 'text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 cursor-not-allowed'
                    : 'text-white bg-blue-600 hover:bg-blue-700'
                }`}
              >
                <Download className="w-4 h-4" />
                {isInstalled ? '已安装' : '安装'}
              </button>
            )}
          </div>
        </Card>
      </motion.div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* 页面头部 */}
      <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
                工作流插件市场
              </h1>
              <p className="text-gray-600 dark:text-gray-400 mt-2">
                发现和安装强大的工作流插件，将现有工作流作为子流程复用
              </p>
            </div>
            <button
              onClick={() => {
                if (!isAuthenticated) {
                  showAlert('请先登录后再发布工作流', 'warning', '请先登录')
                  return
                }
                console.log('发布工作流按钮被点击')
                try {
                  setShowCreatePlugin(true)
                  console.log('模态框状态已设置为true')
                } catch (error) {
                  console.error('设置模态框状态时出错:', error)
                }
              }}
              disabled={!isAuthenticated}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-green-600 hover:bg-green-700 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Upload className="w-4 h-4" />
              {isAuthenticated ? '发布工作流' : '请先登录'}
            </button>
          </div>
        </div>
      </div>

      {/* 搜索和过滤 */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div className="flex flex-col sm:flex-row gap-4 mb-6">
          {/* 搜索框 */}
          <div className="flex-1">
            <Input
              placeholder="搜索插件..."
              value={searchKeyword}
              onChange={(e) => setSearchKeyword(e.target.value)}
              leftIcon={<Search className="w-4 h-4" />}
            />
          </div>

          {/* 分类过滤 */}
          <select
            value={selectedCategory}
            onChange={(e) => setSelectedCategory(e.target.value as WorkflowPluginCategory | '')}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
          >
            <option value="">所有分类</option>
            <option value="data_processing">数据处理</option>
            <option value="api_integration">API集成</option>
            <option value="ai_service">AI服务</option>
            <option value="notification">通知服务</option>
            <option value="utility">工具类</option>
            <option value="business">业务逻辑</option>
            <option value="custom">自定义</option>
          </select>

          {/* 状态过滤 */}
          <select
            value={selectedStatus}
            onChange={(e) => setSelectedStatus(e.target.value as 'published' | 'draft' | '')}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
          >
            <option value="">所有状态</option>
            <option value="published">已发布</option>
            <option value="draft">草稿</option>
          </select>

          {/* 我的插件按钮 */}
            <button
              onClick={() => {
                setShowMyPlugins(!showMyPlugins)
                setCurrentPage(1) // 重置页码
              }}
              className={`inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                showMyPlugins
                  ? 'text-white bg-blue-600 hover:bg-blue-700'
                  : 'text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600'
              }`}
            >
              <Package className="w-4 h-4" />
              我的插件
            </button>

          {/* 视图模式 */}
          <div className="flex border border-gray-300 dark:border-gray-600 rounded-lg overflow-hidden">
            <button
              onClick={() => setViewMode('grid')}
              className={`p-2 ${viewMode === 'grid' ? 'bg-blue-500 text-white' : 'bg-white dark:bg-gray-800 text-gray-600 dark:text-gray-400'}`}
            >
              <Grid3X3 className="w-4 h-4" />
            </button>
            <button
              onClick={() => setViewMode('list')}
              className={`p-2 ${viewMode === 'list' ? 'bg-blue-500 text-white' : 'bg-white dark:bg-gray-800 text-gray-600 dark:text-gray-400'}`}
            >
              <List className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* 插件列表 */}
        {loading ? (
          <div className="flex justify-center items-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
          </div>
        ) : (
          <>
            <motion.div 
              layout
              className={`grid gap-6 ${
                viewMode === 'grid' 
                  ? 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4' 
                  : 'grid-cols-1'
              }`}
            >
              <AnimatePresence>
                {plugins.map(renderPluginCard)}
              </AnimatePresence>
            </motion.div>

            {/* 分页 */}
            {totalPages > 1 && (
              <div className="flex justify-center mt-8">
                <div className="flex gap-2">
                  {Array.from({ length: totalPages }, (_, i) => i + 1).map(page => (
                    <button
                      key={page}
                      onClick={() => setCurrentPage(page)}
                      className={`px-3 py-2 rounded-lg ${
                        page === currentPage
                          ? 'bg-blue-500 text-white'
                          : 'bg-white dark:bg-gray-800 text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-700'
                      }`}
                    >
                      {page}
                    </button>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </div>

      {/* 插件详情模态框 */}
      <Modal
        isOpen={showPluginDetail}
        onClose={() => setShowPluginDetail(false)}
        title="工作流插件详情"
        size="lg"
      >
        {selectedPlugin && (
          <PluginDetailModal 
            plugin={selectedPlugin}
            isInstalled={installedPlugins.has(selectedPlugin.id)}
            onInstall={() => handleInstallPlugin(selectedPlugin)}
          />
        )}
      </Modal>

      {/* 发布工作流模态框 */}
      <Modal
        isOpen={showCreatePlugin}
        onClose={() => setShowCreatePlugin(false)}
        title="发布工作流为插件"
        size="xl"
      >
        <PublishWorkflowModal onClose={() => setShowCreatePlugin(false)} />
      </Modal>

      {/* 编辑插件模态框 */}
      <Modal
        isOpen={showEditPlugin}
        onClose={() => setShowEditPlugin(false)}
        title="编辑插件"
        size="xl"
      >
        {editingPlugin && (
          <EditPluginModal 
            plugin={editingPlugin}
            formData={editFormData}
            setFormData={setEditFormData}
            onSubmit={handleSubmitEdit}
            onCancel={() => setShowEditPlugin(false)}
            loading={editLoading}
          />
        )}
      </Modal>

      {/* 删除确认对话框 */}
      <ConfirmDialog
        isOpen={showDeleteConfirm}
        onClose={cancelDelete}
        onConfirm={confirmDeletePlugin}
        title="删除插件"
        message={`确定要删除插件 "${pluginToDelete?.displayName}" 吗？此操作不可撤销。`}
        confirmText="删除"
        cancelText="取消"
        type="danger"
        loading={deleteLoading}
      />
    </div>
  )
}

// 插件详情组件
const PluginDetailModal: React.FC<{
  plugin: WorkflowPlugin
  isInstalled: boolean
  onInstall: () => void
}> = ({ plugin, isInstalled, onInstall }) => {
  const CategoryIcon = categoryIcons[plugin.category] || Package

  return (
    <div className="space-y-6">
      {/* 插件头部信息 */}
      <div className="flex items-start gap-4">
        <div 
          className="p-4 rounded-xl shadow-md flex-shrink-0"
          style={{ 
            backgroundColor: plugin.color || '#6366f1',
            color: 'white'
          }}
        >
          <CategoryIcon className="w-8 h-8" />
        </div>
        <div className="flex-1">
          <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
            {plugin.displayName}
          </h2>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            by {plugin.author || '未知作者'} • v{plugin.version || '1.0.0'}
          </p>
          <div className="flex items-center gap-4 mt-2 text-sm text-gray-500 dark:text-gray-400">
            <div className="flex items-center gap-1">
              <Download className="w-4 h-4" />
              <span>{(plugin.downloadCount || 0)} 下载</span>
            </div>
            <div className="flex items-center gap-1">
              <Star className="w-4 h-4" />
              <span>{(plugin.rating || 0).toFixed(1)} 评分</span>
            </div>
            <div className="flex items-center gap-1">
              <Calendar className="w-4 h-4" />
              <span>{new Date(plugin.createdAt).toLocaleDateString()}</span>
            </div>
          </div>
        </div>
        <button
          onClick={onInstall}
          disabled={isInstalled}
          className={`inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-md transition-colors ${
            isInstalled
              ? 'text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 cursor-not-allowed'
              : 'text-white bg-blue-600 hover:bg-blue-700'
          }`}
        >
          <Download className="w-4 h-4" />
          {isInstalled ? '已安装' : '安装'}
        </button>
      </div>

      {/* 插件描述 */}
      <div>
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
          描述
        </h3>
        <p className="text-gray-600 dark:text-gray-400">
          {plugin.description || '暂无描述'}
        </p>
      </div>

      {/* 输入输出参数 */}
      <div className="grid grid-cols-2 gap-6">
        <div>
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
            输入参数
          </h3>
          {plugin.inputSchema?.parameters && plugin.inputSchema.parameters.length > 0 ? (
            <div className="space-y-2">
              {plugin.inputSchema.parameters.map((param, index) => (
                <div key={index} className="p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-900 dark:text-white">
                      {param.name}
                    </span>
                    <Badge variant="secondary" className="text-xs">
                      {param.type}
                    </Badge>
                    {param.required && (
                      <Badge variant="error" className="text-xs">
                        必需
                      </Badge>
                    )}
                  </div>
                  {param.description && (
                    <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                      {param.description}
                    </p>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <p className="text-gray-500 dark:text-gray-400">无输入参数</p>
          )}
        </div>

        <div>
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
            输出参数
          </h3>
          {plugin.outputSchema?.parameters && plugin.outputSchema.parameters.length > 0 ? (
            <div className="space-y-2">
              {plugin.outputSchema.parameters.map((param, index) => (
                <div key={index} className="p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-900 dark:text-white">
                      {param.name}
                    </span>
                    <Badge variant="secondary" className="text-xs">
                      {param.type}
                    </Badge>
                  </div>
                  {param.description && (
                    <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
                      {param.description}
                    </p>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <p className="text-gray-500 dark:text-gray-400">无输出参数</p>
          )}
        </div>
      </div>

      {/* 标签 */}
      {plugin.tags && plugin.tags.length > 0 && (
        <div>
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
            标签
          </h3>
          <div className="flex flex-wrap gap-2">
            {plugin.tags.map((tag, index) => (
              <Badge key={index} variant="secondary" className="flex items-center gap-1">
                <Tag className="w-3 h-3" />
                <span>{tag}</span>
              </Badge>
            ))}
          </div>
        </div>
      )}

      {/* 链接 */}
      <div className="flex gap-4">
        {plugin.homepage && (
          <a
            href={plugin.homepage}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
          >
            <ExternalLink className="w-4 h-4" />
            主页
          </a>
        )}
        {plugin.repository && (
          <a
            href={plugin.repository}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
          >
            <Code className="w-4 h-4" />
            源码
          </a>
        )}
      </div>
    </div>
  )
}

// 编辑插件组件
const EditPluginModal: React.FC<{
  plugin: WorkflowPlugin
  formData: any
  setFormData: (data: any) => void
  onSubmit: () => void
  onCancel: () => void
  loading: boolean
}> = ({formData, setFormData, onSubmit, onCancel, loading }) => {
  return (
    <div className="space-y-4 max-h-[70vh] overflow-y-auto">
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-2">显示名称 *</label>
          <Input
            value={formData.displayName}
            onChange={(e) => setFormData({...formData, displayName: e.target.value})}
            placeholder="插件显示名称"
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-2">版本</label>
          <Input
            value={formData.version}
            onChange={(e) => setFormData({...formData, version: e.target.value})}
            placeholder="1.0.0"
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium mb-2">描述 *</label>
        <textarea
          value={formData.description}
          onChange={(e) => setFormData({...formData, description: e.target.value})}
          placeholder="插件功能描述..."
          className="w-full px-3 py-2 border border-gray-300 rounded-lg resize-none h-20"
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-2">分类 *</label>
          <select
            value={formData.category}
            onChange={(e) => setFormData({...formData, category: e.target.value as WorkflowPluginCategory})}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg"
          >
            <option value="data_processing">数据处理</option>
            <option value="api_integration">API集成</option>
            <option value="ai_service">AI服务</option>
            <option value="notification">通知服务</option>
            <option value="utility">工具类</option>
            <option value="business">业务逻辑</option>
            <option value="custom">自定义</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium mb-2">颜色</label>
          <input
            type="color"
            value={formData.color}
            onChange={(e) => setFormData({...formData, color: e.target.value})}
            className="w-full h-10 border border-gray-300 rounded-lg"
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium mb-2">标签</label>
        <Input
          value={formData.tags}
          onChange={(e) => setFormData({...formData, tags: e.target.value})}
          placeholder="标签1, 标签2, 标签3"
        />
        <p className="text-xs text-gray-500 mt-1">用逗号分隔多个标签</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div>
          <label className="block text-sm font-medium mb-2">作者</label>
          <Input
            value={formData.author}
            onChange={(e) => setFormData({...formData, author: e.target.value})}
            placeholder="作者名称"
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-2">主页</label>
          <Input
            value={formData.homepage}
            onChange={(e) => setFormData({...formData, homepage: e.target.value})}
            placeholder="https://..."
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-2">仓库</label>
          <Input
            value={formData.repository}
            onChange={(e) => setFormData({...formData, repository: e.target.value})}
            placeholder="https://github.com/..."
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium mb-2">更新日志</label>
        <textarea
          value={formData.changeLog}
          onChange={(e) => setFormData({...formData, changeLog: e.target.value})}
          placeholder="本次更新的内容..."
          className="w-full px-3 py-2 border border-gray-300 rounded-lg resize-none h-16"
        />
      </div>

      <div className="grid grid-cols-2 gap-6">
        {/* 输入参数 */}
        <div>
          <div className="flex items-center justify-between mb-3">
            <h4 className="font-medium">输入参数</h4>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                const currentParams = formData.inputSchema?.parameters || []
                setFormData({
                  ...formData,
                  inputSchema: {
                    parameters: [
                      ...currentParams,
                      { name: '', type: 'string', required: false, description: '' }
                    ]
                  }
                })
              }}
            >
              添加
            </Button>
          </div>
          
          <div className="space-y-3 max-h-48 overflow-y-auto">
            {(formData.inputSchema?.parameters || []).map((param: any, index: number) => (
              <div key={index} className="p-3 border border-gray-200 rounded-lg">
                <div className="grid grid-cols-2 gap-2 mb-2">
                  <Input
                    placeholder="参数名"
                    value={param.name}
                    onChange={(e) => {
                      const newParams = [...formData.inputSchema.parameters]
                      newParams[index] = { ...param, name: e.target.value }
                      setFormData({
                        ...formData,
                        inputSchema: { parameters: newParams }
                      })
                    }}
                  />
                  <select
                    value={param.type}
                    onChange={(e) => {
                      const newParams = [...formData.inputSchema.parameters]
                      newParams[index] = { ...param, type: e.target.value }
                      setFormData({
                        ...formData,
                        inputSchema: { parameters: newParams }
                      })
                    }}
                    className="px-2 py-1 border border-gray-300 rounded text-sm"
                  >
                    <option value="string">字符串</option>
                    <option value="number">数字</option>
                    <option value="boolean">布尔值</option>
                    <option value="object">对象</option>
                    <option value="array">数组</option>
                  </select>
                </div>
                <Input
                  placeholder="描述"
                  value={param.description}
                  onChange={(e) => {
                    const newParams = [...formData.inputSchema.parameters]
                    newParams[index] = { ...param, description: e.target.value }
                    setFormData({
                      ...formData,
                      inputSchema: { parameters: newParams }
                    })
                  }}
                />
                <div className="flex items-center justify-between mt-2">
                  <label className="flex items-center text-sm gap-2">
                    <input
                      type="checkbox"
                      checked={param.required}
                      onChange={(e) => {
                        const newParams = [...formData.inputSchema.parameters]
                        newParams[index] = { ...param, required: e.target.checked }
                        setFormData({
                          ...formData,
                          inputSchema: { parameters: newParams }
                        })
                      }}
                    />
                    <span>必需</span>
                  </label>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      const newParams = formData.inputSchema.parameters.filter((_: any, i: number) => i !== index)
                      setFormData({
                        ...formData,
                        inputSchema: { parameters: newParams }
                      })
                    }}
                  >
                    删除
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* 输出参数 */}
        <div>
          <div className="flex items-center justify-between mb-3">
            <h4 className="font-medium">输出参数</h4>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                const currentParams = formData.outputSchema?.parameters || []
                setFormData({
                  ...formData,
                  outputSchema: {
                    parameters: [
                      ...currentParams,
                      { name: '', type: 'string', required: false, description: '' }
                    ]
                  }
                })
              }}
            >
              添加
            </Button>
          </div>
          
          <div className="space-y-3 max-h-48 overflow-y-auto">
            {(formData.outputSchema?.parameters || []).map((param: any, index: number) => (
              <div key={index} className="p-3 border border-gray-200 rounded-lg">
                <div className="grid grid-cols-2 gap-2 mb-2">
                  <Input
                    placeholder="参数名"
                    value={param.name}
                    onChange={(e) => {
                      const newParams = [...formData.outputSchema.parameters]
                      newParams[index] = { ...param, name: e.target.value }
                      setFormData({
                        ...formData,
                        outputSchema: { parameters: newParams }
                      })
                    }}
                  />
                  <select
                    value={param.type}
                    onChange={(e) => {
                      const newParams = [...formData.outputSchema.parameters]
                      newParams[index] = { ...param, type: e.target.value }
                      setFormData({
                        ...formData,
                        outputSchema: { parameters: newParams }
                      })
                    }}
                    className="px-2 py-1 border border-gray-300 rounded text-sm"
                  >
                    <option value="string">字符串</option>
                    <option value="number">数字</option>
                    <option value="boolean">布尔值</option>
                    <option value="object">对象</option>
                    <option value="array">数组</option>
                  </select>
                </div>
                <Input
                  placeholder="描述"
                  value={param.description}
                  onChange={(e) => {
                    const newParams = [...formData.outputSchema.parameters]
                    newParams[index] = { ...param, description: e.target.value }
                    setFormData({
                      ...formData,
                      outputSchema: { parameters: newParams }
                    })
                  }}
                />
                <div className="flex justify-end mt-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      const newParams = formData.outputSchema.parameters.filter((_: any, i: number) => i !== index)
                      setFormData({
                        ...formData,
                        outputSchema: { parameters: newParams }
                      })
                    }}
                  >
                    删除
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* 操作按钮 */}
      <div className="flex justify-end gap-2 pt-4 border-t">
        <button
          onClick={onCancel}
          disabled={loading}
          className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          取消
        </button>
        <button
          onClick={onSubmit}
          disabled={loading || !formData.displayName || !formData.description}
          className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Save className="w-4 h-4" />
          保存更改
        </button>
      </div>
    </div>
  )
}

// 发布工作流组件
const PublishWorkflowModal: React.FC<{
  onClose: () => void
}> = ({ onClose }) => {
  const [workflows, setWorkflows] = useState<any[]>([])
  const [selectedWorkflow, setSelectedWorkflow] = useState<any>(null)
  const [publishedPlugin, setPublishedPlugin] = useState<any>(null)
  const [isCheckingPublished, setIsCheckingPublished] = useState(false)
  const [publishMode, setPublishMode] = useState<'new' | 'version'>('new') // 'new' = 新插件, 'version' = 新版本
  const [formData, setFormData] = useState({
    name: '',
    displayName: '',
    description: '',
    category: 'utility' as WorkflowPluginCategory,
    icon: '',
    color: '#6366f1',
    tags: '',
    author: '',
    homepage: '',
    repository: '',
    license: 'MIT',
    version: '',
    inputSchema: {
      parameters: [] as any[]
    },
    outputSchema: {
      parameters: [] as any[]
    }
  })

  const [currentStep, setCurrentStep] = useState(1)
  const [loading, setLoading] = useState(false)

  // 加载用户的工作流列表
  useEffect(() => {
    const loadWorkflows = async () => {
      try {
        const response = await workflowService.listDefinitions()
        if (response.data) {
          setWorkflows(response.data)
        }
      } catch (error: any) {
        console.error('加载工作流失败:', error)
        showAlert('无法加载工作流列表', 'error', '加载失败')
      }
    }
    
    loadWorkflows()
  }, [])

  // 选择工作流时自动填充表单并检查是否已发布
  useEffect(() => {
    if (selectedWorkflow) {
      console.log('Selected workflow:', selectedWorkflow)
      console.log('Input parameters:', selectedWorkflow.inputParameters)
      console.log('Output parameters:', selectedWorkflow.outputParameters)
      
      setFormData(prev => ({
        ...prev,
        name: selectedWorkflow.slug || selectedWorkflow.name,
        displayName: selectedWorkflow.name,
        description: selectedWorkflow.description || '',
        inputSchema: {
          parameters: selectedWorkflow.inputParameters && selectedWorkflow.inputParameters.length > 0 
            ? selectedWorkflow.inputParameters 
            : []
        },
        outputSchema: {
          parameters: selectedWorkflow.outputParameters && selectedWorkflow.outputParameters.length > 0 
            ? selectedWorkflow.outputParameters 
            : []
        }
      }))
      
      // 检查是否已发布过插件
      checkPublishedPlugin(selectedWorkflow.id)
    }
  }, [selectedWorkflow])

  // 检查工作流是否已发布过插件
  const checkPublishedPlugin = async (workflowId: number) => {
    setIsCheckingPublished(true)
    try {
      const response = await workflowPluginService.getWorkflowPublishedPlugin(workflowId)
      if (response.code === 200 && response.data) {
        const pluginData = response.data
        setPublishedPlugin(pluginData)
        setPublishMode('version')
        // 自动填充已发布的插件信息，但保留工作流的参数定义
        setFormData(prev => ({
          ...prev,
          name: pluginData.name || '',
          displayName: pluginData.displayName || '',
          description: pluginData.description || '',
          category: pluginData.category || 'utility',
          icon: pluginData.icon || '',
          color: pluginData.color || '#6366f1',
          tags: pluginData.tags?.join(', ') || '',
          author: pluginData.author || '',
          homepage: pluginData.homepage || '',
          repository: pluginData.repository || '',
          license: pluginData.license || 'MIT',
          version: incrementVersion(pluginData.version || '1.0.0'),
          inputSchema: prev.inputSchema,
          outputSchema: prev.outputSchema
        }))
      } else {
        setPublishedPlugin(null)
        setPublishMode('new')
      }
    } catch (error: any) {
      console.error('检查已发布插件失败:', error)
      setPublishedPlugin(null)
      setPublishMode('new')
    } finally {
      setIsCheckingPublished(false)
    }
  }

  // 版本号自增
  const incrementVersion = (version: string): string => {
    const parts = version.split('.')
    if (parts.length === 3) {
      parts[2] = String(parseInt(parts[2]) + 1)
      return parts.join('.')
    }
    return '1.0.1'
  }

  // 处理表单提交
  const handleSubmit = async () => {
    if (!selectedWorkflow) return
    
    setLoading(true)
    try {
      // 确保使用工作流的参数，而不是表单中的参数
      const inputParams = (formData.inputSchema?.parameters && formData.inputSchema.parameters.length > 0)
        ? formData.inputSchema.parameters
        : (selectedWorkflow.inputParameters || [])
      
      const outputParams = (formData.outputSchema?.parameters && formData.outputSchema.parameters.length > 0)
        ? formData.outputSchema.parameters
        : (selectedWorkflow.outputParameters || [])

      const pluginData = {
        ...formData,
        tags: formData.tags.split(',').map((tag: string) => tag.trim()).filter(Boolean),
        // 确保 inputSchema 和 outputSchema 有 parameters 字段
        inputSchema: {
          parameters: inputParams
        },
        outputSchema: {
          parameters: outputParams
        }
      }

      console.log('发送的插件数据:', pluginData)
      console.log('Input parameters being sent:', inputParams)
      console.log('Output parameters being sent:', outputParams)

      let response
      if (publishMode === 'new') {
        // 发布新插件
        response = await workflowPluginService.publishWorkflowAsPlugin(selectedWorkflow.id, pluginData)
        if (response.code === 200 && response.data) {
          showAlert('插件已创建为草稿状态，请在状态过滤器中选择"草稿"查看', 'success', '发布成功')
        }
      } else {
        // 更新已发布的插件
        response = await workflowPluginService.updateWorkflowPlugin(publishedPlugin.id, {
          displayName: formData.displayName,
          description: formData.description,
          category: formData.category,
          icon: formData.icon,
          color: formData.color,
          tags: pluginData.tags,
          inputSchema: pluginData.inputSchema,
          outputSchema: pluginData.outputSchema,
          version: formData.version,
          author: formData.author,
          homepage: formData.homepage,
          repository: formData.repository,
          license: formData.license
        })
        if (response.code === 200) {
          showAlert('插件已更新', 'success', '更新成功')
        }
      }
      
      if (response.code === 200) {
        onClose()
        // 刷新插件列表
        window.location.reload()
      } else {
        showAlert(response.msg || '未知错误', 'error', '操作失败')
      }
    } catch (error: any) {
      console.error('发布/更新工作流插件失败:', error)
      showAlert(error.msg || error.message || '网络错误', 'error', '操作失败')
    } finally {
      setLoading(false)
    }
  }

  // 渲染步骤1：选择工作流
  const renderStep1 = () => (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold mb-4">选择要发布的工作流</h3>
      
      {workflows.length === 0 ? (
        <div className="text-center py-8">
          <Workflow className="w-12 h-12 text-gray-400 mx-auto mb-4" />
          <p className="text-gray-500">您还没有创建任何工作流</p>
          <p className="text-sm text-gray-400 mt-2">请先在工作流管理页面创建工作流</p>
        </div>
      ) : (
        <div className="grid gap-3 max-h-96 overflow-y-auto">
          {workflows.map((workflow) => (
            <div
              key={workflow.id}
              onClick={() => setSelectedWorkflow(workflow)}
              className={`p-4 border rounded-lg cursor-pointer transition-all ${
                selectedWorkflow?.id === workflow.id
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                  : 'border-gray-200 hover:border-gray-300 dark:border-gray-700'
              }`}
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <h4 className="font-medium text-gray-900 dark:text-white">
                    {workflow.name}
                  </h4>
                  <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                    {workflow.description || '暂无描述'}
                  </p>
                  <div className="flex items-center gap-2 mt-2">
                    <span className={`px-2 py-1 text-xs rounded-full ${
                      workflow.status === 'active' 
                        ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                        : workflow.status === 'draft'
                        ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400'
                        : 'bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-400'
                    }`}>
                      {workflow.status === 'active' ? '已激活' : workflow.status === 'draft' ? '草稿' : '已归档'}
                    </span>
                    <p className="text-xs text-gray-400">
                      创建于 {new Date(workflow.createdAt).toLocaleDateString()}
                    </p>
                  </div>
                </div>
                {selectedWorkflow?.id === workflow.id && (
                  <div className="w-5 h-5 bg-blue-500 rounded-full flex items-center justify-center">
                    <div className="w-2 h-2 bg-white rounded-full" />
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* 已发布插件信息 */}
      {selectedWorkflow && isCheckingPublished && (
        <div className="p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
          <p className="text-sm text-blue-700 dark:text-blue-400">检查已发布插件信息中...</p>
        </div>
      )}

      {selectedWorkflow && !isCheckingPublished && publishedPlugin && (
        <div className="p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
          <div className="flex items-start justify-between">
            <div>
              <h4 className="font-medium text-green-900 dark:text-green-400 mb-2">已发布的插件</h4>
              <div className="text-sm text-green-800 dark:text-green-300 space-y-1">
                <p><strong>名称:</strong> {publishedPlugin.displayName}</p>
                <p><strong>版本:</strong> {publishedPlugin.version || '1.0.0'}</p>
                <p><strong>状态:</strong> {publishedPlugin.status}</p>
              </div>
            </div>
            <div className="text-right">
              <p className="text-xs text-green-700 dark:text-green-400 mb-2">将创建新版本</p>
              <Badge variant="success" size="sm">版本模式</Badge>
            </div>
          </div>
        </div>
      )}

      {/* 工作流参数信息 */}
      {selectedWorkflow && (
        <div className="p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
          <p className="text-sm text-blue-700 dark:text-blue-400 mb-2">
            <strong>工作流参数:</strong> 输入参数 {selectedWorkflow.inputParameters?.length || 0} 个，输出参数 {selectedWorkflow.outputParameters?.length || 0} 个
          </p>
          {(!selectedWorkflow.inputParameters || selectedWorkflow.inputParameters.length === 0) && 
           (!selectedWorkflow.outputParameters || selectedWorkflow.outputParameters.length === 0) && (
            <p className="text-xs text-blue-600 dark:text-blue-300">
              💡 提示：如果工作流中有参数定义，请先在工作流编辑器中保存工作流，参数才会被同步到这里。
            </p>
          )}
        </div>
      )}
    </div>
  )

  // 渲染步骤2：插件信息
  const renderStep2 = () => (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold mb-4">插件信息</h3>
      
      {publishMode === 'version' && (
        <div className="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
          <p className="text-sm text-blue-700 dark:text-blue-400">
            <strong>版本模式:</strong> 您正在为已发布的插件创建新版本
          </p>
        </div>
      )}

      {/* 显示参数信息 */}
      <div className="p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
        <p className="text-sm text-green-700 dark:text-green-400">
          <strong>参数信息:</strong> 输入参数 {formData.inputSchema?.parameters?.length || 0} 个，输出参数 {formData.outputSchema?.parameters?.length || 0} 个
        </p>
        {(formData.inputSchema?.parameters?.length || 0) > 0 && (
          <div className="text-xs text-green-600 dark:text-green-300 mt-2">
            <strong>输入参数:</strong> {formData.inputSchema.parameters.map(p => p.name).join(', ')}
          </div>
        )}
        {(formData.outputSchema?.parameters?.length || 0) > 0 && (
          <div className="text-xs text-green-600 dark:text-green-300 mt-1">
            <strong>输出参数:</strong> {formData.outputSchema.parameters.map(p => p.name).join(', ')}
          </div>
        )}
      </div>
      
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-2">插件名称 *</label>
          <Input
            value={formData.name}
            onChange={(e) => setFormData({...formData, name: e.target.value})}
            placeholder="my-awesome-workflow"
            disabled={publishMode === 'version'}
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-2">显示名称 *</label>
          <Input
            value={formData.displayName}
            onChange={(e) => setFormData({...formData, displayName: e.target.value})}
            placeholder="我的超棒工作流"
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium mb-2">描述 *</label>
        <textarea
          value={formData.description}
          onChange={(e) => setFormData({...formData, description: e.target.value})}
          placeholder="工作流功能描述..."
          className="w-full px-3 py-2 border border-gray-300 rounded-lg resize-none h-20"
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium mb-2">分类 *</label>
          <select
            value={formData.category}
            onChange={(e) => setFormData({...formData, category: e.target.value as WorkflowPluginCategory})}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg"
          >
            <option value="data_processing">数据处理</option>
            <option value="api_integration">API集成</option>
            <option value="ai_service">AI服务</option>
            <option value="notification">通知服务</option>
            <option value="utility">工具类</option>
            <option value="business">业务逻辑</option>
            <option value="custom">自定义</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium mb-2">颜色</label>
          <input
            type="color"
            value={formData.color}
            onChange={(e) => setFormData({...formData, color: e.target.value})}
            className="w-full h-10 border border-gray-300 rounded-lg"
          />
        </div>
      </div>

      {publishMode === 'version' && (
        <div>
          <label className="block text-sm font-medium mb-2">版本号 *</label>
          <Input
            value={formData.version}
            onChange={(e) => setFormData({...formData, version: e.target.value})}
            placeholder="1.0.1"
          />
          <p className="text-xs text-gray-500 mt-1">格式: major.minor.patch (如: 1.0.1)</p>
        </div>
      )}

      <div>
        <label className="block text-sm font-medium mb-2">标签</label>
        <Input
          value={formData.tags}
          onChange={(e) => setFormData({...formData, tags: e.target.value})}
          placeholder="标签1, 标签2, 标签3"
        />
        <p className="text-xs text-gray-500 mt-1">用逗号分隔多个标签</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div>
          <label className="block text-sm font-medium mb-2">作者</label>
          <Input
            value={formData.author}
            onChange={(e) => setFormData({...formData, author: e.target.value})}
            placeholder="作者名称"
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-2">主页</label>
          <Input
            value={formData.homepage}
            onChange={(e) => setFormData({...formData, homepage: e.target.value})}
            placeholder="https://..."
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-2">仓库</label>
          <Input
            value={formData.repository}
            onChange={(e) => setFormData({...formData, repository: e.target.value})}
            placeholder="https://github.com/..."
          />
        </div>
      </div>
    </div>
  )

  // 渲染步骤3：参数定义
  const renderStep3 = () => (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold mb-4">参数定义</h3>
      
      <div className="grid grid-cols-2 gap-6">
        {/* 输入参数 */}
        <div>
          <div className="flex items-center justify-between mb-3">
            <h4 className="font-medium">输入参数</h4>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                const currentParams = formData.inputSchema?.parameters || []
                setFormData({
                  ...formData,
                  inputSchema: {
                    parameters: [
                      ...currentParams,
                      { name: '', type: 'string', required: false, description: '' }
                    ]
                  }
                })
              }}
            >
              添加
            </Button>
          </div>
          
          <div className="space-y-3 max-h-64 overflow-y-auto">
            {(formData.inputSchema?.parameters || []).map((param, index) => (
              <div key={index} className="p-3 border border-gray-200 rounded-lg">
                <div className="grid grid-cols-2 gap-2 mb-2">
                  <Input
                    placeholder="参数名"
                    value={param.name}
                    onChange={(e) => {
                      const newParams = [...formData.inputSchema.parameters]
                      newParams[index] = { ...param, name: e.target.value }
                      setFormData({
                        ...formData,
                        inputSchema: { parameters: newParams }
                      })
                    }}
                  />
                  <select
                    value={param.type}
                    onChange={(e) => {
                      const newParams = [...formData.inputSchema.parameters]
                      newParams[index] = { ...param, type: e.target.value }
                      setFormData({
                        ...formData,
                        inputSchema: { parameters: newParams }
                      })
                    }}
                    className="px-2 py-1 border border-gray-300 rounded text-sm"
                  >
                    <option value="string">字符串</option>
                    <option value="number">数字</option>
                    <option value="boolean">布尔值</option>
                    <option value="object">对象</option>
                    <option value="array">数组</option>
                  </select>
                </div>
                <Input
                  placeholder="描述"
                  value={param.description}
                  onChange={(e) => {
                    const newParams = [...formData.inputSchema.parameters]
                    newParams[index] = { ...param, description: e.target.value }
                    setFormData({
                      ...formData,
                      inputSchema: { parameters: newParams }
                    })
                  }}
                />
                <div className="flex items-center justify-between mt-2">
                  <label className="flex items-center text-sm gap-2">
                    <input
                      type="checkbox"
                      checked={param.required}
                      onChange={(e) => {
                        const newParams = [...formData.inputSchema.parameters]
                        newParams[index] = { ...param, required: e.target.checked }
                        setFormData({
                          ...formData,
                          inputSchema: { parameters: newParams }
                        })
                      }}
                    />
                    <span>必需</span>
                  </label>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      const newParams = formData.inputSchema.parameters.filter((_, i) => i !== index)
                      setFormData({
                        ...formData,
                        inputSchema: { parameters: newParams }
                      })
                    }}
                  >
                    删除
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* 输出参数 */}
        <div>
          <div className="flex items-center justify-between mb-3">
            <h4 className="font-medium">输出参数</h4>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                const currentParams = formData.outputSchema?.parameters || []
                setFormData({
                  ...formData,
                  outputSchema: {
                    parameters: [
                      ...currentParams,
                      { name: '', type: 'string', required: false, description: '' }
                    ]
                  }
                })
              }}
            >
              添加
            </Button>
          </div>
          
          <div className="space-y-3 max-h-64 overflow-y-auto">
            {(formData.outputSchema?.parameters || []).map((param, index) => (
              <div key={index} className="p-3 border border-gray-200 rounded-lg">
                <div className="grid grid-cols-2 gap-2 mb-2">
                  <Input
                    placeholder="参数名"
                    value={param.name}
                    onChange={(e) => {
                      const newParams = [...formData.outputSchema.parameters]
                      newParams[index] = { ...param, name: e.target.value }
                      setFormData({
                        ...formData,
                        outputSchema: { parameters: newParams }
                      })
                    }}
                  />
                  <select
                    value={param.type}
                    onChange={(e) => {
                      const newParams = [...formData.outputSchema.parameters]
                      newParams[index] = { ...param, type: e.target.value }
                      setFormData({
                        ...formData,
                        outputSchema: { parameters: newParams }
                      })
                    }}
                    className="px-2 py-1 border border-gray-300 rounded text-sm"
                  >
                    <option value="string">字符串</option>
                    <option value="number">数字</option>
                    <option value="boolean">布尔值</option>
                    <option value="object">对象</option>
                    <option value="array">数组</option>
                  </select>
                </div>
                <Input
                  placeholder="描述"
                  value={param.description}
                  onChange={(e) => {
                    const newParams = [...formData.outputSchema.parameters]
                    newParams[index] = { ...param, description: e.target.value }
                    setFormData({
                      ...formData,
                      outputSchema: { parameters: newParams }
                    })
                  }}
                />
                <div className="flex justify-end mt-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      const newParams = formData.outputSchema.parameters.filter((_, i) => i !== index)
                      setFormData({
                        ...formData,
                        outputSchema: { parameters: newParams }
                      })
                    }}
                  >
                    删除
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )

  return (
    <div className="max-w-4xl mx-auto">
      {/* 步骤指示器 */}
      <div className="flex items-center justify-center mb-6">
        <div className="flex items-center">
          <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
            currentStep >= 1 ? 'bg-blue-500 text-white' : 'bg-gray-200 text-gray-600'
          }`}>
            1
          </div>
          <div className={`w-16 h-1 ${currentStep >= 2 ? 'bg-blue-500' : 'bg-gray-200'}`} />
          <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
            currentStep >= 2 ? 'bg-blue-500 text-white' : 'bg-gray-200 text-gray-600'
          }`}>
            2
          </div>
          <div className={`w-16 h-1 ${currentStep >= 3 ? 'bg-blue-500' : 'bg-gray-200'}`} />
          <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
            currentStep >= 3 ? 'bg-blue-500 text-white' : 'bg-gray-200 text-gray-600'
          }`}>
            3
          </div>
        </div>
      </div>

      {/* 步骤内容 */}
      <div className="min-h-[500px]">
        {currentStep === 1 && renderStep1()}
        {currentStep === 2 && renderStep2()}
        {currentStep === 3 && renderStep3()}
      </div>

      {/* 操作按钮 */}
      <div className="flex justify-between mt-6 pt-4 border-t">
        <button
          onClick={currentStep === 1 ? onClose : () => setCurrentStep(currentStep - 1)}
          className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-md transition-colors"
        >
          {currentStep === 1 ? '取消' : (
            <>
              <ChevronLeft className="w-4 h-4" />
              上一步
            </>
          )}
        </button>
        
        <div className="flex gap-2">
          {currentStep < 3 && (
            <button
              onClick={() => setCurrentStep(currentStep + 1)}
              disabled={currentStep === 1 && !selectedWorkflow}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              下一步
              <ChevronRight className="w-4 h-4" />
            </button>
          )}
          
          {currentStep === 3 && (
            <button
              onClick={handleSubmit}
              disabled={loading || !formData.name || !formData.displayName || !formData.description}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-green-600 hover:bg-green-700 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Upload className="w-4 h-4" />
              {publishMode === 'new' ? '发布插件' : '更新插件'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

export default NodePluginMarket