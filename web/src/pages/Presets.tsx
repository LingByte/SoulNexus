import React, { useEffect, useState, useMemo, useCallback } from 'react'
import {
  listPresets,
  getPreset,
  createPreset,
  updatePreset,
  deletePreset,
  applyPreset,
  getAssistantList,
  type PresetTemplate,
  type PresetType,
  type AssistantListItem,
  type SystemPromptPresetPayload,
  type PresetVariable,
} from '@/api/assistant'
import { showAlert } from '@/utils/notification'
import { useAuthStore } from '@/stores/authStore'
import {
  Wand2, Plus, Eye, Edit3, Trash2, Zap, Search,
  Layers, Bot, Mic, BookOpen, Sparkles, X,
  Play, Download, Upload,
} from 'lucide-react'
import Button from '@/components/UI/Button'
import PageHeader from '@/components/Layout/PageHeader'
import { motion, AnimatePresence } from 'framer-motion'
import { Input as ArcoInput, Drawer, Select, Tag, Popconfirm } from '@arco-design/web-react'
import { cn } from '@/utils/cn'

// ==================== 常量 ====================

const TYPE_CONFIG: Record<PresetType, { label: string; icon: React.FC<any>; color: string; desc: string }> = {
  agent: { label: '完整 Agent', icon: Bot, color: '#7c3aed', desc: '含提示词+语音+模型参数的完整预设' },
  system_prompt: { label: '提示词模板', icon: Sparkles, color: '#2563eb', desc: '可替换变量的系统提示词模板' },
  voice: { label: '语音配置', icon: Mic, color: '#059669', desc: 'TTS / VAD / 音色预设' },
  knowledge: { label: '知识库配置', icon: BookOpen, color: '#d97706', desc: '向量库 / Embedding / 文档处理预设' },
}

const PAGE_SIZE = 20

// ==================== 预设模板管理页面 ====================

const Presets: React.FC = () => {
  const { isAuthenticated } = useAuthStore()

  // 列表状态
  const [presets, setPresets] = useState<PresetTemplate[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [keyword, setKeyword] = useState('')
  const [typeFilter, setTypeFilter] = useState<PresetType | ''>('')
  const [totalPage, setTotalPage] = useState(0)

  // 详情抽屉
  const [detailPreset, setDetailPreset] = useState<PresetTemplate | null>(null)
  const [detailVisible, setDetailVisible] = useState(false)

  // 编辑/创建 Modal
  const [editVisible, setEditVisible] = useState(false)
  const [editingPreset, setEditingPreset] = useState<Partial<PresetTemplate> | null>(null)
  const [editName, setEditName] = useState('')
  const [editDesc, setEditDesc] = useState('')
  const [editType, setEditType] = useState<PresetType>('system_prompt')
  const [editCategory, setEditCategory] = useState('')
  const [editTags, setEditTags] = useState('')
  const [editVisibility, setEditVisibility] = useState('private')
  const [editContent, setEditContent] = useState('')
  const [editSaving, setEditSaving] = useState(false)

  // 应用 Modal（提示词模板 → Agent）
  const [applyVisible, setApplyVisible] = useState(false)
  const [applyPreset, setApplyPreset] = useState<PresetTemplate | null>(null)
  const [applyAgentId, setApplyAgentId] = useState<number | undefined>(undefined)
  const [applyVariables, setApplyVariables] = useState<Record<string, string>>({})
  const [agents, setAgents] = useState<AssistantListItem[]>([])
  const [applyLoading, setApplyLoading] = useState(false)

  // ==================== 数据加载 ====================

  const fetchPresets = useCallback(async () => {
    setLoading(true)
    try {
      const params: any = { page, pageSize: PAGE_SIZE }
      if (keyword.trim()) params.keyword = keyword.trim()
      if (typeFilter) params.type = typeFilter
      const res = await listPresets(params)
      if (res.code === 200 && res.data) {
        setPresets(res.data.list)
        setTotal(res.data.total)
        setTotalPage(res.data.totalPage)
      }
    } catch (err: any) {
      showAlert(err?.msg || '加载模板失败', 'error')
    } finally {
      setLoading(false)
    }
  }, [page, keyword, typeFilter])

  useEffect(() => {
    fetchPresets()
  }, [fetchPresets])

  // ==================== 详情 ====================

  const handleViewDetail = async (preset: PresetTemplate) => {
    try {
      const res = await getPreset(Number(preset.id))
      if (res.code === 200 && res.data) {
        setDetailPreset(res.data)
        setDetailVisible(true)
      }
    } catch (err: any) {
      showAlert(err?.msg || '获取详情失败', 'error')
    }
  }

  // ==================== 编辑/创建 ====================

  const openCreate = () => {
    setEditingPreset(null)
    setEditName('')
    setEditDesc('')
    setEditType('system_prompt')
    setEditCategory('')
    setEditTags('')
    setEditVisibility('private')
    setEditContent('{}')
    setEditVisible(true)
  }

  const openEdit = async (preset: PresetTemplate) => {
    if (preset.isBuiltin) {
      showAlert('内置模板不可编辑', 'warning')
      return
    }
    setEditingPreset(preset)
    setEditName(preset.name)
    setEditDesc(preset.description || '')
    setEditType(preset.type)
    setEditCategory(preset.category || '')
    setEditTags(preset.tags || '')
    setEditVisibility(preset.visibility)
    setEditContent(preset.content)
    setEditVisible(true)
  }

  const handleSave = async () => {
    if (!editName.trim()) { showAlert('请输入模板名称', 'warning'); return }
    if (!editContent.trim()) { showAlert('请输入模板内容', 'warning'); return }
    try {
      JSON.parse(editContent)
    } catch {
      showAlert('模板内容必须是有效的 JSON', 'warning')
      return
    }
    setEditSaving(true)
    try {
      if (editingPreset) {
        await updatePreset(Number(editingPreset.id), {
          name: editName,
          description: editDesc,
          category: editCategory,
          tags: editTags,
          visibility: editVisibility,
          content: editContent,
        })
        showAlert('更新成功', 'success')
      } else {
        await createPreset({
          name: editName,
          description: editDesc,
          type: editType,
          category: editCategory,
          tags: editTags,
          visibility: editVisibility,
          content: editContent,
        })
        showAlert('创建成功', 'success')
      }
      setEditVisible(false)
      fetchPresets()
    } catch (err: any) {
      showAlert(err?.msg || '保存失败', 'error')
    } finally {
      setEditSaving(false)
    }
  }

  const handleDelete = async (preset: PresetTemplate) => {
    try {
      await deletePreset(Number(preset.id))
      showAlert('已归档', 'success')
      fetchPresets()
    } catch (err: any) {
      showAlert(err?.msg || '删除失败', 'error')
    }
  }

  // ==================== 应用模板到 Agent ====================

  const openApply = async (preset: PresetTemplate) => {
    setApplyPreset(preset)
    setApplyAgentId(undefined)
    setApplyVariables({})
    setApplyLoading(false)

    // 加载 Agent 列表（仅 system_prompt / voice 类型需要）
    if (preset.type === 'system_prompt' || preset.type === 'voice') {
      try {
        const res = await getAssistantList()
        if (res.code === 200 && res.data) {
          setAgents(res.data)
        }
      } catch {}
    }

    // 如果是 system_prompt 类型，预填变量默认值
    if (preset.type === 'system_prompt') {
      try {
        const payload: SystemPromptPresetPayload = JSON.parse(preset.content)
        const vars: Record<string, string> = {}
        if (payload.variables) {
          for (const v of payload.variables) {
            vars[v.name] = v.defaultVal || ''
          }
        }
        setApplyVariables(vars)
      } catch {}
    }

    setApplyVisible(true)
  }

  const handleApply = async () => {
    if (!applyPreset) return
    setApplyLoading(true)
    try {
      const data: any = { presetId: Number(applyPreset.id) }
      if (Object.keys(applyVariables).length > 0) {
        data.variables = applyVariables
      }
      if (applyAgentId) {
        data.agentId = applyAgentId
      }
      const res = await applyPreset(Number(applyPreset.id), data)
      if (res.code === 200) {
        showAlert('应用成功', 'success')
        setApplyVisible(false)
        fetchPresets()
      }
    } catch (err: any) {
      showAlert(err?.msg || '应用失败', 'error')
    } finally {
      setApplyLoading(false)
    }
  }

  // ==================== 搜索 ====================

  const handleSearch = () => {
    setPage(1)
    fetchPresets()
  }

  // ==================== 格式化 ====================

  const fmtDate = (iso?: string) => (iso ? iso.slice(0, 16).replace('T', ' ') : '')

  const renderContentPreview = (preset: PresetTemplate) => {
    try {
      const obj = JSON.parse(preset.content)
      if (preset.type === 'system_prompt') {
        return (obj.systemPrompt || '').slice(0, 120) + ((obj.systemPrompt || '').length > 120 ? '…' : '')
      }
      if (preset.type === 'agent') {
        return `"${obj.name || '未命名'}" — ${(obj.systemPrompt || '').slice(0, 80)}…`
      }
      return JSON.stringify(obj).slice(0, 120) + '…'
    } catch {
      return preset.content.slice(0, 120) + '…'
    }
  }

  const parseVariables = (content: string): PresetVariable[] => {
    try {
      const obj = JSON.parse(content)
      return obj.variables || []
    } catch {
      return []
    }
  }

  const visibilityLabel: Record<string, string> = {
    private: '私有',
    group: '组织',
    public: '公开',
  }

  const visibilityColor: Record<string, string> = {
    private: 'gray',
    group: 'blue',
    public: 'green',
  }

  // ==================== 渲染 ====================

  return (
    <div className="h-full flex flex-col">
      <PageHeader
        title="预设模板"
        description="使用内置或自定义模板，快速创建和配置 AI Agent"
        actions={
          isAuthenticated ? (
            <Button variant="primary" onClick={openCreate} leftIcon={<Plus className="w-4 h-4" />}>
              创建模板
            </Button>
          ) : null
        }
      />

      {/* 搜索和过滤栏 */}
      <div className="flex items-center gap-3 px-4 pb-4 flex-wrap">
        <ArcoInput
          value={keyword}
          onChange={setKeyword}
          onPressEnter={handleSearch}
          placeholder="搜索模板名称、描述…"
          prefix={<Search className="w-4 h-4 text-gray-400" />}
          style={{ width: 260 }}
          allowClear
        />
        <Select
          value={typeFilter || undefined}
          onChange={(val) => { setTypeFilter((val as PresetType) || ''); setPage(1) }}
          placeholder="全部类型"
          style={{ width: 140 }}
          allowClear
          options={Object.entries(TYPE_CONFIG).map(([k, v]) => ({ label: v.label, value: k }))}
        />
        <Button variant="secondary" size="sm" onClick={handleSearch}>
          搜索
        </Button>
        <span className="text-xs text-gray-400 ml-auto">共 {total} 个模板</span>
      </div>

      {/* 卡片列表 */}
      <div className="flex-1 overflow-y-auto px-4 pb-4">
        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500" />
          </div>
        ) : presets.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-gray-400">
            <Layers className="w-12 h-12 mb-3 opacity-30" />
            <p className="text-sm">暂无模板</p>
            <p className="text-xs mt-1">{keyword ? '尝试更换搜索词' : '点击"创建模板"开始'}</p>
          </div>
        ) : (
          <>
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {presets.map((preset) => {
                const tcfg = TYPE_CONFIG[preset.type]
                const Icon = tcfg.icon
                const vars = parseVariables(preset.content)
                return (
                  <motion.div
                    key={preset.id}
                    initial={{ opacity: 0, y: 8 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.25 }}
                    className={cn(
                      'bg-white dark:bg-neutral-800 rounded-xl border border-gray-200 dark:border-neutral-700',
                      'hover:shadow-md transition-shadow p-4 flex flex-col gap-3',
                    )}
                  >
                    {/* 头部：类型标签 + 可见性 */}
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <div className="w-7 h-7 rounded-lg flex items-center justify-center" style={{ backgroundColor: tcfg.color + '18' }}>
                          <Icon className="w-4 h-4" style={{ color: tcfg.color }} />
                        </div>
                        <span className="text-xs font-medium" style={{ color: tcfg.color }}>{tcfg.label}</span>
                        {preset.isBuiltin && (
                          <Tag size="small" color="orange">内置</Tag>
                        )}
                      </div>
                      <div className="flex items-center gap-1">
                        <Tag size="small" color={visibilityColor[preset.visibility] as any || 'gray'}>
                          {visibilityLabel[preset.visibility] || preset.visibility}
                        </Tag>
                      </div>
                    </div>

                    {/* 标题 */}
                    <h3 className="font-semibold text-sm text-gray-900 dark:text-gray-100 line-clamp-1">
                      {preset.name}
                    </h3>

                    {/* 描述 */}
                    {preset.description && (
                      <p className="text-xs text-gray-500 dark:text-gray-400 line-clamp-2">
                        {preset.description}
                      </p>
                    )}

                    {/* 内容预览 */}
                    <p className="text-xs text-gray-400 dark:text-gray-500 bg-gray-50 dark:bg-neutral-700/50 rounded px-2 py-1.5 line-clamp-2 font-mono">
                      {renderContentPreview(preset)}
                    </p>

                    {/* 变量标签（仅提示词模板） */}
                    {preset.type === 'system_prompt' && vars.length > 0 && (
                      <div className="flex items-center gap-1 flex-wrap">
                        <span className="text-[10px] text-gray-400">变量:</span>
                        {vars.map((v) => (
                          <Tag key={v.name} size="small" color="arcoblue">{`{{${v.name}}}`}</Tag>
                        ))}
                      </div>
                    )}

                    {/* 底部信息 + 操作 */}
                    <div className="flex items-center justify-between mt-auto pt-2 border-t border-gray-100 dark:border-neutral-700">
                      <div className="flex items-center gap-3 text-[10px] text-gray-400">
                        <span>使用 {preset.useCount} 次</span>
                        <span>{fmtDate(preset.createdAt)}</span>
                      </div>
                      <div className="flex items-center gap-0.5">
                        <button
                          onClick={() => handleViewDetail(preset)}
                          className="p-1.5 rounded hover:bg-gray-100 dark:hover:bg-neutral-700 text-gray-400 hover:text-gray-600"
                          title="查看详情"
                        >
                          <Eye className="w-3.5 h-3.5" />
                        </button>
                        {isAuthenticated && (
                          <button
                            onClick={() => openApply(preset)}
                            className="p-1.5 rounded hover:bg-green-50 dark:hover:bg-green-900/20 text-green-500"
                            title="应用模板"
                          >
                            <Play className="w-3.5 h-3.5" />
                          </button>
                        )}
                        {isAuthenticated && !preset.isBuiltin && (
                          <>
                            <button
                              onClick={() => openEdit(preset)}
                              className="p-1.5 rounded hover:bg-gray-100 dark:hover:bg-neutral-700 text-gray-400 hover:text-gray-600"
                              title="编辑"
                            >
                              <Edit3 className="w-3.5 h-3.5" />
                            </button>
                            <Popconfirm
                              title="确定归档此模板？"
                              onOk={() => handleDelete(preset)}
                            >
                              <button
                                className="p-1.5 rounded hover:bg-red-50 dark:hover:bg-red-900/20 text-gray-400 hover:text-red-500"
                                title="归档"
                              >
                                <Trash2 className="w-3.5 h-3.5" />
                              </button>
                            </Popconfirm>
                          </>
                        )}
                      </div>
                    </div>
                  </motion.div>
                )
              })}
            </div>

            {/* 分页 */}
            {totalPage > 1 && (
              <div className="flex items-center justify-center gap-2 mt-6">
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  上一页
                </Button>
                <span className="text-xs text-gray-500">
                  {page} / {totalPage}
                </span>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={page >= totalPage}
                  onClick={() => setPage((p) => p + 1)}
                >
                  下一页
                </Button>
              </div>
            )}
          </>
        )}
      </div>

      {/* ========== 详情抽屉 ========== */}
      <Drawer
        title={detailPreset?.name || '模板详情'}
        visible={detailVisible}
        onCancel={() => setDetailVisible(false)}
        width={480}
        footer={null}
      >
        {detailPreset && (
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <Tag color={visibilityColor[detailPreset.visibility] as any}>
                {visibilityLabel[detailPreset.visibility]}
              </Tag>
              {detailPreset.isBuiltin && <Tag color="orange">内置</Tag>}
              <Tag color="arcoblue">{TYPE_CONFIG[detailPreset.type]?.label}</Tag>
            </div>
            {detailPreset.description && (
              <div>
                <h4 className="text-xs font-medium text-gray-400 mb-1">描述</h4>
                <p className="text-sm text-gray-700 dark:text-gray-300">{detailPreset.description}</p>
              </div>
            )}
            {detailPreset.category && (
              <div>
                <h4 className="text-xs font-medium text-gray-400 mb-1">分类</h4>
                <p className="text-sm">{detailPreset.category}</p>
              </div>
            )}
            {detailPreset.tags && (
              <div>
                <h4 className="text-xs font-medium text-gray-400 mb-1">标签</h4>
                <div className="flex gap-1 flex-wrap">
                  {detailPreset.tags.split(',').filter(Boolean).map((t) => (
                    <Tag key={t} size="small">{t.trim()}</Tag>
                  ))}
                </div>
              </div>
            )}
            <div>
              <h4 className="text-xs font-medium text-gray-400 mb-1">模板内容 (JSON)</h4>
              <pre className="bg-gray-50 dark:bg-neutral-800 rounded-lg p-3 text-xs overflow-auto max-h-80 whitespace-pre-wrap font-mono">
                {(() => {
                  try { return JSON.stringify(JSON.parse(detailPreset.content), null, 2) }
                  catch { return detailPreset.content }
                })()}
              </pre>
            </div>
            <div className="flex items-center gap-4 text-xs text-gray-400">
              <span>使用 {detailPreset.useCount} 次</span>
              <span>创建: {fmtDate(detailPreset.createdAt)}</span>
              <span>更新: {fmtDate(detailPreset.updatedAt)}</span>
            </div>
            {isAuthenticated && (
              <div className="flex gap-2 pt-2 border-t border-gray-100 dark:border-neutral-700">
                <Button variant="primary" size="sm" onClick={() => { setDetailVisible(false); openApply(detailPreset) }} leftIcon={<Play className="w-3.5 h-3.5" />}>
                  应用此模板
                </Button>
                {!detailPreset.isBuiltin && (
                  <Button variant="secondary" size="sm" onClick={() => { setDetailVisible(false); openEdit(detailPreset) }} leftIcon={<Edit3 className="w-3.5 h-3.5" />}>
                    编辑
                  </Button>
                )}
              </div>
            )}
          </div>
        )}
      </Drawer>

      {/* ========== 编辑/创建 Drawer ========== */}
      <Drawer
        title={editingPreset ? '编辑模板' : '创建模板'}
        visible={editVisible}
        onCancel={() => setEditVisible(false)}
        width={520}
        footer={
          <div className="flex justify-end gap-3">
            <Button variant="secondary" onClick={() => setEditVisible(false)}>
              取消
            </Button>
            <Button variant="primary" onClick={handleSave} loading={editSaving}>
              保存
            </Button>
          </div>
        }
      >
        <div className="space-y-5 pr-2">
          <div>
            <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">模板名称 *</label>
            <ArcoInput value={editName} onChange={setEditName} placeholder="如: 通用客服助手" />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">描述</label>
            <ArcoInput.TextArea value={editDesc} onChange={setEditDesc} rows={3} placeholder="模板的用途和使用说明…" />
          </div>
          {!editingPreset && (
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">类型 *</label>
              <Select
                value={editType}
                onChange={(val) => setEditType(val as PresetType)}
                style={{ width: '100%' }}
                options={Object.entries(TYPE_CONFIG).map(([k, v]) => ({
                  label: `${v.label} — ${v.desc}`,
                  value: k,
                }))}
              />
            </div>
          )}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">分类</label>
              <ArcoInput value={editCategory} onChange={setEditCategory} placeholder="如: 客服" />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">标签（逗号分隔）</label>
              <ArcoInput value={editTags} onChange={setEditTags} placeholder="如: 通用, 中文" />
            </div>
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">可见性</label>
            <Select
              value={editVisibility}
              onChange={setEditVisibility}
              style={{ width: '100%' }}
              options={[
                { label: '私有 — 仅自己可见', value: 'private' },
                { label: '组织 — 同组织成员可见', value: 'group' },
                { label: '公开 — 所有人可见', value: 'public' },
              ]}
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1.5">模板内容 (JSON) *</label>
            <ArcoInput.TextArea
              value={editContent}
              onChange={setEditContent}
              rows={12}
              placeholder='{"systemPrompt": "你是一个...", "variables": [...]}'
              style={{ fontFamily: 'monospace', fontSize: 12 }}
            />
          </div>
        </div>
      </Drawer>

      {/* ========== 应用模板 Drawer ========== */}
      <Drawer
        title={`应用模板: ${applyPreset?.name || ''}`}
        visible={applyVisible}
        onCancel={() => setApplyVisible(false)}
        width={480}
        footer={
          <div className="flex justify-end gap-3">
            <Button variant="secondary" onClick={() => setApplyVisible(false)}>
              取消
            </Button>
            <Button variant="primary" onClick={handleApply} loading={applyLoading}>
              应用
            </Button>
          </div>
        }
      >
        {applyPreset && (
          <div className="space-y-4 pr-2">
            <p className="text-xs text-gray-500">
              类型: {TYPE_CONFIG[applyPreset.type]?.label} — {applyPreset.description}
            </p>

            {/* Agent 类型：提示创建新 Agent */}
            {applyPreset.type === 'agent' && (
              <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-3 text-xs text-blue-700 dark:text-blue-300">
                此模板将创建一个新的 Agent，包含完整的提示词、语音和模型配置。
              </div>
            )}

            {/* system_prompt / voice 类型：选择目标 Agent */}
            {(applyPreset.type === 'system_prompt' || applyPreset.type === 'voice') && (
              <div>
                <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
                  选择目标 Agent（可选）
                </label>
                <Select
                  value={applyAgentId}
                  onChange={(val) => setApplyAgentId(val as number)}
                  style={{ width: '100%' }}
                  allowClear
                  placeholder="不选则仅生成内容"
                  options={agents.map((a) => ({ label: `#${a.id} ${a.name}`, value: a.id }))}
                />
                {!applyAgentId && (
                  <p className="text-[10px] text-gray-400 mt-1">不选择 Agent 时，仅返回模板内容供复制使用</p>
                )}
              </div>
            )}

            {/* system_prompt 类型：变量填写 */}
            {applyPreset.type === 'system_prompt' && (() => {
              const vars = parseVariables(applyPreset.content)
              return vars.length > 0 ? (
                <div>
                  <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">
                    模板变量 ({vars.length} 个)
                  </label>
                  <div className="space-y-3">
                    {vars.map((v) => (
                      <div key={v.name}>
                        <div className="flex items-center gap-1 mb-1">
                          <code className="text-xs bg-gray-100 dark:bg-neutral-700 px-1.5 py-0.5 rounded font-mono">{`{{${v.name}}}`}</code>
                          {v.required && <span className="text-[10px] text-red-500">*必填</span>}
                          {v.description && <span className="text-[10px] text-gray-400">— {v.description}</span>}
                        </div>
                        <ArcoInput
                          value={applyVariables[v.name] || ''}
                          onChange={(val) => setApplyVariables((prev) => ({ ...prev, [v.name]: val }))}
                          placeholder={v.defaultVal || v.label}
                          size="small"
                        />
                      </div>
                    ))}
                  </div>
                </div>
              ) : (
                <div className="bg-gray-50 dark:bg-neutral-700/50 rounded-lg p-3 text-xs text-gray-500">
                  此提示词模板没有变量，将直接应用原始内容。
                </div>
              )
            })()}

            {/* knowledge 类型 */}
            {applyPreset.type === 'knowledge' && (
              <div className="bg-amber-50 dark:bg-amber-900/20 rounded-lg p-3 text-xs text-amber-700 dark:text-amber-300">
                此模板将创建一个新的知识库配置，包含向量库和 Embedding 模型设置。
              </div>
            )}

            {/* voice 类型 */}
            {applyPreset.type === 'voice' && (
              <div className="bg-green-50 dark:bg-green-900/20 rounded-lg p-3 text-xs text-green-700 dark:text-green-300">
                此模板将应用语音配置（TTS Provider、VAD 参数）到选定的 Agent。
              </div>
            )}
          </div>
        )}
      </Drawer>
    </div>
  )
}

export default Presets
