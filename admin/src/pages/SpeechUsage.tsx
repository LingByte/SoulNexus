// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// SpeechUsage 管理页：ASR / TTS 调用审计与计量。
//   - 顶部统计卡片：按 (kind, provider) 聚合的成功/失败次数
//   - 主表：可按 kind / token / group / channel / provider / 状态 / 时间筛选
//   - 详情 Drawer：展示请求快照、响应快照（不含原始音频）

import { useEffect, useMemo, useState } from 'react'
import {
  Table,
  Input,
  Select,
  Button,
  Tag,
  Drawer,
  Message,
  Space,
  DatePicker,
} from '@arco-design/web-react'
import { Search, RefreshCw, Eye } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listAdminSpeechUsage,
  listAdminSpeechUsageStats,
  type AdminSpeechUsage,
  type AdminSpeechUsageStatsRow,
} from '@/services/adminApi'

const Option = Select.Option

const fmt = (s?: string) => (s ? new Date(s).toLocaleString('zh-CN') : '-')
const fmtBytes = (n?: number) => {
  if (!n || n <= 0) return '-'
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(2)} MB`
}

const SpeechUsage = () => {
  const [rows, setRows] = useState<AdminSpeechUsage[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [kindFilter, setKindFilter] = useState<'asr' | 'tts' | ''>('')
  const [successFilter, setSuccessFilter] = useState<'true' | 'false' | ''>('')
  const [providerFilter, setProviderFilter] = useState('')
  const [range, setRange] = useState<string[]>([])

  const [stats, setStats] = useState<AdminSpeechUsageStatsRow[]>([])
  const [detail, setDetail] = useState<AdminSpeechUsage | null>(null)

  const fetchRows = async () => {
    setLoading(true)
    try {
      const r = await listAdminSpeechUsage({
        page,
        pageSize,
        search: search || undefined,
        kind: kindFilter || undefined,
        success: successFilter || undefined,
        provider: providerFilter || undefined,
        from: range[0] ? new Date(range[0]).toISOString() : undefined,
        to: range[1] ? new Date(range[1]).toISOString() : undefined,
      })
      setRows(r.items || [])
      setTotal(r.total || 0)
    } catch (e: any) {
      Message.error(`加载失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  const fetchStats = async () => {
    try {
      const r = await listAdminSpeechUsageStats()
      setStats(r.items || [])
    } catch {
      // 统计失败不阻塞主流程
    }
  }

  useEffect(() => {
    fetchRows()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, search, kindFilter, successFilter, providerFilter, range])

  useEffect(() => {
    fetchStats()
  }, [])

  const onSearch = () => {
    setPage(1)
    setSearch(searchInput)
  }

  const totalsByKind = useMemo(() => {
    const m: Record<string, { total: number; success: number; failed: number }> = {}
    stats.forEach((s) => {
      const k = s.kind || 'unknown'
      if (!m[k]) m[k] = { total: 0, success: 0, failed: 0 }
      m[k].total += s.total
      m[k].success += s.success
      m[k].failed += s.failed
    })
    return m
  }, [stats])

  const columns = [
    {
      title: '时间',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) => fmt(v),
    },
    {
      title: '类型',
      dataIndex: 'kind',
      width: 80,
      render: (v: string) => <Tag color={v === 'asr' ? 'arcoblue' : 'purple'}>{(v || '').toUpperCase()}</Tag>,
    },
    {
      title: '供应商 / 模型',
      dataIndex: 'provider',
      width: 220,
      ellipsis: true,
      render: (_: any, r: AdminSpeechUsage) => (
        <span>
          <Tag>{r.provider || '-'}</Tag>
          <span className="ml-1 text-xs text-[var(--color-text-3)]">{r.model || '-'}</span>
        </span>
      ),
    },
    {
      title: '租户 / 路由',
      dataIndex: 'group_key',
      width: 160,
      render: (_: any, r: AdminSpeechUsage) => (
        <span className="text-xs">
          org #{r.group_id || 0} <span className="text-[var(--color-text-3)]">/</span> {r.group_key || '-'}
        </span>
      ),
    },
    {
      title: '渠道',
      dataIndex: 'channel_id',
      width: 80,
      render: (v: number) => (v ? `#${v}` : '-'),
    },
    {
      title: '用户 / Token',
      dataIndex: 'user_id',
      width: 130,
      render: (_: any, r: AdminSpeechUsage) => (
        <span className="text-xs">
          u#{r.user_id} / t#{r.token_id}
        </span>
      ),
    },
    {
      title: '状态',
      dataIndex: 'success',
      width: 80,
      render: (v: boolean, r: AdminSpeechUsage) => (
        <Tag color={v ? 'green' : 'red'}>{v ? '成功' : `失败 ${r.status || ''}`}</Tag>
      ),
    },
    { title: '音频', dataIndex: 'audio_bytes', width: 100, render: (v: number) => fmtBytes(v) },
    {
      title: '文本',
      dataIndex: 'text_chars',
      width: 90,
      render: (v: number) => (v ? `${v} 字` : '-'),
    },
    {
      title: '耗时',
      dataIndex: 'duration_ms',
      width: 90,
      render: (v: number) => (v ? `${v}ms` : '-'),
    },
    { title: 'IP', dataIndex: 'client_ip', width: 130, ellipsis: true },
    {
      title: '操作',
      key: '__actions__',
      width: 80,
      fixed: 'right' as const,
      render: (_: any, r: AdminSpeechUsage) => (
        <Button size="mini" type="text" onClick={() => setDetail(r)}>
          <span className="inline-flex items-center gap-1">
            <Eye size={14} /> 详情
          </span>
        </Button>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title="语音用量审计"
        description="ASR / TTS 网关全量调用明细：失败/成功、计量字节与字符、命中渠道、租户归属。"
        actions={
          <Space wrap>
            <Input
              prefix={<Search size={14} />}
              placeholder="模糊：暂仅依赖筛选项"
              value={searchInput}
              onChange={(v) => setSearchInput(v)}
              onPressEnter={onSearch}
              allowClear
              onClear={() => {
                setPage(1)
                setSearch('')
              }}
              style={{ width: 220 }}
            />
            <Select
              value={kindFilter}
              onChange={(v) => {
                setPage(1)
                setKindFilter(v)
              }}
              placeholder="类型"
              style={{ width: 110 }}
            >
              <Option value="">全部</Option>
              <Option value="asr">ASR</Option>
              <Option value="tts">TTS</Option>
            </Select>
            <Select
              value={successFilter}
              onChange={(v) => {
                setPage(1)
                setSuccessFilter(v)
              }}
              placeholder="状态"
              style={{ width: 110 }}
            >
              <Option value="">全部</Option>
              <Option value="true">成功</Option>
              <Option value="false">失败</Option>
            </Select>
            <Input
              placeholder="provider"
              value={providerFilter}
              onChange={(v) => {
                setPage(1)
                setProviderFilter(v)
              }}
              allowClear
              style={{ width: 140 }}
            />
            <DatePicker.RangePicker
              showTime
              onChange={(v) => {
                setPage(1)
                setRange(v || [])
              }}
            />
            <Button type="primary" onClick={onSearch}>
              搜索
            </Button>
            <Button
              onClick={() => {
                fetchRows()
                fetchStats()
              }}
            >
              <span className="inline-flex items-center gap-1">
                <RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新
              </span>
            </Button>
          </Space>
        }
      />

      {/* 统计卡片 */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {(['asr', 'tts'] as const).map((k) => {
          const v = totalsByKind[k] || { total: 0, success: 0, failed: 0 }
          const rate = v.total ? ((v.success / v.total) * 100).toFixed(1) : '0.0'
          return (
            <div
              key={k}
              className="rounded-lg border border-[var(--color-border-2)] bg-[var(--color-bg-2)] p-4"
            >
              <div className="flex items-center justify-between">
                <span className="text-sm text-[var(--color-text-3)]">{k.toUpperCase()} 调用</span>
                <Tag color={k === 'asr' ? 'arcoblue' : 'purple'}>{k.toUpperCase()}</Tag>
              </div>
              <div className="mt-2 text-2xl font-medium">{v.total}</div>
              <div className="mt-1 text-xs text-[var(--color-text-3)]">
                成功 {v.success} · 失败 {v.failed} · 成功率 {rate}%
              </div>
            </div>
          )
        })}
        <div className="rounded-lg border border-[var(--color-border-2)] bg-[var(--color-bg-2)] p-4 md:col-span-2">
          <div className="text-sm text-[var(--color-text-3)] mb-2">按 Provider 拆分</div>
          <div className="flex flex-wrap gap-2">
            {stats.length === 0 ? (
              <span className="text-xs text-[var(--color-text-3)]">暂无数据</span>
            ) : (
              stats.map((s) => (
                <Tag key={`${s.kind}-${s.provider}`} bordered color={s.failed > 0 ? 'orange' : 'green'}>
                  <span className="font-medium">{s.provider || 'unknown'}</span>
                  <span className="ml-1 text-[10px]">
                    [{s.kind}] {s.success}/{s.total}
                  </span>
                </Tag>
              ))
            )}
          </div>
        </div>
      </div>

      <Table
        rowKey={(r: AdminSpeechUsage) => r.id}
        loading={loading}
        columns={columns}
        data={rows}
        scroll={{ x: 1600 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showTotal: (t: number) => `共 ${t} 条`,
          sizeCanChange: true,
          sizeOptions: [20, 50, 100, 200],
          onChange: (p: number, ps: number) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
      />

      <Drawer
        title="语音用量详情"
        visible={!!detail}
        width={720}
        onCancel={() => setDetail(null)}
        footer={null}
        autoFocus={false}
      >
        {detail && (
          <div className="space-y-3 text-sm">
            <Row label="ID" value={detail.id} />
            <Row label="Request ID" value={detail.request_id} />
            <Row
              label="类型"
              value={<Tag color={detail.kind === 'asr' ? 'arcoblue' : 'purple'}>{(detail.kind || '').toUpperCase()}</Tag>}
            />
            <Row
              label="状态"
              value={
                <Tag color={detail.success ? 'green' : 'red'}>
                  {detail.success ? '成功' : `失败 HTTP ${detail.status || '-'}`}
                </Tag>
              }
            />
            {detail.error_msg ? (
              <Row
                label="错误信息"
                value={
                  <pre className="text-xs whitespace-pre-wrap bg-[var(--color-fill-2)] p-2 rounded">
                    {detail.error_msg}
                  </pre>
                }
              />
            ) : null}
            <Row label="供应商" value={detail.provider || '-'} />
            <Row label="模型" value={detail.model || '-'} />
            <Row label="渠道 ID" value={detail.channel_id ? `#${detail.channel_id}` : '-'} />
            <Row label="租户(Org) ID" value={String(detail.group_id || 0)} />
            <Row label="路由 group" value={detail.group_key || '-'} />
            <Row label="User / Token" value={`u#${detail.user_id} / t#${detail.token_id}`} />
            <Row label="音频字节" value={fmtBytes(detail.audio_bytes)} />
            <Row label="文本字符" value={detail.text_chars ? `${detail.text_chars} 字` : '-'} />
            <Row label="耗时" value={detail.duration_ms ? `${detail.duration_ms} ms` : '-'} />
            <Row label="客户端 IP" value={detail.client_ip || '-'} />
            <Row label="UserAgent" value={detail.user_agent || '-'} />
            <Row label="创建时间" value={fmt(detail.created_at)} />
            {detail.req_snap ? (
              <Row
                label="请求快照"
                value={
                  <pre className="text-xs whitespace-pre-wrap bg-[var(--color-fill-2)] p-2 rounded max-h-60 overflow-auto">
                    {detail.req_snap}
                  </pre>
                }
              />
            ) : null}
            {detail.resp_snap ? (
              <Row
                label="响应快照"
                value={
                  <pre className="text-xs whitespace-pre-wrap bg-[var(--color-fill-2)] p-2 rounded max-h-60 overflow-auto">
                    {detail.resp_snap}
                  </pre>
                }
              />
            ) : null}
          </div>
        )}
      </Drawer>
    </div>
  )
}

const Row = ({ label, value }: { label: string; value: React.ReactNode }) => (
  <div className="flex items-start gap-3">
    <div className="w-28 shrink-0 text-[var(--color-text-3)]">{label}</div>
    <div className="flex-1 break-all text-[var(--color-text-1)]">{value}</div>
  </div>
)

export default SpeechUsage
