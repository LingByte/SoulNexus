// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 会话明细：Tabs 切换 chat_sessions / chat_messages。
// 用量统计已迁移到 /llm-usage 页面，本页不再展示 llm_usage。
import { useEffect, useState } from 'react'
import { Tabs, Table, Input, Button, Tag, Message, Space, Drawer } from '@arco-design/web-react'
import { Search, RefreshCw, Eye } from 'lucide-react'
import PageHeader from '@/components/Layout/PageHeader'
import {
  listAdminChatMessages,
  listAdminChatSessions,
  type AdminChatMessage,
  type AdminChatSession,
} from '@/services/adminApi'

const TabPane = Tabs.TabPane

type TabType = 'sessions' | 'messages'

const ChatData = () => {
  const [tab, setTab] = useState<TabType>('sessions')
  const [search, setSearch] = useState('')
  const [sessionId, setSessionId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const [sessions, setSessions] = useState<AdminChatSession[]>([])
  const [messages, setMessages] = useState<AdminChatMessage[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  // 详情抽屉：消息内容 / 完整响应等长字段。
  const [detail, setDetail] = useState<{ title: string; rows: Array<{ label: string; value: any }> } | null>(null)

  const fetchData = async () => {
    setLoading(true)
    try {
      if (tab === 'sessions') {
        const res = await listAdminChatSessions({ page, pageSize, search: search || undefined })
        setSessions(res.items || [])
        setTotal(res.total || 0)
      } else {
        const res = await listAdminChatMessages({
          page,
          pageSize,
          search: search || undefined,
          session_id: sessionId || undefined,
        })
        setMessages(res.items || [])
        setTotal(res.total || 0)
      }
    } catch (e: any) {
      Message.error(`加载失败：${e?.msg || e?.message || e}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    setPage(1)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab])

  useEffect(() => {
    fetchData()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab, page, pageSize, sessionId])

  const handleSearch = () => {
    setPage(1)
    fetchData()
  }

  const sessionColumns = [
    { title: 'ID', dataIndex: 'id', width: 220, ellipsis: true },
    { title: '用户', dataIndex: 'user_id', width: 100 },
    {
      title: '智能体',
      dataIndex: 'agent_id',
      width: 120,
      render: (_: any, r: AdminChatSession) => (r as any).agent_id ?? (r as any).agentId ?? '-',
    },
    {
      title: 'Provider / Model',
      dataIndex: 'provider',
      ellipsis: true,
      render: (_: any, r: AdminChatSession) => (
        <span>
          <Tag color="arcoblue">{r.provider}</Tag>
          <span className="ml-1 text-xs text-[var(--color-text-3)]">{r.model}</span>
        </span>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (s: string) => <Tag color={s === 'active' ? 'green' : 'gray'}>{s || '-'}</Tag>,
    },
    { title: '更新时间', dataIndex: 'updated_at', width: 170, render: (v: string) => v || '-' },
  ]

  const messageColumns = [
    { title: 'ID', dataIndex: 'id', width: 200, ellipsis: true },
    { title: 'Session', dataIndex: 'session_id', width: 200, ellipsis: true },
    {
      title: 'Role',
      dataIndex: 'role',
      width: 100,
      render: (v: string) => <Tag color={v === 'user' ? 'arcoblue' : v === 'assistant' ? 'green' : 'gray'}>{v}</Tag>,
    },
    { title: '内容', dataIndex: 'content', ellipsis: true },
    { title: 'RequestID', dataIndex: 'request_id', width: 220, ellipsis: true },
    { title: '时间', dataIndex: 'created_at', width: 170 },
    {
      title: '操作',
      key: '__actions__',
      width: 90,
      fixed: 'right' as const,
      render: (_: any, r: AdminChatMessage) => (
        <Button
          size="mini"
          type="text"
          onClick={() =>
            setDetail({
              title: '消息详情',
              rows: [
                { label: 'ID', value: r.id },
                { label: 'Session', value: r.session_id },
                { label: 'Role', value: r.role },
                { label: 'RequestID', value: r.request_id },
                { label: '时间', value: r.created_at },
                { label: '内容', value: r.content },
              ],
            })
          }
        >
          <span className="inline-flex items-center gap-1"><Eye size={14} /> 详情</span>
        </Button>
      ),
    },
  ]

  const data: any[] = tab === 'sessions' ? sessions : messages
  const columns: any[] = tab === 'sessions' ? sessionColumns : messageColumns
  const rowKey = (r: any) => String(r.id || r.request_id)

  return (
    <div className="space-y-4">
      <PageHeader
        title="会话明细"
        description="查看 chat_sessions / chat_messages 两类明细数据。Token / 费用统计请到 「LLM 用量」页面查看。"
        actions={
          <Space>
            <Input
              prefix={<Search size={14} />}
              placeholder="关键词"
              value={search}
              onChange={(v) => setSearch(v)}
              onPressEnter={handleSearch}
              allowClear
              style={{ width: 200 }}
            />
            {tab === 'messages' && (
              <Input
                placeholder="session_id"
                value={sessionId}
                onChange={(v) => setSessionId(v)}
                onPressEnter={handleSearch}
                allowClear
                style={{ width: 200 }}
              />
            )}
            <Button type="primary" onClick={handleSearch}>搜索</Button>
            <Button onClick={fetchData}>
              <span className="inline-flex items-center gap-1"><RefreshCw size={14} /> 刷新</span>
            </Button>
          </Space>
        }
      />

      <Tabs activeTab={tab} onChange={(k) => setTab(k as TabType)}>
        <TabPane key="sessions" title="会话" />
        <TabPane key="messages" title="消息" />
      </Tabs>

      <Table
        rowKey={rowKey}
        loading={loading}
        columns={columns}
        data={data}
        scroll={{ x: 1300 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showTotal: (t: number) => `共 ${t} 条`,
          sizeCanChange: true,
          sizeOptions: [10, 20, 50, 100],
          onChange: (p: number, ps: number) => { setPage(p); setPageSize(ps) },
        }}
      />

      <Drawer
        title={detail?.title || '详情'}
        visible={!!detail}
        width={720}
        onCancel={() => setDetail(null)}
        footer={null}
        autoFocus={false}
      >
        {detail && (
          <div className="space-y-3 text-sm">
            {detail.rows.map((row) => (
              <div key={row.label} className="rounded-md border border-[var(--color-border-2)] p-3">
                <div className="text-xs uppercase tracking-wide text-[var(--color-text-3)] mb-1">{row.label}</div>
                <pre className="m-0 break-all whitespace-pre-wrap text-[var(--color-text-1)] text-xs">
                  {row.value === null || row.value === undefined || row.value === '' ? '-' : String(row.value)}
                </pre>
              </div>
            ))}
          </div>
        )}
      </Drawer>
    </div>
  )
}

export default ChatData
