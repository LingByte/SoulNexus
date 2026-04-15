import { useState, useEffect } from 'react'
import { Search, FileText, Calendar, User, Globe, RefreshCw, ChevronDown, ChevronUp } from 'lucide-react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import { getOperationLogs, type OperationLog } from '@/services/adminApi'
import { showAlert } from '@/utils/notification'

const METHOD_COLORS: Record<string, string> = {
  POST: 'bg-green-100 text-green-700 dark:bg-green-900/20 dark:text-green-400',
  PUT: 'bg-blue-100 text-blue-700 dark:bg-blue-900/20 dark:text-blue-400',
  PATCH: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/20 dark:text-yellow-400',
  DELETE: 'bg-red-100 text-red-700 dark:bg-red-900/20 dark:text-red-400',
  GET: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400',
}

const OperationLogs = () => {
  const [logs, setLogs] = useState<OperationLog[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [total, setTotal] = useState(0)
  const [expandedId, setExpandedId] = useState<number | null>(null)
  const [filters, setFilters] = useState({ user_id: '', action: '', target: '' })

  useEffect(() => { fetchLogs() }, [page])

  const fetchLogs = async () => {
    try {
      setLoading(true)
      const params: any = { page, page_size: pageSize }
      if (filters.user_id) params.user_id = parseInt(filters.user_id)
      if (filters.action) params.action = filters.action
      if (filters.target) params.target = filters.target
      const data = await getOperationLogs(params)
      setLogs(data.logs || [])
      setTotal(data.total || 0)
    } catch (error: any) {
      showAlert('获取操作日志失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = () => {
    setPage(1)
    fetchLogs()
  }

  const formatDate = (d: string) => new Date(d).toLocaleString('zh-CN')

  const totalPages = Math.ceil(total / pageSize)

  return (
    <AdminLayout title="操作日志" description="查看系统操作日志记录">
      <div className="space-y-6">
        {/* 筛选 */}
        <Card>
          <div className="flex flex-col sm:flex-row gap-3 items-end">
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 flex-1">
              <Input
                placeholder="用户ID"
                value={filters.user_id}
                onChange={(e) => setFilters({ ...filters, user_id: e.target.value })}
              />
              <Input
                placeholder="操作类型"
                value={filters.action}
                onChange={(e) => setFilters({ ...filters, action: e.target.value })}
              />
              <Input
                placeholder="操作路径"
                value={filters.target}
                onChange={(e) => setFilters({ ...filters, target: e.target.value })}
              />
            </div>
            <div className="flex gap-2 shrink-0">
              <Button onClick={handleSearch} leftIcon={<Search className="w-4 h-4" />}>搜索</Button>
              <Button variant="outline" onClick={fetchLogs} leftIcon={<RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />}>刷新</Button>
            </div>
          </div>
        </Card>

        {/* 列表 */}
        <Card>
          {loading && logs.length === 0 ? (
            <div className="flex items-center justify-center py-16">
              <RefreshCw className="w-6 h-6 animate-spin text-slate-400" />
            </div>
          ) : logs.length === 0 ? (
            <div className="text-center py-16">
              <FileText className="w-12 h-12 mx-auto mb-3 text-slate-300 dark:text-slate-600" />
              <p className="text-slate-500 dark:text-slate-400">暂无操作日志</p>
            </div>
          ) : (
            <div className="divide-y divide-slate-100 dark:divide-slate-800">
              {logs.map((log) => {
                const isExpanded = expandedId === log.id
                return (
                  <div key={log.id} className="py-3 px-1">
                    <div
                      className="flex items-start gap-3 cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-800/40 rounded-lg px-2 py-1 transition-colors"
                      onClick={() => setExpandedId(isExpanded ? null : log.id)}
                    >
                      {/* method badge */}
                      <span className={`mt-0.5 shrink-0 px-2 py-0.5 text-xs font-mono rounded font-semibold ${METHOD_COLORS[log.request_method] || METHOD_COLORS.GET}`}>
                        {log.request_method}
                      </span>

                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="font-medium text-slate-900 dark:text-white text-sm">{log.details}</span>
                          <span className="text-xs text-slate-400 font-mono truncate max-w-xs">{log.target}</span>
                        </div>
                        <div className="flex items-center gap-4 mt-1 text-xs text-slate-500 dark:text-slate-400 flex-wrap">
                          <span className="flex items-center gap-1"><User className="w-3 h-3" />{log.username || '-'} #{log.user_id}</span>
                          <span className="flex items-center gap-1"><Globe className="w-3 h-3" />{log.ip_address}</span>
                          {log.location && <span>{log.location}</span>}
                          <span className="flex items-center gap-1"><Calendar className="w-3 h-3" />{formatDate(log.created_at)}</span>
                        </div>
                      </div>

                      <span className="shrink-0 text-slate-400">
                        {isExpanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
                      </span>
                    </div>

                    {/* 展开详情 */}
                    {isExpanded && (
                      <div className="mt-2 mx-2 p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-xs space-y-1.5 text-slate-600 dark:text-slate-400">
                        <div className="flex gap-2"><span className="w-20 shrink-0 font-medium text-slate-500">设备</span><span>{log.device || '-'}</span></div>
                        <div className="flex gap-2"><span className="w-20 shrink-0 font-medium text-slate-500">浏览器</span><span>{log.browser || '-'}</span></div>
                        <div className="flex gap-2"><span className="w-20 shrink-0 font-medium text-slate-500">操作系统</span><span>{log.operating_system || '-'}</span></div>
                        <div className="flex gap-2"><span className="w-20 shrink-0 font-medium text-slate-500">来源页面</span><span className="break-all">{log.referer || '-'}</span></div>
                        <div className="flex gap-2"><span className="w-20 shrink-0 font-medium text-slate-500">User-Agent</span><span className="break-all">{log.user_agent || '-'}</span></div>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}

          {/* 分页 */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4 pt-4 border-t border-slate-200 dark:border-slate-700">
              <span className="text-sm text-slate-500">共 {total} 条</span>
              <div className="flex items-center gap-2">
                <Button variant="outline" size="sm" onClick={() => setPage(p => p - 1)} disabled={page === 1}>上一页</Button>
                <span className="text-sm text-slate-500">第 {page} / {totalPages} 页</span>
                <Button variant="outline" size="sm" onClick={() => setPage(p => p + 1)} disabled={page >= totalPages}>下一页</Button>
              </div>
            </div>
          )}
        </Card>
      </div>
    </AdminLayout>
  )
}

export default OperationLogs
