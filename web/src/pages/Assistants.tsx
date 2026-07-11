import React, { useEffect, useMemo, useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  getAssistantList,
  getAssistant,
  createAssistant,
  updateAssistant,
  deleteAssistant,
  importCharacterCard,
  exportCharacterCard,
  uploadAgentAvatar,
  listPresets,
  applyPreset,
  type AssistantListItem,
  type Assistant,
  type PresetTemplate,
} from '@/api/assistant'
import AddAssistantModal from '@/components/Voice/AddAssistantModal'
import { showAlert } from '@/utils/notification'
import { useI18nStore } from '@/stores/i18nStore'
import {
  Bot, Users, Zap, Plus, Sparkles, Rocket, Wand2, Search,
  Eye, Edit3, Trash2, Download, Upload as UploadIcon,
  FileInput, Star, MoreHorizontal, Image as ImageIcon,
  Globe, Lock,
} from 'lucide-react'
import Button from '@/components/UI/Button'
import PageHeader from '@/components/Layout/PageHeader'
import { motion } from 'framer-motion'
import { Input as ArcoInput, Drawer, Modal, Upload, Tag, Select, Switch, InputNumber } from '@arco-design/web-react'

// 根据 id 稳定生成渐变色
const GRADIENT_POOL = [
  'from-purple-500 to-pink-500',
  'from-blue-500 to-cyan-500',
  'from-emerald-500 to-teal-500',
  'from-amber-500 to-orange-500',
  'from-rose-500 to-red-500',
  'from-indigo-500 to-violet-500',
  'from-sky-500 to-blue-600',
  'from-fuchsia-500 to-purple-600',
]

const pickGradient = (id: number) => GRADIENT_POOL[Math.abs(id) % GRADIENT_POOL.length]

const visibilityConfig: Record<string, { label: string; color: string }> = {
  public: { label: '公开', color: 'green' },
  group: { label: '组织', color: 'blue' },
  private: { label: '私有', color: 'gray' },
}

const Assistants: React.FC = () => {
  const { t } = useI18nStore()
  const [assistants, setAssistants] = useState<AssistantListItem[]>([])
  const [showAddModal, setShowAddModal] = useState(false)
  const [keyword, setKeyword] = useState('')
  const navigate = useNavigate()

  // 详情抽屉
  const [detailVisible, setDetailVisible] = useState(false)
  const [detailAgent, setDetailAgent] = useState<Assistant | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  // 编辑抽屉
  const [editVisible, setEditVisible] = useState(false)
  const [editAgent, setEditAgent] = useState<Assistant | null>(null)
  const [editSaving, setEditSaving] = useState(false)

  // 导入 Modal
  const [importVisible, setImportVisible] = useState(false)
  const [importFile, setImportFile] = useState<File | null>(null)
  const [importGroupId] = useState<number | undefined>(undefined)
  const [importLoading, setImportLoading] = useState(false)

  // 卡片菜单
  const [menuAgentId, setMenuAgentId] = useState<number | null>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuAgentId(null)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const filtered = useMemo(() => {
    const k = keyword.trim().toLowerCase()
    if (!k) return assistants
    return assistants.filter(a =>
      (a.name || '').toLowerCase().includes(k) ||
      (a.description || '').toLowerCase().includes(k) ||
      (a.tags || '').toLowerCase().includes(k)
    )
  }, [assistants, keyword])

  const fetchAssistants = async () => {
    try {
      const res = await getAssistantList()
      setAssistants(res.data || [])
    } catch {
      showAlert(t('assistants.messages.fetchFailed'), 'error')
      setAssistants([])
    }
  }

  useEffect(() => {
    fetchAssistants()
  }, [])

  const handleAddAssistant = async (assistant: { name: string; groupId?: number | null }) => {
    try {
      await createAssistant(assistant)
      await fetchAssistants()
      setShowAddModal(false)
      showAlert(t('assistants.messages.createSuccess'), 'success')
    } catch (err: any) {
      showAlert(err?.msg || err?.message || t('assistants.messages.createFailed'), 'error')
    }
  }

  // ---- 详情 ----
  const openDetail = async (id: number) => {
    setDetailLoading(true)
    setDetailVisible(true)
    try {
      const res = await getAssistant(id)
      setDetailAgent(res.data)
    } catch {
      showAlert('获取详情失败', 'error')
      setDetailVisible(false)
    } finally {
      setDetailLoading(false)
    }
  }

  // ---- 编辑 ----
  const openEdit = async (id: number) => {
    try {
      const res = await getAssistant(id)
      setEditAgent(res.data)
      setEditVisible(true)
    } catch {
      showAlert('获取智能体信息失败', 'error')
    }
  }

  const handleEditSave = async () => {
    if (!editAgent) return
    setEditSaving(true)
    try {
      await updateAssistant(editAgent.id, {
        name: editAgent.name,
        systemPrompt: editAgent.systemPrompt,
        persona_tag: editAgent.personaTag,
        temperature: editAgent.temperature,
        maxTokens: editAgent.maxTokens,
        speaker: editAgent.speaker,
        ttsProvider: editAgent.ttsProvider,
        llmModel: editAgent.llmModel,
        enableJSONOutput: editAgent.enableJSONOutput,
        openingStatement: editAgent.openingStatement,
        description: editAgent.description,
        personality: editAgent.personality,
        scenario: editAgent.scenario,
        exampleDialogues: editAgent.exampleDialogues,
        tags: editAgent.tags,
        creatorNote: editAgent.creatorNote,
        specVersion: editAgent.specVersion,
        visibility: editAgent.visibility,
      })
      showAlert('保存成功', 'success')
      setEditVisible(false)
      await fetchAssistants()
    } catch (err: any) {
      showAlert(err?.msg || '保存失败', 'error')
    } finally {
      setEditSaving(false)
    }
  }

  // ---- 删除 ----
  const handleDelete = async (id: number) => {
    try {
      await deleteAssistant(id)
      showAlert('删除成功', 'success')
      setMenuAgentId(null)
      await fetchAssistants()
    } catch {
      showAlert('删除失败', 'error')
    }
  }

  // ---- 导入 ----
  const handleImport = async () => {
    if (!importFile) return
    setImportLoading(true)
    try {
      await importCharacterCard(importFile, importGroupId)
      showAlert('导入成功', 'success')
      setImportVisible(false)
      setImportFile(null)
      await fetchAssistants()
    } catch (err: any) {
      showAlert(err?.msg || '导入失败', 'error')
    } finally {
      setImportLoading(false)
    }
  }

  // ---- 导出 ----
  const handleExport = async (id: number, format: 'json' | 'png') => {
    try {
      const blob = await exportCharacterCard(id, format)
      const ext = format === 'png' ? 'png' : 'json'
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `character-card-${id}.${ext}`
      a.click()
      URL.revokeObjectURL(url)
      showAlert('导出成功', 'success')
      setMenuAgentId(null)
    } catch {
      showAlert('导出失败', 'error')
    }
  }

  // ---- 头像上传 ----
  const handleAvatarUpload = async (id: number, file: File) => {
    try {
      await uploadAgentAvatar(id, file)
      showAlert('头像上传成功', 'success')
      await fetchAssistants()
      // 刷新详情
      if (detailAgent?.id === id) {
        const detail = await getAssistant(id)
        setDetailAgent(detail.data)
      }
    } catch {
      showAlert('头像上传失败', 'error')
    }
  }

  // ---- 发布 / 下架 ----
  const handleTogglePublish = async (agent: AssistantListItem) => {
    const newVisibility = agent.visibility === 'public' ? 'group' : 'public'
    const actionLabel = newVisibility === 'public' ? '发布到市场' : '从市场下架'
    try {
      await updateAssistant(agent.id, { visibility: newVisibility })
      showAlert(`${actionLabel}成功`, 'success')
      setMenuAgentId(null)
      await fetchAssistants()
    } catch (err: any) {
      showAlert(err?.msg || `${actionLabel}失败`, 'error')
    }
  }

  // ---- 预设模板快捷加载 ----
  const [promptPresetVisible, setPromptPresetVisible] = useState(false)
  const [promptPresets, setPromptPresets] = useState<PresetTemplate[]>([])
  const [promptPresetLoading, setPromptPresetLoading] = useState(false)

  const openPromptPresetModal = async () => {
    setPromptPresetVisible(true)
    setPromptPresetLoading(true)
    try {
      const res = await listPresets({ type: 'system_prompt', pageSize: 50 })
      if (res.code === 200 && res.data) {
        setPromptPresets(res.data.list)
      }
    } catch {
      // ignore
    } finally {
      setPromptPresetLoading(false)
    }
  }

  const handleApplyPromptPreset = async (preset: PresetTemplate) => {
    try {
      const res = await applyPreset(Number(preset.id), { presetId: Number(preset.id) })
      if (res.code === 200 && res.data) {
        const prompt = (res.data as any)?.systemPrompt || ''
        if (editAgent) {
          setEditAgent({ ...editAgent, systemPrompt: prompt })
        }
        showAlert('提示词已加载', 'success')
      }
    } catch (err: any) {
      showAlert(err?.msg || '加载失败', 'error')
    }
    setPromptPresetVisible(false)
  }

  const fmtDate = (iso?: string) => (iso ? iso.slice(0, 10) : '')

  const tagsToList = (tags?: string): string[] => {
    if (!tags) return []
    return tags.split(',').map(t => t.trim()).filter(Boolean)
  }

  return (
    <div className="flex flex-col h-full dark:bg-neutral-900">
      <PageHeader
        title={t('assistants.title')}
        actions={
          <div className="flex items-center gap-2">
            <div className="w-64">
              <ArcoInput
                size="large"
                className="!h-10 !text-base"
                value={keyword}
                onChange={(val) => setKeyword(val)}
                placeholder={t('assistants.searchPlaceholder') || '搜索智能体'}
                prefix={<Search className="h-4 w-4 text-muted-foreground" />}
              />
            </div>
            <Button
              onClick={() => setImportVisible(true)}
              variant="outline"
              size="sm"
              leftIcon={<FileInput className="w-4 h-4" />}
            >
              导入
            </Button>
            <Button
              onClick={() => setShowAddModal(true)}
              variant="primary"
              size="sm"
              leftIcon={<Plus className="w-4 h-4" />}
            >
              {t('assistants.add')}
            </Button>
          </div>
        }
      />

      <div className="flex-1 overflow-auto">
        <div className="max-w-6xl w-full mx-auto px-4 pt-8 pb-10 flex flex-col">
          <div className="w-full grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {/* 空状态 */}
            {(filtered?.length === 0) && (assistants.length === 0) && (
              <motion.div
                className="col-span-full"
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5 }}
              >
                <div className="relative max-w-2xl mx-auto py-20 px-6">
                  <div className="absolute inset-0 bg-gradient-to-br from-purple-50 via-pink-50 to-blue-50 dark:from-purple-900/10 dark:via-pink-900/10 dark:to-blue-900/10 rounded-3xl blur-3xl opacity-50" />
                  <div className="relative text-center">
                    <motion.div
                      initial={{ scale: 0.8, opacity: 0 }}
                      animate={{ scale: 1, opacity: 1 }}
                      transition={{ delay: 0.2, duration: 0.5, type: 'spring' }}
                      className="inline-flex items-center justify-center mb-6"
                    >
                      <div className="relative">
                        <div className="absolute inset-0 bg-gradient-to-r from-purple-400 via-pink-400 to-blue-400 rounded-full blur-2xl opacity-30 animate-pulse" />
                        <div className="absolute inset-0 bg-gradient-to-br from-purple-500 via-pink-500 to-blue-500 rounded-full blur-xl opacity-50" />
                        <div className="relative w-32 h-32 rounded-full bg-gradient-to-br from-purple-500 via-pink-500 to-blue-500 flex items-center justify-center shadow-2xl">
                          <div className="absolute inset-0 rounded-full bg-gradient-to-br from-white/20 to-transparent" />
                          <Rocket className="w-16 h-16 animate-bounce" style={{ animationDuration: '2s' }} />
                        </div>
                        <motion.div
                          initial={{ scale: 0, rotate: 0 }}
                          animate={{ scale: 1, rotate: 360 }}
                          transition={{ delay: 0.5, duration: 0.8 }}
                          className="absolute -top-2 -right-2"
                        >
                          <Sparkles className="w-8 h-8 text-yellow-400" />
                        </motion.div>
                        <motion.div
                          initial={{ scale: 0, rotate: 0 }}
                          animate={{ scale: 1, rotate: -360 }}
                          transition={{ delay: 0.7, duration: 0.8 }}
                          className="absolute -bottom-2 -left-2"
                        >
                          <Wand2 className="w-6 h-6 text-purple-400" />
                        </motion.div>
                      </div>
                    </motion.div>
                    <motion.h2
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ delay: 0.4, duration: 0.5 }}
                      className="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-3"
                    >
                      {t('assistants.emptyState.title')}
                    </motion.h2>
                    <motion.p
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ delay: 0.5, duration: 0.5 }}
                      className="text-gray-600 dark:text-gray-400 text-lg mb-8 max-w-md mx-auto leading-relaxed"
                    >
                      {t('assistants.emptyState.description')}
                    </motion.p>
                    <motion.div
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ delay: 0.6, duration: 0.5 }}
                      className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8 max-w-2xl mx-auto"
                    >
                      {[
                        { icon: Bot, text: t('assistants.emptyState.features.smartDialogue'), color: 'from-purple-500 to-pink-500' },
                        { icon: Zap, text: t('assistants.emptyState.features.fastResponse'), color: 'from-yellow-500 to-orange-500' },
                        { icon: Users, text: t('assistants.emptyState.features.multiScenario'), color: 'from-blue-500 to-cyan-500' },
                      ].map((item, index) => (
                        <motion.div
                          key={index}
                          initial={{ opacity: 0, scale: 0.9 }}
                          animate={{ opacity: 1, scale: 1 }}
                          transition={{ delay: 0.7 + index * 0.1, duration: 0.3 }}
                          className="flex flex-col items-center p-4 rounded-xl bg-white/50 dark:bg-neutral-800/50 backdrop-blur-sm border border-gray-200/50 dark:border-neutral-700/50"
                        >
                          <div className={`w-12 h-12 rounded-lg bg-gradient-to-br ${item.color} flex items-center justify-center mb-2 shadow-lg`}>
                            <item.icon className="w-6 h-6" />
                          </div>
                          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{item.text}</span>
                        </motion.div>
                      ))}
                    </motion.div>
                    <motion.div
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ delay: 0.8, duration: 0.5 }}
                    >
                      <Button
                        onClick={() => setShowAddModal(true)}
                        variant="primary"
                        size="lg"
                        leftIcon={<Plus className="w-5 h-5" />}
                        className="bg-gradient-to-r from-purple-500 to-pink-500 hover:from-purple-600 hover:to-pink-600 shadow-lg hover:shadow-xl transform hover:scale-105 transition-all duration-200"
                      >
                        {t('assistants.emptyState.createButton')}
                      </Button>
                    </motion.div>
                  </div>
                </div>
              </motion.div>
            )}

            {/* 卡片列表 */}
            {filtered.map((assistant, index) => {
              const gradient = pickGradient(assistant.id)
              return (
                <motion.div
                  key={assistant.id}
                  initial={{ opacity: 0, y: 12 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: index * 0.03, duration: 0.2 }}
                  whileHover={{ y: -3 }}
                  className="group relative bg-white dark:bg-neutral-800/80 rounded-2xl overflow-hidden border border-gray-200/70 dark:border-neutral-700/60 hover:border-primary/40 dark:hover:border-primary/40 shadow-[0_1px_2px_rgba(0,0,0,0.03)] hover:shadow-lg hover:shadow-primary/5 transition-all duration-200 cursor-pointer"
                  onClick={() => navigate(`/voice-assistant/${assistant.id}`)}
                >
                  <div className="p-5">
                    <div className="flex items-start gap-3">
                      {/* 头像 */}
                      <div
                        className={`w-12 h-12 rounded-xl bg-gradient-to-br ${gradient} flex items-center justify-center shadow-sm overflow-hidden flex-shrink-0`}
                      >
                        {assistant.avatarUrl ? (
                          <img src={assistant.avatarUrl} alt={assistant.name} className="w-full h-full object-cover" />
                        ) : (
                          <Bot className="h-6 w-6" />
                        )}
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <h3 className="font-semibold text-[15px] text-gray-900 dark:text-gray-100 truncate group-hover:text-primary transition-colors">
                            {assistant.name}
                          </h3>
                        </div>
                        <div className="mt-0.5 text-[11px] text-gray-400 dark:text-gray-500 font-mono">
                          #{assistant.id}
                        </div>
                      </div>

                      {/* 更多操作按钮 */}
                      <div className="relative flex-shrink-0" onClick={e => e.stopPropagation()}>
                        <button
                          onClick={() => setMenuAgentId(menuAgentId === assistant.id ? null : assistant.id)}
                          className="w-7 h-7 flex items-center justify-center rounded-md opacity-0 group-hover:opacity-100 hover:bg-gray-100 dark:hover:bg-neutral-700 transition-all"
                        >
                          <MoreHorizontal className="w-4 h-4 text-gray-400" />
                        </button>
                        {menuAgentId === assistant.id && (
                          <div
                            ref={menuRef}
                            className="absolute right-0 top-8 w-40 bg-white dark:bg-neutral-800 rounded-lg shadow-xl border border-gray-200 dark:border-neutral-700 z-50 py-1"
                          >
                            <button
                              onClick={() => { openDetail(assistant.id); setMenuAgentId(null) }}
                              className="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-neutral-700"
                            >
                              <Eye className="w-4 h-4" /> 查看详情
                            </button>
                            <button
                              onClick={() => { openEdit(assistant.id); setMenuAgentId(null) }}
                              className="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-neutral-700"
                            >
                              <Edit3 className="w-4 h-4" /> 编辑
                            </button>
                            <button
                              onClick={() => handleDelete(assistant.id)}
                              className="w-full flex items-center gap-2 px-3 py-2 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20"
                            >
                              <Trash2 className="w-4 h-4" /> 删除
                            </button>
                            <div className="border-t border-gray-100 dark:border-neutral-700 my-1" />
                            <button
                              onClick={() => handleTogglePublish(assistant)}
                              className="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-neutral-700"
                            >
                              {assistant.visibility === 'public' ? (
                                <><Lock className="w-4 h-4 text-orange-500" /> 从市场下架</>
                              ) : (
                                <><Globe className="w-4 h-4 text-green-500" /> 发布到市场</>
                              )}
                            </button>
                            <div className="border-t border-gray-100 dark:border-neutral-700 my-1" />
                            <button
                              onClick={() => { handleExport(assistant.id, 'json'); setMenuAgentId(null) }}
                              className="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-neutral-700"
                            >
                              <Download className="w-4 h-4" /> 导出 JSON
                            </button>
                            <button
                              onClick={() => { handleExport(assistant.id, 'png'); setMenuAgentId(null) }}
                              className="w-full flex items-center gap-2 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-neutral-700"
                            >
                              <Download className="w-4 h-4" /> 导出 PNG
                            </button>
                          </div>
                        )}
                      </div>
                    </div>

                    {/* 描述 */}
                    {assistant.description && (
                      <p className="mt-3 text-xs text-gray-500 dark:text-gray-400 line-clamp-2 leading-relaxed">
                        {assistant.description}
                      </p>
                    )}

                    {/* 标签 + 可见性 */}
                    <div className="mt-3 flex items-center gap-1.5 flex-wrap">
                      {assistant.visibility && (
                        <Tag size="small" color={visibilityConfig[assistant.visibility]?.color || 'gray'}>
                          {visibilityConfig[assistant.visibility]?.label || assistant.visibility}
                        </Tag>
                      )}
                      {assistant.tags && tagsToList(assistant.tags).slice(0, 3).map(tag => (
                        <span
                          key={tag}
                          className="px-2 py-0.5 rounded-md bg-indigo-50 dark:bg-indigo-900/20 text-indigo-600 dark:text-indigo-300 text-[11px] font-medium"
                        >
                          {tag.length > 10 ? tag.slice(0, 10) + '…' : tag}
                        </span>
                      ))}
                      {assistant.personaTag && (
                        <span className="px-2 py-0.5 rounded-md bg-purple-50 dark:bg-purple-900/20 text-purple-600 dark:text-purple-300 text-[11px] font-medium">
                          {assistant.personaTag.length > 14 ? assistant.personaTag.slice(0, 14) + '…' : assistant.personaTag}
                        </span>
                      )}
                      {typeof assistant.temperature === 'number' && (
                        <span className="px-2 py-0.5 rounded-md bg-orange-50 dark:bg-orange-900/20 text-orange-600 dark:text-orange-300 text-[11px] font-medium">
                          T {assistant.temperature}
                        </span>
                      )}
                    </div>

                    {/* 底栏：下载/评分 + 日期 */}
                    <div className="mt-4 pt-3 border-t border-gray-100 dark:border-neutral-700/60 flex items-center justify-between text-[11px] text-gray-400 dark:text-gray-500">
                      <div className="flex items-center gap-3">
                        {typeof assistant.downloadCount === 'number' && assistant.downloadCount > 0 && (
                          <span className="flex items-center gap-0.5">
                            <Download className="w-3 h-3" /> {assistant.downloadCount}
                          </span>
                        )}
                        {typeof assistant.rating === 'number' && assistant.rating > 0 && (
                          <span className="flex items-center gap-0.5 text-amber-500">
                            <Star className="w-3 h-3 fill-current" /> {assistant.rating.toFixed(1)}
                          </span>
                        )}
                      </div>
                      {assistant.createdAt && <span>{fmtDate(assistant.createdAt)}</span>}
                    </div>
                  </div>
                </motion.div>
              )
            })}

            {/* 搜索无结果 */}
            {assistants.length > 0 && filtered.length === 0 && (
              <div className="col-span-full text-center py-16 text-sm text-gray-500 dark:text-gray-400">
                {t('assistants.noMatch') || '没有匹配的智能体'}
              </div>
            )}
          </div>
        </div>
      </div>

      <AddAssistantModal isOpen={showAddModal} onClose={() => setShowAddModal(false)} onAdd={handleAddAssistant} />

      {/* ========== 详情抽屉 ========== */}
      <Drawer
        width={480}
        title={null}
        visible={detailVisible}
        onCancel={() => { setDetailVisible(false); setDetailAgent(null) }}
        footer={null}
        className="!p-0"
      >
        {detailLoading ? (
          <div className="flex items-center justify-center py-20">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
          </div>
        ) : detailAgent ? (
          <div className="flex flex-col h-full">
            {/* 头像 + 名称头部 */}
            <div className="relative px-6 pt-8 pb-6 bg-gradient-to-b from-primary/5 to-transparent">
              <div className="flex items-center gap-4">
                <div className="relative group/avatar">
                  {detailAgent.avatarUrl ? (
                    <img src={detailAgent.avatarUrl} alt={detailAgent.name} className="w-16 h-16 rounded-2xl object-cover shadow-md" />
                  ) : (
                    <div className={`w-16 h-16 rounded-2xl bg-gradient-to-br ${pickGradient(detailAgent.id)} flex items-center justify-center shadow-md`}>
                      <Bot className="h-8 w-8" />
                    </div>
                  )}
                  <label className="absolute inset-0 flex items-center justify-center bg-black/40 rounded-2xl opacity-0 group-hover/avatar:opacity-100 cursor-pointer transition-opacity">
                    <UploadIcon className="w-5 h-5 text-white" />
                    <input
                      type="file"
                      accept="image/*"
                      className="hidden"
                      onChange={async (e) => {
                        const file = e.target.files?.[0]
                        if (file) await handleAvatarUpload(detailAgent.id, file)
                      }}
                    />
                  </label>
                </div>
                <div className="flex-1 min-w-0">
                  <h2 className="text-xl font-bold text-gray-900 dark:text-gray-100 truncate">{detailAgent.name}</h2>
                  <div className="flex items-center gap-2 mt-1">
                    <span className="text-xs text-gray-400 font-mono">#{detailAgent.id}</span>
                    {detailAgent.visibility && (
                      <Tag size="small" color={visibilityConfig[detailAgent.visibility]?.color || 'gray'}>
                        {visibilityConfig[detailAgent.visibility]?.label}
                      </Tag>
                    )}
                  </div>
                </div>
              </div>
              {/* 快捷发布/下架按钮 */}
              <div className="mt-3">
                <button
                  onClick={() => handleTogglePublish({ id: detailAgent.id, name: detailAgent.name, visibility: detailAgent.visibility })}
                  className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                    detailAgent.visibility === 'public'
                      ? 'bg-orange-50 dark:bg-orange-900/20 text-orange-600 hover:bg-orange-100 dark:hover:bg-orange-900/40'
                      : 'bg-green-50 dark:bg-green-900/20 text-green-600 hover:bg-green-100 dark:hover:bg-green-900/40'
                  }`}
                >
                  {detailAgent.visibility === 'public' ? (
                    <><Lock className="w-3.5 h-3.5" /> 从市场下架</>
                  ) : (
                    <><Globe className="w-3.5 h-3.5" /> 发布到市场</>
                  )}
                </button>
              </div>
            </div>

            <div className="flex-1 overflow-auto px-6 pb-8 space-y-5">
              {/* 描述 */}
              {detailAgent.description && (
                <section>
                  <h4 className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">描述</h4>
                  <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap">{detailAgent.description}</p>
                </section>
              )}

              {/* 开场白 */}
              {detailAgent.openingStatement && (
                <section>
                  <h4 className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">开场白</h4>
                  <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap bg-gray-50 dark:bg-neutral-800/50 rounded-lg p-3">{detailAgent.openingStatement}</p>
                </section>
              )}

              {/* 人格设定 */}
              {detailAgent.personality && (
                <section>
                  <h4 className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">人格设定</h4>
                  <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap">{detailAgent.personality}</p>
                </section>
              )}

              {/* 世界观/场景 */}
              {detailAgent.scenario && (
                <section>
                  <h4 className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">世界观 / 场景</h4>
                  <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap">{detailAgent.scenario}</p>
                </section>
              )}

              {/* 示例对话 */}
              {detailAgent.exampleDialogues && (
                <section>
                  <h4 className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">示例对话</h4>
                  <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed whitespace-pre-wrap bg-gray-50 dark:bg-neutral-800/50 rounded-lg p-3">{detailAgent.exampleDialogues}</p>
                </section>
              )}

              {/* 标签 */}
              {detailAgent.tags && (
                <section>
                  <h4 className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">标签</h4>
                  <div className="flex flex-wrap gap-1.5">
                    {tagsToList(detailAgent.tags).map(tag => (
                      <span key={tag} className="px-2.5 py-1 rounded-md bg-indigo-50 dark:bg-indigo-900/20 text-indigo-600 dark:text-indigo-300 text-xs font-medium">{tag}</span>
                    ))}
                  </div>
                </section>
              )}

              {/* 创作者备注 */}
              {detailAgent.creatorNote && (
                <section>
                  <h4 className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">创作者备注</h4>
                  <p className="text-sm text-gray-500 dark:text-gray-400 leading-relaxed">{detailAgent.creatorNote}</p>
                </section>
              )}

              {/* 系统提示词 */}
              {detailAgent.systemPrompt && (
                <section>
                  <h4 className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">系统提示词</h4>
                  <pre className="text-xs text-gray-600 dark:text-gray-400 bg-gray-50 dark:bg-neutral-800/50 rounded-lg p-3 overflow-auto max-h-40 whitespace-pre-wrap">{detailAgent.systemPrompt}</pre>
                </section>
              )}

              {/* 基本信息 */}
              <section>
                <h4 className="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2">基本信息</h4>
                <div className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
                  {detailAgent.speaker && (
                    <>
                      <span className="text-gray-400 dark:text-gray-500">音色</span>
                      <span className="text-gray-700 dark:text-gray-300">{detailAgent.speaker}</span>
                    </>
                  )}
                  {detailAgent.llmModel && (
                    <>
                      <span className="text-gray-400 dark:text-gray-500">模型</span>
                      <span className="text-gray-700 dark:text-gray-300">{detailAgent.llmModel}</span>
                    </>
                  )}
                  {typeof detailAgent.temperature === 'number' && (
                    <>
                      <span className="text-gray-400 dark:text-gray-500">Temperature</span>
                      <span className="text-gray-700 dark:text-gray-300">{detailAgent.temperature}</span>
                    </>
                  )}
                  {typeof detailAgent.maxTokens === 'number' && (
                    <>
                      <span className="text-gray-400 dark:text-gray-500">Max Tokens</span>
                      <span className="text-gray-700 dark:text-gray-300">{detailAgent.maxTokens}</span>
                    </>
                  )}
                  {detailAgent.specVersion && (
                    <>
                      <span className="text-gray-400 dark:text-gray-500">规范版本</span>
                      <span className="text-gray-700 dark:text-gray-300">{detailAgent.specVersion}</span>
                    </>
                  )}
                  <span className="text-gray-400 dark:text-gray-500">创建时间</span>
                  <span className="text-gray-700 dark:text-gray-300">{fmtDate(detailAgent.createdAt)}</span>
                </div>
              </section>
            </div>
          </div>
        ) : null}
      </Drawer>

      {/* ========== 编辑抽屉 ========== */}
      <Drawer
        width={520}
        title={<span className="text-lg font-semibold">编辑智能体</span>}
        visible={editVisible}
        onCancel={() => { setEditVisible(false); setEditAgent(null) }}
        footer={
          <div className="flex justify-end gap-3">
            <Button variant="ghost" onClick={() => { setEditVisible(false); setEditAgent(null) }}>
              取消
            </Button>
            <Button variant="primary" onClick={handleEditSave} loading={editSaving}>
              保存
            </Button>
          </div>
        }
      >
        {editAgent && (
          <div className="space-y-4">
            {/* 头像 */}
            <div className="flex items-center gap-4">
              <div className="relative group/avatar">
                {editAgent.avatarUrl ? (
                  <img src={editAgent.avatarUrl} alt={editAgent.name} className="w-14 h-14 rounded-xl object-cover shadow-sm" />
                ) : (
                  <div className={`w-14 h-14 rounded-xl bg-gradient-to-br ${pickGradient(editAgent.id)} flex items-center justify-center shadow-sm`}>
                    <Bot className="h-7 w-7" />
                  </div>
                )}
                <label className="absolute inset-0 flex items-center justify-center bg-black/40 rounded-xl opacity-0 group-hover/avatar:opacity-100 cursor-pointer transition-opacity">
                  <ImageIcon className="w-4 h-4 text-white" />
                  <input
                    type="file"
                    accept="image/*"
                    className="hidden"
                    onChange={async (e) => {
                      const file = e.target.files?.[0]
                      if (file) await handleAvatarUpload(editAgent.id, file)
                    }}
                  />
                </label>
              </div>
              <div className="flex-1">
                <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">名称</label>
                <ArcoInput
                  value={editAgent.name}
                  onChange={(val) => setEditAgent({ ...editAgent, name: val })}
                  className="!h-9"
                />
              </div>
            </div>

            {/* 可见性 */}
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">可见性</label>
              <Select
                value={editAgent.visibility || 'group'}
                onChange={(val) => setEditAgent({ ...editAgent, visibility: val })}
                style={{ width: '100%' }}
              >
                <Select.Option value="private">私有</Select.Option>
                <Select.Option value="group">组织</Select.Option>
                <Select.Option value="public">公开</Select.Option>
              </Select>
            </div>

            {/* 描述 */}
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">描述</label>
              <ArcoInput.TextArea
                value={editAgent.description || ''}
                onChange={(val) => setEditAgent({ ...editAgent, description: val })}
                rows={3}
                placeholder="简短描述角色特征..."
              />
            </div>

            {/* 开场白 */}
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">开场白</label>
              <ArcoInput.TextArea
                value={editAgent.openingStatement || ''}
                onChange={(val) => setEditAgent({ ...editAgent, openingStatement: val })}
                rows={3}
                placeholder="角色首次对话的开场白..."
              />
            </div>

            {/* 人格设定 */}
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">人格设定</label>
              <ArcoInput.TextArea
                value={editAgent.personality || ''}
                onChange={(val) => setEditAgent({ ...editAgent, personality: val })}
                rows={4}
                placeholder="角色的性格、背景、说话方式..."
              />
            </div>

            {/* 世界观 */}
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">世界观 / 场景</label>
              <ArcoInput.TextArea
                value={editAgent.scenario || ''}
                onChange={(val) => setEditAgent({ ...editAgent, scenario: val })}
                rows={3}
                placeholder="角色所处的世界观或场景设定..."
              />
            </div>

            {/* 示例对话 */}
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">示例对话</label>
              <ArcoInput.TextArea
                value={editAgent.exampleDialogues || ''}
                onChange={(val) => setEditAgent({ ...editAgent, exampleDialogues: val })}
                rows={4}
                placeholder="角色对话示例，展示对话风格..."
              />
            </div>

            {/* 标签 */}
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">标签（逗号分隔）</label>
              <ArcoInput
                value={editAgent.tags || ''}
                onChange={(val) => setEditAgent({ ...editAgent, tags: val })}
                placeholder="如: 助手, 友善, 知识渊博"
              />
            </div>

            {/* 创作者备注 */}
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">创作者备注</label>
              <ArcoInput.TextArea
                value={editAgent.creatorNote || ''}
                onChange={(val) => setEditAgent({ ...editAgent, creatorNote: val })}
                rows={2}
                placeholder="创作者备注信息..."
              />
            </div>

            {/* 系统提示词 */}
            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="text-xs font-medium text-gray-500 dark:text-gray-400">系统提示词</label>
                <button
                  onClick={openPromptPresetModal}
                  className="flex items-center gap-1 text-xs text-blue-500 hover:text-blue-600 transition-colors"
                >
                  <Sparkles className="w-3 h-3" />
                  从模板加载
                </button>
              </div>
              <ArcoInput.TextArea
                value={editAgent.systemPrompt || ''}
                onChange={(val) => setEditAgent({ ...editAgent, systemPrompt: val })}
                rows={5}
                placeholder="AI 助手的系统提示词..."
              />
            </div>

            {/* 模型参数 */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Temperature</label>
                <InputNumber
                  value={editAgent.temperature}
                  onChange={(val) => setEditAgent({ ...editAgent, temperature: val as number })}
                  min={0}
                  max={2}
                  step={0.1}
                  style={{ width: '100%' }}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">Max Tokens</label>
                <InputNumber
                  value={editAgent.maxTokens}
                  onChange={(val) => setEditAgent({ ...editAgent, maxTokens: val as number })}
                  min={1}
                  max={32768}
                  style={{ width: '100%' }}
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">音色</label>
                <ArcoInput
                  value={editAgent.speaker || ''}
                  onChange={(val) => setEditAgent({ ...editAgent, speaker: val })}
                  placeholder="音色编码"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">LLM 模型</label>
                <ArcoInput
                  value={editAgent.llmModel || ''}
                  onChange={(val) => setEditAgent({ ...editAgent, llmModel: val })}
                  placeholder="模型名称"
                />
              </div>
            </div>

            <div>
              <label className="flex items-center gap-2 cursor-pointer">
                <Switch
                  checked={editAgent.enableJSONOutput || false}
                  onChange={(val) => setEditAgent({ ...editAgent, enableJSONOutput: val })}
                  size="small"
                />
                <span className="text-xs font-medium text-gray-500 dark:text-gray-400">启用 JSON 输出</span>
              </label>
            </div>
          </div>
        )}
      </Drawer>

      {/* ========== 导入 Modal ========== */}
      <Modal
        title={<span className="text-lg font-semibold">导入角色卡</span>}
        visible={importVisible}
        onCancel={() => { setImportVisible(false); setImportFile(null) }}
        onOk={handleImport}
        confirmLoading={importLoading}
        okText="导入"
        cancelText="取消"
      >
        <div className="space-y-4 py-2">
          <p className="text-sm text-gray-500 dark:text-gray-400">
            支持 SillyTavern Character Card 格式（JSON / PNG / YAML）
          </p>
          <Upload
            drag
            accept=".json,.png,.yaml,.yml"
            limit={1}
            onChange={(_files, file) => {
              if (file.originFile) setImportFile(file.originFile)
            }}
            onRemove={() => setImportFile(null)}
          />
        </div>
      </Modal>

      {/* ========== 提示词模板选择 Modal ========== */}
      <Modal
        title="选择提示词模板"
        visible={promptPresetVisible}
        onCancel={() => setPromptPresetVisible(false)}
        footer={null}
        style={{ width: 560 }}
      >
        {promptPresetLoading ? (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-500" />
          </div>
        ) : promptPresets.length === 0 ? (
          <div className="text-center py-12 text-gray-400">
            <Sparkles className="w-10 h-10 mx-auto mb-2 opacity-30" />
            <p className="text-sm">暂无可用提示词模板</p>
            <p className="text-xs mt-1">前往「预设模板」页面创建</p>
          </div>
        ) : (
          <div className="space-y-2 max-h-96 overflow-y-auto py-2">
            {promptPresets.map((preset) => {
              let preview = ''
              try {
                const obj = JSON.parse(preset.content)
                preview = (obj.systemPrompt || '').slice(0, 100)
              } catch {}
              return (
                <div
                  key={preset.id}
                  onClick={() => handleApplyPromptPreset(preset)}
                  className="flex items-start gap-3 p-3 rounded-lg border border-gray-200 dark:border-neutral-700 hover:border-blue-300 dark:hover:border-blue-600 hover:bg-blue-50/50 dark:hover:bg-blue-900/10 cursor-pointer transition-colors"
                >
                  <div className="w-8 h-8 rounded-lg bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center shrink-0 mt-0.5">
                    <Sparkles className="w-4 h-4 text-blue-500" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{preset.name}</span>
                      {preset.isBuiltin && <Tag size="small" color="orange">内置</Tag>}
                    </div>
                    {preset.description && (
                      <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5 line-clamp-1">{preset.description}</p>
                    )}
                    <p className="text-xs text-gray-400 dark:text-gray-500 mt-1 font-mono line-clamp-2">
                      {preview || '(空模板)'}
                    </p>
                  </div>
                  <span className="text-[10px] text-gray-400 shrink-0">使用 {preset.useCount} 次</span>
                </div>
              )
            })}
          </div>
        )}
      </Modal>
    </div>
  )
}

export default Assistants
