import { useEffect, useMemo, useState } from 'react'
import { Key, Plus, RefreshCw, Trash2, Copy, Check } from 'lucide-react'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Card from '@/components/UI/Card'
import Badge from '@/components/UI/Badge'
import Modal from '@/components/UI/Modal'
import ConfirmDialog from '@/components/UI/ConfirmDialog'
import { showAlert } from '@/utils/notification'
import {
  listMyLLMTokens,
  createMyLLMToken,
  updateMyLLMToken,
  regenerateMyLLMToken,
  deleteMyLLMToken,
  type LLMToken,
} from '@/api/llmTokens'

const STATUS_LABEL: Record<string, string> = {
  active: '启用',
  disabled: '停用',
  expired: '已过期',
}

const LLMTokenManager = () => {
  const [tokens, setTokens] = useState<LLMToken[]>([])
  const [loading, setLoading] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [editing, setEditing] = useState<LLMToken | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<LLMToken | null>(null)
  const [revealKey, setRevealKey] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const [name, setName] = useState('')
  const [tokenType, setTokenType] = useState<'llm' | 'asr' | 'tts'>('llm')
  const [group, setGroup] = useState('default')
  const [whitelist, setWhitelist] = useState('')

  const fetchTokens = async () => {
    setLoading(true)
    try {
      const r = await listMyLLMTokens({ pageSize: 100 })
      if (r.code === 200) setTokens(r.data?.tokens || [])
      else showAlert(r.msg || '加载失败', 'error')
    } catch (e: any) {
      showAlert(e?.msg || e?.message || '加载失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchTokens()
  }, [])

  const resetForm = () => {
    setName('')
    setTokenType('llm')
    setGroup('default')
    setWhitelist('')
    setEditing(null)
  }

  const openCreate = () => {
    resetForm()
    setCreateOpen(true)
  }

  const openEdit = (t: LLMToken) => {
    setEditing(t)
    setName(t.name)
    setTokenType((t.type as 'llm' | 'asr' | 'tts') || 'llm')
    setGroup(t.group || 'default')
    setWhitelist(t.model_whitelist || '')
    setCreateOpen(true)
  }

  const handleSave = async () => {
    try {
      const body = {
        name: name.trim(),
        type: tokenType,
        group: group.trim() || 'default',
        model_whitelist: whitelist.trim(),
      }
      if (editing) {
        const r = await updateMyLLMToken(editing.id, body)
        if (r.code === 200) {
          showAlert('更新成功', 'success')
          setCreateOpen(false)
          resetForm()
          fetchTokens()
        } else throw new Error(r.msg)
      } else {
        const r = await createMyLLMToken(body)
        if (r.code === 200) {
          showAlert('创建成功', 'success')
          setCreateOpen(false)
          resetForm()
          fetchTokens()
          if (r.data?.raw_api_key) setRevealKey(r.data.raw_api_key)
        } else throw new Error(r.msg)
      }
    } catch (e: any) {
      showAlert(e?.msg || e?.message || '保存失败', 'error')
    }
  }

  const handleRegenerate = async (t: LLMToken) => {
    try {
      const r = await regenerateMyLLMToken(t.id)
      if (r.code === 200) {
        showAlert('已重置 API Key', 'success')
        fetchTokens()
        if (r.data?.raw_api_key) setRevealKey(r.data.raw_api_key)
      } else throw new Error(r.msg)
    } catch (e: any) {
      showAlert(e?.msg || e?.message || '重置失败', 'error')
    }
  }

  const handleDelete = async () => {
    if (!confirmDelete) return
    try {
      const r = await deleteMyLLMToken(confirmDelete.id)
      if (r.code === 200) {
        showAlert('已删除', 'success')
        setConfirmDelete(null)
        fetchTokens()
      } else throw new Error(r.msg)
    } catch (e: any) {
      showAlert(e?.msg || e?.message || '删除失败', 'error')
    }
  }

  const copyKey = async (key: string) => {
    try {
      await navigator.clipboard.writeText(key)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      showAlert('复制失败，请手动选中', 'error')
    }
  }

  const stats = useMemo(() => {
    const total = tokens.length
    const active = tokens.filter((t) => t.status === 'active').length
    const used = tokens.reduce((s, t) => s + (t.token_used || 0), 0)
    const requests = tokens.reduce((s, t) => s + (t.request_used || 0), 0)
    return { total, active, used, requests }
  }, [tokens])

  return (
    <div className="space-y-6">
      <Card>
        <div className="p-5 md:p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <Key className="w-4 h-4 text-sky-600" />
              <h3 className="text-base font-semibold text-gray-900 dark:text-white">LLM API Token</h3>
              <Badge variant="primary" className="text-[10px]">/v1/* 网关</Badge>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" leftIcon={<RefreshCw className="w-3.5 h-3.5" />} onClick={fetchTokens} disabled={loading}>
                刷新
              </Button>
              <Button variant="primary" size="sm" leftIcon={<Plus className="w-3.5 h-3.5" />} onClick={openCreate}>
                新建 Token
              </Button>
            </div>
          </div>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            用于调用统一的 OpenAI / Anthropic 兼容 API（<code className="font-mono">/v1/chat/completions</code> 等）。
            按模型计费倍率自动扣减额度，与传统密钥（Credential）相互独立。
          </p>
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 mt-5">
            <StatCell label="Token 数量" value={`#${stats.total}`} />
            <StatCell label="启用中" value={String(stats.active)} />
            <StatCell label="累计 Token 用量" value={stats.used.toLocaleString()} />
            <StatCell label="累计请求数" value={stats.requests.toLocaleString()} />
          </div>
        </div>
      </Card>

      <Card>
        <div className="p-5 md:p-6">
          {tokens.length === 0 ? (
            <div className="text-center py-12">
              <Key className="w-10 h-10 text-gray-400 mx-auto mb-3" />
              <h4 className="text-base font-medium text-gray-900 dark:text-white mb-1">还没有 API Token</h4>
              <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">创建一个 Token 即可调用统一 LLM 网关。</p>
              <Button variant="primary" size="sm" leftIcon={<Plus className="w-3.5 h-3.5" />} onClick={openCreate}>
                创建第一个 Token
              </Button>
            </div>
          ) : (
            <div className="space-y-3">
              {tokens.map((t) => {
                const ratio = t.unlimited_quota || t.token_quota <= 0 ? 0 : Math.min(100, Math.round((t.token_used / t.token_quota) * 100))
                return (
                  <div
                    key={t.id}
                    className="rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50/60 dark:bg-gray-800/40 px-4 py-3"
                  >
                    <div className="flex items-start justify-between gap-3 flex-wrap">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="font-medium text-gray-900 dark:text-white">{t.name || `Token #${t.id}`}</span>
                          <Badge
                            variant={t.status === 'active' ? 'success' : t.status === 'expired' ? 'warning' : 'secondary'}
                            className="text-[10px]"
                          >
                            {STATUS_LABEL[t.status] || t.status}
                          </Badge>
                          <Badge variant="primary" className="text-[10px]">{(t.type || 'llm').toUpperCase()}</Badge>
                          <Badge variant="secondary" className="text-[10px]">分组：{t.group}</Badge>
                          {t.unlimited_quota && <Badge variant="primary" className="text-[10px]">不限额</Badge>}
                        </div>
                        <div className="mt-1.5 font-mono text-xs text-gray-600 dark:text-gray-300 break-all">{t.api_key}</div>
                        <div className="mt-2 grid grid-cols-2 md:grid-cols-4 gap-2 text-xs text-gray-500 dark:text-gray-400">
                          <Stat label="Token" value={t.unlimited_quota ? `${t.token_used.toLocaleString()} / ∞` : `${t.token_used.toLocaleString()} / ${t.token_quota.toLocaleString()}`} />
                          <Stat label="请求" value={t.request_quota > 0 ? `${t.request_used.toLocaleString()} / ${t.request_quota.toLocaleString()}` : t.request_used.toLocaleString()} />
                          <Stat label="过期" value={t.expires_at ? new Date(t.expires_at).toLocaleString('zh-CN') : '永久'} />
                          <Stat label="最近使用" value={t.last_used_at ? new Date(t.last_used_at).toLocaleString('zh-CN') : '从未'} />
                        </div>
                        {!t.unlimited_quota && t.token_quota > 0 && (
                          <div className="mt-2 h-1.5 bg-gray-200 dark:bg-gray-700 rounded">
                            <div
                              className={`h-1.5 rounded ${ratio >= 100 ? 'bg-red-500' : ratio >= 80 ? 'bg-amber-500' : 'bg-sky-500'}`}
                              style={{ width: `${ratio}%` }}
                            />
                          </div>
                        )}
                        {t.model_whitelist && (
                          <div className="mt-2 text-[11px] text-gray-500 dark:text-gray-400">
                            白名单：{t.model_whitelist}
                          </div>
                        )}
                      </div>
                      <div className="flex items-center gap-1.5 shrink-0">
                        <Button variant="outline" size="sm" onClick={() => openEdit(t)}>编辑</Button>
                        <Button variant="outline" size="sm" leftIcon={<RefreshCw className="w-3 h-3" />} onClick={() => handleRegenerate(t)}>
                          重置 Key
                        </Button>
                        <Button variant="outline" size="sm" leftIcon={<Trash2 className="w-3 h-3" />} onClick={() => setConfirmDelete(t)}>
                          删除
                        </Button>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </Card>

      <Modal isOpen={createOpen} onClose={() => setCreateOpen(false)} title={editing ? '编辑 Token' : '新建 Token'} size="md">
        <div className="p-6 space-y-4">
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">名称</label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="便于识别用途" />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">类型</label>
            <div className="grid grid-cols-3 gap-2">
              {([
                { v: 'llm', label: 'LLM · 聊天补全', desc: '/v1/chat/completions /v1/messages' },
                { v: 'asr', label: 'ASR · 语音识别', desc: '/v1/audio/transcriptions' },
                { v: 'tts', label: 'TTS · 语音合成', desc: '/v1/audio/speech' },
              ] as const).map((opt) => {
                const active = tokenType === opt.v
                return (
                  <button
                    key={opt.v}
                    type="button"
                    onClick={() => setTokenType(opt.v)}
                    className={`text-left rounded-lg border px-3 py-2 transition ${
                      active
                        ? 'border-sky-500 bg-sky-50 dark:bg-sky-900/20'
                        : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
                    }`}
                  >
                    <div className="text-xs font-medium text-gray-900 dark:text-white">{opt.label}</div>
                    <div className="mt-0.5 text-[10px] font-mono text-gray-500 dark:text-gray-400 break-all">{opt.desc}</div>
                  </button>
                )
              })}
            </div>
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">分组 group</label>
            <Input value={group} onChange={(e) => setGroup(e.target.value)} placeholder="default" />
            <p className="text-[11px] text-gray-500 dark:text-gray-400 mt-1">决定路由到哪一批 abilities / 语音渠道，由管理员预先配置。</p>
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">模型白名单（可选）</label>
            <textarea
              value={whitelist}
              onChange={(e) => setWhitelist(e.target.value)}
              rows={2}
              placeholder="逗号分隔，如 gpt-4o-mini,gpt-4o；留空则不限制"
              className="w-full px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="outline" size="sm" onClick={() => setCreateOpen(false)}>取消</Button>
            <Button variant="primary" size="sm" onClick={handleSave}>保存</Button>
          </div>
        </div>
      </Modal>

      <Modal isOpen={!!revealKey} onClose={() => setRevealKey(null)} title="API Key 已生成" size="md">
        <div className="p-6 space-y-3">
          <p className="text-sm text-amber-600 dark:text-amber-400">
            完整 Key 仅展示一次，请立即复制保存。离开后将无法再次查看明文。
          </p>
          <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 p-3 break-all font-mono text-sm">
            {revealKey}
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={() => setRevealKey(null)}>关闭</Button>
            <Button
              variant="primary"
              size="sm"
              leftIcon={copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
              onClick={() => revealKey && copyKey(revealKey)}
            >
              {copied ? '已复制' : '复制 API Key'}
            </Button>
          </div>
        </div>
      </Modal>

      <ConfirmDialog
        isOpen={!!confirmDelete}
        onClose={() => setConfirmDelete(null)}
        onConfirm={handleDelete}
        title="删除 Token"
        message={`确认删除「${confirmDelete?.name || `Token #${confirmDelete?.id}`}」？删除后无法恢复，使用该 Key 的客户端将立即失效。`}
        confirmText="删除"
        cancelText="取消"
        type="danger"
      />
    </div>
  )
}

const Stat = ({ label, value }: { label: string; value: string }) => (
  <div>
    <div className="text-[10px] uppercase tracking-wide">{label}</div>
    <div className="text-gray-700 dark:text-gray-200 font-medium">{value}</div>
  </div>
)

const StatCell = ({ label, value }: { label: string; value: string }) => (
  <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50/80 dark:bg-gray-800/40 px-4 py-3">
    <div className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">{label}</div>
    <div className="text-2xl font-semibold tabular-nums text-gray-900 dark:text-white mt-1 font-mono">{value}</div>
  </div>
)

export default LLMTokenManager
