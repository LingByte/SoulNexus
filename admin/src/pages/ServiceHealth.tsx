import { useEffect, useState, useCallback } from 'react'
import { Activity, RefreshCw } from 'lucide-react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Badge from '@/components/UI/Badge'

type ServiceEndpoint = {
  id: string
  label: string
  pingUrl: string
}

type EndpointsConfig = {
  services: ServiceEndpoint[]
  refreshIntervalMs?: number
}

type CheckState = {
  loading: boolean
  ok: boolean
  status?: number
  body?: Record<string, unknown>
  error?: string
  latencyMs?: number
}

function sanitizePingPreview(body: Record<string, unknown>) {
  const cpu = body.cpu_percent
  let cpuRounded: unknown = cpu
  if (typeof cpu === 'number' && Number.isFinite(cpu)) {
    cpuRounded = Math.round(cpu * 1000) / 1000
  }
  return {
    ping: body.ping,
    service: body.service,
    uptime_human: body.uptime_human,
    goroutines: body.goroutines,
    cpu_percent: cpuRounded,
    memory_go: body.memory_go,
    memory_host: body.memory_host,
    load_average: body.load_average,
    database: body.database,
  }
}

const ServiceHealth = () => {
  const [config, setConfig] = useState<EndpointsConfig | null>(null)
  const [cfgError, setCfgError] = useState<string | null>(null)
  const [results, setResults] = useState<Record<string, CheckState>>({})
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null)

  useEffect(() => {
    let cancelled = false
    fetch('/service-health-endpoints.json')
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      })
      .then((data: EndpointsConfig) => {
        if (!cancelled) setConfig(data)
      })
      .catch((e: Error) => {
        if (!cancelled) setCfgError(e.message || '无法加载 service-health-endpoints.json')
      })
    return () => {
      cancelled = true
    }
  }, [])

  const probeOne = useCallback(async (svc: ServiceEndpoint) => {
    const t0 = performance.now()
    try {
      const res = await fetch(svc.pingUrl, { method: 'GET', cache: 'no-store' })
      const latencyMs = Math.round(performance.now() - t0)
      let body: Record<string, unknown> | undefined
      try {
        body = (await res.json()) as Record<string, unknown>
      } catch {
        body = undefined
      }
      return {
        loading: false,
        ok: res.ok,
        status: res.status,
        body,
        latencyMs,
      } satisfies CheckState
    } catch (e) {
      return {
        loading: false,
        ok: false,
        error: e instanceof Error ? e.message : String(e),
        latencyMs: Math.round(performance.now() - t0),
      } satisfies CheckState
    }
  }, [])

  const runAll = useCallback(async () => {
    if (!config?.services?.length) return
    const next: Record<string, CheckState> = {}
    for (const svc of config.services) {
      next[svc.id] = { loading: true, ok: false }
    }
    setResults({ ...next })
    for (const svc of config.services) {
      const r = await probeOne(svc)
      setResults((prev) => ({ ...prev, [svc.id]: r }))
    }
    setLastRefresh(new Date())
  }, [config, probeOne])

  useEffect(() => {
    if (!config?.services?.length) return
    runAll()
    const ms = config.refreshIntervalMs && config.refreshIntervalMs >= 3000 ? config.refreshIntervalMs : 8000
    const id = window.setInterval(runAll, ms)
    return () => window.clearInterval(id)
  }, [config, runAll])

  const renderBodyPreview = (body?: Record<string, unknown>) => {
    if (!body) return null
    const subset = sanitizePingPreview(body)
    return (
      <pre className="mt-3 max-h-48 overflow-auto rounded-lg bg-slate-900/80 p-3 text-xs text-slate-100 dark:bg-slate-950">
        {JSON.stringify(subset, null, 2)}
      </pre>
    )
  }

  return (
    <AdminLayout>
      <div className="space-y-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="flex items-center gap-2 text-2xl font-semibold text-slate-900 dark:text-white">
              <Activity className="h-7 w-7 text-emerald-500" />
              服务存活检测
            </h1>
            <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
              地址列表来自{' '}
              <code className="rounded bg-slate-100 px-1 font-mono text-xs dark:bg-slate-800">
                public/service-health-endpoints.json
              </code>
              ，可自行增删服务。
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            size="md"
            leftIcon={<RefreshCw className="h-4 w-4" />}
            onClick={() => runAll()}
            disabled={!config?.services?.length}
          >
            立即检测
          </Button>
        </div>

        {cfgError && (
          <Card variant="outlined" className="border-amber-200 bg-amber-50 dark:border-amber-900 dark:bg-amber-950">
            <p className="text-sm text-amber-900 dark:text-amber-100">配置文件加载失败：{cfgError}</p>
          </Card>
        )}

        {lastRefresh && (
          <p className="text-xs text-slate-400">上次刷新：{lastRefresh.toLocaleString()}</p>
        )}

        <div className="grid gap-4 md:grid-cols-2">
          {config?.services?.map((svc) => {
            const r = results[svc.id]
            const alive = r?.ok && r.status === 200
            const statusBadge = r?.loading ? (
              <Badge variant="secondary">检测中…</Badge>
            ) : alive ? (
              <Badge variant="success">存活</Badge>
            ) : (
              <Badge variant="error">不可用</Badge>
            )

            return (
              <Card
                key={svc.id}
                title={svc.label}
                subtitle={svc.pingUrl}
                actions={statusBadge}
                padding="md"
                animation="none"
              >
                {r?.loading && <p className="text-sm text-slate-500">正在请求…</p>}
                {r && !r.loading && (
                  <div className="space-y-1 text-sm text-slate-600 dark:text-slate-300">
                    <div>
                      HTTP {r.status ?? '—'}
                      {typeof r.latencyMs === 'number' && ` · ${r.latencyMs} ms`}
                    </div>
                    {r.error && <div className="text-red-600 dark:text-red-400">{r.error}</div>}
                    {renderBodyPreview(r.body)}
                  </div>
                )}
              </Card>
            )
          })}
        </div>
      </div>
    </AdminLayout>
  )
}

export default ServiceHealth
