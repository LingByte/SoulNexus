import { useEffect, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import Button from '@/components/UI/Button'
import Badge from '@/components/UI/Badge'
import Modal from '@/components/UI/Modal'
import { showAlert } from '@/utils/notification'
import { cn } from '@/utils/cn'
import {
  getMyLLMUsageSummary,
  listMyLLMUsage,
  type LLMUsageRecord,
  type LLMUsageSummary,
} from '@/api/llmUsage'

const fmtTime = (s?: string) => (s ? new Date(s).toLocaleString('zh-CN') : '—')

function formatJSONIfPossible(s: string): string {
  try {
    return JSON.stringify(JSON.parse(s), null, 2)
  } catch {
    return s
  }
}

const LLMUsagePanel = () => {
  const [summary, setSummary] = useState<LLMUsageSummary | null>(null)
  const [items, setItems] = useState<LLMUsageRecord[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [loading, setLoading] = useState(false)
  const [detail, setDetail] = useState<LLMUsageRecord | null>(null)

  const load = async (p = page) => {
    setLoading(true)
    try {
      const [sumRes, listRes] = await Promise.all([
        getMyLLMUsageSummary(),
        listMyLLMUsage({ page: p, pageSize }),
      ])
      if (sumRes.code === 200 && sumRes.data) {
        setSummary(sumRes.data.summary)
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
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-xs text-gray-500 dark:text-gray-400 max-w-3xl">
          通过「LLM Token」调用本平台 <code className="text-[10px]">/v1</code> 开放接口的用量记录，与智能体无关。
        </p>
        <Button
          variant="outline"
          size="sm"
          onClick={() => void load(page)}
          disabled={loading}
          leftIcon={<RefreshCw className={cn('h-3.5 w-3.5', loading && 'animate-spin')} />}
        >
          刷新
        </Button>
      </div>

      {summary && (
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 rounded-lg border border-slate-200 bg-slate-50/80 px-3 py-2 text-xs dark:border-neutral-700 dark:bg-neutral-900/40">
          <InlineStat label="总请求" value={summary.total_requests} />
          <InlineStat label="成功" value={summary.success_requests} />
          <InlineStat label="总 Tokens" value={summary.total_tokens} />
          <InlineStat label="额度扣减" value={summary.quota_delta} />
        </div>
      )}

      <div className="overflow-hidden rounded-lg border border-slate-200 dark:border-neutral-800">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-slate-50 text-[10px] uppercase tracking-wide text-slate-500 dark:bg-neutral-900/80 dark:text-slate-400">
              <tr>
                <th className="px-2.5 py-2 font-medium">时间</th>
                <th className="px-2.5 py-2 font-medium">模型</th>
                <th className="px-2.5 py-2 font-medium">类型</th>
                <th className="px-2.5 py-2 font-medium">Tokens</th>
                <th className="px-2.5 py-2 font-medium">延迟</th>
                <th className="px-2.5 py-2 font-medium">状态</th>
                <th className="px-2.5 py-2 font-medium w-16">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
              {items.length === 0 && !loading && (
                <tr>
                  <td colSpan={7} className="px-3 py-10 text-center text-slate-400">
                    暂无记录
                  </td>
                </tr>
              )}
              {items.map((row) => (
                <tr
                  key={row.id}
                  className="cursor-pointer hover:bg-slate-50/90 dark:hover:bg-neutral-900/50"
                  onClick={() => setDetail(row)}
                >
                  <td className="whitespace-nowrap px-2.5 py-2 tabular-nums text-slate-600 dark:text-slate-300">
                    {fmtTime(row.requested_at)}
                  </td>
                  <td className="max-w-[7rem] truncate px-2.5 py-2 font-mono" title={row.model}>
                    {row.model}
                  </td>
                  <td
                    className="max-w-[6rem] truncate px-2.5 py-2 text-slate-500"
                    title={row.request_type}
                  >
                    {row.request_type.replace(/^openapi_/, '')}
                  </td>
                  <td className="whitespace-nowrap px-2.5 py-2 tabular-nums">
                    <span className="text-slate-500">{row.input_tokens}</span>
                    <span className="mx-0.5 text-slate-300">/</span>
                    <span>{row.output_tokens}</span>
                  </td>
                  <td className="whitespace-nowrap px-2.5 py-2 tabular-nums text-slate-600 dark:text-slate-300">
                    {row.latency_ms ? `${row.latency_ms} ms` : '—'}
                  </td>
                  <td className="px-2.5 py-2">
                    <Badge variant={row.success ? 'success' : 'error'} size="xs">
                      {row.success ? '成功' : '失败'}
                    </Badge>
                  </td>
                  <td className="px-2.5 py-2">
                    <button
                      type="button"
                      className="text-sky-600 hover:underline dark:text-sky-400"
                      onClick={(e) => {
                        e.stopPropagation()
                        setDetail(row)
                      }}
                    >
                      详情
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {(totalPages > 1 || total > 0) && (
          <div className="flex items-center justify-between border-t border-slate-100 px-2.5 py-2 text-[11px] dark:border-neutral-800">
            <span className="text-slate-500">共 {total} 条</span>
            {totalPages > 1 && (
              <div className="flex items-center gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={page <= 1 || loading}
                  onClick={() => void load(page - 1)}
                >
                  上一页
                </Button>
                <span className="tabular-nums text-slate-600 dark:text-slate-300">
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
            )}
          </div>
        )}
      </div>

      <Modal
        isOpen={!!detail}
        onClose={() => setDetail(null)}
        title="请求详情"
        size="xl"
      >
        {detail && <UsageDetailContent row={detail} />}
      </Modal>
    </div>
  )
}

function InlineStat({ label, value }: { label: string; value: number }) {
  return (
    <span className="inline-flex items-center gap-1.5 text-slate-600 dark:text-slate-300">
      <span className="text-slate-400 dark:text-slate-500">{label}</span>
      <span className="font-semibold tabular-nums text-slate-800 dark:text-slate-100">
        {value.toLocaleString()}
      </span>
    </span>
  )
}

function UsageDetailContent({ row }: { row: LLMUsageRecord }) {
  return (
    <div className="max-h-[70vh] space-y-3 overflow-y-auto pr-1 text-sm">
      <DetailGrid>
        <DetailRow label="Request ID" value={row.request_id} mono />
        <DetailRow label="Model" value={row.model} mono />
        <DetailRow label="类型" value={row.request_type.replace(/^openapi_/, '')} />
        <DetailRow
          label="状态"
          value={
            <Badge variant={row.success ? 'success' : 'error'} size="sm">
              {row.success ? '成功' : '失败'}
            </Badge>
          }
        />
        <DetailRow
          label="Tokens"
          value={`in ${row.input_tokens} / out ${row.output_tokens} / Σ ${row.total_tokens}`}
        />
        <DetailRow label="额度扣减" value={String(row.quota_delta ?? 0)} />
        <DetailRow label="延迟" value={row.latency_ms ? `${row.latency_ms} ms` : '—'} />
        <DetailRow label="TTFT" value={row.ttft_ms ? `${row.ttft_ms} ms` : '—'} />
        <DetailRow label="TPS" value={row.tps ? row.tps.toFixed(2) : '—'} />
        <DetailRow label="HTTP" value={row.status_code ? String(row.status_code) : '—'} />
        <DetailRow label="请求时间" value={fmtTime(row.requested_at)} />
        <DetailRow label="完成时间" value={fmtTime(row.completed_at)} />
      </DetailGrid>

      {row.error_code || row.error_message ? (
        <DetailBlock label="错误信息">
          <pre className="max-h-32 overflow-auto whitespace-pre-wrap break-words rounded-md bg-red-50 p-2 text-xs text-red-700 dark:bg-red-950/40 dark:text-red-300">
            {[row.error_code, row.error_message].filter(Boolean).join(': ')}
          </pre>
        </DetailBlock>
      ) : null}

      {row.request_content ? (
        <DetailBlock label="请求体">
          <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-words rounded-md bg-slate-100 p-2 font-mono text-[11px] dark:bg-neutral-900">
            {formatJSONIfPossible(row.request_content)}
          </pre>
        </DetailBlock>
      ) : null}

      {row.response_content ? (
        <DetailBlock label="响应体">
          <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-words rounded-md bg-slate-100 p-2 font-mono text-[11px] dark:bg-neutral-900">
            {formatJSONIfPossible(row.response_content)}
          </pre>
        </DetailBlock>
      ) : null}
    </div>
  )
}

function DetailGrid({ children }: { children: React.ReactNode }) {
  return <div className="grid gap-2 sm:grid-cols-2">{children}</div>
}

function DetailRow({
  label,
  value,
  mono,
  breakAll,
}: {
  label: string
  value: React.ReactNode
  mono?: boolean
  breakAll?: boolean
}) {
  return (
    <div className="flex gap-2 text-xs sm:flex-col sm:gap-0.5">
      <span className="shrink-0 text-slate-500 dark:text-slate-400">{label}</span>
      <span
        className={cn(
          'min-w-0 text-slate-800 dark:text-slate-100',
          mono && 'font-mono',
          breakAll && 'break-all',
        )}
      >
        {value}
      </span>
    </div>
  )
}

function DetailBlock({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <p className="mb-1 text-xs font-medium text-slate-500 dark:text-slate-400">{label}</p>
      {children}
    </div>
  )
}

export default LLMUsagePanel
