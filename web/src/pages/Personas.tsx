import React, { useState, useEffect, useCallback } from 'react'
import {
  Card, Table, Button, Modal, Input, Form, Switch, Space,
  Popconfirm, Tag, Empty, Avatar, Tooltip,
} from '@arco-design/web-react'
import { IconPlus, IconEdit, IconDelete, IconUser, IconStar, IconStarFill } from '@arco-design/web-react/icon'
import type { UserPersona } from '@/api/chat'
import {
  listPersonas, createPersona, updatePersona, deletePersona,
  setDefaultPersona,
} from '@/api/chat'
import { alert } from '@/utils/alert'

const { TextArea } = Input

const PersonasPage: React.FC = () => {
  const [personas, setPersonas] = useState<UserPersona[]>([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingPersona, setEditingPersona] = useState<Partial<UserPersona> | null>(null)
  const [saving, setSaving] = useState(false)

  const fetchPersonas = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listPersonas()
      if (res.code === 200 && res.data) {
        setPersonas(res.data.personas || [])
      }
    } catch (err) {
      alert.error('Failed to load personas')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchPersonas()
  }, [fetchPersonas])

  const handleCreate = () => {
    setEditingPersona({
      name: '',
      description: '',
      personality: '',
      scenario: '',
      avatarUrl: '',
      isDefault: personas.length === 0,
    })
    setModalVisible(true)
  }

  const handleEdit = (persona: UserPersona) => {
    setEditingPersona({ ...persona })
    setModalVisible(true)
  }

  const handleSave = async () => {
    if (!editingPersona) return
    if (!editingPersona.name?.trim()) {
      alert.warning('Name is required')
      return
    }

    setSaving(true)
    try {
      if (editingPersona.id) {
        const res = await updatePersona(editingPersona.id, editingPersona)
        if (res.code === 200) {
          alert.success('Persona updated')
          fetchPersonas()
          setModalVisible(false)
        }
      } else {
        const res = await createPersona(editingPersona)
        if (res.code === 200) {
          alert.success('Persona created')
          fetchPersonas()
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
      const res = await deletePersona(id)
      if (res.code === 200) {
        alert.success('Persona deleted')
        fetchPersonas()
      }
    } catch (err) {
      alert.error('Delete failed')
    }
  }

  const handleSetDefault = async (id: string) => {
    try {
      const res = await setDefaultPersona(id)
      if (res.code === 200) {
        alert.success('Default persona set')
        fetchPersonas()
      }
    } catch (err) {
      alert.error('Failed to set default')
    }
  }

  const columns = [
    {
      title: '',
      key: 'default',
      width: 50,
      render: (_: any, record: UserPersona) =>
        record.isDefault ? (
          <IconStarFill className="text-yellow-500 text-lg" />
        ) : (
          <Tooltip content="Set as default">
            <IconStar
              className="text-gray-300 hover:text-yellow-500 cursor-pointer text-lg"
              onClick={() => handleSetDefault(record.id)}
            />
          </Tooltip>
        ),
    },
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      width: 180,
      render: (text: string, record: UserPersona) => (
        <div className="flex items-center gap-2">
          {record.avatarUrl ? (
            <Avatar size={28} shape="square">
              <img src={record.avatarUrl} alt={text} />
            </Avatar>
          ) : (
            <Avatar size={28} className="bg-purple-100 text-purple-600">
              <IconUser />
            </Avatar>
          )}
          <span className="font-medium">{text}</span>
        </div>
      ),
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      width: 250,
      render: (text: string) => (
        <span className="text-gray-500 text-sm">{text || '—'}</span>
      ),
    },
    {
      title: 'Personality',
      dataIndex: 'personality',
      key: 'personality',
      ellipsis: true,
      width: 200,
      render: (text: string) => (
        <span className="text-gray-500 text-sm">{text || '—'}</span>
      ),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 140,
      render: (text: string) => {
        const d = new Date(text)
        return d.toLocaleDateString()
      },
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 150,
      render: (_: any, record: UserPersona) => (
        <Space>
          <Button type="text" size="small" icon={<IconEdit />} onClick={() => handleEdit(record)} />
          {!record.isDefault && (
            <Popconfirm title="Delete this persona?" onOk={() => handleDelete(record.id)}>
              <Button type="text" size="small" status="danger" icon={<IconDelete />} />
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <Card
        title={
          <div className="flex items-center gap-2">
            <IconUser className="text-blue-500" />
            <span>User Personas</span>
            <Tag size="small" color="blue">Role-Play Identity</Tag>
          </div>
        }
        extra={
          <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
            New Persona
          </Button>
        }
      >
        <Table
          columns={columns}
          data={personas}
          rowKey="id"
          loading={loading}
          pagination={false}
          noDataElement={
            <Empty description="No personas yet. Define your identity for role-play conversations." />
          }
          scroll={{ x: 1000 }}
        />
      </Card>

      {/* Create/Edit Modal */}
      <Modal
        title={editingPersona?.id ? 'Edit Persona' : 'New Persona'}
        visible={modalVisible}
        onOk={handleSave}
        onCancel={() => setModalVisible(false)}
        confirmLoading={saving}
        style={{ width: 640 }}
      >
        {editingPersona && (
          <Form layout="vertical" className="mt-4">
            <Form.Item label="Name" required>
              <Input
                value={editingPersona.name}
                onChange={v => setEditingPersona(prev => prev ? { ...prev, name: v } : null)}
                placeholder="e.g., A wandering knight, a curious scholar..."
              />
            </Form.Item>

            <Form.Item label="Avatar URL">
              <Input
                value={editingPersona.avatarUrl}
                onChange={v => setEditingPersona(prev => prev ? { ...prev, avatarUrl: v } : null)}
                placeholder="https://example.com/avatar.png"
              />
            </Form.Item>

            <Form.Item label="Description (appearance, background)">
              <TextArea
                value={editingPersona.description}
                onChange={v => setEditingPersona(prev => prev ? { ...prev, description: v } : null)}
                placeholder="Physical appearance, backstory, occupation..."
                rows={3}
              />
            </Form.Item>

            <Form.Item label="Personality (traits, temperament, speech patterns)">
              <TextArea
                value={editingPersona.personality}
                onChange={v => setEditingPersona(prev => prev ? { ...prev, personality: v } : null)}
                placeholder="Brave but reckless, speaks in short sentences, always polite..."
                rows={3}
              />
            </Form.Item>

            <Form.Item label="Scenario (current situation / context)">
              <TextArea
                value={editingPersona.scenario}
                onChange={v => setEditingPersona(prev => prev ? { ...prev, scenario: v } : null)}
                placeholder="You are at a bustling tavern, seeking information about..."
                rows={2}
              />
            </Form.Item>

            <Form.Item label="Set as default persona">
              <Switch
                checked={editingPersona.isDefault}
                onChange={v => setEditingPersona(prev => prev ? { ...prev, isDefault: v } : null)}
              />
            </Form.Item>
          </Form>
        )}
      </Modal>
    </div>
  )
}

export default PersonasPage
