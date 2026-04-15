import { useEffect, useState } from 'react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Modal from '@/components/UI/Modal'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/UI/Select'
import { listAdminCredentials, updateAdminCredentialStatus, deleteAdminCredential, type AdminCredential } from '@/services/adminApi'
import { showAlert } from '@/utils/notification'

const Credentials = () => {
  const [list, setList] = useState<AdminCredential[]>([])
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [detail, setDetail] = useState<AdminCredential | null>(null)

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

  const exportCsv = () => {
    const headers = ['id', 'name', 'userId', 'status', 'llmProvider', 'usageCount', 'createdAt']
    const rows = list.map(i => [i.id, i.name, i.userId, i.status, i.llmProvider || '', i.usageCount, i.createdAt])
    const csv = [headers, ...rows].map(row => row.map(v => `"${String(v ?? '').replaceAll('"', '""')}"`).join(',')).join('\n')
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
                  <Button size="sm" variant="ghost" onClick={() => setDetail(item)}>详情</Button>
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
          </div>
        )}
      </Modal>
    </AdminLayout>
  )
}

export default Credentials
