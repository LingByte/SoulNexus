import { useEffect, useState } from 'react'
import { Plus, Edit, Trash2, RefreshCw, Save, KeyRound } from 'lucide-react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Badge from '@/components/UI/Badge'
import EmptyState from '@/components/UI/EmptyState'
import Modal from '@/components/UI/Modal'
import ConfirmDialog from '@/components/UI/ConfirmDialog'
import { showAlert } from '@/utils/notification'
import {
  listOAuthClients,
  createOAuthClient,
  updateOAuthClient,
  deleteOAuthClient,
  type OAuthClient,
  type UpsertOAuthClientRequest,
} from '@/services/adminApi'

const OAuthClients = () => {
  const [clients, setClients] = useState<OAuthClient[]>([])
  const [loading, setLoading] = useState(false)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [selectedClient, setSelectedClient] = useState<OAuthClient | null>(null)
  const [editingClient, setEditingClient] = useState<OAuthClient | null>(null)
  const [formData, setFormData] = useState<UpsertOAuthClientRequest>({
    clientId: '',
    clientSecret: '',
    name: '',
    redirectUri: '',
    status: 1,
  })

  const fetchClients = async () => {
    try {
      setLoading(true)
      const result = await listOAuthClients({ page: 1, pageSize: 200 })
      setClients(result.clients || [])
    } catch (error: any) {
      showAlert('获取 OAuth 客户端失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchClients()
  }, [])

  const openCreateModal = () => {
    setFormData({
      clientId: '',
      clientSecret: '',
      name: '',
      redirectUri: '',
      status: 1,
    })
    setShowCreateModal(true)
  }

  const submitCreate = async () => {
    if (!formData.clientId || !formData.name || !formData.redirectUri) {
      showAlert('请填写 clientId、名称和回调地址', 'error')
      return
    }
    try {
      await createOAuthClient(formData)
      showAlert('创建 OAuth 客户端成功', 'success')
      setShowCreateModal(false)
      fetchClients()
    } catch (error: any) {
      showAlert('创建 OAuth 客户端失败', 'error', error?.msg || error?.message)
    }
  }

  const openEditModal = (client: OAuthClient) => {
    setEditingClient(client)
    setFormData({
      clientId: client.clientId,
      clientSecret: client.clientSecret || '',
      name: client.name,
      redirectUri: client.redirectUri,
      status: client.status,
    })
    setShowEditModal(true)
  }

  const submitEdit = async () => {
    if (!editingClient) return
    try {
      await updateOAuthClient(editingClient.id, formData)
      showAlert('更新 OAuth 客户端成功', 'success')
      setShowEditModal(false)
      setEditingClient(null)
      fetchClients()
    } catch (error: any) {
      showAlert('更新 OAuth 客户端失败', 'error', error?.msg || error?.message)
    }
  }

  const askDelete = (client: OAuthClient) => {
    setSelectedClient(client)
    setShowDeleteConfirm(true)
  }

  const confirmDelete = async () => {
    if (!selectedClient) return
    try {
      await deleteOAuthClient(selectedClient.id)
      showAlert('删除 OAuth 客户端成功', 'success')
      setShowDeleteConfirm(false)
      setSelectedClient(null)
      fetchClients()
    } catch (error: any) {
      showAlert('删除 OAuth 客户端失败', 'error', error?.msg || error?.message)
    }
  }

  return (
    <AdminLayout>
      <div className="space-y-6">
        <Card>
          <div className="flex items-center justify-between">
            <div className="text-sm text-slate-600 dark:text-slate-300">管理后台 OAuth Client 列表</div>
            <div className="flex items-center gap-2">
              <Button variant="outline" onClick={fetchClients} leftIcon={<RefreshCw className="w-4 h-4" />}>
                刷新
              </Button>
              <Button onClick={openCreateModal} leftIcon={<Plus className="w-4 h-4" />}>
                新建客户端
              </Button>
            </div>
          </div>
        </Card>

        <Card>
          {loading ? (
            <div className="flex justify-center py-12">
              <RefreshCw className="w-6 h-6 animate-spin text-slate-400" />
            </div>
          ) : clients.length === 0 ? (
            <EmptyState icon={KeyRound} title="暂无 OAuth 客户端" description="点击上方按钮创建" />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-slate-700">
                    <th className="text-left py-3 px-4">Client ID</th>
                    <th className="text-left py-3 px-4">名称</th>
                    <th className="text-left py-3 px-4">回调地址</th>
                    <th className="text-left py-3 px-4">状态</th>
                    <th className="text-right py-3 px-4">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {clients.map((client) => (
                    <tr key={client.id} className="border-b border-slate-100 dark:border-slate-800">
                      <td className="py-3 px-4 font-mono text-xs">{client.clientId}</td>
                      <td className="py-3 px-4">{client.name}</td>
                      <td className="py-3 px-4 text-xs whitespace-pre-wrap break-all">
                        {client.redirectUri?.split(';').map((uri) => uri.trim()).filter(Boolean).join('\n')}
                      </td>
                      <td className="py-3 px-4">
                        <Badge variant={client.status === 1 ? 'success' : 'secondary'}>
                          {client.status === 1 ? '启用' : '禁用'}
                        </Badge>
                      </td>
                      <td className="py-3 px-4">
                        <div className="flex items-center justify-end gap-2">
                          <Button variant="ghost" size="sm" onClick={() => openEditModal(client)} leftIcon={<Edit className="w-4 h-4" />}>
                            编辑
                          </Button>
                          <Button variant="ghost" size="sm" onClick={() => askDelete(client)} leftIcon={<Trash2 className="w-4 h-4" />}>
                            删除
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>

        <Modal isOpen={showCreateModal} onClose={() => setShowCreateModal(false)} title="新建 OAuth 客户端" size="lg">
          <div className="space-y-4">
            <Input label="Client ID *" value={formData.clientId || ''} onChange={(e) => setFormData({ ...formData, clientId: e.target.value })} />
            <Input label="名称 *" value={formData.name || ''} onChange={(e) => setFormData({ ...formData, name: e.target.value })} />
            <Input label="Client Secret（可选）" value={formData.clientSecret || ''} onChange={(e) => setFormData({ ...formData, clientSecret: e.target.value })} />
            <Input
              label="Redirect URI *（多个用 ; 分隔）"
              placeholder="https://a.com/callback;https://b.com/callback"
              value={formData.redirectUri || ''}
              onChange={(e) => setFormData({ ...formData, redirectUri: e.target.value })}
            />
            <div className="flex justify-end gap-3">
              <Button variant="outline" onClick={() => setShowCreateModal(false)}>取消</Button>
              <Button onClick={submitCreate} leftIcon={<Save className="w-4 h-4" />}>创建</Button>
            </div>
          </div>
        </Modal>

        <Modal isOpen={showEditModal} onClose={() => setShowEditModal(false)} title="编辑 OAuth 客户端" size="lg">
          <div className="space-y-4">
            <Input label="Client ID *" value={formData.clientId || ''} onChange={(e) => setFormData({ ...formData, clientId: e.target.value })} />
            <Input label="名称 *" value={formData.name || ''} onChange={(e) => setFormData({ ...formData, name: e.target.value })} />
            <Input label="Client Secret（留空则不改）" value={formData.clientSecret || ''} onChange={(e) => setFormData({ ...formData, clientSecret: e.target.value })} />
            <Input
              label="Redirect URI *（多个用 ; 分隔）"
              placeholder="https://a.com/callback;https://b.com/callback"
              value={formData.redirectUri || ''}
              onChange={(e) => setFormData({ ...formData, redirectUri: e.target.value })}
            />
            <div className="flex justify-end gap-3">
              <Button variant="outline" onClick={() => setShowEditModal(false)}>取消</Button>
              <Button onClick={submitEdit} leftIcon={<Save className="w-4 h-4" />}>保存</Button>
            </div>
          </div>
        </Modal>

        <ConfirmDialog
          isOpen={showDeleteConfirm}
          onClose={() => setShowDeleteConfirm(false)}
          onConfirm={confirmDelete}
          title="确认删除"
          message={`确定删除 OAuth 客户端 "${selectedClient?.name}" 吗？`}
          confirmText="删除"
          cancelText="取消"
          variant="danger"
        />
      </div>
    </AdminLayout>
  )
}

export default OAuthClients
