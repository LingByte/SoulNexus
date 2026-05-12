// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// LLM 渠道管理：Arco Table + Form Modal + Drawer 详情。
import { useEffect, useMemo, useState } from 'react'
import {
  Table,
  Input,
  Select,
  Button,
  Tag,
  Modal,
  Form,
  Drawer,
  Switch,
  Popconfirm,
  Message,
  Space,
  InputNumber,
} from '@arco-design/web-react'
import { Search, RefreshCw, Plus, Edit, Trash2, Eye, Zap, ExternalLink } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listLLMChannels,
  getLLMChannel,
  createLLMChannel,
  updateLLMChannel,
  deleteLLMChannel,
  syncLLMChannelAbilities,
  type LLMChannel,
} from '@/services/adminApi'
import { LLM_PROVIDER_HINTS, findLLMHint } from '@/data/providerSchemas'

const Option = Select.Option
const FormItem = Form.Item
const TextArea = Input.TextArea

// 供下拉列表使用：走 LLM_PROVIDER_HINTS，额外补一个 coze / lmstudio 占位
const PROTOCOLS = [
  ...LLM_PROVIDER_HINTS.map((h) => ({ value: h.protocol, label: h.label })),
  { value: 'coze', label: 'Coze' },
  { value: 'lmstudio', label: 'LM Studio' },
]

const protocolColor = (p: string) => {
  switch (p) {
    case 'openai': return 'arcoblue'
    case 'azure': return 'cyan'
    case 'anthropic': return 'orangered'
    case 'gemini': return 'purple'
    case 'qwen': return 'orange'
    case 'deepseek': return 'magenta'
    case 'zhipu': return 'lime'
    case 'moonshot': return 'gold'
    case 'ollama': return 'green'
    case 'coze': return 'purple'
    case 'lmstudio': return 'cyan'
    default: return 'gray'
  }
}

interface FormShape {
  protocol: string
  name: string
  base_url?: string
  key: string
  group: string
  models: string
  status: number
  priority: number
  weight: number
  test_model?: string
  openai_organization?: string
  tag?: string
  is_multi_key?: boolean
  multi_key_mode?: 'polling' | 'random'
}

const LlmChannels = () => {
  const [rows, setRows] = useState<LLMChannel[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [protocolFilter, setProtocolFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<LLMChannel | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm<FormShape>()

  const [detail, setDetail] = useState<LLMChannel | null>(null)
  const [syncingId, setSyncingId] = useState<number | null>(null)

  // 当前 Modal 内选中的协议，驱动占位 / 条件字段
  const [currentProtocol, setCurrentProtocol] = useState<string>('openai')
  const activeHint = useMemo(() => findLLMHint(currentProtocol), [currentProtocol])

  const fetchRows = async () => {
    setLoading(true)
    try {
      const r = await listLLMChannels({
        page,
        pageSize,
        search: search || undefined,
        protocol: protocolFilter || undefined,
        status: statusFilter || undefined,
        mask_key: 'true',
      })
      setRows(r.channels || [])
      setTotal(r.total || 0)
    } catch (e: any) {
      Message.error(`加载失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchRows()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, search, protocolFilter, statusFilter])

  const onSearch = () => { setPage(1); setSearch(searchInput) }

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ protocol: 'openai', status: 1, priority: 0, weight: 1, group: 'default', models: '', is_multi_key: false, multi_key_mode: 'polling' } as any)
    setCurrentProtocol('openai')
    setModalOpen(true)
  }

  const openEdit = async (row: LLMChannel) => {
    try {
      const full = await getLLMChannel(row.id) // 拿未脱敏 Key
      setEditing(full)
      form.setFieldsValue({
        protocol: full.protocol,
        name: full.name,
        base_url: full.base_url || '',
        key: full.key,
        group: full.group || 'default',
        models: full.models || '',
        status: full.status,
        priority: (full.priority as any) ?? 0,
        weight: (full.weight as any) ?? 1,
        test_model: full.test_model || '',
        openai_organization: full.openai_organization || '',
        tag: full.tag || '',
        is_multi_key: !!full.channel_info?.is_multi_key,
        multi_key_mode: (full.channel_info?.multi_key_mode as any) || 'polling',
      } as any)
      setCurrentProtocol(full.protocol || 'openai')
      setModalOpen(true)
    } catch (e: any) {
      Message.error(`加载详情失败：${e?.msg || e?.message || e}`)
    }
  }

  const save = async () => {
    try {
      const v = await form.validate()
      setSaving(true)
      const body: Partial<LLMChannel> = {
        protocol: v.protocol,
        name: v.name,
        base_url: v.base_url || null,
        key: v.key,
        group: v.group || 'default',
        models: v.models || '',
        status: v.status,
        priority: v.priority,
        weight: v.weight,
        test_model: v.test_model || null,
        openai_organization: v.openai_organization || null,
        tag: v.tag || null,
        channel_info: {
          is_multi_key: !!v.is_multi_key,
          multi_key_mode: v.multi_key_mode || 'polling',
          multi_key_size: v.is_multi_key
            ? (v.key || '').split(/[\n,]/).map((s) => s.trim()).filter(Boolean).length
            : 0,
        },
      }
      if (editing) {
        await updateLLMChannel(editing.id, body)
        Message.success('已更新')
      } else {
        await createLLMChannel(body)
        Message.success('已创建')
      }
      setModalOpen(false)
      fetchRows()
    } catch (e: any) {
      if (e?.message || e?.msg) Message.error(`保存失败：${e?.msg || e?.message}`)
    } finally {
      setSaving(false)
    }
  }

  const onDelete = async (row: LLMChannel) => {
    try {
      await deleteLLMChannel(row.id)
      Message.success('已删除')
      fetchRows()
    } catch (e: any) {
      Message.error(`删除失败：${e?.msg || e?.message || e}`)
    }
  }

  const onSync = async (row: LLMChannel) => {
    setSyncingId(row.id)
    try {
      await syncLLMChannelAbilities(row.id)
      Message.success(`已根据 models 重建 abilities`)
    } catch (e: any) {
      Message.error(`同步失败：${e?.msg || e?.message || e}`)
    } finally {
      setSyncingId(null)
    }
  }

  const openDetail = async (row: LLMChannel) => {
    try {
      const full = await getLLMChannel(row.id)
      setDetail(full)
    } catch (e: any) {
      Message.error(`加载详情失败：${e?.msg || e?.message || e}`)
    }
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '名称', dataIndex: 'name', width: 180, ellipsis: true },
    {
      title: '协议',
      dataIndex: 'protocol',
      width: 120,
      render: (v: string) => <Tag color={protocolColor(v)}>{v}</Tag>,
    },
    { title: '分组', dataIndex: 'group', width: 110 },
    { title: 'BaseURL', dataIndex: 'base_url', ellipsis: true, render: (v: string) => v || '-' },
    {
      title: '模型',
      dataIndex: 'models',
      ellipsis: true,
      render: (v: string) => {
        const list = String(v || '').split(/[,\n;]/).map((s) => s.trim()).filter(Boolean)
        if (list.length === 0) return '-'
        return (
          <Space wrap size="mini">
            {list.slice(0, 3).map((m) => <Tag key={m} size="small">{m}</Tag>)}
            {list.length > 3 && <Tag size="small">+{list.length - 3}</Tag>}
          </Space>
        )
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 80,
      render: (v: number) => <Tag color={v === 1 ? 'green' : 'gray'}>{v === 1 ? '启用' : '停用'}</Tag>,
    },
    { title: '优先级', dataIndex: 'priority', width: 80, render: (v: number) => v ?? 0 },
    {
      title: '操作',
      key: '__actions__',
      width: 280,
      fixed: 'right' as const,
      render: (_: any, row: LLMChannel) => (
        <Space size="mini">
          <Button size="mini" type="text" onClick={() => openDetail(row)}>
            <span className="inline-flex items-center gap-1"><Eye size={14} /> 详情</span>
          </Button>
          <Button size="mini" type="text" onClick={() => openEdit(row)}>
            <span className="inline-flex items-center gap-1"><Edit size={14} /> 编辑</span>
          </Button>
          <Button size="mini" type="text" loading={syncingId === row.id} onClick={() => onSync(row)}>
            <span className="inline-flex items-center gap-1"><Zap size={14} /> 同步</span>
          </Button>
          <Popconfirm
            title={`确定删除渠道「${row.name || row.id}」？相关 abilities 也会一并清除。`}
            okText="删除"
            cancelText="取消"
            onOk={() => onDelete(row)}
          >
            <Button size="mini" type="text" status="danger">
              <span className="inline-flex items-center gap-1"><Trash2 size={14} /> 删除</span>
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title="LLM 渠道管理"
        description="维护对外 OpenAI / Anthropic 兼容 API 的上游渠道。模型按 (group, model) 路由，按 priority 失败重试。"
        actions={
          <Space wrap>
            <Input
              prefix={<Search size={14} />}
              placeholder="搜索 名称 / 分组 / BaseURL / 模型"
              value={searchInput}
              onChange={(v) => setSearchInput(v)}
              onPressEnter={onSearch}
              allowClear
              onClear={() => { setPage(1); setSearch('') }}
              style={{ width: 260 }}
            />
            <Select value={protocolFilter} onChange={(v) => { setPage(1); setProtocolFilter(v) }} placeholder="协议" style={{ width: 140 }}>
              <Option value="">全部协议</Option>
              {PROTOCOLS.map((p) => <Option key={p.value} value={p.value}>{p.label}</Option>)}
            </Select>
            <Select value={statusFilter} onChange={(v) => { setPage(1); setStatusFilter(v) }} placeholder="状态" style={{ width: 110 }}>
              <Option value="">全部</Option>
              <Option value="1">启用</Option>
              <Option value="0">停用</Option>
            </Select>
            <Button type="primary" onClick={onSearch}>搜索</Button>
            <Button onClick={fetchRows}>
              <span className="inline-flex items-center gap-1"><RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新</span>
            </Button>
            <Button type="primary" onClick={openCreate}>
              <span className="inline-flex items-center gap-1"><Plus size={14} /> 新建渠道</span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(r: LLMChannel) => String(r.id)}
        loading={loading}
        columns={columns}
        data={rows}
        scroll={{ x: 1500 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showTotal: (t: number) => `共 ${t} 条`,
          sizeCanChange: true,
          sizeOptions: [10, 20, 50, 100],
          onChange: (p: number, ps: number) => { setPage(p); setPageSize(ps) },
        }}
      />

      <Modal
        title={editing ? '编辑渠道' : '新建渠道'}
        visible={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={save}
        confirmLoading={saving}
        okText="保存"
        cancelText="取消"
        autoFocus={false}
        style={{ width: 720 }}
      >
        <Form form={form} layout="vertical" autoComplete="off">
          {/* 协议改变时通过 onChange 同步 currentProtocol，用以驱动占位 / 字段可见性 */}
          <div className="grid grid-cols-2 gap-3">
            <FormItem label="协议 Provider" field="protocol" rules={[{ required: true }]}>
              <Select
                onChange={(v) => setCurrentProtocol(String(v || ''))}
                placeholder="选择上游协议"
              >
                {PROTOCOLS.map((p) => (
                  <Option key={p.value} value={p.value}>{p.label}</Option>
                ))}
              </Select>
            </FormItem>
            <FormItem label="名称" field="name" rules={[{ required: true, message: '请填写名称' }]}>
              <Input placeholder="如 OpenAI 主渠道" />
            </FormItem>
          </div>

          {activeHint?.docs ? (
            <div className="mb-2 flex items-center gap-2 text-xs text-[var(--color-text-3)]">
              <span>{activeHint.label}</span>
              <a className="inline-flex items-center gap-0.5 text-[var(--color-link)]" href={activeHint.docs} target="_blank" rel="noreferrer">
                <ExternalLink size={12} /> 官方文档
              </a>
              {activeHint.notes ? <span className="ml-2 italic">{activeHint.notes}</span> : null}
            </div>
          ) : null}

          <FormItem
            label="Base URL"
            field="base_url"
            extra={
              activeHint?.baseUrlPlaceholder
                ? `留空使用 ${activeHint.label} 官方默认；自建代理可填代理根路径`
                : '留空使用各协议官方默认；自建 OneAPI / 第三方代理在此填入根路径'
            }
          >
            <Input placeholder={activeHint?.baseUrlPlaceholder || 'https://api.openai.com'} />
          </FormItem>
          <FormItem label="API Key" field="key" rules={[{ required: true, message: '请填写 API Key' }]} extra="开启「多 Key」后支持每行 / 逗号一条，多 Key 自动按下方策略调度">
            <TextArea rows={3} placeholder={'sk-xxx\nsk-yyy（每行一个或逗号分隔）'} />
          </FormItem>
          <div className="grid grid-cols-2 gap-3">
            <FormItem label="启用多 Key 调度" field="is_multi_key" triggerPropName="checked">
              <Switch />
            </FormItem>
            <FormItem label="调度策略" field="multi_key_mode" extra="多 Key 启用时生效；polling=轮询，random=随机">
              <Select>
                <Option value="polling">轮询 polling</Option>
                <Option value="random">随机 random</Option>
              </Select>
            </FormItem>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <FormItem label="分组 group" field="group" rules={[{ required: true }]} extra="路由按 (group, model) 解析；默认 default">
              <Input placeholder="default" />
            </FormItem>
            {activeHint?.showOpenAIOrganization ? (
              <FormItem label="OpenAI Organization" field="openai_organization" extra="仅在使用 OpenAI 多组织账号时需要">
                <Input placeholder="org-xxxx，可留空" />
              </FormItem>
            ) : (
              <div />
            )}
          </div>
          <FormItem
            label="模型列表 models"
            field="models"
            rules={[{ required: true, message: '至少一个模型' }]}
            extra={
              activeHint?.modelsHint
                ? `示例：${activeHint.modelsHint}`
                : '逗号、分号或换行分隔；保存后会自动同步到 abilities 表'
            }
          >
            <TextArea rows={4} placeholder={activeHint?.modelsHint || 'gpt-4o-mini, gpt-4o\ngpt-3.5-turbo'} />
          </FormItem>
          <div className="grid grid-cols-3 gap-3">
            <FormItem label="状态" field="status" rules={[{ required: true }]}>
              <Select>
                <Option value={1}>启用</Option>
                <Option value={0}>停用</Option>
              </Select>
            </FormItem>
            <FormItem label="优先级" field="priority" extra="高在前">
              <InputNumber min={0} step={1} />
            </FormItem>
            <FormItem label="权重" field="weight">
              <InputNumber min={1} step={1} />
            </FormItem>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <FormItem label="测试模型 test_model" field="test_model">
              <Input placeholder="例如 gpt-4o-mini" />
            </FormItem>
            <FormItem label="标签 tag" field="tag">
              <Input placeholder="可选" />
            </FormItem>
          </div>
        </Form>
      </Modal>

      <Drawer
        title="渠道详情"
        visible={!!detail}
        width={720}
        onCancel={() => setDetail(null)}
        footer={null}
        autoFocus={false}
      >
        {detail && (
          <div className="space-y-3 text-sm">
            <Row label="ID" value={String(detail.id)} />
            <Row label="名称" value={detail.name || '-'} />
            <Row label="协议" value={<Tag color={protocolColor(detail.protocol)}>{detail.protocol}</Tag>} />
            <Row label="分组" value={detail.group || '-'} />
            <Row label="状态" value={<Tag color={detail.status === 1 ? 'green' : 'gray'}>{detail.status === 1 ? '启用' : '停用'}</Tag>} />
            <Row label="Base URL" value={detail.base_url || '-'} />
            <Row label="API Key" value={detail.key || '-'} />
            <Row label="OpenAI Org" value={detail.openai_organization || '-'} />
            <Row label="测试模型" value={detail.test_model || '-'} />
            <Row label="优先级" value={String(detail.priority ?? 0)} />
            <Row label="权重" value={String(detail.weight ?? 1)} />
            <Row label="标签" value={detail.tag || '-'} />
            <Row
              label="模型"
              value={
                <Space wrap size="mini">
                  {String(detail.models || '').split(/[,\n;]/).map((s) => s.trim()).filter(Boolean).map((m) => (
                    <Tag key={m}>{m}</Tag>
                  ))}
                </Space>
              }
            />
            <Row
              label="创建时间"
              value={detail.created_time ? new Date(detail.created_time * 1000).toLocaleString('zh-CN') : '-'}
            />
            <Row label="已用额度" value={String(detail.used_quota ?? 0)} />
          </div>
        )}
      </Drawer>
    </div>
  )
}

const Row = ({ label, value }: { label: string; value: React.ReactNode }) => (
  <div className="flex items-start gap-3">
    <div className="w-28 shrink-0 text-[var(--color-text-3)]">{label}</div>
    <div className="flex-1 break-all text-[var(--color-text-1)]">{value}</div>
  </div>
)

export default LlmChannels
