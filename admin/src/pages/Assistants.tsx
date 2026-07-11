import { useEffect, useState, useCallback } from 'react'
import {
  Table,
  Card,
  Input,
  Button,
  Tag,
  Space,
  Modal,
  Form,
  Drawer,
  Select,
  Upload,
  Image,
  Popconfirm,
  Message,
  Descriptions,
  Typography,
  Tabs,
  InputNumber,
  Switch,
} from '@arco-design/web-react'
import {
  RefreshCw,
  Eye,
  Trash2,
  Edit3,
  Upload as UploadIcon,
  Download,
  FileInput,
  User,
  Star,
} from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import { useMediaQuery } from '@/hooks/useMediaQuery'
import {
  listAdminAssistants,
  getAdminAssistant,
  updateAdminAssistant,
  deleteAdminAssistant,
  importCharacterCard,
  exportCharacterCard,
  uploadAgentAvatar,
  type AdminAssistant,
} from '@/services/adminApi'

const { Text, Paragraph } = Typography
const FormItem = Form.Item
const TabPane = Tabs.TabPane

const VISIBILITY_MAP: Record<string, { label: string; color: string }> = {
  public: { label: '公开', color: 'green' },
  group: { label: '组织', color: 'arcoblue' },
  private: { label: '私有', color: 'gray' },
}

const Assistants = () => {
  const isSmallScreen = useMediaQuery('(max-width: 639px)')

  // 列表状态
  const [loading, setLoading] = useState(false)
  const [rows, setRows] = useState<AdminAssistant[]>([])
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [search, setSearch] = useState('')

  // 编辑抽屉
  const [editOpen, setEditOpen] = useState(false)
  const [editRow, setEditRow] = useState<AdminAssistant | null>(null)
  const [editSubmitting, setEditSubmitting] = useState(false)
  const [editForm] = Form.useForm()

  // 详情抽屉
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailRow, setDetailRow] = useState<AdminAssistant | null>(null)

  // 导入模态框
  const [importOpen, setImportOpen] = useState(false)
  const [importFile, setImportFile] = useState<File | null>(null)
  const [importSubmitting, setImportSubmitting] = useState(false)
  const [importGroupId, setImportGroupId] = useState<number | undefined>()

  // 头像上传
  const [avatarUploading, setAvatarUploading] = useState<Record<number, boolean>>({})

  // ==================== 数据加载 ====================

  const loadList = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listAdminAssistants({ page, pageSize, search: search || undefined })
      setRows(res.agents || [])
      setTotal(res.total || 0)
    } catch (e: any) {
      Message.error(e?.msg || e?.message || '加载失败')
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, search])

  useEffect(() => {
    loadList()
  }, [loadList])

  // ==================== 详情 ====================

  const openDetail = async (row: AdminAssistant) => {
    try {
      const detail = await getAdminAssistant(row.id)
      setDetailRow(detail)
    } catch {
      setDetailRow(row)
    }
    setDetailOpen(true)
  }

  // ==================== 编辑 ====================

  const openEdit = async (row: AdminAssistant) => {
    try {
      const detail = await getAdminAssistant(row.id)
      setEditRow(detail)
      editForm.setFieldsValue({
        name: detail.name,
        systemPrompt: detail.systemPrompt || '',
        description: detail.description || '',
        personality: detail.personality || '',
        scenario: detail.scenario || '',
        exampleDialogues: detail.exampleDialogues || '',
        tags: detail.tags || '',
        creatorNote: detail.creatorNote || '',
        specVersion: detail.specVersion || '',
        visibility: detail.visibility || 'private',
        temperature: detail.temperature ?? 0.7,
        maxTokens: detail.maxTokens ?? 2048,
        llmModel: detail.llmModel || '',
        speaker: detail.speaker || '',
        ttsProvider: detail.ttsProvider || '',
        enableJSONOutput: detail.enableJSONOutput ?? false,
        openingStatement: detail.openingStatement || '',
        personaTag: detail.personaTag || '',
      })
    } catch {
      setEditRow(row)
      editForm.setFieldsValue({
        name: row.name,
        systemPrompt: row.systemPrompt || '',
        visibility: 'private',
        temperature: 0.7,
        maxTokens: 2048,
        enableJSONOutput: false,
      })
    }
    setEditOpen(true)
  }

  const submitEdit = async () => {
    try {
      const values = await editForm.validate()
      if (!editRow) return
      setEditSubmitting(true)

      const payload: Record<string, any> = {
        name: String(values.name || '').trim(),
      }
      // 基础字段
      if (values.systemPrompt !== undefined) payload.systemPrompt = values.systemPrompt
      if (values.temperature !== undefined) payload.temperature = values.temperature
      if (values.maxTokens !== undefined) payload.maxTokens = values.maxTokens
      if (values.llmModel !== undefined) payload.llmModel = values.llmModel
      if (values.speaker !== undefined) payload.speaker = values.speaker
      if (values.ttsProvider !== undefined) payload.ttsProvider = values.ttsProvider
      if (values.enableJSONOutput !== undefined) payload.enableJSONOutput = values.enableJSONOutput
      // 角色卡字段
      if (values.description !== undefined) payload.description = values.description
      if (values.personality !== undefined) payload.personality = values.personality
      if (values.scenario !== undefined) payload.scenario = values.scenario
      if (values.exampleDialogues !== undefined) payload.exampleDialogues = values.exampleDialogues
      if (values.tags !== undefined) payload.tags = values.tags
      if (values.creatorNote !== undefined) payload.creatorNote = values.creatorNote
      if (values.specVersion !== undefined) payload.specVersion = values.specVersion
      if (values.visibility !== undefined) payload.visibility = values.visibility
      if (values.openingStatement !== undefined) payload.openingStatement = values.openingStatement
      if (values.personaTag !== undefined) payload.personaTag = values.personaTag

      await updateAdminAssistant(editRow.id, payload)
      Message.success('更新成功')
      setEditOpen(false)
      loadList()
    } catch (e: any) {
      if ((e as { errors?: unknown })?.errors) return
      Message.error(e?.msg || e?.message || '更新失败')
    } finally {
      setEditSubmitting(false)
    }
  }

  // ==================== 删除 ====================

  const handleDelete = async (row: AdminAssistant) => {
    try {
      await deleteAdminAssistant(row.id)
      Message.success('删除成功')
      loadList()
    } catch (e: any) {
      Message.error(e?.msg || e?.message || '删除失败')
    }
  }

  // ==================== 导入 ====================

  const handleImport = async () => {
    if (!importFile) {
      Message.warning('请选择文件')
      return
    }
    setImportSubmitting(true)
    try {
      await importCharacterCard({
        file: importFile,
        groupId: importGroupId,
      })
      Message.success('导入成功！角色卡已添加到智能体列表')
      setImportOpen(false)
      setImportFile(null)
      setImportGroupId(undefined)
      loadList()
    } catch (e: any) {
      Message.error(e?.msg || e?.message || '导入失败')
    } finally {
      setImportSubmitting(false)
    }
  }

  // ==================== 导出 ====================

  const handleExport = async (row: AdminAssistant, format: 'json' | 'png') => {
    try {
      const blob = await exportCharacterCard(row.id, format)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${row.name}_character_card.${format}`
      a.click()
      URL.revokeObjectURL(url)
      Message.success('导出成功')
    } catch (e: any) {
      Message.error(e?.msg || e?.message || '导出失败')
    }
  }

  // ==================== 头像上传 ====================

  const handleAvatarUpload = async (agentId: number, file: File) => {
    setAvatarUploading((prev) => ({ ...prev, [agentId]: true }))
    try {
      await uploadAgentAvatar(agentId, file)
      Message.success('头像上传成功')
      loadList()
    } catch (e: any) {
      Message.error(e?.msg || e?.message || '上传头像失败')
    } finally {
      setAvatarUploading((prev) => ({ ...prev, [agentId]: false }))
    }
  }

  // ==================== 表格列 ====================

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: '角色',
      dataIndex: 'name',
      width: 220,
      render: (_: any, record: AdminAssistant) => (
        <Space>
          <div
            style={{
              width: 36,
              height: 36,
              borderRadius: 8,
              overflow: 'hidden',
              background: record.avatarUrl ? undefined : 'var(--color-fill-2)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
            }}
          >
            {record.avatarUrl ? (
              <Image
                src={record.avatarUrl}
                width={36}
                height={36}
                preview={false}
                style={{ objectFit: 'cover' }}
              />
            ) : (
              <User size={18} style={{ color: 'var(--color-text-3)' }} />
            )}
          </div>
          <div>
            <Text bold>{record.name}</Text>
            {record.visibility && record.visibility !== 'private' && (
              <Tag
                size="small"
                color={VISIBILITY_MAP[record.visibility]?.color || 'gray'}
                style={{ marginLeft: 4 }}
              >
                {VISIBILITY_MAP[record.visibility]?.label || record.visibility}
              </Tag>
            )}
          </div>
        </Space>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      width: 180,
      ellipsis: true,
      render: (val: string) => (
        <Text type="secondary" style={{ fontSize: 12 }}>
          {val || '-'}
        </Text>
      ),
    },
    {
      title: '标签',
      dataIndex: 'tags',
      width: 140,
      render: (val: string) => {
        const tags = val?.split(',').filter(Boolean) || []
        if (!tags.length) return '-'
        return (
          <Space wrap size="mini">
            {tags.slice(0, 2).map((t) => (
              <Tag key={t} size="small" color="arcoblue">
                {t.trim()}
              </Tag>
            ))}
            {tags.length > 2 && (
              <Text type="secondary" style={{ fontSize: 11 }}>
                +{tags.length - 2}
              </Text>
            )}
          </Space>
        )
      },
    },
    {
      title: '版本',
      dataIndex: 'specVersion',
      width: 60,
      render: (val: string) => (val ? <Tag size="small">{val}</Tag> : '-'),
    },
    {
      title: '下载',
      dataIndex: 'downloadCount',
      width: 60,
      render: (val: number) => (
        <Space size={4}>
          <Download size={12} />
          <Text>{val || 0}</Text>
        </Space>
      ),
    },
    {
      title: '评分',
      dataIndex: 'rating',
      width: 80,
      render: (val: number, record: AdminAssistant) => {
        if (!val && !record.ratingCount) return '-'
        return (
          <Space size={4}>
            <Star size={12} style={{ color: '#f7ba1e' }} />
            <Text>{val?.toFixed(1) || '0'}</Text>
            <Text type="secondary" style={{ fontSize: 11 }}>
              ({record.ratingCount || 0})
            </Text>
          </Space>
        )
      },
    },
    {
      title: '模型',
      dataIndex: 'llmModel',
      width: 100,
      ellipsis: true,
      render: (val: string) => val || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 120,
      render: (val: string) => {
        if (!val) return '-'
        const d = new Date(val)
        return Number.isNaN(d.getTime()) ? val : d.toLocaleString('zh-CN')
      },
    },
    {
      title: '操作',
      key: 'actions',
      width: 280,
      fixed: 'right' as const,
      render: (_: any, record: AdminAssistant) => (
        <Space size="mini" wrap>
          <Button size="mini" type="text" onClick={() => openDetail(record)}>
            <span className="inline-flex items-center gap-1"><Eye size={14} /> 详情</span>
          </Button>
          <Button size="mini" type="text" onClick={() => openEdit(record)}>
            <span className="inline-flex items-center gap-1"><Edit3 size={14} /> 编辑</span>
          </Button>
          <Popconfirm
            title="确认删除该智能体？"
            okText="删除"
            cancelText="取消"
            onOk={() => handleDelete(record)}
          >
            <Button size="mini" type="text" status="danger">
              <Trash2 size={14} />
            </Button>
          </Popconfirm>
          <Upload
            accept="image/*"
            showUploadList={false}
            customRequest={(opt) => {
              const file = (opt as { file?: File }).file
              if (file) handleAvatarUpload(record.id, file)
            }}
          >
            <Button
              size="mini"
              type="outline"
              loading={avatarUploading[record.id]}
            >
              <UploadIcon size={12} />
            </Button>
          </Upload>
          <Popconfirm
            title="选择导出格式"
            okText="PNG"
            cancelText="JSON"
            onOk={() => handleExport(record, 'png')}
            onCancel={() => handleExport(record, 'json')}
          >
            <Button size="mini" type="outline">
              <Download size={12} />
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  // ==================== 渲染 ====================

  return (
    <div className="space-y-4">
      <PageHeader
        title="智能体管理"
        description="管理智能体配置、角色卡信息，支持导入/导出角色卡、上传头像。"
        actions={
          <div className="flex w-full max-w-full flex-col gap-2 sm:w-auto sm:flex-row sm:flex-wrap sm:items-center">
            <Input.Search
              allowClear
              placeholder="搜索智能体名称 / 描述"
              className="w-full min-w-0 sm:w-[240px]"
              value={search}
              onChange={setSearch}
              onSearch={(v) => {
                setPage(1)
                setSearch(String(v || '').trim())
              }}
            />
            <Button type="primary" onClick={() => setImportOpen(true)}>
              <span className="inline-flex items-center gap-1"><FileInput size={16} /> 导入角色卡</span>
            </Button>
            <Button onClick={() => loadList()}>
              <span className="inline-flex items-center gap-1">
                <RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新
              </span>
            </Button>
          </div>
        }
      />

      {/* 数据表格 */}
      <Card>
        <Table
          rowKey={(r) => String(r.id)}
          loading={loading}
          columns={columns}
          data={rows}
          scroll={{ x: 1400 }}
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
      </Card>

      {/* ====== 详情抽屉 ====== */}
      <Drawer
        title={detailRow?.name || '智能体详情'}
        visible={detailOpen}
        placement="right"
        width={isSmallScreen ? '100%' : 640}
        onCancel={() => {
          setDetailOpen(false)
          setDetailRow(null)
        }}
        footer={null}
      >
        {detailRow && (
          <div className="space-y-4">
            {/* 头像 + 名称 */}
            <div className="flex items-center gap-3">
              <div
                style={{
                  width: 64,
                  height: 64,
                  borderRadius: 12,
                  overflow: 'hidden',
                  background: detailRow.avatarUrl ? undefined : 'var(--color-fill-2)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  flexShrink: 0,
                }}
              >
                {detailRow.avatarUrl ? (
                  <Image
                    src={detailRow.avatarUrl}
                    width={64}
                    height={64}
                    preview={true}
                    style={{ objectFit: 'cover' }}
                  />
                ) : (
                  <User size={28} style={{ color: 'var(--color-text-3)' }} />
                )}
              </div>
              <div>
                <div className="text-lg font-semibold">{detailRow.name}</div>
                <Space size={4}>
                  {detailRow.visibility && (
                    <Tag size="small" color={VISIBILITY_MAP[detailRow.visibility]?.color || 'gray'}>
                      {VISIBILITY_MAP[detailRow.visibility]?.label || detailRow.visibility}
                    </Tag>
                  )}
                  {detailRow.specVersion && <Tag size="small">{detailRow.specVersion}</Tag>}
                </Space>
              </div>
            </div>

            {/* 基本信息 */}
            <Descriptions
              column={2}
              border
              size="small"
              data={[
                { label: 'ID', value: detailRow.id },
                { label: '用户ID', value: detailRow.userId },
                { label: '组织ID', value: detailRow.groupId || '-' },
                { label: '模型', value: detailRow.llmModel || '-' },
                { label: '温度', value: detailRow.temperature ?? '-' },
                { label: '最大Token', value: detailRow.maxTokens ?? '-' },
                { label: '说话人', value: detailRow.speaker || '-' },
                { label: 'TTS Provider', value: detailRow.ttsProvider || '-' },
                { label: 'JSON输出', value: detailRow.enableJSONOutput ? '是' : '否' },
                { label: 'Persona Tag', value: detailRow.personaTag || '-' },
                { label: '下载次数', value: detailRow.downloadCount || 0 },
                {
                  label: '评分',
                  value: detailRow.rating
                    ? `${detailRow.rating.toFixed(1)} (${detailRow.ratingCount || 0}人)`
                    : '-',
                },
                { label: 'Fork 来源', value: detailRow.forkedFrom || '-' },
                { label: '创建时间', value: detailRow.createdAt ? new Date(detailRow.createdAt).toLocaleString('zh-CN') : '-' },
              ]}
            />

            {/* 描述 */}
            {detailRow.description && (
              <Card title="描述" size="small">
                <Paragraph>{detailRow.description}</Paragraph>
              </Card>
            )}

            {/* 开场白 */}
            {detailRow.openingStatement && (
              <Card title="开场白" size="small">
                <Paragraph>{detailRow.openingStatement}</Paragraph>
              </Card>
            )}

            {/* 人格设定 */}
            {detailRow.personality && (
              <Card title="人格设定" size="small">
                <Paragraph style={{ whiteSpace: 'pre-wrap' }}>{detailRow.personality}</Paragraph>
              </Card>
            )}

            {/* 世界观 */}
            {detailRow.scenario && (
              <Card title="世界观/场景" size="small">
                <Paragraph style={{ whiteSpace: 'pre-wrap' }}>{detailRow.scenario}</Paragraph>
              </Card>
            )}

            {/* 示例对话 */}
            {detailRow.exampleDialogues && (
              <Card title="示例对话" size="small">
                <Paragraph style={{ whiteSpace: 'pre-wrap' }}>{detailRow.exampleDialogues}</Paragraph>
              </Card>
            )}

            {/* 创作者备注 */}
            {detailRow.creatorNote && (
              <Card title="创作者备注" size="small">
                <Paragraph>{detailRow.creatorNote}</Paragraph>
              </Card>
            )}

            {/* 标签 */}
            {detailRow.tags && (
              <Card title="标签" size="small">
                <Space wrap>
                  {detailRow.tags.split(',').filter(Boolean).map((t) => (
                    <Tag key={t} color="arcoblue">{t.trim()}</Tag>
                  ))}
                </Space>
              </Card>
            )}

            {/* 系统提示词 */}
            {detailRow.systemPrompt && (
              <Card title="系统提示词" size="small">
                <pre
                  style={{
                    whiteSpace: 'pre-wrap',
                    maxHeight: 300,
                    overflow: 'auto',
                    background: 'var(--color-fill-1)',
                    padding: 12,
                    borderRadius: 8,
                    fontSize: 12,
                  }}
                >
                  {detailRow.systemPrompt}
                </pre>
              </Card>
            )}

            {/* 操作按钮 */}
            <Space>
              <Popconfirm
                title="选择导出格式"
                okText="PNG"
                cancelText="JSON"
                onOk={() => handleExport(detailRow, 'png')}
                onCancel={() => handleExport(detailRow, 'json')}
              >
                <Button type="outline" icon={<Download size={16} />}>
                  导出角色卡
                </Button>
              </Popconfirm>
              <Upload
                accept="image/*"
                showUploadList={false}
                customRequest={(opt) => {
                  const file = (opt as { file?: File }).file
                  if (file) {
                    handleAvatarUpload(detailRow.id, file).then(() => {
                      // refresh detail
                      getAdminAssistant(detailRow.id).then(setDetailRow).catch(() => {})
                    })
                  }
                }}
              >
                <Button type="outline" icon={<UploadIcon size={16} />}>
                  上传头像
                </Button>
              </Upload>
            </Space>
          </div>
        )}
      </Drawer>

      {/* ====== 编辑抽屉 ====== */}
      <Drawer
        title={`编辑智能体：${editRow?.name || ''}`}
        visible={editOpen}
        placement="right"
        width={isSmallScreen ? '100%' : 600}
        onCancel={() => setEditOpen(false)}
        footer={
          <Space>
            <Button onClick={() => setEditOpen(false)}>取消</Button>
            <Button type="primary" loading={editSubmitting} onClick={() => { void submitEdit() }}>
              保存
            </Button>
          </Space>
        }
      >
        <Form form={editForm} layout="vertical">
          <Tabs defaultActiveTab="basic">
            <TabPane key="basic" title="基本信息">
              <div className="space-y-0 pt-2">
                <FormItem label="名称" field="name" rules={[{ required: true, message: '必填' }]}>
                  <Input placeholder="智能体名称" />
                </FormItem>
                <FormItem label="可见性" field="visibility">
                  <Select
                    options={[
                      { label: '私有 - 仅自己可见', value: 'private' },
                      { label: '组织 - 同组织可见', value: 'group' },
                      { label: '公开 - 所有人可见', value: 'public' },
                    ]}
                  />
                </FormItem>
                <FormItem label="模型" field="llmModel">
                  <Input placeholder="如：gpt-4o" />
                </FormItem>
                <FormItem label="温度 (Temperature)" field="temperature">
                  <InputNumber min={0} max={2} step={0.1} style={{ width: '100%' }} />
                </FormItem>
                <FormItem label="最大 Token" field="maxTokens">
                  <InputNumber min={1} max={128000} step={256} style={{ width: '100%' }} />
                </FormItem>
                <FormItem label="说话人" field="speaker">
                  <Input placeholder="说话人名称" />
                </FormItem>
                <FormItem label="TTS Provider" field="ttsProvider">
                  <Input placeholder="TTS 提供商" />
                </FormItem>
                <FormItem label="Persona Tag" field="personaTag">
                  <Input placeholder="如：assistant, companion" />
                </FormItem>
                <FormItem label="启用 JSON 输出" field="enableJSONOutput">
                  <Switch />
                </FormItem>
              </div>
            </TabPane>

            <TabPane key="character" title="角色卡">
              <div className="space-y-0 pt-2">
                <FormItem label="描述" field="description">
                  <Input.TextArea
                    placeholder="角色的简短描述"
                    autoSize={{ minRows: 2, maxRows: 6 }}
                  />
                </FormItem>
                <FormItem label="人格设定" field="personality">
                  <Input.TextArea
                    placeholder="角色的人格、性格描述"
                    autoSize={{ minRows: 4, maxRows: 12 }}
                  />
                </FormItem>
                <FormItem label="世界观/场景" field="scenario">
                  <Input.TextArea
                    placeholder="角色所处的世界观或场景设定"
                    autoSize={{ minRows: 3, maxRows: 10 }}
                  />
                </FormItem>
                <FormItem label="示例对话" field="exampleDialogues">
                  <Input.TextArea
                    placeholder='示例对话，格式如：\n{{user}}: 你好\n{{char}}: 你好呀！'
                    autoSize={{ minRows: 4, maxRows: 12 }}
                  />
                </FormItem>
                <FormItem label="标签" field="tags">
                  <Input placeholder="逗号分隔，如：助手,治愈,专业" />
                </FormItem>
                <FormItem label="创作者备注" field="creatorNote">
                  <Input.TextArea
                    placeholder="创作者备注信息"
                    autoSize={{ minRows: 2, maxRows: 6 }}
                  />
                </FormItem>
                <FormItem label="角色卡版本" field="specVersion">
                  <Input placeholder="如：v2" />
                </FormItem>
              </div>
            </TabPane>

            <TabPane key="prompt" title="系统提示词">
              <div className="pt-2">
                <FormItem label="系统提示词" field="systemPrompt">
                  <Input.TextArea
                    placeholder="输入系统提示词..."
                    autoSize={{ minRows: 8, maxRows: 30 }}
                    style={{ fontFamily: 'monospace', fontSize: 12 }}
                  />
                </FormItem>
                <FormItem label="开场白" field="openingStatement">
                  <Input.TextArea
                    placeholder="角色的第一句话"
                    autoSize={{ minRows: 2, maxRows: 6 }}
                  />
                </FormItem>
              </div>
            </TabPane>
          </Tabs>
        </Form>
      </Drawer>

      {/* ====== 导入角色卡模态框 ====== */}
      <Modal
        title="导入角色卡"
        visible={importOpen}
        onCancel={() => {
          setImportOpen(false)
          setImportFile(null)
          setImportGroupId(undefined)
        }}
        onOk={() => { void handleImport() }}
        confirmLoading={importSubmitting}
        okText="导入"
        autoFocus={false}
        style={{ width: 480 }}
      >
        <div className="space-y-4">
          <Text type="secondary">
            支持 JSON、PNG（SillyTavern 兼容）、YAML 格式的角色卡文件。
          </Text>
          <FormItem label="选择文件">
            <Upload
              accept=".json,.png,.yaml,.yml"
              showUploadList={true}
              limit={1}
              autoUpload={false}
              onChange={(files) => {
                const f = files[0]?.originFile
                setImportFile(f || null)
              }}
              tip="拖拽或点击选择文件"
            />
          </FormItem>
          <FormItem label="目标组织（可选）">
            <Input
              placeholder="输入组织 ID，留空则导入到默认组织"
              value={importGroupId !== undefined ? String(importGroupId) : ''}
              onChange={(v) => {
                const n = parseInt(v, 10)
                setImportGroupId(Number.isNaN(n) ? undefined : n)
              }}
            />
          </FormItem>
        </div>
      </Modal>
    </div>
  )
}

export default Assistants
