// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// LLM 模型元数据：模型展示信息、计费倍率，独立于 abilities 路由表，
// 主要给前端模型广场 / 计费模块使用。
import { useEffect, useState } from 'react'
import {
  Table,
  Input,
  Select,
  Button,
  Tag,
  Modal,
  Form,
  Popconfirm,
  Message,
  Space,
  InputNumber,
  Switch,
} from '@arco-design/web-react'
import { Search, RefreshCw, Plus, Edit, Trash2 } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listLLMModelMetas,
  upsertLLMModelMeta,
  deleteLLMModelMeta,
  type LLMModelMeta,
} from '@/services/adminApi'

const Option = Select.Option
const FormItem = Form.Item

interface FormShape {
  id?: number
  model_name: string
  vendor?: string
  description?: string
  tags?: string
  icon_url?: string
  status: boolean
  sort_order: number
  context_length?: number | null
  max_output_tokens?: number | null
  quota_billing_mode?: string
  quota_model_ratio: number
  quota_prompt_ratio: number
  quota_completion_ratio: number
  quota_cache_read_ratio: number
}

const VENDORS = ['openai', 'anthropic', 'deepseek', 'moonshot', 'qwen', 'doubao', 'gemini', 'meta', 'mistral']

const LlmModelMetas = () => {
  const [rows, setRows] = useState<LLMModelMeta[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [vendorFilter, setVendorFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const [modalOpen, setModalOpen] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm<FormShape>()

  const fetchRows = async () => {
    setLoading(true)
    try {
      const r = await listLLMModelMetas({
        page,
        pageSize,
        search: search || undefined,
        vendor: vendorFilter || undefined,
        status: statusFilter || undefined,
      })
      setRows(r.items || [])
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
  }, [page, pageSize, search, vendorFilter, statusFilter])

  const onSearch = () => { setPage(1); setSearch(searchInput) }

  const openCreate = () => {
    setEditingId(null)
    form.resetFields()
    form.setFieldsValue({
      status: true,
      sort_order: 0,
      quota_model_ratio: 1,
      quota_prompt_ratio: 1,
      quota_completion_ratio: 1,
      quota_cache_read_ratio: 0.25,
      quota_billing_mode: '',
    } as any)
    setModalOpen(true)
  }

  const openEdit = (row: LLMModelMeta) => {
    setEditingId(row.id)
    form.setFieldsValue({
      ...row,
      status: row.status === 1,
    } as any)
    setModalOpen(true)
  }

  const save = async () => {
    try {
      const v = await form.validate()
      setSaving(true)
      const body: Partial<LLMModelMeta> = {
        ...v,
        id: editingId || undefined,
        status: v.status ? 1 : 0,
      } as any
      await upsertLLMModelMeta(body)
      Message.success(editingId ? '已更新' : '已创建')
      setModalOpen(false)
      fetchRows()
    } catch (e: any) {
      if (e?.message || e?.msg) Message.error(`保存失败：${e?.msg || e?.message}`)
    } finally {
      setSaving(false)
    }
  }

  const onDelete = async (row: LLMModelMeta) => {
    try {
      await deleteLLMModelMeta(row.id)
      Message.success('已删除')
      fetchRows()
    } catch (e: any) {
      Message.error(`删除失败：${e?.msg || e?.message || e}`)
    }
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '模型名', dataIndex: 'model_name', width: 200, ellipsis: true },
    {
      title: '供应商',
      dataIndex: 'vendor',
      width: 120,
      render: (v: string) => (v ? <Tag>{v}</Tag> : '-'),
    },
    { title: '描述', dataIndex: 'description', ellipsis: true, render: (v: string) => v || '-' },
    {
      title: '状态',
      dataIndex: 'status',
      width: 80,
      render: (v: number) => <Tag color={v === 1 ? 'green' : 'gray'}>{v === 1 ? '启用' : '停用'}</Tag>,
    },
    {
      title: '计费倍率',
      dataIndex: 'quota_model_ratio',
      width: 110,
      render: (_: any, r: LLMModelMeta) => (
        <Space size="mini">
          <Tag size="small">×{r.quota_model_ratio}</Tag>
        </Space>
      ),
    },
    {
      title: '上下文',
      dataIndex: 'context_length',
      width: 100,
      render: (v: number) => (v ? `${(v / 1024).toFixed(0)}K` : '-'),
    },
    { title: '排序', dataIndex: 'sort_order', width: 80 },
    {
      title: '操作',
      key: '__actions__',
      width: 180,
      fixed: 'right' as const,
      render: (_: any, row: LLMModelMeta) => (
        <Space size="mini">
          <Button size="mini" type="text" onClick={() => openEdit(row)}>
            <span className="inline-flex items-center gap-1"><Edit size={14} /> 编辑</span>
          </Button>
          <Popconfirm
            title={`确定删除模型「${row.model_name}」？`}
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
        title="LLM 模型元数据"
        description="模型展示信息（描述 / 图标 / 上下文 / 计费倍率）。路由仍以渠道 abilities 为准；这里主要给模型广场 / 计费模块使用。"
        actions={
          <Space wrap>
            <Input
              prefix={<Search size={14} />}
              placeholder="搜索 模型名 / 描述 / 标签"
              value={searchInput}
              onChange={(v) => setSearchInput(v)}
              onPressEnter={onSearch}
              allowClear
              onClear={() => { setPage(1); setSearch('') }}
              style={{ width: 240 }}
            />
            <Select value={vendorFilter} onChange={(v) => { setPage(1); setVendorFilter(v) }} placeholder="供应商" style={{ width: 140 }}>
              <Option value="">全部</Option>
              {VENDORS.map((x) => <Option key={x} value={x}>{x}</Option>)}
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
              <span className="inline-flex items-center gap-1"><Plus size={14} /> 新建</span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(r: LLMModelMeta) => String(r.id)}
        loading={loading}
        columns={columns}
        data={rows}
        scroll={{ x: 1300 }}
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
        title={editingId ? '编辑模型元数据' : '新建模型元数据'}
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
            <FormItem label="模型名称 model_name" field="model_name" rules={[{ required: true }]}>
              <Input placeholder="例如 gpt-4o-mini" />
            </FormItem>
            <FormItem label="供应商 vendor" field="vendor">
              <Select allowClear placeholder="选择或输入">
                {VENDORS.map((x) => <Option key={x} value={x}>{x}</Option>)}
              </Select>
            </FormItem>
          </div>
          <FormItem label="描述" field="description">
            <Input.TextArea rows={3} placeholder="模型简介、能力卖点" />
          </FormItem>
          <div className="grid grid-cols-2 gap-3">
            <FormItem label="标签 tags" field="tags" extra="逗号分隔">
              <Input placeholder="multimodal, fast" />
            </FormItem>
            <FormItem label="图标 URL" field="icon_url">
              <Input placeholder="https://..." />
            </FormItem>
          </div>
          <div className="grid grid-cols-3 gap-3">
            <FormItem label="启用" field="status" triggerPropName="checked"><Switch /></FormItem>
            <FormItem label="排序" field="sort_order">
              <InputNumber min={0} step={1} />
            </FormItem>
            <FormItem label="计费模式" field="quota_billing_mode">
              <Select allowClear placeholder="留空 = 按 token">
                <Option value="">按 token</Option>
                <Option value="per_request">按次</Option>
                <Option value="hybrid">混合</Option>
              </Select>
            </FormItem>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <FormItem label="上下文长度 context_length" field="context_length">
              <InputNumber min={0} step={1024} placeholder="例如 128000" />
            </FormItem>
            <FormItem label="最大输出 max_output_tokens" field="max_output_tokens">
              <InputNumber min={0} step={256} />
            </FormItem>
          </div>
          <div className="grid grid-cols-4 gap-3">
            <FormItem label="模型倍率" field="quota_model_ratio">
              <InputNumber min={0} step={0.1} />
            </FormItem>
            <FormItem label="输入倍率" field="quota_prompt_ratio">
              <InputNumber min={0} step={0.1} />
            </FormItem>
            <FormItem label="输出倍率" field="quota_completion_ratio">
              <InputNumber min={0} step={0.1} />
            </FormItem>
            <FormItem label="缓存读" field="quota_cache_read_ratio">
              <InputNumber min={0} max={1} step={0.05} />
            </FormItem>
          </div>
        </Form>
      </Modal>
    </div>
  )
}

export default LlmModelMetas
