import { useEffect, useState } from 'react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Modal from '@/components/UI/Modal'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/UI/Select'
import { listAdminBills, updateAdminBill, deleteAdminBill, type AdminBill } from '@/services/adminApi'
import { showAlert } from '@/utils/notification'

const Bills = () => {
  const [list, setList] = useState<AdminBill[]>([])
  const [status, setStatus] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [detail, setDetail] = useState<AdminBill | null>(null)

  const fetchData = async () => {
    try {
      const res = await listAdminBills({ status: status || undefined, page, pageSize, search: search || undefined } as any)
      const all = res.bills || []
      const filtered = search ? all.filter(b => b.billNo.includes(search) || b.title.includes(search)) : all
      setList(filtered)
      setTotal(res.total || filtered.length)
    } catch (e: any) {
      showAlert('加载账单失败', 'error', e?.msg || e?.message)
    }
  }
  useEffect(() => { fetchData() }, [status, search, page])

  const exportCsv = () => {
    const headers = ['id', 'billNo', 'title', 'userId', 'status', 'startTime', 'endTime', 'totalLLMCalls', 'totalLLMTokens', 'totalAPICalls']
    const rows = list.map(b => [b.id, b.billNo, b.title, b.userId, b.status, b.startTime, b.endTime, b.totalLLMCalls, b.totalLLMTokens, b.totalAPICalls])
    const csv = [headers, ...rows].map(row => row.map(v => `"${String(v ?? '').replaceAll('"', '""')}"`).join(',')).join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `bills_${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <AdminLayout title="账单管理" description="管理账单状态与信息">
      <Card className="space-y-4">
        <div className="flex gap-3">
          <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="搜索账单编号/标题" className="max-w-sm" />
          <Select value={status} onValueChange={setStatus}>
            <SelectTrigger className="w-40"><SelectValue placeholder="全部状态" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="">全部状态</SelectItem><SelectItem value="draft">draft</SelectItem><SelectItem value="generated">generated</SelectItem><SelectItem value="exported">exported</SelectItem><SelectItem value="archived">archived</SelectItem>
            </SelectContent>
          </Select>
          <Button variant="outline" onClick={fetchData}>刷新</Button>
          <Button variant="outline" onClick={exportCsv}>导出 CSV</Button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead><tr className="border-b"><th className="text-left py-2">编号</th><th className="text-left">标题</th><th className="text-left">用户</th><th className="text-left">状态</th><th className="text-left">LLM调用</th><th className="text-left">操作</th></tr></thead>
            <tbody>{list.map(b => (
              <tr key={b.id} className="border-b">
                <td className="py-2">{b.billNo}</td><td>{b.title}</td><td>{b.userId}</td><td>{b.status}</td><td>{b.totalLLMCalls}</td>
                <td className="flex gap-2 py-2">
                  <Button size="sm" variant="ghost" onClick={() => setDetail(b)}>详情</Button>
                  <Button size="sm" variant="outline" onClick={async () => { await updateAdminBill(b.id, { status: b.status === 'archived' ? 'generated' : 'archived' }); fetchData() }}>{b.status === 'archived' ? '恢复' : '归档'}</Button>
                  <Button size="sm" variant="ghost" className="text-red-600" onClick={async () => { await deleteAdminBill(b.id); fetchData() }}>删除</Button>
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
      <Modal isOpen={!!detail} onClose={() => setDetail(null)} title="账单详情" size="md">
        {detail && (
          <div className="space-y-2 text-sm">
            <div>账单编号: {detail.billNo}</div>
            <div>标题: {detail.title}</div>
            <div>状态: {detail.status}</div>
            <div>用户: {detail.userId}</div>
            <div>开始: {detail.startTime}</div>
            <div>结束: {detail.endTime}</div>
            <div>LLM 调用: {detail.totalLLMCalls}</div>
            <div>Token: {detail.totalLLMTokens}</div>
            <div>API 调用: {detail.totalAPICalls}</div>
          </div>
        )}
      </Modal>
    </AdminLayout>
  )
}

export default Bills
