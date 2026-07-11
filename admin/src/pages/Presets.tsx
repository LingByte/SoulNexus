// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Table,
  Button,
  Drawer,
  Form,
  Input,
  Select,
  Message,
  Space,
  Tag,
  Modal,
  Tabs,
  InputNumber,
  Popconfirm,
  Switch,
  Textarea,
} from '@arco-design/web-react'
import {
  RefreshCw,
  Plus,
  Eye,
  Trash2,
  Play,
  Copy,
  Edit3,
} from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import { useMediaQuery } from '@/hooks/useMediaQuery'
import {
  listPresetTemplates,
  getPresetTemplate,
  createPresetTemplate,
  updatePresetTemplate,
  deletePresetTemplate,
  applyPresetTemplate,
  listAdminAssistants,
  type PresetTemplateRow,
  type AdminAssistant,
} from '@/services/adminApi'

const FormItem = Form.Item
const TabPane = Tabs.TabPane

// ==================== 类型映射 ====================

const TYPE_MAP: Record<string, { label: string; color: string; icon: string }> = {
  agent: { label: 'Agent 预设', color: 'arcoblue', icon: '🤖' },
  system_prompt: { label: '提示词模板', color: 'purple', icon: '💬' },
  voice: { label: '语音配置', color: 'green', icon: '🎤' },
  knowledge: { label: '知识库预设', color: 'orange', icon: '📚' },
}

const VISIBILITY_MAP: Record<string, { label: string; color: string }> = {
  public: { label: '公开', color: 'green' },
  group: { label: '组织', color: 'arcoblue' },
  private: { label: '私有', color: 'gray' },
}

// ==================== 主组件 ====================

const Presets = () => {
  const isSmallScreen = useMediaQuery('(max-width: 639px)')

  // 列表状态
  const [loading, setLoading] = useState(false)
  const [rows, setRows] = useState<PresetTemplateRow[]>([])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [typeFilter, setTypeFilter] = useState('')
  const [keywordInput, setKeywordInput] = useState('')
  const [keyword, setKeyword] = useState('')

  // 详情抽屉
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailRow, setDetailRow] = useState<PresetTemplateRow | null>(null)

  // 创建/编辑抽屉
  const [formOpen, setFormOpen] = useState(false)
  const [formMode, setFormMode] = useState<'create' | 'edit'>('create')
  const [formSubmitting, setFormSubmitting] = useState(false)
  const [form] = Form.useForm()

  // 应用模态框
  const [applyOpen, setApplyOpen] = useState(false)
  const [applyRow, setApplyRow] = useState<PresetTemplateRow | null>(null)
  const [applySubmitting, setApplySubmitting] = useState(false)
  const [applyAgentId, setApplyAgentId] = useState<number | undefined>()
  const [applyVariables, setApplyVariables] = useState<Record<string, string>>({})
  const [agents, setAgents] = useState<AdminAssistant[]>([])

  // ==================== 数据加载 ====================

  const loadList = useCallback(async () => {
    setLoading(true)
    try {
      const out = await listPresetTemplates({
        page,
        pageSize,
        type: typeFilter || undefined,
        keyword: keyword.trim() || undefined,
      })
      setRows(out.list || [])
      setTotal(out.total || 0)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      Message.error(`加载模板失败：${msg}`)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, typeFilter, keyword])

  useEffect(() => {
    void loadList()
  }, [loadList])

  // ==================== 详情 ====================

  const openDetail = async (row: PresetTemplateRow) => {
    try {
      const detail = await getPresetTemplate(row.id)
      setDetailRow(detail)
      setDetailOpen(true)
    } catch {
      // fallback: 使用列表数据
      setDetailRow(row)
      setDetailOpen(true)
    }
  }

  // 格式化 JSON 内容
  const formatContent = (content: string): string => {
    try {
      return JSON.stringify(JSON.parse(content), null, 2)
    } catch {
      return content
    }
  }

  // 复制内容到剪贴板
  const copyContent = (content: string) => {
    navigator.clipboard.writeText(formatContent(content)).then(
      () => Message.success('已复制到剪贴板'),
      () => Message.error('复制失败'),
    )
  }

  // ==================== 创建 / 编辑 ====================

  const openCreate = () => {
    setFormMode('create')
    form.resetFields()
    form.setFieldsValue({
      type: 'system_prompt',
      visibility: 'private',
    })
    setFormOpen(true)
  }

  const openEdit = (row: PresetTemplateRow) => {
    if (row.isBuiltin) {
      Message.warning('内置模板不可编辑')
      return
    }
    setFormMode('edit')
    form.resetFields()
    form.setFieldsValue({
      id: row.id,
      name: row.name,
      description: row.description || '',
      type: row.type,
      category: row.category || '',
      tags: row.tags || '',
      visibility: row.visibility,
      content: formatContent(row.content),
    })
    setFormOpen(true)
  }

  const submitForm = async () => {
    try {
      const v = await form.validate()
      setFormSubmitting(true)

      // 验证 content 是合法 JSON
      let contentStr = String(v.content || '').trim()
      try {
        JSON.parse(contentStr)
      } catch {
        Message.error('模板内容必须是有效的 JSON 格式')
        setFormSubmitting(false)
        return
      }

      if (formMode === 'create') {
        await createPresetTemplate({
          name: String(v.name || '').trim(),
          description: v.description ? String(v.description).trim() : undefined,
          type: String(v.type || '').trim(),
          category: v.category ? String(v.category).trim() : undefined,
          tags: v.tags ? String(v.tags).trim() : undefined,
          visibility: v.visibility || 'private',
          content: contentStr,
        })
        Message.success('创建成功')
      } else {
        await updatePresetTemplate(v.id, {
          name: String(v.name || '').trim(),
          description: v.description ? String(v.description).trim() : undefined,
          category: v.category ? String(v.category).trim() : undefined,
          tags: v.tags ? String(v.tags).trim() : undefined,
          visibility: v.visibility || undefined,
          content: contentStr,
        })
        Message.success('更新成功')
      }

      setFormOpen(false)
      void loadList()
    } catch (e: unknown) {
      if ((e as { errors?: unknown })?.errors) return // 表单校验错误
      const msg = e instanceof Error ? e.message : String(e)
      Message.error(msg)
    } finally {
      setFormSubmitting(false)
    }
  }

  // ==================== 删除 ====================

  const handleDelete = async (row: PresetTemplateRow) => {
    if (row.isBuiltin) {
      Message.warning('内置模板不可删除')
      return
    }
    try {
      await deletePresetTemplate(row.id)
      Message.success('删除成功')
      void loadList()
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      Message.error(`删除失败：${msg}`)
    }
  }

  // ==================== 应用模板 ====================

  const openApply = async (row: PresetTemplateRow) => {
    setApplyRow(row)
    setApplyAgentId(undefined)
    setApplyVariables({})

    // 如果是语音/提示词类型，需要加载 Agent 列表供选择
    if (row.type === 'voice' || row.type === 'system_prompt') {
      try {
        const res = await listAdminAssistants({ page: 1, pageSize: 200 })
        setAgents(res.agents || [])
      } catch {
        setAgents([])
      }
    }

    // 如果是提示词类型，尝试解析变量
    if (row.type === 'system_prompt') {
      try {
        const payload = JSON.parse(row.content)
        if (payload.variables && Array.isArray(payload.variables)) {
          const defaults: Record<string, string> = {}
          payload.variables.forEach((v: any) => {
            defaults[v.name] = v.defaultVal || ''
          })
          setApplyVariables(defaults)
        }
      } catch {
        setApplyVariables({})
      }
    }

    setApplyOpen(true)
  }

  const submitApply = async () => {
    if (!applyRow) return
    setApplySubmitting(true)
    try {
      const body: { variables?: Record<string, string>; agentId?: number } = {}

      if (applyRow.type === 'voice' || applyRow.type === 'system_prompt') {
        if (applyAgentId) {
          body.agentId = applyAgentId
        }
      }

      if (applyRow.type === 'system_prompt' && Object.keys(applyVariables).length > 0) {
        body.variables = applyVariables
      }

      const result = await applyPresetTemplate(applyRow.id, Object.keys(body).length > 0 ? body : undefined)
      Message.success('模板应用成功！')

      // 显示结果详情
      let resultMsg = ''
      if (result.agent) {
        resultMsg = `已创建 Agent: ${(result.agent as any)?.name || ''}`
      } else if (result.agentId) {
        resultMsg = `已更新 Agent #${result.agentId}`
      } else if (result.systemPrompt) {
        resultMsg = '提示词已应用'
      } else if (result.voice) {
        resultMsg = '语音配置已应用'
      } else {
        resultMsg = '操作完成'
      }

      Modal.info({
        title: '应用成功',
        content: resultMsg,
      })

      setApplyOpen(false)
      void loadList()
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      Message.error(`应用失败：${msg}`)
    } finally {
      setApplySubmitting(false)
    }
  }

  // ==================== 表格列 ====================

  const columns = useMemo(
    () => [
      {
        title: 'ID',
        dataIndex: 'id',
        width: 80,
        sorter: (a: PresetTemplateRow, b: PresetTemplateRow) => Number(a.id) - Number(b.id),
      },
      {
        title: '类型',
        dataIndex: 'type',
        width: 120,
        render: (v: string) => {
          const t = TYPE_MAP[v] || { label: v, color: 'gray' }
          return <Tag color={t.color}>{t.label}</Tag>
        },
      },
      {
        title: '名称',
        dataIndex: 'name',
        ellipsis: true,
        render: (_: unknown, r: PresetTemplateRow) => (
          <span className="cursor-pointer text-[rgb(var(--primary-6))] hover:underline" onClick={() => openDetail(r)}>
            {r.isBuiltin && <Tag color="red" size="small" style={{ marginRight: 6 }}>内置</Tag>}
            {r.name}
          </span>
        ),
      },
      {
        title: '分类',
        dataIndex: 'category',
        width: 80,
        ellipsis: true,
      },
      {
        title: '可见性',
        dataIndex: 'visibility',
        width: 80,
        render: (v: string) => {
          const vm = VISIBILITY_MAP[v] || { label: v, color: 'gray' }
          return <Tag color={vm.color}>{vm.label}</Tag>
        },
      },
      {
        title: '使用次数',
        dataIndex: 'useCount',
        width: 80,
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 80,
        render: (v: string) => {
          if (v === 'active') return <Tag color="green">active</Tag>
          if (v === 'archived') return <Tag color="gray">archived</Tag>
          return <Tag>{v}</Tag>
        },
      },
      {
        title: '操作',
        key: 'actions',
        width: 200,
        fixed: 'right' as const,
        render: (_: unknown, r: PresetTemplateRow) => (
          <Space size="mini">
            <Button size="mini" type="text" onClick={() => openDetail(r)}>
              <span className="inline-flex items-center gap-1"><Eye size={14} /> 详情</span>
            </Button>
            <Button size="mini" type="text" onClick={() => openApply(r)}>
              <span className="inline-flex items-center gap-1"><Play size={14} /> 应用</span>
            </Button>
            {!r.isBuiltin && (
              <>
                <Button size="mini" type="text" onClick={() => openEdit(r)}>
                  <span className="inline-flex items-center gap-1"><Edit3 size={14} /> 编辑</span>
                </Button>
                <Popconfirm
                  title="确认删除该模板？"
                  okText="删除"
                  cancelText="取消"
                  onOk={() => handleDelete(r)}
                >
                  <Button size="mini" type="text" status="danger">
                    <Trash2 size={14} />
                  </Button>
                </Popconfirm>
              </>
            )}
          </Space>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rows],
  )

  // ==================== 渲染 ====================

  const typeTabs = [
    { key: '', label: '全部' },
    { key: 'agent', label: '🤖 Agent' },
    { key: 'system_prompt', label: '💬 提示词' },
    { key: 'voice', label: '🎤 语音' },
    { key: 'knowledge', label: '📚 知识库' },
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title="预设模板"
        description="管理 Agent / 提示词 / 语音 / 知识库的预设模板，快速复用配置。"
        actions={
          <div className="flex w-full max-w-full flex-col gap-2 sm:w-auto sm:flex-row sm:flex-wrap sm:items-center">
            <Input.Search
              allowClear
              placeholder="搜索模板名称 / 描述 / 标签"
              className="w-full min-w-0 sm:w-[240px]"
              value={keywordInput}
              onChange={setKeywordInput}
              onSearch={(v) => {
                setPage(1)
                setKeyword(String(v || '').trim())
              }}
            />
            <Button type="primary" onClick={openCreate}>
              <span className="inline-flex items-center gap-1"><Plus size={16} /> 新建模板</span>
            </Button>
            <Button onClick={() => void loadList()}>
              <span className="inline-flex items-center gap-1">
                <RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新
              </span>
            </Button>
          </div>
        }
      />

      {/* 类型筛选 Tabs */}
      <Tabs
        activeTab={typeFilter}
        onChange={(v) => {
          setPage(1)
          setTypeFilter(v)
        }}
        type="card-gutter"
        size="small"
      >
        {typeTabs.map((t) => (
          <TabPane key={t.key} title={t.label} />
        ))}
      </Tabs>

      {/* 数据表格 */}
      <Table
        rowKey={(r) => String(r.id)}
        loading={loading}
        columns={columns}
        data={rows}
        scroll={{ x: 900 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showTotal: (t: number) => `共 ${t} 条`,
          sizeCanChange: true,
          pageSizeChangeResetCurrent: true,
          onChange: (p: number, ps: number) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />

      {/* ====== 详情抽屉 ====== */}
      <Drawer
        title="模板详情"
        visible={detailOpen}
        placement="right"
        width={isSmallScreen ? '100%' : 600}
        onCancel={() => {
          setDetailOpen(false)
          setDetailRow(null)
        }}
        footer={null}
      >
        {detailRow && (
          <div className="space-y-4">
            <div className="flex items-center gap-2 flex-wrap">
              {detailRow.isBuiltin && <Tag color="red">内置</Tag>}
              <Tag color={TYPE_MAP[detailRow.type]?.color || 'gray'}>
                {TYPE_MAP[detailRow.type]?.label || detailRow.type}
              </Tag>
              <Tag color={VISIBILITY_MAP[detailRow.visibility]?.color || 'gray'}>
                {VISIBILITY_MAP[detailRow.visibility]?.label || detailRow.visibility}
              </Tag>
              {detailRow.status === 'archived' && <Tag color="gray">已归档</Tag>}
            </div>

            <div className="grid grid-cols-2 gap-3 text-sm">
              <div className="rounded-md border border-[var(--color-border-2)] p-3">
                <div className="text-xs text-[var(--color-text-3)] mb-1">名称</div>
                <div className="font-medium text-[var(--color-text-1)]">{detailRow.name}</div>
              </div>
              <div className="rounded-md border border-[var(--color-border-2)] p-3">
                <div className="text-xs text-[var(--color-text-3)] mb-1">分类</div>
                <div className="text-[var(--color-text-1)]">{detailRow.category || '-'}</div>
              </div>
              <div className="rounded-md border border-[var(--color-border-2)] p-3">
                <div className="text-xs text-[var(--color-text-3)] mb-1">标签</div>
                <div className="text-[var(--color-text-1)]">{detailRow.tags || '-'}</div>
              </div>
              <div className="rounded-md border border-[var(--color-border-2)] p-3">
                <div className="text-xs text-[var(--color-text-3)] mb-1">使用次数</div>
                <div className="text-[var(--color-text-1)]">{detailRow.useCount}</div>
              </div>
              {detailRow.description && (
                <div className="col-span-2 rounded-md border border-[var(--color-border-2)] p-3">
                  <div className="text-xs text-[var(--color-text-3)] mb-1">描述</div>
                  <div className="text-[var(--color-text-1)] whitespace-pre-wrap">{detailRow.description}</div>
                </div>
              )}
              <div className="col-span-2 rounded-md border border-[var(--color-border-2)] p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="text-xs text-[var(--color-text-3)]">模板内容 (JSON)</div>
                  <Button size="mini" type="text" onClick={() => copyContent(detailRow.content)}>
                    <span className="inline-flex items-center gap-1"><Copy size={12} /> 复制</span>
                  </Button>
                </div>
                <pre className="text-xs bg-[var(--color-fill-2)] p-3 rounded overflow-auto max-h-96 whitespace-pre-wrap break-all">
                  {formatContent(detailRow.content)}
                </pre>
              </div>
            </div>
          </div>
        )}
      </Drawer>

      {/* ====== 创建/编辑抽屉 ====== */}
      <Drawer
        title={formMode === 'create' ? '新建预设模板' : '编辑预设模板'}
        visible={formOpen}
        placement="right"
        width={isSmallScreen ? '100%' : 520}
        onCancel={() => setFormOpen(false)}
        footer={
          <Space>
            <Button onClick={() => setFormOpen(false)}>取消</Button>
            <Button type="primary" loading={formSubmitting} onClick={() => void submitForm()}>
              {formMode === 'create' ? '创建' : '保存'}
            </Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical">
          <FormItem label="模板名称" field="name" rules={[{ required: true, message: '必填' }]}>
            <Input placeholder="输入模板名称" />
          </FormItem>
          <FormItem label="模板类型" field="type" rules={[{ required: true, message: '必填' }]}>
            <Select
              placeholder="选择模板类型"
              disabled={formMode === 'edit'}
              options={[
                { label: '💬 提示词模板', value: 'system_prompt' },
                { label: '🎤 语音配置', value: 'voice' },
                { label: '📚 知识库预设', value: 'knowledge' },
                { label: '🤖 Agent 预设', value: 'agent' },
              ]}
            />
          </FormItem>
          <FormItem label="分类" field="category">
            <Input placeholder="如：客服、教育、通用" />
          </FormItem>
          <FormItem label="标签" field="tags">
            <Input placeholder="逗号分隔，如：专业,低延迟" />
          </FormItem>
          <FormItem label="可见性" field="visibility">
            <Select
              options={[
                { label: '🔒 私有 - 仅自己可见', value: 'private' },
                { label: '👥 组织 - 同组织可见', value: 'group' },
                { label: '🌐 公开 - 所有人可见', value: 'public' },
              ]}
            />
          </FormItem>
          <FormItem label="描述" field="description">
            <Input.TextArea placeholder="模板使用说明" autoSize={{ minRows: 2, maxRows: 4 }} />
          </FormItem>
          <FormItem
            label="模板内容 (JSON)"
            field="content"
            rules={[
              { required: true, message: '必填' },
              {
                validator: (val: string, cb: (err?: string) => void) => {
                  if (!val || !String(val).trim()) return cb('必填')
                  try {
                    JSON.parse(String(val).trim())
                    cb()
                  } catch {
                    cb('内容必须是有效的 JSON 格式')
                  }
                },
              },
            ]}
            extra="请按照对应类型的 JSON Schema 填写模板内容"
          >
            <Textarea
              placeholder={`// 示例：system_prompt 类型
{
  "systemPrompt": "你是一个专业的...",
  "personaTag": "assistant",
  "variables": [{"name": "language", "label": "语言", "defaultVal": "中文"}]
}`}
              autoSize={{ minRows: 8, maxRows: 20 }}
              style={{ fontFamily: 'monospace', fontSize: '12px' }}
            />
          </FormItem>
        </Form>
      </Drawer>

      {/* ====== 应用模板模态框 ====== */}
      <Modal
        title={applyRow ? `应用模板：${applyRow.name}` : '应用模板'}
        visible={applyOpen}
        onCancel={() => setApplyOpen(false)}
        onOk={() => void submitApply()}
        confirmLoading={applySubmitting}
        okText="确认应用"
        autoFocus={false}
        style={{ width: 520 }}
      >
        {applyRow && (
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <Tag color={TYPE_MAP[applyRow.type]?.color || 'gray'}>
                {TYPE_MAP[applyRow.type]?.label || applyRow.type}
              </Tag>
              <span className="text-sm text-[var(--color-text-2)]">{applyRow.description || ''}</span>
            </div>

            {/* 语音类型：选择目标 Agent */}
            {applyRow.type === 'voice' && (
              <FormItem label="选择目标 Agent" required>
                <Select
                  placeholder="选择要应用语音配置的 Agent"
                  value={applyAgentId}
                  onChange={(v) => setApplyAgentId(v as number | undefined)}
                  options={agents.map((a) => ({ label: `${a.name} (#${a.id})`, value: a.id }))}
                  allowClear
                />
              </FormItem>
            )}

            {/* 提示词类型：选择目标 Agent + 填写变量 */}
            {applyRow.type === 'system_prompt' && (
              <>
                <FormItem label="选择目标 Agent（可选）" extra="不选则仅预览替换后的提示词">
                  <Select
                    placeholder="选择要更新提示词的 Agent"
                    value={applyAgentId}
                    onChange={(v) => setApplyAgentId(v as number | undefined)}
                    options={agents.map((a) => ({ label: `${a.name} (#${a.id})`, value: a.id }))}
                    allowClear
                  />
                </FormItem>
                {Object.keys(applyVariables).length > 0 && (
                  <div className="rounded-md border border-[var(--color-border-2)] p-3 space-y-3">
                    <div className="text-sm font-medium text-[var(--color-text-1)]">模板变量</div>
                    {Object.entries(applyVariables).map(([name, val]) => (
                      <FormItem key={name} label={name}>
                        <Input
                          value={val}
                          onChange={(v) =>
                            setApplyVariables((prev) => ({ ...prev, [name]: v }))
                          }
                          placeholder={`输入 ${name} 的值`}
                        />
                      </FormItem>
                    ))}
                  </div>
                )}
              </>
            )}

            {/* Agent / Knowledge 类型 */}
            {(applyRow.type === 'agent' || applyRow.type === 'knowledge') && (
              <div className="rounded-md bg-[var(--color-fill-2)] p-3 text-sm text-[var(--color-text-2)]">
                {applyRow.type === 'agent'
                  ? '将使用此模板创建一个全新的 Agent，包含系统提示词、语音配置、模型参数等。'
                  : '将使用此模板创建一个新的知识库命名空间。'}
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  )
}

export default Presets
