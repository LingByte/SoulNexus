import React, { useState, useEffect, useCallback } from 'react'
import {
  Table, Button, Modal, Input, Form, Switch, Select,
  Space, Popconfirm, Tag, Card, Empty, Tooltip,
} from '@arco-design/web-react'
import { IconPlus, IconEdit, IconDelete, IconSearch, IconBulb } from '@arco-design/web-react/icon'
import type { WorldInfoEntry } from '@/api/chat'
import {
  listWorldInfo, createWorldInfo, updateWorldInfo, deleteWorldInfo,
} from '@/api/chat'
import { alert } from '@/utils/alert'

const { TextArea } = Input

const WorldInfoPage: React.FC = () => {
  const [entries, setEntries] = useState<WorldInfoEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingEntry, setEditingEntry] = useState<Partial<WorldInfoEntry> | null>(null)
  const [saving, setSaving] = useState(false)

  const fetchEntries = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listWorldInfo()
      if (res.code === 200 && res.data) {
        setEntries(res.data.entries || [])
      }
    } catch (err) {
      alert.error('Failed to load world info entries')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchEntries()
  }, [fetchEntries])

  const handleCreate = () => {
    setEditingEntry({
      name: '',
      content: '',
      keys: '',
      regex: '',
      selective: false,
      constant: false,
      depth: 4,
      order: 0,
      enabled: true,
      agentId: 0,
    })
    setModalVisible(true)
  }

  const handleEdit = (entry: WorldInfoEntry) => {
    setEditingEntry({ ...entry })
    setModalVisible(true)
  }

  const handleSave = async () => {
    if (!editingEntry) return
    if (!editingEntry.name?.trim()) {
      alert.warning('Name is required')
      return
    }
    if (!editingEntry.content?.trim()) {
      alert.warning('Content is required')
      return
    }

    setSaving(true)
    try {
      if (editingEntry.id) {
        const res = await updateWorldInfo(editingEntry.id, editingEntry)
        if (res.code === 200) {
          alert.success('Entry updated')
          fetchEntries()
          setModalVisible(false)
        }
      } else {
        const res = await createWorldInfo(editingEntry)
        if (res.code === 200) {
          alert.success('Entry created')
          fetchEntries()
          setModalVisible(false)
        }
      }
    } catch (err) {
      alert.error('Save failed')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      const res = await deleteWorldInfo(id)
      if (res.code === 200) {
        alert.success('Entry deleted')
        fetchEntries()
      }
    } catch (err) {
      alert.error('Delete failed')
    }
  }

  const columns = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      width: 200,
      render: (text: string, record: WorldInfoEntry) => (
        <div className="flex items-center gap-2">
          <span className="font-medium">{text}</span>
          {record.constant && <Tag size="small" color="purple">Constant</Tag>}
          {record.selective && <Tag size="small" color="blue">Chain</Tag>}
        </div>
      ),
    },
    {
      title: 'Keys / Regex',
      dataIndex: 'keys',
      key: 'keys',
      width: 250,
      render: (_: string, record: WorldInfoEntry) => (
        <div className="space-y-0.5">
          {record.keys && (
            <div className="flex flex-wrap gap-1">
              {record.keys.split(',').filter(Boolean).map((k, i) => (
                <Tag key={i} size="small" color="arcoblue">{k.trim()}</Tag>
              ))}
            </div>
          )}
          {record.regex && (
            <Tooltip content={record.regex}>
              <Tag size="small" color="green">/regex/</Tag>
            </Tooltip>
          )}
        </div>
      ),
    },
    {
      title: 'Content',
      dataIndex: 'content',
      key: 'content',
      ellipsis: true,
      width: 300,
      render: (text: string) => (
        <span className="text-gray-500 text-sm line-clamp-2">{text}</span>
      ),
    },
    {
      title: 'Depth',
      dataIndex: 'depth',
      key: 'depth',
      width: 70,
    },
    {
      title: 'Order',
      dataIndex: 'order',
      key: 'order',
      width: 70,
    },
    {
      title: 'Status',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (enabled: boolean) => (
        <Tag size="small" color={enabled ? 'green' : 'gray'}>
          {enabled ? 'Enabled' : 'Disabled'}
        </Tag>
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 120,
      render: (_: any, record: WorldInfoEntry) => (
        <Space>
          <Button type="text" size="small" icon={<IconEdit />} onClick={() => handleEdit(record)} />
          <Popconfirm title="Delete this entry?" onOk={() => handleDelete(record.id)}>
            <Button type="text" size="small" status="danger" icon={<IconDelete />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <Card
        title={
          <div className="flex items-center gap-2">
            <IconBulb className="text-purple-500" />
            <span>World Info / Lorebook</span>
            <Tag size="small" color="purple">Lightweight Context Injection</Tag>
          </div>
        }
        extra={
          <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
            New Entry
          </Button>
        }
      >
        <Table
          columns={columns}
          data={entries}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 20, sizeCanChange: true }}
          noDataElement={<Empty description="No world info entries yet. Create one to inject lore into conversations." />}
          scroll={{ x: 1100 }}
        />
      </Card>

      {/* Create/Edit Modal */}
      <Modal
        title={editingEntry?.id ? 'Edit World Info Entry' : 'New World Info Entry'}
        visible={modalVisible}
        onOk={handleSave}
        onCancel={() => setModalVisible(false)}
        confirmLoading={saving}
        style={{ width: 640 }}
      >
        {editingEntry && (
          <Form layout="vertical" className="mt-4">
            <Form.Item label="Name" required>
              <Input
                value={editingEntry.name}
                onChange={v => setEditingEntry(prev => prev ? { ...prev, name: v } : null)}
                placeholder="e.g., Magic System, Kingdom of Arathor"
              />
            </Form.Item>

            <Form.Item label="Content" required>
              <TextArea
                value={editingEntry.content}
                onChange={v => setEditingEntry(prev => prev ? { ...prev, content: v } : null)}
                placeholder="The lore content that gets injected into the prompt..."
                rows={4}
              />
            </Form.Item>

            <Form.Item label="Trigger Keywords (comma-separated)">
              <Input
                value={editingEntry.keys}
                onChange={v => setEditingEntry(prev => prev ? { ...prev, keys: v } : null)}
                placeholder="magic, wizard, spell, arcane"
              />
            </Form.Item>

            <Form.Item label="Regex Trigger (optional)">
              <Input
                value={editingEntry.regex}
                onChange={v => setEditingEntry(prev => prev ? { ...prev, regex: v } : null)}
                placeholder="\\b(magic|wizard)\\b"
              />
            </Form.Item>

            <div className="grid grid-cols-2 gap-4">
              <Form.Item label="Depth (chain recursion)">
                <Input
                  type="number"
                  value={String(editingEntry.depth ?? 4)}
                  onChange={v => setEditingEntry(prev => prev ? { ...prev, depth: parseInt(v) || 4 } : null)}
                />
              </Form.Item>

              <Form.Item label="Order (priority)">
                <Input
                  type="number"
                  value={String(editingEntry.order ?? 0)}
                  onChange={v => setEditingEntry(prev => prev ? { ...prev, order: parseInt(v) || 0 } : null)}
                />
              </Form.Item>
            </div>

            <div className="grid grid-cols-3 gap-4">
              <Form.Item label="Enabled">
                <Switch
                  checked={editingEntry.enabled}
                  onChange={v => setEditingEntry(prev => prev ? { ...prev, enabled: v } : null)}
                />
              </Form.Item>

              <Form.Item label="Constant (always active)">
                <Switch
                  checked={editingEntry.constant}
                  onChange={v => setEditingEntry(prev => prev ? { ...prev, constant: v } : null)}
                />
              </Form.Item>

              <Form.Item label="Selective (chain)">
                <Switch
                  checked={editingEntry.selective}
                  onChange={v => setEditingEntry(prev => prev ? { ...prev, selective: v } : null)}
                />
              </Form.Item>
            </div>
          </Form>
        )}
      </Modal>
    </div>
  )
}

export default WorldInfoPage
