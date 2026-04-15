import { useEffect, useMemo, useState } from 'react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Modal from '@/components/UI/Modal'
import { deleteAdminDevice, listAdminDevices } from '@/services/adminApi'
import { showAlert } from '@/utils/notification'

type DeviceRow = Record<string, any>

const toText = (v: any) => {
  if (v === null || v === undefined || v === '') return '-'
  if (typeof v === 'object') return JSON.stringify(v)
  return String(v)
}

const yesNo = (v: any) => {
  if (v === true || v === 1 || v === '1') return '是'
  if (v === false || v === 0 || v === '0') return '否'
  return '-'
}

const Devices = () => {
  const [list, setList] = useState<DeviceRow[]>([])
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [detail, setDetail] = useState<DeviceRow | null>(null)

  const fetchData = async () => {
    try {
      setLoading(true)
      const res = await listAdminDevices({ page, pageSize, search: search || undefined })
      setList(res.items || [])
      setTotal(res.total || 0)
    } catch (e: any) {
      showAlert('加载设备列表失败', 'error', e?.msg || e?.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [page, search])

  const exportCsv = () => {
    const headers = [
      'id', 'alias', 'device_name', 'board', 'app_version', 'assistant_id', 'is_online', 'last_connected',
      'error_count', 'last_error_at', 'created_at'
    ]
    const rows = list.map((i) => headers.map((h) => toText(i[h])))
    const csv = [headers, ...rows].map((row) => row.map((v) => `"${String(v).replaceAll('"', '""')}"`).join(',')).join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `devices_${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  const deleteOne = async (item: DeviceRow) => {
    try {
      await deleteAdminDevice(String(item.id))
      showAlert('删除设备成功', 'success')
      fetchData()
    } catch (e: any) {
      showAlert('删除设备失败', 'error', e?.msg || e?.message)
    }
  }

  const baseInfo = useMemo(() => {
    if (!detail) return []
    return [
      ['设备ID', detail.id],
      ['别名', detail.alias],
      ['设备名称', detail.device_name],
      ['Board', detail.board],
      ['App Version', detail.app_version],
      ['Assistant ID', detail.assistant_id],
      ['组织ID', detail.group_id],
      ['自动更新', yesNo(detail.auto_update)],
      ['创建时间', detail.created_at],
    ]
  }, [detail])

  const runtimeInfo = useMemo(() => {
    if (!detail) return []
    return [
      ['在线状态', yesNo(detail.is_online)],
      ['CPU 使用率', detail.cpu_usage],
      ['Audio Status', detail.audio_status],
      ['最后连接', detail.last_connected],
      ['硬件信息', detail.hardware_info],
    ]
  }, [detail])

  const errorInfo = useMemo(() => {
    if (!detail) return []
    return [
      ['错误次数', detail.error_count],
      ['最后错误时间', detail.last_error_at],
      ['最后错误', detail.last_error],
    ]
  }, [detail])

  return (
    <AdminLayout title="设备管理" description="管理平台设备资产与运行状态">
      <Card className="space-y-4">
        <div className="flex gap-3">
          <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="搜索 ID / MAC / 设备名 / 别名" className="max-w-sm" />
          <Button variant="outline" onClick={fetchData}>刷新</Button>
          <Button variant="outline" onClick={exportCsv}>导出 CSV</Button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b">
                <th className="text-left py-2 px-2">设备ID</th>
                <th className="text-left py-2 px-2">别名/名称</th>
                <th className="text-left py-2 px-2">Board</th>
                <th className="text-left py-2 px-2">版本</th>
                <th className="text-left py-2 px-2">在线</th>
                <th className="text-left py-2 px-2">错误次数</th>
                <th className="text-left py-2 px-2">最后连接</th>
                <th className="text-left py-2 px-2">操作</th>
              </tr>
            </thead>
            <tbody>
              {list.map((item) => (
                <tr key={String(item.id)} className="border-b">
                  <td className="py-2 px-2 font-mono text-xs">{toText(item.id)}</td>
                  <td className="py-2 px-2">{toText(item.alias || item.device_name)}</td>
                  <td className="py-2 px-2">{toText(item.board)}</td>
                  <td className="py-2 px-2">{toText(item.app_version)}</td>
                  <td className="py-2 px-2">{yesNo(item.is_online)}</td>
                  <td className="py-2 px-2">{toText(item.error_count)}</td>
                  <td className="py-2 px-2">{toText(item.last_connected)}</td>
                  <td className="py-2 px-2 flex gap-2">
                    <Button size="sm" variant="ghost" onClick={() => setDetail(item)}>详情</Button>
                    <Button size="sm" variant="ghost" className="text-red-600" onClick={() => deleteOne(item)}>删除</Button>
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
      <Modal isOpen={!!detail} onClose={() => setDetail(null)} title="设备详情" size="xl">
        {detail && (
          <div className="space-y-6">
            <div>
              <h4 className="font-semibold mb-2">基础信息</h4>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-2 text-sm">
                {baseInfo.map(([k, v]) => (
                  <div key={k}><span className="text-slate-500">{k}: </span><span className="break-all">{toText(v)}</span></div>
                ))}
              </div>
            </div>
            <div>
              <h4 className="font-semibold mb-2">运行状态</h4>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-2 text-sm">
                {runtimeInfo.map(([k, v]) => (
                  <div key={k}><span className="text-slate-500">{k}: </span><span className="break-all">{toText(v)}</span></div>
                ))}
              </div>
            </div>
            <div>
              <h4 className="font-semibold mb-2">错误信息</h4>
              <div className="space-y-2 text-sm">
                {errorInfo.map(([k, v]) => (
                  <div key={k}><span className="text-slate-500">{k}: </span><span className="break-all">{toText(v)}</span></div>
                ))}
              </div>
            </div>
          </div>
        )}
      </Modal>
    </AdminLayout>
  )
}

export default Devices
