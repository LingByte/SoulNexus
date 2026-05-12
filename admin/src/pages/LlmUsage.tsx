// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// LLM 用量统计：复用现有 /api/admin/llm-usage 接口；
// 重点把 channel_attempts JSON、success/失败、TTFT/TPS 等清晰展示出来。
import { useEffect, useState } from 'react'
import {
  Table,
  Input,
  Select,
  Button,
  Tag,
  Drawer,
  Message,
  Space,
} from '@arco-design/web-react'
import { Search, RefreshCw, Eye } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import { listAdminLLMUsage, type AdminLLMUsage } from '@/services/adminApi'

const Option = Select.Option

interface ExtendedUsage extends AdminLLMUsage {
  user_id?: string
  base_url?: string
  channel_id?: number
  channel_attempts?: any[]
  ttft_ms?: number
  tps?: number
  quota_delta?: number
  status_code?: number
  error_code?: string
  completed_at?: string
}

// 尝试把存为字符串的 JSON 美化输出；非 JSON 则原样返回。
const formatJSONIfPossible = (s: string): string => {
  if (!s) return ''
  const trimmed = s.trim()
  if (!trimmed.startsWith('{') && !trimmed.startsWith('[')) return s
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2)
  } catch {
    return s
  }
}

const LlmUsage = () => {
  const [rows, setRows] = useState<ExtendedUsage[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [successFilter, setSuccessFilter] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const [detail, setDetail] = useState<ExtendedUsage | null>(null)

  const fetchRows = async () => {
    setLoading(true)
    try {
      const r = await listAdminLLMUsage({
        page,
        pageSize,
        search: search || undefined,
        success: successFilter || undefined,
      })
      setRows((r.items || []) as ExtendedUsage[])
      setTotal(r.total || 0)
    } catch (e: any) {
      Message.error(`加载失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchRows()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, search, successFilter])

  const onSearch = () => { setPage(1); setSearch(searchInput) }

  const fmt = (s?: string) => (s ? new Date(s).toLocaleString('zh-CN') : '-')

  const columns = [
    {
      title: '请求时间',
      dataIndex: 'requested_at',
      width: 170,
      render: (v: string) => fmt(v),
    },
    { title: '模型', dataIndex: 'model', width: 160, ellipsis: true },
    {
      title: '供应商',
      dataIndex: 'provider',
      width: 100,
      render: (v: string) => <Tag>{v || '-'}</Tag>,
    },
    {
      title: '渠道',
      dataIndex: 'channel_id',
      width: 100,
      render: (v: number) => (v ? `#${v}` : '-'),
    },
    {
      title: '状态',
      dataIndex: 'success',
      width: 90,
      render: (v: boolean) => <Tag color={v ? 'green' : 'red'}>{v ? '成功' : '失败'}</Tag>,
    },
    {
      title: 'Tokens',
      dataIndex: 'total_tokens',
      width: 130,
      render: (_: any, r: ExtendedUsage) => (
        <Space size="mini">
          <Tag size="small" color="arcoblue">in {r.input_tokens}</Tag>
          <Tag size="small" color="purple">out {r.output_tokens}</Tag>
        </Space>
      ),
    },
    { title: '延迟', dataIndex: 'latency_ms', width: 90, render: (v: number) => `${v}ms` },
    {
      title: 'TTFT',
      dataIndex: 'ttft_ms',
      width: 90,
      render: (v: number) => (v ? `${v}ms` : '-'),
    },
    { title: '用户', dataIndex: 'user_id', width: 110, render: (v: string) => v || '-' },
    { title: 'IP', dataIndex: 'ip_address', width: 130, render: (v: string) => v || '-' },
    {
      title: '操作',
      key: '__actions__',
      width: 100,
      fixed: 'right' as const,
      render: (_: any, row: ExtendedUsage) => (
        <Button size="mini" type="text" onClick={() => setDetail(row)}>
          <span className="inline-flex items-center gap-1"><Eye size={14} /> 详情</span>
        </Button>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title="LLM 用量统计"
        description="对外 /v1/* 网关与内部 in-process 调用的全量明细。流式调用包含 TTFT；多渠道重试在 channel_attempts 里。"
        actions={
          <Space wrap>
            <Input
              prefix={<Search size={14} />}
              placeholder="搜索 模型 / request_id / IP / UA"
              value={searchInput}
              onChange={(v) => setSearchInput(v)}
              onPressEnter={onSearch}
              allowClear
              onClear={() => { setPage(1); setSearch('') }}
              style={{ width: 280 }}
            />
            <Select value={successFilter} onChange={(v) => { setPage(1); setSuccessFilter(v) }} placeholder="状态" style={{ width: 130 }}>
              <Option value="">全部</Option>
              <Option value="true">成功</Option>
              <Option value="false">失败</Option>
            </Select>
            <Button type="primary" onClick={onSearch}>搜索</Button>
            <Button onClick={fetchRows}>
              <span className="inline-flex items-center gap-1"><RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新</span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(r: ExtendedUsage) => r.id}
        loading={loading}
        columns={columns}
        data={rows}
        scroll={{ x: 1500 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showTotal: (t: number) => `共 ${t} 条`,
          sizeCanChange: true,
          sizeOptions: [20, 50, 100, 200],
          onChange: (p: number, ps: number) => { setPage(p); setPageSize(ps) },
        }}
      />

      <Drawer
        title="用量详情"
        visible={!!detail}
        width={760}
        onCancel={() => setDetail(null)}
        footer={null}
        autoFocus={false}
      >
        {detail && (
          <div className="space-y-3 text-sm">
            <Row label="Request ID" value={detail.request_id} />
            <Row label="Session ID" value={detail.session_id || '-'} />
            <Row label="User" value={detail.user_id || '-'} />
            <Row label="Provider" value={detail.provider} />
            <Row label="Model" value={detail.model} />
            <Row label="Base URL" value={detail.base_url || '-'} />
            <Row label="Request Type" value={detail.request_type} />
            <Row
              label="状态"
              value={<Tag color={detail.success ? 'green' : 'red'}>{detail.success ? '成功' : '失败'}</Tag>}
            />
            {detail.error_code ? <Row label="错误码" value={detail.error_code} /> : null}
            {detail.error_message ? (
              <Row
                label="错误"
                value={
                  <pre className="text-xs whitespace-pre-wrap bg-[var(--color-fill-2)] p-2 rounded">
                    {detail.error_message}
                  </pre>
                }
              />
            ) : null}
            <Row label="Tokens" value={`in ${detail.input_tokens} / out ${detail.output_tokens} / total ${detail.total_tokens}`} />
            <Row label="额度扣减" value={String(detail.quota_delta ?? 0)} />
            <Row label="延迟" value={`${detail.latency_ms}ms`} />
            <Row label="TTFT" value={detail.ttft_ms ? `${detail.ttft_ms}ms` : '-'} />
            <Row label="TPS" value={detail.tps ? detail.tps.toFixed(2) : '-'} />
            <Row label="HTTP" value={detail.status_code ? String(detail.status_code) : '-'} />
            <Row label="渠道" value={detail.channel_id ? `#${detail.channel_id}` : '-'} />
            <Row label="UserAgent" value={detail.user_agent || '-'} />
            <Row label="IP" value={detail.ip_address || '-'} />
            <Row label="请求时间" value={fmt(detail.requested_at)} />
            <Row label="完成时间" value={fmt(detail.completed_at)} />
            {Array.isArray(detail.channel_attempts) && detail.channel_attempts.length > 0 ? (
              <Row
                label="多渠道尝试"
                value={
                  <div className="space-y-2">
                    {detail.channel_attempts.map((a: any, i: number) => (
                      <div
                        key={i}
                        className="rounded border border-[var(--color-border-2)] p-2 text-xs"
                      >
                        <Space size="mini" wrap>
                          <Tag size="small">#{a.order}</Tag>
                          <Tag size="small">channel {a.channel_id}</Tag>
                          <Tag size="small" color={a.success ? 'green' : 'red'}>
                            {a.success ? '成功' : '失败'}
                          </Tag>
                          {a.status_code ? <Tag size="small">HTTP {a.status_code}</Tag> : null}
                          {a.latency_ms ? <Tag size="small">{a.latency_ms}ms</Tag> : null}
                        </Space>
                        {a.base_url ? (
                          <div className="text-[var(--color-text-3)] mt-1 break-all">{a.base_url}</div>
                        ) : null}
                        {a.error_message ? (
                          <div className="text-red-500 mt-1 break-all">{a.error_code}: {a.error_message}</div>
                        ) : null}
                      </div>
                    ))}
                  </div>
                }
              />
            ) : null}
            {detail.request_content ? (
              <Row
                label="请求体"
                value={
                  <pre className="text-xs whitespace-pre-wrap bg-[var(--color-fill-2)] p-2 rounded max-h-96 overflow-auto">
                    {formatJSONIfPossible(detail.request_content)}
                  </pre>
                }
              />
            ) : null}
            {detail.response_content ? (
              <Row
                label="响应体"
                value={
                  <pre className="text-xs whitespace-pre-wrap bg-[var(--color-fill-2)] p-2 rounded max-h-96 overflow-auto">
                    {formatJSONIfPossible(detail.response_content)}
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

export default LlmUsage
