import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Building2, AlertCircle } from 'lucide-react'
import { cn } from '@/utils/cn'
import { getGroupList, type Group } from '@/api/group'
import { useAuthStore } from '@/stores/authStore'
import { useI18nStore } from '@/stores/i18nStore'
import Button from '@/components/UI/Button'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/UI/Select'

interface AddAssistantModalProps {
  isOpen: boolean
  onClose: () => void
  onAdd: (assistant: { name: string; description: string; groupId?: number | null }) => void
}

const AddAssistantModal: React.FC<AddAssistantModalProps> = ({ isOpen, onClose, onAdd }) => {
  const { user } = useAuthStore()
  const { t } = useI18nStore()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [groups, setGroups] = useState<Group[]>([])
  const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null)
  const [shareToGroup, setShareToGroup] = useState(false)
  const [errors, setErrors] = useState<{ name?: string; description?: string }>({})

  useEffect(() => {
    if (isOpen) {
      void fetchGroups()
      setErrors({})
    }
  }, [isOpen])

  const fetchGroups = async () => {
    try {
      const res = await getGroupList()
      const adminGroups = (res.data || []).filter((g) => {
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
      newErrors.name = t('assistants.validation.nameRequired') || '请输入智能体名称'
    }
    if (!description.trim()) {
      newErrors.description = t('assistants.validation.descriptionRequired') || '请输入智能体描述'
    }
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = () => {
    if (!validateForm()) return
    onAdd({
      name,
      description,
      groupId: shareToGroup && selectedGroupId ? selectedGroupId : null,
    })
    setName('')
    setDescription('')
    setShareToGroup(false)
    setSelectedGroupId(null)
    setErrors({})
    onClose()
  }

  return (
    <AnimatePresence>
      {isOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <motion.div
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.9 }}
            className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-neutral-800"
          >
            <h3 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">{t('assistants.add')}</h3>
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {t('assistants.name') || '智能体名称'}
                </label>
                <input
                  value={name}
                  onChange={(e) => {
                    setName(e.target.value)
                    if (errors.name) setErrors({ ...errors, name: undefined })
                  }}
                  className={cn(
                    'w-full rounded-lg border px-3 py-2 transition-colors dark:bg-neutral-700 dark:text-gray-100',
                    errors.name
                      ? 'border-red-500 bg-red-50 dark:border-red-500 dark:bg-red-900/10'
                      : 'border-gray-300 dark:border-neutral-600',
                  )}
                  placeholder={t('assistants.namePlaceholder') || '请输入智能体名称'}
                />
                {errors.name && (
                  <div className="mt-1 flex items-center gap-1 text-xs text-red-600 dark:text-red-400">
                    <AlertCircle className="h-3 w-3" />
                    {errors.name}
                  </div>
                )}
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {t('assistants.description') || '智能体描述'}
                </label>
                <textarea
                  value={description}
                  onChange={(e) => {
                    setDescription(e.target.value)
                    if (errors.description) setErrors({ ...errors, description: undefined })
                  }}
                  rows={2}
                  className={cn(
                    'w-full resize-none rounded-lg border px-3 py-2 transition-colors dark:bg-neutral-700 dark:text-gray-100',
                    errors.description
                      ? 'border-red-500 bg-red-50 dark:border-red-500 dark:bg-red-900/10'
                      : 'border-gray-300 dark:border-neutral-600',
                  )}
                  placeholder={t('assistants.descriptionPlaceholder') || '请输入智能体描述'}
                />
                {errors.description && (
                  <div className="mt-1 flex items-center gap-1 text-xs text-red-600 dark:text-red-400">
                    <AlertCircle className="h-3 w-3" />
                    {errors.description}
                  </div>
                )}
              </div>
              {groups.length > 0 && (
                <div className="space-y-2">
                  <label className="flex cursor-pointer items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
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
                      className="h-4 w-4 cursor-pointer rounded border-gray-300 dark:border-neutral-600"
                    />
                    <span className="flex items-center gap-1">
                      <Building2 className="h-4 w-4" />
                      {t('assistants.shareToGroup') || '共享到组织'}
                    </span>
                  </label>
                  {shareToGroup && (
                    <Select
                      value={selectedGroupId != null ? String(selectedGroupId) : ''}
                      onValueChange={(v) => setSelectedGroupId(v ? Number(v) : null)}
                    >
                      <SelectTrigger className="w-full">
                        <SelectValue placeholder={t('assistants.selectGroup') || '选择组织'} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="">{t('assistants.selectGroup') || '选择组织'}</SelectItem>
                        {groups.map((group) => (
                          <SelectItem key={group.id} value={String(group.id)}>
                            {group.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                </div>
              )}
              <div className="flex justify-end gap-3 pt-2">
                <Button onClick={onClose} variant="ghost" size="md">
                  {t('common.cancel') || '取消'}
                </Button>
                <Button onClick={handleSubmit} variant="primary" size="md">
                  {t('assistants.save') || '保存智能体'}
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
