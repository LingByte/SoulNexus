// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 操作日志：Arco Table，支持按用户/动作/路径筛选；可展开行查看详细 UA / 设备等。
import { useEffect, useState } from 'react'
import { Table, Input, Button, Tag, Message, Space } from '@arco-design/web-react'
import { Search, RefreshCw } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import { getOperationLogs, type OperationLog } from '@/services/adminApi'

const methodColor = (m: string): any => {
  switch ((m || '').toUpperCase()) {
    case 'POST': return 'green'
    case 'PUT': return 'arcoblue'
    case 'PATCH': return 'orange'
    case 'DELETE': return 'red'
    default: return 'gray'
  }
}

const formatDate = (d?: string) => (d ? new Date(d).toLocaleString('zh-CN') : '-')

const OperationLogs = () => {
  const [logs, setLogs] = useState<OperationLog[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)

  const [userId, setUserId] = useState('')
  const [action, setAction] = useState('')
  const [target, setTarget] = useState('')

  const fetchLogs = async () => {
    setLoading(true)
    try {
      const params: any = { page, page_size: pageSize }
      if (userId) params.user_id = parseInt(userId, 10)
      if (action) params.action = action
      if (target) params.target = target
      const data = await getOperationLogs(params)
      setLogs(data.logs || [])
      setTotal(data.total || 0)
    } catch (e: any) {
      Message.error(`获取操作日志失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchLogs()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize])

  const handleSearch = () => {
    setPage(1)
    fetchLogs()
  }

  const columns = [
    {
      title: '时间',
      dataIndex: 'created_at',
      width: 170,
      render: (v: string) => formatDate(v),
    },
    {
      title: '方法',
      dataIndex: 'request_method',
      width: 80,
      render: (m: string) => <Tag color={methodColor(m)}>{m || '-'}</Tag>,
    },
    {
      title: '动作',
      dataIndex: 'details',
      ellipsis: true,
    },
    {
      title: '路径',
      dataIndex: 'target',
      ellipsis: true,
      width: 280,
      render: (v: string) => (
        <span style={{ fontFamily: 'ui-monospace, monospace', fontSize: 12 }}>{v || '-'}</span>
      ),
    },
    {
      title: '用户',
      dataIndex: 'user_id',
      width: 160,
      render: (_: any, r: OperationLog) => (
        <span>
          {r.username || '-'}
          <span className="ml-1 text-xs text-[var(--color-text-3)]">#{r.user_id}</span>
        </span>
      ),
    },
    { title: 'IP', dataIndex: 'ip_address', width: 130 },
    { title: '位置', dataIndex: 'location', width: 140, render: (v: string) => v || '-' },
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title="操作日志"
        description="审计后台与业务接口的写操作；点击行可展开查看 UA / 设备 / 来源页面。"
        actions={
          <Space>
            <Input
              prefix={<Search size={14} />}
              placeholder="用户 ID"
              value={userId}
              onChange={(v) => setUserId(v)}
              onPressEnter={handleSearch}
              allowClear
              style={{ width: 130 }}
            />
            <Input
              placeholder="操作类型"
              value={action}
              onChange={(v) => setAction(v)}
              onPressEnter={handleSearch}
              allowClear
              style={{ width: 160 }}
            />
            <Input
              placeholder="路径关键字"
              value={target}
              onChange={(v) => setTarget(v)}
              onPressEnter={handleSearch}
              allowClear
              style={{ width: 200 }}
            />
            <Button type="primary" onClick={handleSearch}>搜索</Button>
            <Button onClick={fetchLogs}>
              <span className="inline-flex items-center gap-1">
                <RefreshCw size={14} className={loading ? 'animate-spin' : ''} /> 刷新
              </span>
            </Button>
          </Space>
        }
      />

      <Table
        rowKey={(r: OperationLog) => String(r.id)}
        loading={loading}
        columns={columns}
        data={logs}
        scroll={{ x: 1300 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showTotal: (t: number) => `共 ${t} 条`,
          sizeCanChange: true,
          sizeOptions: [10, 20, 50, 100],
          onChange: (p: number, ps: number) => {
            setPage(p)
            setPageSize(ps)
          },
        }}
        expandedRowRender={(r: OperationLog) => (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-2 text-xs text-[var(--color-text-2)]">
            <Detail label="设备" value={r.device || '-'} />
            <Detail label="浏览器" value={r.browser || '-'} />
            <Detail label="操作系统" value={r.operating_system || '-'} />
            <Detail label="来源页面" value={r.referer || '-'} />
            <div className="md:col-span-2">
              <Detail label="User-Agent" value={r.user_agent || '-'} />
            </div>
          </div>
        )}
      />
    </div>
  )
}

const Detail = ({ label, value }: { label: string; value: React.ReactNode }) => (
  <div className="flex gap-2 break-all">
    <span className="w-20 shrink-0 text-[var(--color-text-3)]">{label}</span>
    <span className="flex-1">{value}</span>
  </div>
)

export default OperationLogs
