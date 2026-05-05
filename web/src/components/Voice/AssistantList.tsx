import React, { useState, useMemo } from 'react'
import { motion } from 'framer-motion'
import { Users, Plus, ChevronRight, Bot, MessageCircle, Circle, Zap, Search, X, Settings } from 'lucide-react'
import { cn } from '@/utils/cn'
import CollapsibleSectionHeader from '@/components/UI/CollapsibleSectionHeader'

interface Assistant {
  id: number
  name: string
  description: string
  icon: string
  active?: boolean
}

interface AssistantListProps {
  assistants: Assistant[]
  selectedAssistant: number
  onSelectAssistant: (id: number) => void
  onAddAssistant: () => void
  onConfigAssistant?: (id: number) => void
  className?: string
}

const ICON_MAP = {
  Bot: <Bot className="w-5 h-5" />,
  MessageCircle: <MessageCircle className="w-5 h-5" />,
  Users: <Users className="w-5 h-5" />,
  Zap: <Zap className="w-5 h-5" />,
  Circle: <Circle className="w-5 h-5" />
}

const AssistantList: React.FC<AssistantListProps> = ({
  assistants,
  selectedAssistant,
  onSelectAssistant,
  onAddAssistant,
  onConfigAssistant,
  className = ''
}) => {
  const [searchQuery, setSearchQuery] = useState('')

  // 过滤助手列表
  const filteredAssistants = useMemo(() => {
    if (!searchQuery.trim()) {
      return assistants
    }
    
    const query = searchQuery.toLowerCase()
    return assistants.filter(assistant => 
      assistant.name.toLowerCase().includes(query) ||
      assistant.description.toLowerCase().includes(query)
    )
  }, [assistants, searchQuery])

  const clearSearch = () => {
    setSearchQuery('')
  }

  return (
    <div className={cn('flex-1 p-3', className)}>
      {/* 标题栏 */}
      <CollapsibleSectionHeader
        title="虚拟人物列表"
        icon={<Users className="w-4 h-4" />}
        expanded={true}
        onToggle={() => {}}
        showChevron={false}
        clickable={false}
        compact
        titleSize="md"
        withDivider
        rightContent={(
          <div className="flex items-end space-x-2">
            <motion.button
              onClick={onAddAssistant}
              className="p-1 hover:bg-gray-100 dark:hover:bg-neutral-700 rounded-md transition-colors"
              title="添加智能体"
            >
              <Plus className="w-4 h-4 text-purple-600" />
            </motion.button>
          </div>
        )}
      />

      {/* 搜索框 */}
      <div className="relative mb-3">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 transform -translate-y-1/2 w-3.5 h-3.5 text-gray-400" />
          <input
            type="text"
            placeholder="搜索智能体..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-8 pr-8 py-1.5 text-xs border border-gray-200 dark:border-neutral-600 rounded-md 
                     bg-white dark:bg-neutral-700 text-gray-900 dark:text-gray-100
                     focus:ring-2 focus:ring-purple-500 focus:border-transparent
                     placeholder-gray-500 dark:placeholder-gray-400"
          />
          {searchQuery && (
            <button
              onClick={clearSearch}
              className="absolute right-2 top-1/2 transform -translate-y-1/2 p-1 hover:bg-gray-100 dark:hover:bg-neutral-600 rounded"
              title="清除搜索"
            >
              <X className="w-3.5 h-3.5 text-gray-400" />
            </button>
          )}
        </div>
      </div>
      
      {/* 助手列表内容 */}
      <div className="space-y-2 max-h-[calc(100vh-300px)] overflow-y-auto custom-scrollbar">
        {filteredAssistants.length > 0 ? (
          filteredAssistants.map((assistant, index) => (
            <motion.div
              key={assistant.id}
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: index * 0.05 }}
              onClick={() => onSelectAssistant(assistant.id)}
              className={cn(
                'p-3 rounded-lg cursor-pointer transition-all duration-200',
                selectedAssistant === assistant.id
                  ? 'bg-purple-50 dark:bg-neutral-700 text-purple-600 dark:text-purple-400 shadow-sm'
                  : 'hover:bg-gray-50 dark:hover:bg-neutral-600 hover:shadow-sm'
              )}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center">
                  <div className={cn(
                    'p-2 rounded-md transition-colors',
                    selectedAssistant === assistant.id
                      ? 'bg-purple-100 dark:bg-neutral-600'
                      : 'bg-gray-100 dark:bg-neutral-700'
                  )}>
                    {ICON_MAP[assistant.icon as keyof typeof ICON_MAP] || <Circle className="w-5 h-5" />}
                  </div>
                  <div className="ml-3 flex-1 min-w-0">
                    <div className="text-xs font-medium truncate" title={assistant.name}>
                      {assistant.name && assistant.name.length > 5 
                        ? `${assistant.name.slice(0, 5)}...` 
                        : assistant.name}
                    </div>
                    <div className="text-xs text-gray-500 dark:text-gray-400 line-clamp-1">
                      {assistant.description}
                    </div>
                  </div>
                </div>
                <div className="flex items-center space-x-1">
                  {onConfigAssistant && (
                    <motion.button
                      onClick={(e) => {
                        e.stopPropagation()
                        onConfigAssistant(assistant.id)
                      }}
                      className="p-1.5 hover:bg-gray-100 dark:hover:bg-neutral-600 rounded-lg transition-colors"
                      whileHover={{ scale: 1.05 }}
                      whileTap={{ scale: 0.95 }}
                      title="配置智能体"
                    >
                      <Settings className="w-4 h-4 text-gray-500 hover:text-purple-600" />
                    </motion.button>
                  )}
                  <ChevronRight className="w-4 h-4 text-gray-400" />
                </div>
              </div>
            </motion.div>
          ))
        ) : (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400">
            <Search className="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">
              {searchQuery ? '未找到匹配的智能体' : '暂无智能体'}
            </p>
            {searchQuery && (
              <button
                onClick={clearSearch}
                className="text-purple-600 hover:text-purple-700 text-sm mt-2"
              >
                清除搜索条件
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

export default AssistantList
