import { useEffect, useState } from 'react'
import { Activity, RefreshCw } from 'lucide-react'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Badge from '@/components/UI/Badge'
import { showAlert } from '@/utils/notification'
import {
  getMyLLMUsageSummary,
  listMyLLMUsage,
  type LLMUsageRecord,
  type LLMUsageSummary,
} from '@/api/llmUsage'

const fmtTime = (s?: string) => (s ? new Date(s).toLocaleString('zh-CN') : '—')

const LLMUsagePanel = () => {
  const [summary, setSummary] = useState<LLMUsageSummary | null>(null)
  const [byModel, setByModel] = useState<Array<{ model: string; request_count: number; total_tokens: number }>>([])
  const [items, setItems] = useState<LLMUsageRecord[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(15)
  const [loading, setLoading] = useState(false)

  const load = async (p = page) => {
    setLoading(true)
    try {
      const [sumRes, listRes] = await Promise.all([
        getMyLLMUsageSummary(),
        listMyLLMUsage({ page: p, pageSize }),
      ])
      if (sumRes.code === 200 && sumRes.data) {
        setSummary(sumRes.data.summary)
        setByModel(sumRes.data.by_model || [])
      }
      if (listRes.code === 200 && listRes.data) {
        setItems(listRes.data.items || [])
        setTotal(listRes.data.total || 0)
        setPage(listRes.data.page || p)
      }
    } catch (e: unknown) {
      const err = e as { msg?: string; message?: string }
      showAlert(err?.msg || err?.message || '加载失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load(1)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 flex items-center gap-2">
            <Activity className="h-5 w-5 text-sky-500" />
            LLM API 消耗
          </h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400 max-w-2xl">
            统计通过您在「LLM Token」创建的密钥调用本平台 <code className="text-xs">/v1</code>{' '}
            开放接口产生的用量，与语音智能体、业务 API 密钥无关。
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => void load(page)}
          disabled={loading}
          leftIcon={<RefreshCw className={cnIcon(loading)} />}
        >
          刷新
        </Button>
      </div>

      {summary && (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <StatCard label="总请求" value={summary.total_requests} />
          <StatCard label="成功" value={summary.success_requests} />
          <StatCard label="总 Tokens" value={summary.total_tokens} />
          <StatCard label="额度扣减" value={summary.quota_delta} />
        </div>
      )}

      {byModel.length > 0 && (
        <Card className="p-4">
          <h3 className="text-sm font-medium text-gray-800 dark:text-gray-200 mb-3">按模型</h3>
          <div className="flex flex-wrap gap-2">
            {byModel.map((row) => (
              <span
                key={row.model}
                className="inline-flex items-center gap-2 rounded-lg border border-gray-200 dark:border-neutral-700 px-2.5 py-1 text-xs"
              >
                <span className="font-mono font-medium">{row.model}</span>
                <span className="text-gray-500">{row.request_count} 次</span>
                <span className="tabular-nums text-sky-600 dark:text-sky-400">{row.total_tokens} tok</span>
              </span>
            ))}
          </div>
        </Card>
      )}

      <Card className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="bg-gray-50 dark:bg-neutral-900/80 text-xs text-gray-500 uppercase">
              <tr>
                <th className="px-3 py-2">时间</th>
                <th className="px-3 py-2">模型</th>
                <th className="px-3 py-2">类型</th>
                <th className="px-3 py-2">Tokens</th>
                <th className="px-3 py-2">延迟</th>
                <th className="px-3 py-2">状态</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-neutral-800">
              {items.length === 0 && !loading && (
                <tr>
                  <td colSpan={6} className="px-3 py-8 text-center text-gray-400 text-sm">
                    暂无记录。使用 LLM Token 调用 /v1/chat/completions 或 /v1/messages 后将在此展示。
                  </td>
                </tr>
              )}
              {items.map((row) => (
                <tr key={row.id} className="hover:bg-gray-50/80 dark:hover:bg-neutral-900/40">
                  <td className="px-3 py-2 text-xs whitespace-nowrap">{fmtTime(row.requested_at)}</td>
                  <td className="px-3 py-2 font-mono text-xs">{row.model}</td>
                  <td className="px-3 py-2 text-xs text-gray-500 max-w-[8rem] truncate" title={row.request_type}>
                    {row.request_type.replace(/^openapi_/, '')}
                  </td>
                  <td className="px-3 py-2 text-xs tabular-nums">
                    <span className="text-gray-500">in {row.input_tokens}</span>
                    <span className="mx-1">/</span>
                    <span>out {row.output_tokens}</span>
                  </td>
                  <td className="px-3 py-2 text-xs tabular-nums">{row.latency_ms ? `${row.latency_ms} ms` : '—'}</td>
                  <td className="px-3 py-2">
                    <Badge variant={row.success ? 'success' : 'error'} size="sm">
                      {row.success ? '成功' : '失败'}
                    </Badge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {totalPages > 1 && (
          <div className="flex items-center justify-between border-t border-gray-100 dark:border-neutral-800 px-3 py-2 text-xs">
            <span className="text-gray-500">共 {total} 条</span>
            <div className="flex gap-2">
              <Button
                variant="ghost"
                size="sm"
                disabled={page <= 1 || loading}
                onClick={() => void load(page - 1)}
              >
                上一页
              </Button>
              <span className="self-center tabular-nums">
                {page} / {totalPages}
              </span>
              <Button
                variant="ghost"
                size="sm"
                disabled={page >= totalPages || loading}
                onClick={() => void load(page + 1)}
              >
                下一页
              </Button>
            </div>
          </div>
        )}
      </Card>
    </div>
  )
}

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <Card className="p-3">
      <p className="text-xs text-gray-500 dark:text-gray-400">{label}</p>
      <p className="mt-1 text-xl font-semibold tabular-nums text-gray-900 dark:text-gray-100">
        {value.toLocaleString()}
      </p>
    </Card>
  )
}

function cnIcon(loading: boolean) {
  return loading ? 'h-3.5 w-3.5 animate-spin' : 'h-3.5 w-3.5'
}

export default LLMUsagePanel
