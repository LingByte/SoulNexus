import React from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { X, Search, Package } from 'lucide-react'
import { cn } from '@/utils/cn'
import type { WorkflowNode } from '../types'

interface NodeDrawerProps {
  isOpen: boolean
  onClose: () => void
  nodeSearchQuery: string
  onSearchChange: (query: string) => void
  NODE_TYPES: Record<string, any>
  installedPlugins: any[]
  loadingPlugins: boolean
  onAddNode: (type: WorkflowNode['type'], position: { x: number; y: number }) => void
  onAddPluginNode: (plugin: any, position: { x: number; y: number }) => void
  getIconComponent: (iconName: string) => React.ReactNode
}

/**
 * 节点选择抽屉组件
 * 用于选择和添加节点到工作流
 */
export const NodeDrawer: React.FC<NodeDrawerProps> = ({
  isOpen,
  onClose,
  nodeSearchQuery,
  onSearchChange,
  NODE_TYPES,
  installedPlugins,
  loadingPlugins,
  onAddNode,
  onAddPluginNode,
  getIconComponent
}) => {
  const handleAddNode = (type: WorkflowNode['type']) => {
    onAddNode(type, { x: 0, y: 0 })
    setTimeout(() => {
      onClose()
      onSearchChange('')
    }, 100)
  }

  const handleAddPluginNode = (plugin: any) => {
    onAddPluginNode(plugin, { x: 0, y: 0 })
    setTimeout(() => {
      onClose()
      onSearchChange('')
    }, 100)
  }

  const filteredNodeTypes = Object.entries(NODE_TYPES).filter(([type, config]) => 
    type !== 'workflow_plugin' && (
      config.label.toLowerCase().includes(nodeSearchQuery.toLowerCase()) ||
      type.toLowerCase().includes(nodeSearchQuery.toLowerCase())
    )
  )

  const filteredPlugins = installedPlugins.filter(plugin => 
    plugin.plugin?.displayName?.toLowerCase().includes(nodeSearchQuery.toLowerCase()) ||
    plugin.plugin?.name?.toLowerCase().includes(nodeSearchQuery.toLowerCase()) ||
    plugin.plugin?.description?.toLowerCase().includes(nodeSearchQuery.toLowerCase())
  )

  return (
    <AnimatePresence>
      {isOpen && (
        <>
          {/* 背景遮罩 */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
            className="fixed inset-0 bg-black/50 z-40"
          />
          
          {/* 抽屉内容 */}
          <motion.div
            initial={{ y: '100%' }}
            animate={{ y: 0 }}
            exit={{ y: '100%' }}
            transition={{ type: 'spring', damping: 25, stiffness: 200 }}
            className="fixed bottom-0 left-0 right-0 bg-white dark:bg-gray-800 rounded-t-2xl shadow-2xl z-50 max-h-[80vh] flex flex-col"
          >
            {/* 抽屉头部 */}
            <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                选择节点类型
              </h3>
              <button
                onClick={onClose}
                className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
              >
                <X className="w-5 h-5 text-gray-500 dark:text-gray-400" />
              </button>
            </div>
            
            {/* 搜索框 */}
            <div className="p-4 border-b border-gray-200 dark:border-gray-700">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
                <input
                  type="text"
                  value={nodeSearchQuery}
                  onChange={(e) => onSearchChange(e.target.value)}
                  placeholder="搜索节点类型..."
                  className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  autoFocus
                />
              </div>
            </div>
            
            {/* 节点列表 */}
            <div className="flex-1 overflow-y-auto p-6">
              {/* 内置节点类型 */}
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
                {filteredNodeTypes.map(([type, config]) => (
                  <NodeCard
                    key={type}
                    label={config.label}
                    color={config.color}
                    gradient={config.gradient}
                    iconName={config.iconName}
                    getIconComponent={getIconComponent}
                    onClick={() => handleAddNode(type as WorkflowNode['type'])}
                  />
                ))}
              </div>
              
              {/* 插件节点部分 */}
              {installedPlugins.length > 0 && (
                <>
                  <div className="mt-8 mb-4">
                    <h4 className="text-lg font-semibold text-gray-900 dark:text-white mb-2 flex items-center gap-2">
                      <Package className="w-5 h-5" />
                      已安装的插件
                    </h4>
                    <div className="h-px bg-gradient-to-r from-gray-200 via-gray-300 to-gray-200 dark:from-gray-700 dark:via-gray-600 dark:to-gray-700"></div>
                  </div>
                  
                  <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
                    {filteredPlugins.map((plugin) => (
                      <PluginNodeCard
                        key={plugin.id}
                        plugin={plugin}
                        onClick={() => handleAddPluginNode(plugin)}
                      />
                    ))}
                  </div>
                </>
              )}
              
              {/* 加载状态 */}
              {loadingPlugins && (
                <div className="mt-8 text-center">
                  <div className="inline-flex items-center gap-2 text-gray-500 dark:text-gray-400">
                    <div className="animate-spin rounded-full h-4 w-4 border-2 border-gray-300 border-t-blue-500"></div>
                    <span>加载插件中...</span>
                  </div>
                </div>
              )}
              
              {/* 空状态 */}
              {filteredNodeTypes.length === 0 && filteredPlugins.length === 0 && (
                <motion.div 
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  className="text-center py-16 text-gray-500 dark:text-gray-400"
                >
                  <div className="relative">
                    <Search className="w-16 h-16 mx-auto mb-4 opacity-30" />
                    <div className="absolute top-0 left-1/2 transform -translate-x-1/2 w-16 h-16 bg-gradient-to-r from-blue-400 to-purple-500 rounded-full blur-xl opacity-20 animate-pulse" />
                  </div>
                  <p className="text-lg font-medium">未找到匹配的节点类型</p>
                  <p className="text-sm mt-2 opacity-75">尝试使用不同的关键词搜索</p>
                </motion.div>
              )}
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}

/**
 * 节点卡片组件
 */
interface NodeCardProps {
  label: string
  color: string
  gradient: string
  iconName: string
  getIconComponent: (iconName: string) => React.ReactNode
  onClick: () => void
}

const NodeCard: React.FC<NodeCardProps> = ({
  label,
  color,
  gradient,
  iconName,
  getIconComponent,
  onClick
}) => {
  return (
    <motion.div
      className="group relative"
      whileHover={{ scale: 1.05, y: -2 }}
      whileTap={{ scale: 0.95 }}
      onClick={onClick}
    >
      <div className="relative flex flex-col items-center p-4 bg-gradient-to-br from-white via-gray-50 to-white dark:from-gray-800 dark:via-gray-850 dark:to-gray-800 rounded-2xl cursor-pointer transition-all duration-300 border border-gray-200/60 dark:border-gray-700/60 hover:border-gray-300/80 dark:hover:border-gray-600/80 shadow-lg hover:shadow-2xl backdrop-blur-sm overflow-hidden">
        
        {/* 背景装饰 */}
        <div className="absolute top-0 right-0 w-16 h-16 opacity-10">
          <div 
            className="w-full h-full rounded-full blur-xl"
            style={{ backgroundColor: color }}
          />
        </div>
        
        {/* 顶部装饰条 */}
        <div 
          className={cn(
            'absolute top-0 left-0 right-0 h-1 bg-gradient-to-r',
            gradient || 'from-gray-400 to-gray-600'
          )}
        />
        
        {/* 图标容器 */}
        <div className="relative mb-3">
          <div 
            className={cn(
              'p-3 rounded-xl shadow-lg transition-all duration-300',
              'bg-gradient-to-br border border-white/20',
              gradient || 'from-gray-400 to-gray-600',
              'group-hover:scale-110 group-hover:rotate-3'
            )}
            style={{
              boxShadow: `0 8px 16px -4px ${color}30`
            }}
          >
            <div className="text-white drop-shadow-sm">
              {getIconComponent(iconName)}
            </div>
          </div>
          
          {/* 图标光晕效果 */}
          <div 
            className="absolute inset-0 rounded-xl blur-md opacity-30 -z-10 group-hover:opacity-50 transition-opacity duration-300"
            style={{ backgroundColor: color }}
          />
        </div>
        
        {/* 标题 */}
        <span className="text-sm font-bold text-gray-900 dark:text-white text-center leading-tight">
          {label}
        </span>
        
        {/* 悬停效果 */}
        <div className="absolute inset-0 bg-gradient-to-t from-transparent via-transparent to-white/5 opacity-0 group-hover:opacity-100 transition-opacity duration-300 rounded-2xl" />
      </div>
    </motion.div>
  )
}

/**
 * 插件节点卡片组件
 */
interface PluginNodeCardProps {
  plugin: any
  onClick: () => void
}

const PluginNodeCard: React.FC<PluginNodeCardProps> = ({ plugin, onClick }) => {
  const pluginColor = plugin.plugin?.color || '#7c2d12'
  
  return (
    <motion.div
      className="group relative"
      whileHover={{ scale: 1.05, y: -2 }}
      whileTap={{ scale: 0.95 }}
      onClick={onClick}
    >
      <div className="relative flex flex-col items-center p-4 bg-gradient-to-br from-white via-gray-50 to-white dark:from-gray-800 dark:via-gray-850 dark:to-gray-800 rounded-2xl cursor-pointer transition-all duration-300 border border-gray-200/60 dark:border-gray-700/60 hover:border-gray-300/80 dark:hover:border-gray-600/80 shadow-lg hover:shadow-2xl backdrop-blur-sm overflow-hidden">
        
        {/* 背景装饰 */}
        <div className="absolute top-0 right-0 w-16 h-16 opacity-10">
          <div 
            className="w-full h-full rounded-full blur-xl"
            style={{ backgroundColor: pluginColor }}
          />
        </div>
        
        {/* 顶部装饰条 */}
        <div 
          className="absolute top-0 left-0 right-0 h-1 bg-gradient-to-r from-orange-400 to-orange-600"
          style={{ 
            background: `linear-gradient(to right, ${pluginColor}80, ${pluginColor})` 
          }}
        />
        
        {/* 图标容器 */}
        <div className="relative mb-3">
          <div 
            className="p-3 rounded-xl shadow-lg transition-all duration-300 bg-gradient-to-br border border-white/20 from-orange-400 to-orange-600 group-hover:scale-110 group-hover:rotate-3"
            style={{
              background: `linear-gradient(to bottom right, ${pluginColor}CC, ${pluginColor})`,
              boxShadow: `0 8px 16px -4px ${pluginColor}30`
            }}
          >
            <div className="text-white drop-shadow-sm">
              <Package className="w-5 h-5" />
            </div>
          </div>
          
          {/* 图标光晕效果 */}
          <div 
            className="absolute inset-0 rounded-xl blur-md opacity-30 -z-10 group-hover:opacity-50 transition-opacity duration-300"
            style={{ backgroundColor: pluginColor }}
          />
        </div>
        
        {/* 标题 */}
        <span className="text-sm font-bold text-gray-900 dark:text-white text-center leading-tight">
          {plugin.plugin?.displayName || plugin.plugin?.name || '插件节点'}
        </span>
        
        {/* 描述 */}
        {plugin.plugin?.description && (
          <span className="text-xs text-gray-500 dark:text-gray-400 text-center mt-1 line-clamp-2">
            {plugin.plugin.description}
          </span>
        )}
        
        {/* 悬停效果 */}
        <div className="absolute inset-0 bg-gradient-to-t from-transparent via-transparent to-white/5 opacity-0 group-hover:opacity-100 transition-opacity duration-300 rounded-2xl" />
      </div>
    </motion.div>
  )
}
