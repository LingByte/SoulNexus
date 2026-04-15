import { useEffect, useState } from 'react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Modal from '@/components/UI/Modal'
import { listAdminJSTemplates, updateAdminJSTemplate, deleteAdminJSTemplate, type AdminJSTemplate } from '@/services/adminApi'
import { showAlert } from '@/utils/notification'

const JSTemplates = () => {
  const [list, setList] = useState<AdminJSTemplate[]>([])
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [detail, setDetail] = useState<AdminJSTemplate | null>(null)

  const fetchData = async () => {
    try {
      const res = await listAdminJSTemplates({ search, page, pageSize })
      setList(res.templates || [])
      setTotal(res.total || 0)
    } catch (e: any) {
      showAlert('加载 JS 模版失败', 'error', e?.msg || e?.message)
    }
  }
  useEffect(() => { fetchData() }, [search, page])

  const exportCsv = () => {
    const headers = ['id', 'name', 'jsSourceId', 'type', 'status', 'version', 'user_id', 'created_at']
    const rows = list.map(t => [t.id, t.name, t.jsSourceId, t.type, t.status, t.version, t.user_id, t.created_at])
    const csv = [headers, ...rows].map(row => row.map(v => `"${String(v ?? '').replaceAll('"', '""')}"`).join(',')).join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `js_templates_${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <AdminLayout title="JS 模版管理" description="管理系统中的 JS 模版">
      <Card className="space-y-4">
        <div className="flex gap-3">
          <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="搜索模版名 / Source ID" className="max-w-sm" />
          <Button variant="outline" onClick={fetchData}>刷新</Button>
          <Button variant="outline" onClick={exportCsv}>导出 CSV</Button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead><tr className="border-b"><th className="text-left py-2">ID</th><th className="text-left">名称</th><th className="text-left">类型</th><th className="text-left">状态</th><th className="text-left">版本</th><th className="text-left">操作</th></tr></thead>
            <tbody>{list.map(t => (
              <tr key={t.id} className="border-b">
                <td className="py-2">{t.id}</td><td>{t.name}</td><td>{t.type}</td><td>{t.status}</td><td>{t.version}</td>
                <td className="flex gap-2 py-2">
                  <Button size="sm" variant="ghost" onClick={() => setDetail(t)}>详情</Button>
                  <Button size="sm" variant="outline" onClick={async () => { await updateAdminJSTemplate(t.id, { status: t.status === 'active' ? 'archived' : 'active' }); fetchData() }}>
                    {t.status === 'active' ? '归档' : '启用'}
                  </Button>
                  <Button size="sm" variant="ghost" className="text-red-600" onClick={async () => { await deleteAdminJSTemplate(t.id); fetchData() }}>删除</Button>
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
      </Card>
      <Modal isOpen={!!detail} onClose={() => setDetail(null)} title="模版详情" size="md">
        {detail && (
          <div className="space-y-2 text-sm">
            <div>ID: {detail.id}</div>
            <div>名称: {detail.name}</div>
            <div>Source ID: {detail.jsSourceId}</div>
            <div>类型: {detail.type}</div>
            <div>状态: {detail.status}</div>
            <div>版本: {detail.version}</div>
            <div>用户: {detail.user_id}</div>
            <div>创建时间: {detail.created_at}</div>
          </div>
        )}
      </Modal>
    </AdminLayout>
  )
}

export default JSTemplates
