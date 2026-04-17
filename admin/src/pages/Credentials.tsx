import { useEffect, useState } from 'react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Modal from '@/components/UI/Modal'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/UI/Select'
import { listAdminCredentials, updateAdminCredentialStatus, deleteAdminCredential, type AdminCredential } from '@/services/adminApi'
import { showAlert } from '@/utils/notification'

type CredentialEditForm = {
  expiresAt: string
  tokenQuota: number
  requestQuota: number
  useNativeQuota: boolean
  unlimitedQuota: boolean
}

const DATE_NEVER = '1970-01-01 07:59:59'

function formatDateTimeInput(value?: string): string {
  if (!value) return ''
  const dt = new Date(value)
  if (Number.isNaN(dt.getTime())) return value
  const y = dt.getFullYear()
  const m = `${dt.getMonth() + 1}`.padStart(2, '0')
  const d = `${dt.getDate()}`.padStart(2, '0')
  const h = `${dt.getHours()}`.padStart(2, '0')
  const mi = `${dt.getMinutes()}`.padStart(2, '0')
  const s = `${dt.getSeconds()}`.padStart(2, '0')
  return `${y}-${m}-${d} ${h}:${mi}:${s}`
}

function addDays(days: number): string {
  const now = new Date()
  now.setDate(now.getDate() + days)
  return formatDateTimeInput(now.toISOString())
}

const Credentials = () => {
  const [list, setList] = useState<AdminCredential[]>([])
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [detail, setDetail] = useState<AdminCredential | null>(null)
  const [saving, setSaving] = useState(false)
  const [editForm, setEditForm] = useState<CredentialEditForm>({
    expiresAt: '',
    tokenQuota: 0,
    requestQuota: 0,
    useNativeQuota: false,
    unlimitedQuota: true,
  })

  const fetchData = async () => {
    try {
      setLoading(true)
      const res = await listAdminCredentials({ status: status || undefined, search: search || undefined, page, pageSize })
      setList(res.credentials || [])
      setTotal(res.total || 0)
    } catch (e: any) {
      showAlert('加载密钥失败', 'error', e?.msg || e?.message)
    } finally {
      setLoading(false)
    }
  }
  useEffect(() => { fetchData() }, [status, search, page])

  const updateStatus = async (id: number, next: 'active' | 'banned' | 'suspended') => {
    await updateAdminCredentialStatus(id, { status: next })
    fetchData()
  }

  const openDetail = (item: AdminCredential) => {
    setDetail(item)
    setEditForm({
      expiresAt: formatDateTimeInput(item.expiresAt),
      tokenQuota: item.tokenQuota || 0,
      requestQuota: item.requestQuota || 0,
      useNativeQuota: !!item.useNativeQuota,
      unlimitedQuota: item.unlimitedQuota !== false,
    })
  }

  const saveCredentialSettings = async () => {
    if (!detail) return
    try {
      setSaving(true)
      await updateAdminCredentialStatus(detail.id, {
        status: detail.status,
        expiresAt: editForm.expiresAt.trim(),
        tokenQuota: Math.max(0, Number(editForm.tokenQuota || 0)),
        requestQuota: Math.max(0, Number(editForm.requestQuota || 0)),
        useNativeQuota: !!editForm.useNativeQuota,
        unlimitedQuota: !!editForm.unlimitedQuota,
      })
      showAlert('密钥设置已保存', 'success')
      setDetail(null)
      fetchData()
    } catch (e: any) {
      showAlert('保存失败', 'error', e?.msg || e?.message)
    } finally {
      setSaving(false)
    }
  }

  const exportCsv = () => {
    const headers = ['id', 'name', 'userId', 'status', 'llmProvider', 'usageCount', 'createdAt']
    const rows = list.map(i => [i.id, i.name, i.userId, i.status, i.llmProvider || '', i.usageCount, i.createdAt])
    const csv = [headers, ...rows].map(row => row.map(v => `"${String(v ?? '').replace(/"/g, '""')}"`).join(',')).join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `credentials_${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <AdminLayout title="密钥管理" description="管理用户 API 密钥状态">
      <Card className="space-y-4">
        <div className="flex gap-3">
          <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="搜索名称/API Key/Provider" className="max-w-sm" />
          <Select value={status} onValueChange={setStatus}>
            <SelectTrigger className="w-40"><SelectValue placeholder="全部状态" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="">全部状态</SelectItem><SelectItem value="active">active</SelectItem><SelectItem value="banned">banned</SelectItem><SelectItem value="suspended">suspended</SelectItem>
            </SelectContent>
          </Select>
          <Button variant="outline" onClick={fetchData}>刷新</Button>
          <Button variant="outline" onClick={exportCsv}>导出 CSV</Button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead><tr className="border-b"><th className="text-left py-2">ID</th><th className="text-left">名称</th><th className="text-left">所属用户</th><th className="text-left">状态</th><th className="text-left">操作</th></tr></thead>
            <tbody>{list.map(item => (
              <tr key={item.id} className="border-b">
                <td className="py-2">{item.id}</td><td>{item.name}</td><td>{item.userId}</td><td>{item.status}</td>
                <td className="flex gap-2 py-2">
                  <Button size="sm" variant="ghost" onClick={() => openDetail(item)}>详情</Button>
                  <Button size="sm" variant="outline" onClick={() => updateStatus(item.id, 'active')}>启用</Button>
                  <Button size="sm" variant="outline" onClick={() => updateStatus(item.id, 'suspended')}>暂停</Button>
                  <Button size="sm" variant="outline" onClick={() => updateStatus(item.id, 'banned')}>封禁</Button>
                  <Button size="sm" variant="ghost" className="text-red-600" onClick={async () => { await deleteAdminCredential(item.id); fetchData() }}>删除</Button>
                </td>
              </tr>
            ))}</tbody>
          </table>
        </div>
        <div className="flex justify-between text-sm">
          <span>共 {total} 条</span>
          <div className="flex gap-2">
            <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>上一页</Button>
            <Button size="sm" variant="outline" disabled={page * pageSize >= total} onClick={() => setPage(p => p + 1)}>下一页</Button>
          </div>
        </div>
        {loading && <div className="text-sm text-slate-500">加载中...</div>}
      </Card>
      <Modal isOpen={!!detail} onClose={() => setDetail(null)} title="密钥详情" size="md">
        {detail && (
          <div className="space-y-2 text-sm">
            <div>ID: {detail.id}</div>
            <div>名称: {detail.name}</div>
            <div>所属用户: {detail.userId}</div>
            <div>状态: {detail.status}</div>
            <div>API Key: {detail.apiKey}</div>
            <div>Provider: {detail.llmProvider || '-'}</div>
            <div>使用次数: {detail.usageCount}</div>
            <div>最近使用: {detail.lastUsedAt || '-'}</div>
            <div className="pt-2 border-t mt-2">
              <div className="font-medium mb-2">过期时间快捷设置</div>
              <div className="flex flex-wrap gap-2 mb-2">
                <Button size="sm" variant="outline" onClick={() => setEditForm(prev => ({ ...prev, expiresAt: addDays(1) }))}>+1天</Button>
                <Button size="sm" variant="outline" onClick={() => setEditForm(prev => ({ ...prev, expiresAt: addDays(7) }))}>+7天</Button>
                <Button size="sm" variant="outline" onClick={() => setEditForm(prev => ({ ...prev, expiresAt: addDays(30) }))}>+30天</Button>
                <Button size="sm" variant="outline" onClick={() => setEditForm(prev => ({ ...prev, expiresAt: DATE_NEVER }))}>1970-01-01 07:59:59</Button>
                <Button size="sm" variant="ghost" onClick={() => setEditForm(prev => ({ ...prev, expiresAt: '' }))}>清空</Button>
              </div>
              <Input
                label="过期时间"
                value={editForm.expiresAt}
                onChange={(e) => setEditForm(prev => ({ ...prev, expiresAt: e.target.value }))}
                placeholder="YYYY-MM-DD HH:MM:SS"
              />
            </div>

            <div className="pt-2 border-t mt-2">
              <div className="font-medium mb-2">令牌分组额度设置</div>
              <div className="space-y-2">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={editForm.useNativeQuota}
                    onChange={(e) => setEditForm(prev => ({ ...prev, useNativeQuota: e.target.checked }))}
                  />
                  <span>▸ 使用原生额度输入</span>
                </label>
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={editForm.unlimitedQuota}
                    onChange={(e) => setEditForm(prev => ({ ...prev, unlimitedQuota: e.target.checked }))}
                  />
                  <span>无限额度</span>
                </label>
                <Input
                  label="设置令牌可用额度"
                  type="number"
                  value={String(editForm.tokenQuota)}
                  onChange={(e) => setEditForm(prev => ({ ...prev, tokenQuota: Number(e.target.value || 0) }))}
                />
                <Input
                  label="设置令牌可用数量"
                  type="number"
                  value={String(editForm.requestQuota)}
                  onChange={(e) => setEditForm(prev => ({ ...prev, requestQuota: Number(e.target.value || 0) }))}
                />
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-3">
              <Button variant="outline" onClick={() => setDetail(null)}>取消</Button>
              <Button variant="primary" onClick={saveCredentialSettings} disabled={saving}>
                {saving ? '保存中...' : '保存设置'}
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </AdminLayout>
  )
}

export default Credentials
