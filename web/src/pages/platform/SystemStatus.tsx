import { useCallback, useEffect, useMemo, useState } from 'react'
import { Card, Grid, Progress, Space, Statistic, Tabs, Tag, Typography } from '@arco-design/web-react'
import { RefreshCw, CheckCircle, AlertTriangle, XCircle, Server, Cpu, HardDrive, Activity, Database, Wifi, BarChart3, Hash } from 'lucide-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button } from '@/components/ui'
import { useTranslation } from '@/i18n'
import {
  fetchSystemStatus,
  type PreflightLevel,
  type SystemStatusPayload,
} from '@/api/systemStatus'
import { extractApiErrorMessage } from '@/utils/apiError'
import { showAlert } from '@/utils/notification'

const { Row, Col } = Grid
const TabPane = Tabs.TabPane

function formatBytes(n?: number): string {
  if (n == null || !Number.isFinite(n)) return '—'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let v = n
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

function formatDuration(sec?: number): string {
  if (sec == null || sec < 0) return '—'
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = sec % 60
  if (h > 0) return `${h}h ${m}m ${s}s`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

function formatPct(v?: number, digits = 2): string {
  if (v == null || !Number.isFinite(v)) return '—'
  return `${(v * 100).toFixed(digits)}%`
}

function formatMs(v?: number): string {
  if (v == null || !Number.isFinite(v)) return '—'
  return `${v.toFixed(v >= 10 ? 0 : 2)} ms`
}

function levelColor(level: PreflightLevel) {
  if (level === 'ok') return { bg: 'bg-emerald-50', text: 'text-emerald-700', border: 'border-emerald-200', icon: 'text-emerald-500' }
  if (level === 'warn') return { bg: 'bg-amber-50', text: 'text-amber-700', border: 'border-amber-200', icon: 'text-amber-500' }
  return { bg: 'bg-red-50', text: 'text-red-700', border: 'border-red-200', icon: 'text-red-500' }
}

function levelLabel(level: PreflightLevel, t: (k: string) => string): string {
  if (level === 'ok') return t('systemStatus.levelOk')
  if (level === 'warn') return t('systemStatus.levelWarn')
  return t('systemStatus.levelError')
}

function LevelIcon({ level }: { level: PreflightLevel }) {
  if (level === 'ok') return <CheckCircle size={18} className="text-emerald-500" />
  if (level === 'warn') return <AlertTriangle size={18} className="text-amber-500" />
  return <XCircle size={18} className="text-red-500" />
}

function MetricRow({ label, value, icon }: { label: string; value: React.ReactNode; icon?: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border border-neutral-100 px-4 py-3">
      <div className="flex items-center gap-2">
        {icon && <span className="text-neutral-400">{icon}</span>}
        <span className="text-sm text-neutral-600">{label}</span>
      </div>
      <span className="font-medium text-neutral-900">{value}</span>
    </div>
  )
}

function PreflightRow({ level, category, message, latencyMs, detail, t }: {
  level: PreflightLevel
  category: string
  message: string
  latencyMs?: number
  detail?: string
  t: (k: string) => string
}) {
  const colors = levelColor(level)
  return (
    <div className={`flex items-center gap-4 rounded-xl border ${colors.border} ${colors.bg} px-5 py-4 transition hover:shadow-sm`}>
      <div className="shrink-0">
        <LevelIcon level={level} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-medium text-neutral-900">{message}</span>
          <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${colors.bg} ${colors.text} border ${colors.border}`}>
            {levelLabel(level, t)}
          </span>
        </div>
        <div className="mt-1 flex flex-wrap items-center gap-x-4 text-sm text-neutral-500">
          <span className="flex items-center gap-1">
            <Hash size={12} className="text-neutral-400" />
            {category}
          </span>
          {latencyMs != null && latencyMs > 0 && (
            <span className="font-mono text-xs text-neutral-400">{latencyMs} ms</span>
          )}
          {detail && (
            <span className="truncate text-xs text-neutral-400" title={detail}>{detail}</span>
          )}
        </div>
      </div>
    </div>
  )
}

function MetricListItem({ label, value, icon }: { label: string; value: string | number; icon?: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border border-neutral-100 bg-white px-4 py-3 transition hover:border-neutral-200 hover:shadow-sm">
      <div className="flex items-center gap-2.5">
        {icon && <span className="text-neutral-400">{icon}</span>}
        <span className="text-sm text-neutral-500">{label}</span>
      </div>
      <span className="font-mono text-sm font-medium text-neutral-900">{value}</span>
    </div>
  )
}

function OverviewTab({ data, loading }: { data: SystemStatusPayload | null; loading: boolean }) {
  const { t } = useTranslation()
  const preflightRows = useMemo(() => {
    const base = data?.preflight?.checks ?? []
    const deps = (data?.dependencies ?? []).map((d) => ({ ...d, category: d.category || 'dependencies' }))
    return [...base, ...deps]
  }, [data])

  const errorCount = preflightRows.filter((r) => r.level === 'error').length
  const warnCount = preflightRows.filter((r) => r.level === 'warn').length

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Row gutter={16}>
        <Col span={6}>
          <Card className="border border-neutral-100 shadow-sm">
            <Statistic
              title={t('systemStatus.uptime')}
              value={formatDuration(data?.uptimeSeconds)}
              prefix={<Activity size={16} className="text-blue-500" />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card className="border border-neutral-100 shadow-sm">
            <Statistic
              title={t('systemStatus.goroutines')}
              value={data?.goroutineDetail?.numGoroutine ?? data?.goroutines ?? '—'}
              prefix={<Server size={16} className="text-purple-500" />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card className="border border-neutral-100 shadow-sm">
            <Statistic
              title={t('systemStatus.cpu')}
              value={data?.resource?.cpuUsage != null ? `${data.resource.cpuUsage.toFixed(1)}%` : '—'}
              prefix={<Cpu size={16} className="text-emerald-500" />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card className="border border-neutral-100 shadow-sm">
            <Statistic
              title={t('systemStatus.memory')}
              value={data?.resource?.memoryUsage != null ? `${data.resource.memoryUsage.toFixed(1)}%` : '—'}
              prefix={<HardDrive size={16} className="text-amber-500" />}
            />
          </Card>
        </Col>
      </Row>

      <Card
        title={
          <div className="flex items-center gap-2">
            <Server size={16} className="text-neutral-500" />
            <span>{t('systemStatus.serverInfo')}</span>
          </div>
        }
        className="border border-neutral-100 shadow-sm"
      >
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
          <div className="rounded-lg border border-neutral-100 bg-neutral-50 px-3 py-2.5">
            <div className="text-xs text-neutral-400">{t('systemStatus.serverAddr')}</div>
            <div className="mt-0.5 truncate font-mono text-sm font-medium text-neutral-900">{data?.server?.addr || '—'}</div>
          </div>
          <div className="rounded-lg border border-neutral-100 bg-neutral-50 px-3 py-2.5">
            <div className="text-xs text-neutral-400">{t('systemStatus.serverMode')}</div>
            <div className="mt-0.5 font-medium text-neutral-900">{data?.server?.mode || '—'}</div>
          </div>
          <div className="rounded-lg border border-neutral-100 bg-neutral-50 px-3 py-2.5">
            <div className="text-xs text-neutral-400">HTTPS</div>
            <div className="mt-0.5 font-medium text-neutral-900">
              {data?.server?.sslEnabled ? t('systemStatus.enabled') : t('systemStatus.disabled')}
            </div>
          </div>
          <div className="rounded-lg border border-neutral-100 bg-neutral-50 px-3 py-2.5">
            <div className="text-xs text-neutral-400">Go</div>
            <div className="mt-0.5 font-mono text-sm font-medium text-neutral-900">{data?.go || '—'}</div>
          </div>
          {data?.startedAt && (
            <div className="rounded-lg border border-neutral-100 bg-neutral-50 px-3 py-2.5">
              <div className="text-xs text-neutral-400">{t('systemStatus.startedAt')}</div>
              <div className="mt-0.5 text-sm font-medium text-neutral-900">{new Date(data.startedAt).toLocaleString()}</div>
            </div>
          )}
        </div>
      </Card>

      <Card
        title={
          <div className="flex items-center gap-2">
            <CheckCircle size={16} className="text-neutral-500" />
            <span>{t('systemStatus.preflightTitle')}</span>
          </div>
        }
        className="border border-neutral-100 shadow-sm"
        extra={
          <div className="flex items-center gap-3">
            {errorCount > 0 && <Tag color="red" className="!rounded-full">{t('systemStatus.errors', { count: errorCount })}</Tag>}
            {warnCount > 0 && <Tag color="orangered" className="!rounded-full">{t('systemStatus.warnings', { count: warnCount })}</Tag>}
            {data?.preflight?.checkedAt && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {t('systemStatus.lastChecked')}: {new Date(data.preflight.checkedAt).toLocaleString()}
              </Typography.Text>
            )}
          </div>
        }
      >
        {loading ? (
          <div className="py-8 text-center text-sm text-neutral-400">{t('common.loading')}</div>
        ) : preflightRows.length === 0 ? (
          <div className="py-8 text-center text-sm text-neutral-400">—</div>
        ) : (
          <div className="space-y-2">
            {preflightRows.map((row) => (
              <PreflightRow
                key={row.id}
                level={row.level}
                category={row.category}
                message={row.message}
                latencyMs={row.latencyMs}
                detail={row.detail}
                t={t}
              />
            ))}
          </div>
        )}
      </Card>

      <Row gutter={16}>
        <Col span={12}>
          <Card
            title={
              <div className="flex items-center gap-2">
                <HardDrive size={16} className="text-neutral-500" />
                <span>{t('systemStatus.diskTitle')}</span>
              </div>
            }
            className="border border-neutral-100 shadow-sm"
          >
            <Progress
              percent={data?.disk?.usedPercent ?? 0}
              formatText={() =>
                `${formatBytes(data?.disk?.used)} / ${formatBytes(data?.disk?.total)} (${(data?.disk?.usedPercent ?? 0).toFixed(1)}%)`
              }
              className="mb-3"
            />
            <div className="flex items-center gap-2 text-sm text-neutral-600">
              <HardDrive size={14} className="text-neutral-400" />
              <span>{t('systemStatus.diskFree')}: {formatBytes(data?.disk?.free)}</span>
            </div>
          </Card>
        </Col>
        <Col span={12}>
          <Card
            title={
              <div className="flex items-center gap-2">
                <Activity size={16} className="text-neutral-500" />
                <span>{t('systemStatus.hostLoadTitle')}</span>
              </div>
            }
            className="border border-neutral-100 shadow-sm"
          >
            <Space direction="vertical" size={8} style={{ width: '100%' }}>
              <MetricRow
                label={t('systemStatus.load1')}
                value={`${data?.host?.load?.load1?.toFixed(2) ?? '—'} / ${data?.host?.cpu?.numCpu ?? '—'} CPU`}
                icon={<Activity size={14} />}
              />
              <MetricRow
                label={t('systemStatus.swapUsed')}
                value={formatBytes(data?.host?.memory?.swapUsed)}
                icon={<HardDrive size={14} />}
              />
              <MetricRow
                label={t('systemStatus.hostMemAvailable')}
                value={formatBytes(data?.host?.memory?.available)}
                icon={<Server size={14} />}
              />
            </Space>
          </Card>
        </Col>
      </Row>
    </Space>
  )
}

function RuntimeTab({ data }: { data: SystemStatusPayload | null }) {
  const { t } = useTranslation()
  const mem = data?.runtimeMemory
  const gc = data?.gc

  return (
    <Row gutter={16}>
      <Col span={12}>
        <Card
          title={
            <div className="flex items-center gap-2">
              <HardDrive size={16} className="text-neutral-500" />
              <span>{t('systemStatus.runtimeMemTitle')}</span>
            </div>
          }
          className="border border-neutral-100 shadow-sm"
        >
          <div className="space-y-2">
            <MetricListItem label="HeapInuse" value={formatBytes(mem?.heapInuse)} icon={<HardDrive size={14} />} />
            <MetricListItem label="HeapIdle" value={formatBytes(mem?.heapIdle)} icon={<HardDrive size={14} />} />
            <MetricListItem label="HeapReleased" value={formatBytes(mem?.heapReleased)} icon={<HardDrive size={14} />} />
            <MetricListItem label="HeapSys" value={formatBytes(mem?.heapSys)} icon={<HardDrive size={14} />} />
            <MetricListItem label="StackInuse" value={formatBytes(mem?.stackInuse)} icon={<Server size={14} />} />
            <MetricListItem label="StackSys" value={formatBytes(mem?.stackSys)} icon={<Server size={14} />} />
            <MetricListItem label="MSpanSys" value={formatBytes(mem?.mSpanSys)} icon={<Activity size={14} />} />
            <MetricListItem label="MCacheSys" value={formatBytes(mem?.mCacheSys)} icon={<Activity size={14} />} />
            <MetricListItem label="GCSys" value={formatBytes(mem?.gcSys)} icon={<Activity size={14} />} />
            <MetricListItem label="Sys" value={formatBytes(mem?.sys)} icon={<Server size={14} />} />
            <MetricListItem label={t('systemStatus.heapObjects')} value={String(mem?.heapObjects ?? '—')} icon={<Database size={14} />} />
            <MetricListItem label="Alloc" value={formatBytes(mem?.alloc)} icon={<HardDrive size={14} />} />
            <MetricListItem label="TotalAlloc" value={formatBytes(mem?.totalAlloc)} icon={<HardDrive size={14} />} />
            <MetricListItem label="Mallocs / Frees" value={`${mem?.mallocs ?? '—'} / ${mem?.frees ?? '—'}`} icon={<Activity size={14} />} />
          </div>
        </Card>
      </Col>
      <Col span={12}>
        <Card
          title={
            <div className="flex items-center gap-2">
              <Activity size={16} className="text-neutral-500" />
              <span>{t('systemStatus.gcTitle')}</span>
            </div>
          }
          className="border border-neutral-100 shadow-sm"
        >
          <div className="space-y-2">
            <MetricListItem label="NumGC" value={String(gc?.numGC ?? '—')} icon={<Activity size={14} />} />
            <MetricListItem label="NumForcedGC" value={String(gc?.numForcedGC ?? '—')} icon={<Activity size={14} />} />
            <MetricListItem label={t('systemStatus.gcPauseTotal')} value={formatMs(gc?.pauseTotalMs)} icon={<Activity size={14} />} />
            <MetricListItem label={t('systemStatus.gcPauseAvg')} value={formatMs(gc?.recentPauseAvgMs)} icon={<Activity size={14} />} />
            <MetricListItem label={t('systemStatus.gcPauseMax')} value={formatMs(gc?.recentPauseMaxMs)} icon={<Activity size={14} />} />
            <MetricListItem label={t('systemStatus.gcPerMinute')} value={gc?.gcPerMinute?.toFixed(2) ?? '—'} icon={<Activity size={14} />} />
            <MetricListItem label="GCCPUFraction" value={formatPct(gc?.gcCpuFraction)} icon={<Cpu size={14} />} />
            <MetricListItem label="NextGC" value={formatBytes(gc?.nextGC)} icon={<HardDrive size={14} />} />
          </div>
        </Card>
      </Col>
    </Row>
  )
}

function GoroutineTab({ data }: { data: SystemStatusPayload | null }) {
  const { t } = useTranslation()
  const gr = data?.goroutineDetail
  const stateRows = Object.entries(gr?.byState ?? {})
    .sort((a, b) => b[1] - a[1])
    .map(([state, count]) => ({ state, count }))

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Row gutter={16}>
        <Col span={6}>
          <Card className="border border-neutral-100 shadow-sm">
            <Statistic
              title={t('systemStatus.grTotal')}
              value={gr?.numGoroutine ?? '—'}
              prefix={<Server size={16} className="text-blue-500" />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card className="border border-neutral-100 shadow-sm">
            <Statistic
              title={t('systemStatus.grMax')}
              value={gr?.numGoroutineMax ?? '—'}
              prefix={<Activity size={16} className="text-purple-500" />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card className="border border-neutral-100 shadow-sm">
            <Statistic
              title={t('systemStatus.grThreads')}
              value={gr?.numThread ?? '—'}
              prefix={<Cpu size={16} className="text-emerald-500" />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card className="border border-neutral-100 shadow-sm">
            <Statistic
              title="NumCgoCall"
              value={gr?.numCgoCall ?? '—'}
              prefix={<Activity size={16} className="text-amber-500" />}
            />
          </Card>
        </Col>
      </Row>
      {gr?.leakSuspect && (
        <div className="flex items-center gap-2 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3">
          <AlertTriangle size={16} className="text-amber-600" />
          <span className="text-sm font-medium text-amber-800">{t('systemStatus.grLeakSuspect')}</span>
        </div>
      )}
      <Card
        title={
          <div className="flex items-center gap-2">
            <BarChart3 size={16} className="text-neutral-500" />
            <span>{t('systemStatus.grByState')}</span>
          </div>
        }
        className="border border-neutral-100 shadow-sm"
      >
        <div className="space-y-2">
          {stateRows.map((row) => (
            <div key={row.state} className="flex items-center justify-between rounded-lg border border-neutral-100 bg-white px-4 py-3 transition hover:border-neutral-200 hover:shadow-sm">
              <span className="font-mono text-sm text-neutral-700">{row.state}</span>
              <span className="font-mono text-sm font-semibold text-neutral-900">{row.count}</span>
            </div>
          ))}
          {stateRows.length === 0 && (
            <div className="py-8 text-center text-sm text-neutral-400">—</div>
          )}
        </div>
      </Card>
    </Space>
  )
}

function HostTab({ data }: { data: SystemStatusPayload | null }) {
  const { t } = useTranslation()
  const host = data?.host

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Row gutter={16}>
        <Col span={12}>
          <Card
            title={
              <div className="flex items-center gap-2">
                <Cpu size={16} className="text-neutral-500" />
                <span>{t('systemStatus.cpuDetailTitle')}</span>
              </div>
            }
            className="border border-neutral-100 shadow-sm"
          >
            <div className="space-y-2">
              <MetricListItem label={t('systemStatus.cpuUser')} value={`${host?.cpu?.userPercent?.toFixed(1) ?? '—'}%`} icon={<Cpu size={14} />} />
              <MetricListItem label={t('systemStatus.cpuSystem')} value={`${host?.cpu?.systemPercent?.toFixed(1) ?? '—'}%`} icon={<Cpu size={14} />} />
              <MetricListItem label={t('systemStatus.cpuIdle')} value={`${host?.cpu?.idlePercent?.toFixed(1) ?? '—'}%`} icon={<Cpu size={14} />} />
              <MetricListItem label="iowait" value={`${host?.cpu?.iowaitPercent?.toFixed(1) ?? '—'}%`} icon={<Activity size={14} />} />
            </div>
          </Card>
        </Col>
        <Col span={12}>
          <Card
            title={
              <div className="flex items-center gap-2">
                <Server size={16} className="text-neutral-500" />
                <span>{t('systemStatus.processTitle')}</span>
              </div>
            }
            className="border border-neutral-100 shadow-sm"
          >
            <div className="space-y-2">
              <MetricListItem label="RSS" value={formatBytes(host?.process?.rss)} icon={<HardDrive size={14} />} />
              <MetricListItem label="VMS" value={formatBytes(host?.process?.vms)} icon={<HardDrive size={14} />} />
              <MetricListItem label={t('systemStatus.openFds')} value={String(host?.process?.openFds ?? '—')} icon={<Server size={14} />} />
              <MetricListItem label={t('systemStatus.maxFds')} value={String(host?.process?.maxFds ?? '—')} icon={<Server size={14} />} />
              <MetricListItem label={t('systemStatus.grThreads')} value={String(host?.process?.numThreads ?? '—')} icon={<Cpu size={14} />} />
            </div>
          </Card>
        </Col>
      </Row>

      <Card
        title={
          <div className="flex items-center gap-2">
            <HardDrive size={16} className="text-neutral-500" />
            <span>{t('systemStatus.diskPathsTitle')}</span>
          </div>
        }
        className="border border-neutral-100 shadow-sm"
      >
        <div className="space-y-3">
          {(host?.disks ?? []).map((disk) => (
            <div key={disk.path} className="rounded-xl border border-neutral-100 bg-white p-4 transition hover:border-neutral-200 hover:shadow-sm">
              <div className="mb-3 flex items-center justify-between">
                <span className="font-mono text-sm font-medium text-neutral-900">{disk.path}</span>
                <span className="font-mono text-xs text-neutral-400">
                  {disk.inodesFree != null ? `${disk.inodesFree} inodes free` : ''}
                </span>
              </div>
              <Progress percent={disk.usedPercent} size="small" className="mb-2" />
              <div className="text-xs text-neutral-500">
                {formatBytes(disk.used)} / {formatBytes(disk.total)} ({disk.usedPercent.toFixed(1)}%)
                {disk.free != null && <span className="ml-2 text-neutral-400">· {formatBytes(disk.free)} free</span>}
              </div>
            </div>
          ))}
          {(host?.disks ?? []).length === 0 && (
            <div className="py-8 text-center text-sm text-neutral-400">—</div>
          )}
        </div>
      </Card>

      <Card
        title={
          <div className="flex items-center gap-2">
            <Wifi size={16} className="text-neutral-500" />
            <span>{t('systemStatus.networkTitle')}</span>
          </div>
        }
        className="border border-neutral-100 shadow-sm"
      >
        <div className="grid grid-cols-3 gap-4">
          <div className="rounded-lg border border-neutral-100 bg-blue-50 px-4 py-3 text-center">
            <div className="text-xs text-blue-500">↓ 下载</div>
            <div className="mt-1 font-mono text-sm font-semibold text-blue-700">{formatBytes(host?.network?.bytesRecv)}</div>
          </div>
          <div className="rounded-lg border border-neutral-100 bg-emerald-50 px-4 py-3 text-center">
            <div className="text-xs text-emerald-500">↑ 上传</div>
            <div className="mt-1 font-mono text-sm font-semibold text-emerald-700">{formatBytes(host?.network?.bytesSent)}</div>
          </div>
          <div className="rounded-lg border border-neutral-100 bg-neutral-50 px-4 py-3 text-center">
            <div className="text-xs text-neutral-500">{t('systemStatus.netDrops')}</div>
            <div className="mt-1 font-mono text-sm font-semibold text-neutral-700">{host?.network?.dropIn ?? 0} / {host?.network?.dropOut ?? 0}</div>
          </div>
        </div>
      </Card>
    </Space>
  )
}

function BusinessTab({ data }: { data: SystemStatusPayload | null }) {
  const { t } = useTranslation()
  const biz = data?.business

  return (
    <Row gutter={16}>
      <Col span={12}>
        <Card
          title={
            <div className="flex items-center gap-2">
              <Activity size={16} className="text-neutral-500" />
              <span>{t('systemStatus.bizHttpTitle')}</span>
            </div>
          }
          className="border border-neutral-100 shadow-sm"
        >
          <div className="space-y-2">
            <MetricListItem label="QPS (total req)" value={String(biz?.http?.requestsTotal ?? '—')} icon={<Activity size={14} />} />
            <MetricListItem label="4xx" value={String(biz?.http?.errors4xx ?? '—')} icon={<AlertTriangle size={14} />} />
            <MetricListItem label="5xx" value={String(biz?.http?.errors5xx ?? '—')} icon={<XCircle size={14} />} />
            <MetricListItem label="P50" value={formatMs(biz?.http?.latencyMs?.p50)} icon={<Activity size={14} />} />
            <MetricListItem label="P90" value={formatMs(biz?.http?.latencyMs?.p90)} icon={<Activity size={14} />} />
            <MetricListItem label="P99" value={formatMs(biz?.http?.latencyMs?.p99)} icon={<Activity size={14} />} />
          </div>
        </Card>
        <Card
          title={
            <div className="flex items-center gap-2">
              <Database size={16} className="text-neutral-500" />
              <span>{t('systemStatus.bizDbTitle')}</span>
            </div>
          }
          className="mt-4 border border-neutral-100 shadow-sm"
        >
          <div className="space-y-2">
            <MetricListItem label={t('systemStatus.dbOpen')} value={String(biz?.database?.openConnections ?? '—')} icon={<Database size={14} />} />
            <MetricListItem label={t('systemStatus.dbInUse')} value={String(biz?.database?.inUse ?? '—')} icon={<Database size={14} />} />
            <MetricListItem label={t('systemStatus.dbIdle')} value={String(biz?.database?.idle ?? '—')} icon={<Database size={14} />} />
            <MetricListItem label={t('systemStatus.dbWait')} value={String(biz?.database?.waitCount ?? '—')} icon={<Database size={14} />} />
          </div>
        </Card>
      </Col>
      <Col span={12}>
        <Card
          title={
            <div className="flex items-center gap-2">
              <Database size={16} className="text-neutral-500" />
              <span>{t('systemStatus.bizKbTitle')}</span>
            </div>
          }
          className="border border-neutral-100 shadow-sm"
        >
          <div className="space-y-2">
            <MetricListItem label={t('systemStatus.kbQueued')} value={String(biz?.knowledgeWorker?.queued ?? '—')} icon={<Activity size={14} />} />
            <MetricListItem label={t('systemStatus.kbRunning')} value={String(biz?.knowledgeWorker?.running ?? '—')} icon={<Activity size={14} />} />
            <MetricListItem label={t('systemStatus.kbUnfinished')} value={String(biz?.knowledgeWorker?.unfinished ?? '—')} icon={<AlertTriangle size={14} />} />
          </div>
        </Card>
      </Col>
    </Row>
  )
}

function OpsTab({ data }: { data: SystemStatusPayload | null }) {
  const { t } = useTranslation()
  const ops = data?.ops
  const prom = data?.prometheus

  const counterRows = Object.entries(prom?.counters ?? {}).map(([name, value]) => ({ name, value }))
  const gaugeRows = Object.entries(prom?.gauges ?? {}).map(([name, value]) => ({ name, value: value.toFixed(2) }))

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Card
        title={
          <div className="flex items-center gap-2">
            <AlertTriangle size={16} className="text-neutral-500" />
            <span>{t('systemStatus.opsTitle')}</span>
          </div>
        }
        className="border border-neutral-100 shadow-sm"
      >
        <div className="grid grid-cols-3 gap-4">
          <div className="rounded-lg border border-neutral-100 bg-neutral-50 px-4 py-3">
            <div className="text-xs text-neutral-400">panic</div>
            <div className="mt-1 text-2xl font-semibold text-neutral-900">{ops?.panicTotal ?? 0}</div>
          </div>
          <div className="rounded-lg border border-neutral-100 bg-neutral-50 px-4 py-3">
            <div className="text-xs text-neutral-400">JWT fail</div>
            <div className="mt-1 text-2xl font-semibold text-neutral-900">{ops?.jwtVerifyFailTotal ?? 0}</div>
          </div>
          <div className="rounded-lg border border-neutral-100 bg-neutral-50 px-4 py-3">
            <div className="text-xs text-neutral-400">DB reconnect</div>
            <div className="mt-1 text-2xl font-semibold text-neutral-900">{ops?.dbReconnectTotal ?? 0}</div>
          </div>
        </div>
        <div className="mt-4 grid grid-cols-3 gap-4">
          <div className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3">
            <div className="text-xs text-amber-500">HTTP 4xx</div>
            <div className="mt-1 text-2xl font-semibold text-amber-700">{ops?.http4xxTotal ?? 0}</div>
          </div>
          <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3">
            <div className="text-xs text-red-500">HTTP 5xx</div>
            <div className="mt-1 text-2xl font-semibold text-red-700">{ops?.http5xxTotal ?? 0}</div>
          </div>
          <div className="rounded-lg border border-neutral-100 bg-neutral-50 px-4 py-3">
            <div className="text-xs text-neutral-400">dep fail</div>
            <div className="mt-1 text-2xl font-semibold text-neutral-900">{ops?.dependencyFailTotal ?? 0}</div>
          </div>
        </div>
      </Card>

      <Row gutter={16}>
        <Col span={12}>
          <Card
            title={
              <div className="flex items-center gap-2">
                <BarChart3 size={16} className="text-neutral-500" />
                <span>{t('systemStatus.promCounters')}</span>
              </div>
            }
            className="border border-neutral-100 shadow-sm"
          >
            <div className="space-y-2">
              {counterRows.map((row) => (
                <div key={row.name} className="flex items-center justify-between rounded-lg border border-neutral-100 bg-white px-4 py-3 transition hover:border-neutral-200 hover:shadow-sm">
                  <span className="font-mono text-sm text-neutral-700">{row.name}</span>
                  <span className="font-mono text-sm font-semibold text-neutral-900">{row.value}</span>
                </div>
              ))}
              {counterRows.length === 0 && (
                <div className="py-8 text-center text-sm text-neutral-400">—</div>
              )}
            </div>
          </Card>
        </Col>
        <Col span={12}>
          <Card
            title={
              <div className="flex items-center gap-2">
                <BarChart3 size={16} className="text-neutral-500" />
                <span>{t('systemStatus.promGauges')}</span>
              </div>
            }
            className="border border-neutral-100 shadow-sm"
          >
            <div className="space-y-2">
              {gaugeRows.map((row) => (
                <div key={row.name} className="flex items-center justify-between rounded-lg border border-neutral-100 bg-white px-4 py-3 transition hover:border-neutral-200 hover:shadow-sm">
                  <span className="font-mono text-sm text-neutral-700">{row.name}</span>
                  <span className="font-mono text-sm font-semibold text-neutral-900">{row.value}</span>
                </div>
              ))}
              {gaugeRows.length === 0 && (
                <div className="py-8 text-center text-sm text-neutral-400">—</div>
              )}
            </div>
          </Card>
        </Col>
      </Row>
    </Space>
  )
}

export default function SystemStatusPage() {
  const { t } = useTranslation()
  const [data, setData] = useState<SystemStatusPayload | null>(null)
  const [loading, setLoading] = useState(false)

  const load = useCallback(async (refresh = false) => {
    setLoading(true)
    try {
      const res = await fetchSystemStatus(refresh)
      setData(res.data)
    } catch (e) {
      showAlert(extractApiErrorMessage(e, t('systemStatus.loadFailed')), 'error')
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void load(false)
    const timer = window.setInterval(() => void load(false), 30_000)
    return () => window.clearInterval(timer)
  }, [load])

  return (
    <BaseLayout
      title={t('systemStatus.title')}
      description={t('systemStatus.description')}
      actions={
        <Button icon={<RefreshCw size={16} />} loading={loading} onClick={() => void load(true)}>
          {t('systemStatus.refreshChecks')}
        </Button>
      }
    >
      <Tabs defaultActiveTab="overview" type="rounded">
        <TabPane key="overview" title={t('systemStatus.tabOverview')}>
          <OverviewTab data={data} loading={loading} />
        </TabPane>
        <TabPane key="runtime" title={t('systemStatus.tabRuntime')}>
          <RuntimeTab data={data} />
        </TabPane>
        <TabPane key="goroutine" title={t('systemStatus.tabGoroutine')}>
          <GoroutineTab data={data} />
        </TabPane>
        <TabPane key="host" title={t('systemStatus.tabHost')}>
          <HostTab data={data} />
        </TabPane>
        <TabPane key="business" title={t('systemStatus.tabBusiness')}>
          <BusinessTab data={data} />
        </TabPane>
        <TabPane key="ops" title={t('systemStatus.tabOps')}>
          <OpsTab data={data} />
        </TabPane>
      </Tabs>
    </BaseLayout>
  )
}