import { useEffect, useState } from 'react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/UI/Select'
import { listAdminChatMessages, listAdminChatSessions, listAdminLLMUsage, type AdminChatMessage, type AdminChatSession, type AdminLLMUsage } from '@/services/adminApi'

type TabType = 'sessions' | 'messages' | 'usage'

const ChatData = () => {
  const [tab, setTab] = useState<TabType>('sessions')
  const [search, setSearch] = useState('')
  const [sessionId, setSessionId] = useState('')
  const [success, setSuccess] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)

  const [sessions, setSessions] = useState<AdminChatSession[]>([])
  const [messages, setMessages] = useState<AdminChatMessage[]>([])
  const [usage, setUsage] = useState<AdminLLMUsage[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  const fetchData = async () => {
    setLoading(true)
    try {
      if (tab === 'sessions') {
        const res = await listAdminChatSessions({ page, pageSize, search: search || undefined })
        setSessions(res.items || [])
        setTotal(res.total || 0)
      } else if (tab === 'messages') {
        const res = await listAdminChatMessages({ page, pageSize, search: search || undefined, session_id: sessionId || undefined })
        setMessages(res.items || [])
        setTotal(res.total || 0)
      } else {
        const res = await listAdminLLMUsage({ page, pageSize, search: search || undefined, session_id: sessionId || undefined, success: success || undefined })
        setUsage(res.items || [])
        setTotal(res.total || 0)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [tab, page, search, sessionId, success])

  return (
    <AdminLayout title="会话与用量" description="查看 chat_sessions / chat_messages / llm_usage">
      <Card className="space-y-4">
        <div className="flex gap-3 flex-wrap">
          <Select value={tab} onValueChange={(v) => setTab(v as TabType)}>
            <SelectTrigger className="w-44"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="sessions">会话</SelectItem>
              <SelectItem value="messages">消息</SelectItem>
              <SelectItem value="usage">用量</SelectItem>
            </SelectContent>
          </Select>
          <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="搜索关键词" className="max-w-sm" />
          {(tab === 'messages' || tab === 'usage') && (
            <Input value={sessionId} onChange={(e) => setSessionId(e.target.value)} placeholder="按 session_id 过滤" className="max-w-sm" />
          )}
          {tab === 'usage' && (
            <Select value={success} onValueChange={setSuccess}>
              <SelectTrigger className="w-40"><SelectValue placeholder="全部结果" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="">全部结果</SelectItem>
                <SelectItem value="true">成功</SelectItem>
                <SelectItem value="false">失败</SelectItem>
              </SelectContent>
            </Select>
          )}
          <Button variant="outline" onClick={fetchData}>刷新</Button>
        </div>

        {tab === 'sessions' && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead><tr className="border-b"><th className="text-left py-2">ID</th><th>用户</th><th>智能体ID</th><th>Provider/Model</th><th>状态</th><th>更新时间</th></tr></thead>
              <tbody>{sessions.map((s) => (
                <tr key={s.id} className="border-b">
                  <td className="py-2">{s.id}</td><td>{s.user_id}</td><td>{s.agent_id ?? s.agentId}</td><td>{s.provider}/{s.model}</td><td>{s.status}</td><td>{s.updated_at}</td>
                </tr>
              ))}</tbody>
            </table>
          </div>
        )}

        {tab === 'messages' && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead><tr className="border-b"><th className="text-left py-2">ID</th><th>Session</th><th>Role</th><th>内容</th><th>RequestID</th><th>时间</th></tr></thead>
              <tbody>{messages.map((m) => (
                <tr key={m.id} className="border-b">
                  <td className="py-2">{m.id}</td><td>{m.session_id}</td><td>{m.role}</td><td className="max-w-[460px] truncate">{m.content}</td><td>{m.request_id}</td><td>{m.created_at}</td>
                </tr>
              ))}</tbody>
            </table>
          </div>
        )}

        {tab === 'usage' && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead><tr className="border-b"><th className="text-left py-2">RequestID</th><th>Session</th><th>Provider/Model</th><th>Tokens</th><th>UA/IP</th><th>响应摘要</th></tr></thead>
              <tbody>{usage.map((u) => (
                <tr key={u.request_id} className="border-b">
                  <td className="py-2">{u.request_id}</td><td>{u.session_id}</td><td>{u.provider}/{u.model}</td><td>{u.total_tokens}</td><td className="max-w-[340px] truncate">{u.user_agent} / {u.ip_address}</td><td className="max-w-[420px] truncate">{u.response_content}</td>
                </tr>
              ))}</tbody>
            </table>
          </div>
        )}

        <div className="flex justify-between text-sm">
          <span>共 {total} 条</span>
          <div className="flex gap-2">
            <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>上一页</Button>
            <Button size="sm" variant="outline" disabled={page * pageSize >= total} onClick={() => setPage((p) => p + 1)}>下一页</Button>
          </div>
        </div>
        {loading && <div className="text-sm text-slate-500">加载中...</div>}
      </Card>
    </AdminLayout>
  )
}

export default ChatData
