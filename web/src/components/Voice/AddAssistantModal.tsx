import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Bot, MessageCircle, Users, Zap, Circle, Building2, AlertCircle } from 'lucide-react'
import { cn } from '@/utils/cn'
import { getGroupList, type Group } from '@/api/group'
import { useAuthStore } from '@/stores/authStore'
import { useI18nStore } from '@/stores/i18nStore'
import Button from '@/components/UI/Button'

interface AddAssistantModalProps {
  isOpen: boolean
  onClose: () => void
  onAdd: (assistant: { name: string; description: string; icon: string; groupId?: number | null }) => void
}

const ICON_MAP = {
  Bot: <Bot className="w-5 h-5" />,
  MessageCircle: <MessageCircle className="w-5 h-5" />,
  Users: <Users className="w-5 h-5" />,
  Zap: <Zap className="w-5 h-5" />,
  Circle: <Circle className="w-5 h-5" />
}

const AddAssistantModal: React.FC<AddAssistantModalProps> = ({
  isOpen,
  onClose,
  onAdd
}) => {
  const { user } = useAuthStore()
  const { t } = useI18nStore()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [selectedIcon, setSelectedIcon] = useState('Bot')
  const [groups, setGroups] = useState<Group[]>([])
  const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null)
  const [shareToGroup, setShareToGroup] = useState(false)
  const [errors, setErrors] = useState<{ name?: string; description?: string }>({})

  useEffect(() => {
    if (isOpen) {
      fetchGroups()
      setErrors({})
    }
  }, [isOpen])

  const fetchGroups = async () => {
    try {
      const res = await getGroupList()
      // 只显示用户是创建者或管理员的组织
      const adminGroups = (res.data || []).filter(g => {
        const userId = user?.id ? Number(user.id) : null
        return g.creatorId === userId || g.myRole === 'admin'
      })
      setGroups(adminGroups)
    } catch (err) {
      console.error('获取组织列表失败', err)
      setGroups([])
    }
  }

  const validateForm = () => {
    const newErrors: { name?: string; description?: string } = {}
    
    if (!name.trim()) {
      newErrors.name = t('assistants.validation.nameRequired') || '请输入助手名称'
    }
    
    if (!description.trim()) {
      newErrors.description = t('assistants.validation.descriptionRequired') || '请输入助手描述'
    }
    
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = () => {
    if (!validateForm()) return
    
    onAdd({
      name,
      description,
      icon: selectedIcon,
      groupId: shareToGroup && selectedGroupId ? selectedGroupId : null
    })
    
    // 重置表单
    setName('')
    setDescription('')
    setSelectedIcon('Bot')
    setShareToGroup(false)
    setSelectedGroupId(null)
    setErrors({})
    onClose()
  }

  return (
    <AnimatePresence>
      {isOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <motion.div
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.9 }}
            className="bg-white dark:bg-neutral-800 p-6 rounded-xl max-w-md w-full mx-4 shadow-xl"
          >
            <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">
              {t('assistants.add')}
            </h3>
            
            <div className="space-y-4">
              {/* 助手名称 */}
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">
                  {t('assistants.name') || '助手名称'}
                </label>
                <input
                  value={name}
                  onChange={(e) => {
                    setName(e.target.value)
                    if (errors.name) {
                      setErrors({ ...errors, name: undefined })
                    }
                  }}
                  className={cn(
                    'w-full px-3 py-2 border rounded-lg dark:bg-neutral-700 dark:border-neutral-600 transition-colors',
                    errors.name 
                      ? 'border-red-500 dark:border-red-500 bg-red-50 dark:bg-red-900/10' 
                      : 'border-gray-300 dark:border-neutral-600'
                  )}
                  placeholder={t('assistants.namePlaceholder') || '请输入助手名称'}
                />
                {errors.name && (
                  <div className="flex items-center gap-1 mt-1 text-red-600 dark:text-red-400 text-xs">
                    <AlertCircle className="w-3 h-3" />
                    {errors.name}
                  </div>
                )}
              </div>
              
              {/* 助手描述 */}
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-1">
                  {t('assistants.description') || '助手描述'}
                </label>
                <textarea
                  value={description}
                  onChange={(e) => {
                    setDescription(e.target.value)
                    if (errors.description) {
                      setErrors({ ...errors, description: undefined })
                    }
                  }}
                  className={cn(
                    'w-full px-3 py-2 border rounded-lg dark:bg-neutral-700 dark:border-neutral-600 transition-colors resize-none',
                    errors.description 
                      ? 'border-red-500 dark:border-red-500 bg-red-50 dark:bg-red-900/10' 
                      : 'border-gray-300 dark:border-neutral-600'
                  )}
                  rows={2}
                  placeholder={t('assistants.descriptionPlaceholder') || '请输入助手描述'}
                />
                {errors.description && (
                  <div className="flex items-center gap-1 mt-1 text-red-600 dark:text-red-400 text-xs">
                    <AlertCircle className="w-3 h-3" />
                    {errors.description}
                  </div>
                )}
              </div>
              
              {/* 选择图标 */}
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300 block mb-2">
                  {t('assistants.selectIcon') || '选择图标'}
                </label>
                <div className="grid grid-cols-5 gap-2">
                  {Object.keys(ICON_MAP).map(iconName => (
                    <motion.button
                      key={iconName}
                      onClick={() => setSelectedIcon(iconName)}
                      whileHover={{ scale: 1.05 }}
                      whileTap={{ scale: 0.95 }}
                      className={cn(
                        'p-2 rounded-lg transition-all duration-200 flex items-center justify-center',
                        selectedIcon === iconName
                          ? 'bg-purple-500 text-white shadow-md' 
                          : 'bg-gray-100 dark:bg-neutral-700 text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-neutral-600'
                      )}
                    >
                      {ICON_MAP[iconName as keyof typeof ICON_MAP]}
                    </motion.button>
                  ))}
                </div>
              </div>

              {/* 共享到组织 */}
              {groups.length > 0 && (
                <div className="space-y-2">
                  <label className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={shareToGroup}
                      onChange={(e) => {
                        setShareToGroup(e.target.checked)
                        if (!e.target.checked) {
                          setSelectedGroupId(null)
                        } else if (groups.length === 1) {
                          setSelectedGroupId(groups[0].id)
                        }
                      }}
                      className="w-4 h-4 rounded border-gray-300 dark:border-neutral-600 cursor-pointer"
                    />
                    <span className="flex items-center gap-1">
                      <Building2 className="w-4 h-4" />
                      {t('assistants.shareToGroup') || '共享到组织'}
                    </span>
                  </label>
                  {shareToGroup && (
                    <select
                      value={selectedGroupId || ''}
                      onChange={(e) => setSelectedGroupId(e.target.value ? Number(e.target.value) : null)}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-600 rounded-lg dark:bg-neutral-700 dark:text-gray-100"
                    >
                      <option value="">{t('assistants.selectGroup') || '选择组织'}</option>
                      {groups.map(group => (
                        <option key={group.id} value={group.id}>
                          {group.name}
                        </option>
                      ))}
                    </select>
                  )}
                </div>
              )}
              
              {/* 操作按钮 */}
              <div className="flex justify-end gap-3 pt-2">
                <Button
                  onClick={onClose}
                  variant="ghost"
                  size="md"
                >
                  {t('common.cancel') || '取消'}
                </Button>
                <Button
                  onClick={handleSubmit}
                  variant="primary"
                  size="md"
                >
                  {t('assistants.save') || '保存助手'}
                </Button>
              </div>
            </div>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  )
}

export default AddAssistantModal
