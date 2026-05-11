// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// OAuth 客户端管理：Arco Table + Form Modal。
import { useEffect, useState } from 'react'
import {
  Table,
  Input,
  Button,
  Tag,
  Modal,
  Form,
  Popconfirm,
  Message,
  Space,
} from '@arco-design/web-react'
import { Plus, Edit, Trash2, RefreshCw } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listOAuthClients,
  createOAuthClient,
  updateOAuthClient,
  deleteOAuthClient,
  type OAuthClient,
  type UpsertOAuthClientRequest,
} from '@/services/adminApi'

const FormItem = Form.Item

const OAuthClients = () => {
  const [clients, setClients] = useState<OAuthClient[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<OAuthClient | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm<UpsertOAuthClientRequest>()

  const fetchClients = async () => {
    setLoading(true)
    try {
      const result = await listOAuthClients({ page: 1, pageSize: 200 })
      setClients(result.clients || [])
    } catch (e: any) {
      Message.error(`获取 OAuth 客户端失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchClients()
  }, [])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ status: 1 })
    setModalOpen(true)
  }

  const openEdit = (c: OAuthClient) => {
    setEditing(c)
    form.setFieldsValue({
      clientId: c.clientId,
      clientSecret: c.clientSecret || '',
      name: c.name,
      redirectUri: c.redirectUri,
      status: c.status,
    })
    setModalOpen(true)
  }

  const save = async () => {
    try {
      const values = await form.validate()
      setSaving(true)
      if (editing) {
        await updateOAuthClient(editing.id, values)
        Message.success('已更新')
      } else {
        await createOAuthClient(values)
        Message.success('已创建')
      }
      setModalOpen(false)
      fetchClients()
    } catch (e: any) {
      if (e?.message || e?.msg) Message.error(`保存失败：${e?.msg || e?.message}`)
    } finally {
      setSaving(false)
    }
  }

  const onDelete = async (c: OAuthClient) => {
    try {
      await deleteOAuthClient(c.id)
      Message.success('已删除')
      fetchClients()
    } catch (e: any) {
      Message.error(`删除失败：${e?.msg || e?.message || e}`)
    }
  }

  const columns = [
    {
      title: 'Client ID',
      dataIndex: 'clientId',
      width: 240,
      ellipsis: true,
      render: (v: string) => <span style={{ fontFamily: 'ui-monospace, monospace', fontSize: 12 }}>{v}</span>,
    },
    { title: '名称', dataIndex: 'name', width: 200, ellipsis: true },
    {
      title: '回调地址',
      dataIndex: 'redirectUri',
      ellipsis: true,
      render: (v: string) => {
        const list = (v || '').split(';').map((s) => s.trim()).filter(Boolean)
        return (
          <div className="flex flex-col gap-0.5 text-xs break-all">
            {list.map((u, i) => <div key={i}>{u}</div>)}
          </div>
        )
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (v: number) => (v === 1 ? <Tag color="green">启用</Tag> : <Tag color="gray">禁用</Tag>),
    },
    {
      title: '操作',
      key: '__actions__',
      width: 180,
      fixed: 'right' as const,
      render: (_: any, c: OAuthClient) => (
        <Space size="mini">
          <Button size="mini" type="text" onClick={() => openEdit(c)}>
            <span className="inline-flex items-center gap-1"><Edit size={14} /> 编辑</span>
          </Button>
          <Popconfirm title={`确认删除 ${c.name}？`} okText="删除" cancelText="取消" onOk={() => onDelete(c)}>
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
        title="OAuth 客户端"
        description="管理可以接入本平台 OAuth 授权的第三方客户端。"
        actions={
          <Space>
            <Button onClick={fetchClients}>
              <span className="inline-flex items-center gap-1"><RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新</span>
            </Button>
            <Button type="primary" onClick={openCreate}>
              <span className="inline-flex items-center gap-1"><Plus size={14} /> 新建客户端</span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(c: OAuthClient) => String(c.id)}
        loading={loading}
        columns={columns}
        data={clients}
        scroll={{ x: 1100 }}
        pagination={{
          current: page,
          pageSize,
          total: clients.length,
          showTotal: (t: number) => `共 ${t} 条`,
          sizeCanChange: true,
          sizeOptions: [10, 20, 50, 100],
          onChange: (p: number, ps: number) => { setPage(p); setPageSize(ps) },
        }}
      />

      <Modal
        title={editing ? '编辑 OAuth 客户端' : '新建 OAuth 客户端'}
        visible={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={save}
        confirmLoading={saving}
        okText="保存"
        cancelText="取消"
        autoFocus={false}
        style={{ width: 580 }}
      >
        <Form form={form} layout="vertical" autoComplete="off">
          <FormItem label="Client ID" field="clientId" rules={[{ required: true, message: '请填写 Client ID' }]}>
            <Input disabled={!!editing} placeholder="例如：myapp_web" />
          </FormItem>
          <FormItem label="名称" field="name" rules={[{ required: true, message: '请填写名称' }]}>
            <Input placeholder="客户端展示名" />
          </FormItem>
          <FormItem label={editing ? 'Client Secret（留空则不改）' : 'Client Secret（可选）'} field="clientSecret">
            <Input.Password placeholder="OAuth 客户端密钥" />
          </FormItem>
          <FormItem
            label="Redirect URI（多个用 ; 分隔）"
            field="redirectUri"
            rules={[{ required: true, message: '请填写回调地址' }]}
          >
            <Input.TextArea
              rows={3}
              placeholder="https://a.com/callback;https://b.com/callback"
            />
          </FormItem>
        </Form>
      </Modal>
    </div>
  )
}

export default OAuthClients
