// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 语音渠道管理（ASR / TTS）：Tabs 切换两种类型，复用同一 CRUD 逻辑。
// 两类渠道字段一致（provider/name/group/sort_order/enabled/config_json/models），后端走独立路由。

import { useEffect, useMemo, useState } from 'react'
import {
  Table,
  Input,
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
  Select,
  Tabs,
} from '@arco-design/web-react'
import { Search, RefreshCw, Plus, Edit, Trash2, Eye } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  asrChannelsApi,
  ttsChannelsApi,
  type SpeechChannel,
  type SpeechChannelWriteReq,
} from '@/services/adminApi'
import {
  ASR_PROVIDERS,
  TTS_PROVIDERS,
  findASRSchema,
  findTTSSchema,
} from '@/data/providerSchemas'
import {
  ProviderConfigForm,
  parseConfigJSON,
  validateProviderConfig,
} from '@/components/Provider/ProviderConfigForm'

const FormItem = Form.Item
const TextArea = Input.TextArea
const TabPane = Tabs.TabPane
const Option = Select.Option

type Kind = 'asr' | 'tts'

interface FormShape {
  provider: string
  name: string
  enabled: boolean
  group: string
  sort_order: number
  models?: string
  /** 仅当用户显式切到「原始 JSON」模式时才使用 */
  config_json?: string
}

const SpeechChannels = () => {
  const [kind, setKind] = useState<Kind>('asr')
  const [rows, setRows] = useState<SpeechChannel[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const [modalOpen, setModalOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [editing, setEditing] = useState<SpeechChannel | null>(null)
  const [form] = Form.useForm<FormShape>()

  const [detail, setDetail] = useState<SpeechChannel | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)

  // 当前选中的 provider（用于实时切换表单表调用 schema）
  const [currentProvider, setCurrentProvider] = useState<string>('')
  // 结构化表单的 config 状态对象（序列化后写回 config_json）
  const [configObj, setConfigObj] = useState<Record<string, any>>({})
  // 在编辑模式下，已保存过的敏感字段名（用户留空则保留原值）
  const [filledSecrets, setFilledSecrets] = useState<Set<string>>(new Set())
  // 原始 JSON 编辑模式（未注册 provider 或用户手动切换）
  const [rawJsonMode, setRawJsonMode] = useState(false)

  const activeSchema = useMemo(
    () => (kind === 'asr' ? findASRSchema(currentProvider) : findTTSSchema(currentProvider)),
    [kind, currentProvider],
  )
  const providerOptions = useMemo(
    () => (kind === 'asr' ? ASR_PROVIDERS : TTS_PROVIDERS),
    [kind],
  )

  const api = useMemo(() => (kind === 'asr' ? asrChannelsApi : ttsChannelsApi), [kind])

  const fetchRows = async () => {
    setLoading(true)
    try {
      const r = await api.list({ page, pageSize, search: search || undefined })
      setRows(r.channels || [])
      setTotal(r.total || 0)
    } catch (e: any) {
      Message.error(`加载失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    setPage(1)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kind])

  useEffect(() => {
    fetchRows()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kind, page, pageSize])

  const handleSearch = () => {
    setPage(1)
    fetchRows()
  }

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({
      provider: '',
      name: '',
      enabled: true,
      group: 'default',
      sort_order: 0,
    })
    setCurrentProvider('')
    setConfigObj({})
    setFilledSecrets(new Set())
    setRawJsonMode(false)
    setModalOpen(true)
  }

  const openEdit = (row: SpeechChannel) => {
    setEditing(row)
    form.setFieldsValue({
      provider: row.provider,
      name: row.name,
      enabled: row.enabled,
      group: row.group || 'default',
      sort_order: row.sort_order || 0,
      models: row.models || '',
      config_json: row.config_json || '',
    })
    setCurrentProvider(row.provider)
    const parsed = parseConfigJSON(row.config_json)
    // 记录哪些敏感字段已有值，用于编辑校验时跳过「留空表示不修改」
    const schema = kind === 'asr' ? findASRSchema(row.provider) : findTTSSchema(row.provider)
    const filled = new Set<string>()
    schema?.fields.forEach((f) => {
      if (f.secret && parsed[f.name] !== undefined && String(parsed[f.name]).length > 0) {
        filled.add(f.name)
      }
    })
    // 编辑模式：如果是敏感字段，清空带入的值，避免明文回显
    const sanitized: Record<string, any> = { ...parsed }
    schema?.fields.forEach((f) => {
      if (f.secret) sanitized[f.name] = ''
    })
    setConfigObj(sanitized)
    setFilledSecrets(filled)
    setRawJsonMode(!schema)
    setModalOpen(true)
  }

  const save = async () => {
    try {
      const v = await form.validate()

      let configJSON = ''
      if (rawJsonMode || !activeSchema) {
        // 原始 JSON 模式
        const raw = (v.config_json || '').trim()
        if (raw) {
          try { JSON.parse(raw) } catch { Message.error('config_json 不是合法 JSON'); return }
        }
        configJSON = raw
      } else {
        // 结构化表单模式：验证必填，合并原未改动的敏感字段
        const missing = validateProviderConfig(activeSchema, configObj, {
          editing: !!editing,
          alreadyFilledSecrets: filledSecrets,
        })
        if (missing.length) {
          Message.error(`请填写：${missing.join('、')}`)
          return
        }
        // 编辑模式：敏感字段留空表示不修改，混入原 config_json 中的值
        let merged: Record<string, any> = { ...configObj }
        if (editing) {
          const original = parseConfigJSON(editing.config_json)
          activeSchema.fields.forEach((f) => {
            if (f.secret && (merged[f.name] === undefined || String(merged[f.name]).trim() === '')) {
              if (original[f.name] !== undefined) merged[f.name] = original[f.name]
            }
          })
        }
        // 剔除空字段，保持 JSON 净
        merged = Object.fromEntries(
          Object.entries(merged).filter(([, val]) => val !== undefined && val !== ''),
        )
        configJSON = JSON.stringify(merged, null, 2)
      }

      setSaving(true)
      const body: SpeechChannelWriteReq = {
        provider: v.provider.trim(),
        name: v.name.trim(),
        enabled: v.enabled,
        group: (v.group || 'default').trim(),
        sort_order: v.sort_order ?? 0,
        models: v.models || '',
        config_json: configJSON,
      }
      if (editing) {
        await api.update(editing.id, body)
        Message.success('已保存')
      } else {
        await api.create(body)
        Message.success('已创建')
      }
      setModalOpen(false)
      fetchRows()
    } catch (e: any) {
      if (e?.errorFields) return // form validation
      Message.error(`保存失败：${e?.msg || e?.message || e}`)
    } finally {
      setSaving(false)
    }
  }

  const remove = async (id: number) => {
    try {
      await api.remove(id)
      Message.success('已删除')
      fetchRows()
    } catch (e: any) {
      Message.error(`删除失败：${e?.msg || e?.message || e}`)
    }
  }

  const showDetail = async (row: SpeechChannel) => {
    try {
      const d = await api.get(row.id)
      setDetail(d)
      setDrawerOpen(true)
    } catch {
      setDetail(row)
      setDrawerOpen(true)
    }
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: 'Provider', dataIndex: 'provider', width: 160, render: (v: string) => <Tag color="arcoblue">{v}</Tag> },
    { title: '名称', dataIndex: 'name', width: 220, ellipsis: true },
    {
      title: '启用',
      dataIndex: 'enabled',
      width: 80,
      render: (v: boolean) => (v ? <Tag color="green">启用</Tag> : <Tag color="gray">禁用</Tag>),
    },
    { title: 'Group', dataIndex: 'group', width: 120 },
    { title: '优先级', dataIndex: 'sort_order', width: 90 },
    { title: '更新时间', dataIndex: 'updated_at', width: 170 },
    {
      title: '操作',
      key: '__actions__',
      width: 220,
      fixed: 'right' as const,
      render: (_: any, row: SpeechChannel) => (
        <Space size={4}>
          <Button size="mini" type="text" onClick={() => showDetail(row)}>
            <span className="inline-flex items-center gap-1"><Eye size={14} />详情</span>
          </Button>
          <Button size="mini" type="text" onClick={() => openEdit(row)}>
            <span className="inline-flex items-center gap-1"><Edit size={14} />编辑</span>
          </Button>
          <Popconfirm title="确认删除该渠道？" onOk={() => remove(row.id)} okText="删除" cancelText="取消">
            <Button size="mini" type="text" status="danger">
              <span className="inline-flex items-center gap-1"><Trash2 size={14} />删除</span>
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title="语音渠道管理"
        description="维护 ASR（语音识别）/ TTS（语音合成）上游渠道；按 group + sort_order 路由，相同 group 内 sort_order 越大优先级越高。"
        actions={
          <Space>
            <Input
              prefix={<Search size={14} />}
              placeholder="名称 / provider"
              value={search}
              onChange={(v) => setSearch(v)}
              onPressEnter={handleSearch}
              allowClear
              style={{ width: 220 }}
            />
            <Button type="primary" onClick={handleSearch}>搜索</Button>
            <Button onClick={fetchRows}>
              <span className="inline-flex items-center gap-1"><RefreshCw size={14} />刷新</span>
            </Button>
            <Button type="primary" status="success" onClick={openCreate}>
              <span className="inline-flex items-center gap-1"><Plus size={14} />新增{kind.toUpperCase()}</span>
            </Button>
          </Space>
        }
      />

      <Tabs activeTab={kind} onChange={(k) => setKind(k as Kind)}>
        <TabPane key="asr" title="ASR · 语音识别" />
        <TabPane key="tts" title="TTS · 语音合成" />
      </Tabs>

      <Table
        rowKey="id"
        loading={loading}
        columns={columns}
        data={rows}
        scroll={{ x: 1100 }}
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
        title={editing ? `编辑 ${kind.toUpperCase()} 渠道` : `新建 ${kind.toUpperCase()} 渠道`}
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
          <div className="grid grid-cols-2 gap-3">
            <FormItem label="厂商 Provider" field="provider" rules={[{ required: true, message: '请选择厂商' }]}>
              <Select
                allowCreate
                showSearch
                placeholder="选择已支持的厂商，或输入自定义 provider 标识"
                onChange={(v) => setCurrentProvider(String(v || ''))}
                filterOption={(input: string, option: any) =>
                  String(option?.props?.value ?? '')
                    .toLowerCase()
                    .includes(input.toLowerCase())
                }
              >
                {providerOptions.map((p) => (
                  <Option key={p.provider} value={p.provider}>
                    <span className="font-mono text-xs mr-2 text-[var(--color-text-3)]">{p.provider}</span>
                    {p.label}
                  </Option>
                ))}
              </Select>
            </FormItem>
            <FormItem label="名称" field="name" rules={[{ required: true }]}>
              <Input placeholder="展示名称" />
            </FormItem>
          </div>
          <div className="grid grid-cols-3 gap-3">
            <FormItem label="启用" field="enabled" triggerPropName="checked">
              <Switch />
            </FormItem>
            <FormItem label="Group 分组" field="group" rules={[{ required: true }]}>
              <Input placeholder="default" />
            </FormItem>
            <FormItem label="优先级" field="sort_order" extra="越大越优先">
              <InputNumber min={0} step={1} />
            </FormItem>
          </div>
          <FormItem
            label={kind === 'tts' ? '支持模型 / 音色（逗号分隔）' : '支持模型（逗号分隔）'}
            field="models"
            extra={
              activeSchema?.modelsHint
                ? `示例：${activeSchema.modelsHint}`
                : kind === 'tts'
                  ? '示例：cosyvoice-v1,longxiaobai,longxiaochun'
                  : '示例：whisper-1,paraformer-v2'
            }
          >
            <Input placeholder="逗号分隔，留空表示不限" />
          </FormItem>

          {/* 厂商配置：注册到 schema 的厂商使用结构化表单；其余 / 用户切换 → 原始 JSON */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-[var(--color-text-1)]">厂商配置</span>
              {activeSchema ? (
                <Button
                  size="mini"
                  type="text"
                  onClick={() => setRawJsonMode((v) => !v)}
                >
                  {rawJsonMode ? '回到结构化表单' : '切换为原始 JSON'}
                </Button>
              ) : null}
            </div>
            {!rawJsonMode && activeSchema ? (
              <ProviderConfigForm
                schema={activeSchema}
                value={configObj}
                onChange={setConfigObj}
                kind={kind}
                editing={!!editing}
              />
            ) : (
              <FormItem
                field="config_json"
                noStyle={false}
                extra={
                  activeSchema
                    ? '直接编辑 JSON；保存后会取代结构化表单填写的内容'
                    : '当前 provider 未在内置 schema 中，使用原始 JSON 编辑'
                }
              >
                <TextArea rows={8} placeholder={`{\n  "api_key": "xxx",\n  "endpoint": "https://..."\n}`} />
              </FormItem>
            )}
          </div>
        </Form>
      </Modal>

      <Drawer
        title="渠道详情"
        visible={drawerOpen}
        width={640}
        onCancel={() => setDrawerOpen(false)}
        footer={null}
        autoFocus={false}
      >
        {detail && (
          <div className="space-y-3 text-sm">
            <Row label="ID" value={String(detail.id)} />
            <Row label="Provider" value={detail.provider} />
            <Row label="名称" value={detail.name} />
            <Row label="启用" value={detail.enabled ? '是' : '否'} />
            <Row label="Group" value={detail.group || 'default'} />
            <Row label="优先级" value={String(detail.sort_order ?? 0)} />
            <Row label="模型 / 音色" value={detail.models || '-'} />
            <Row label="config_json" value={<pre className="m-0 break-all whitespace-pre-wrap text-xs">{detail.config_json || '-'}</pre>} />
            <Row label="创建时间" value={detail.created_at} />
            <Row label="更新时间" value={detail.updated_at} />
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

export default SpeechChannels
