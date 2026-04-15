import { useEffect, useMemo, useState } from 'react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Modal from '@/components/UI/Modal'
import { showAlert } from '@/utils/notification'

type ListResult = { items: Record<string, any>[]; total: number; page: number; pageSize: number }

interface GenericEntityPageProps {
  title: string
  description: string
  searchPlaceholder: string
  exportName: string
  fetchList: (params: { page: number; pageSize: number; search?: string }) => Promise<ListResult>
  deleteItem: (id: string | number) => Promise<any>
  getId: (item: Record<string, any>) => string | number
}

const GenericEntityPage = ({
  title,
  description,
  searchPlaceholder,
  exportName,
  fetchList,
  deleteItem,
  getId,
}: GenericEntityPageProps) => {
  const [list, setList] = useState<Record<string, any>[]>([])
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [detail, setDetail] = useState<Record<string, any> | null>(null)

  const columns = useMemo(() => {
    const keys = new Set<string>()
    list.forEach((item) => Object.keys(item).slice(0, 8).forEach((k) => keys.add(k)))
    const base = Array.from(keys)
    if (!base.includes('id')) base.unshift('id')
    return base.slice(0, 8)
  }, [list])

  const detailEntries = useMemo(() => {
    if (!detail) return []
    return Object.entries(detail).sort(([a], [b]) => {
      if (a === 'id') return -1
      if (b === 'id') return 1
      if (a.endsWith('_at') && !b.endsWith('_at')) return 1
      if (!a.endsWith('_at') && b.endsWith('_at')) return -1
      return a.localeCompare(b)
    })
  }, [detail])

  const prettyKey = (k: string) =>
    k
      .replaceAll('_', ' ')
      .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
      .replace(/\s+/g, ' ')
      .trim()

  const renderValue = (value: any) => {
    if (value === null || value === undefined || value === '') return '-'
    if (typeof value === 'boolean') return value ? '是' : '否'
    if (value === 0 || value === 1) {
      const n = Number(value)
      if (n === 0 || n === 1) return n === 1 ? '是' : '否'
    }
    if (typeof value === 'string' && /(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})/.test(value)) {
      const d = new Date(value)
      if (!Number.isNaN(d.getTime())) return d.toLocaleString('zh-CN')
    }
    if (typeof value === 'object') return JSON.stringify(value)
    return String(value)
  }

  const fetchData = async () => {
    try {
      setLoading(true)
      const res = await fetchList({ page, pageSize, search: search || undefined })
      setList(res.items || [])
      setTotal(res.total || 0)
    } catch (e: any) {
      showAlert(`加载${title}失败`, 'error', e?.msg || e?.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [page, search])

  const exportCsv = () => {
    if (list.length === 0) return
    const headers = columns
    const rows = list.map((item) => headers.map((h) => item[h] ?? ''))
    const csv = [headers, ...rows]
      .map((row) => row.map((v) => `"${String(v).replaceAll('"', '""')}"`).join(','))
      .join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${exportName}_${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  const onDelete = async (item: Record<string, any>) => {
    try {
      await deleteItem(getId(item))
      showAlert('删除成功', 'success')
      fetchData()
    } catch (e: any) {
      showAlert('删除失败', 'error', e?.msg || e?.message)
    }
  }

  return (
    <AdminLayout title={title} description={description}>
      <Card className="space-y-4">
        <div className="flex gap-3">
          <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder={searchPlaceholder} className="max-w-sm" />
          <Button variant="outline" onClick={fetchData}>刷新</Button>
          <Button variant="outline" onClick={exportCsv}>导出 CSV</Button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b">
                {columns.map((col) => (
                  <th key={col} className="text-left py-2 px-2">{col}</th>
                ))}
                <th className="text-left py-2 px-2">操作</th>
              </tr>
            </thead>
            <tbody>
              {list.map((item, idx) => (
                <tr key={`${getId(item)}-${idx}`} className="border-b">
                  {columns.map((col) => (
                    <td key={col} className="py-2 px-2 max-w-[280px] truncate">{renderValue(item[col])}</td>
                  ))}
                  <td className="py-2 px-2 flex gap-2">
                    <Button size="sm" variant="ghost" onClick={() => setDetail(item)}>详情</Button>
                    <Button size="sm" variant="ghost" className="text-red-600" onClick={() => onDelete(item)}>删除</Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="flex justify-between text-sm">
          <span>共 {total} 条</span>
          <div className="flex gap-2">
            <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>上一页</Button>
            <Button size="sm" variant="outline" disabled={page * pageSize >= total} onClick={() => setPage((p) => p + 1)}>下一页</Button>
          </div>
        </div>
        {loading && <div className="text-sm text-slate-500">加载中...</div>}
      </Card>
      <Modal isOpen={!!detail} onClose={() => setDetail(null)} title={`${title}详情`} size="lg">
        {detail && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-sm">
            {detailEntries.map(([k, v]) => (
              <div key={k} className="rounded-md border border-slate-200 dark:border-slate-700 p-3">
                <div className="text-xs uppercase tracking-wide text-slate-500 mb-1">{prettyKey(k)}</div>
                <span className="break-all text-slate-900 dark:text-slate-100">{renderValue(v)}</span>
              </div>
            ))}
          </div>
        )}
      </Modal>
    </AdminLayout>
  )
}

export default GenericEntityPage
