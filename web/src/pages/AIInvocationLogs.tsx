import { useEffect, useMemo, useState } from 'react'
import { Collapse, Drawer, Tag } from '@arco-design/web-react'
import dayjs from 'dayjs'
import { Eye, RefreshCw, Search } from 'lucide-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, DataList, Input, Select } from '@/components/ui'
import type { DataListColumn } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  getAdminAIInvocation,
  getTenantAIInvocation,
  listAdminAIInvocations,
  listTenantAIInvocations,
  type AIInvocationLog,
} from '@/api/aiInvocations'

const COMPONENT_OPTIONS = [
  { label: '全部', value: '' },
  { label: 'LLM', value: 'llm' },
  { label: 'ASR', value: 'asr' },
  { label: 'TTS', value: 'tts' },
  { label: 'NLU', value: 'nlu' },
]

const STATUS_OPTIONS = [
  { label: '全部', value: '' },
  { label: 'ok', value: 'ok' },
  { label: 'error', value: 'error' },
]

const SOURCE_OPTIONS = [
  { label: '全部来源', value: '' },
  { label: '文本调试', value: 'assistant_debug_text' },
  { label: '语音调试', value: 'assistant_debug_voice' },
  { label: 'JS 模版', value: 'js_template' },
  { label: 'JS 嵌入(API Key)', value: 'js_embed' },
  { label: '语音会话', value: 'voice' },
  { label: '语音会话', value: 'voice_session' },
  { label: 'API', value: 'api' },
]

function formatSourceLabel(source?: string) {
  const s = (source || '').trim()
  if (!s) return '—'
  const base = s.replace(/_stream$/, '')
  const stream = s.endsWith('_stream')
  const map: Record<string, string> = {
    assistant_debug_text: '文本调试',
    assistant_debug_voice: '语音调试',
    js_template: 'JS 模版',
    js_embed: 'JS 嵌入',
    voice: '语音会话',
    voice_session: '语音会话',
    api: 'API',
  }
  const label = map[base] || s
  return stream ? `${label}(流式)` : label
}

const componentColor = (c?: string) => {
  const v = (c || '').toLowerCase()
  if (v === 'llm') return 'arcoblue'
  if (v === 'asr') return 'purple'
  if (v === 'tts') return 'green'
  if (v === 'nlu') return 'orange'
  return 'gray'
}

const statusColor = (s?: string) => ((s || '').toLowerCase() === 'ok' ? 'green' : 'red')

function fmtMs(v?: number) {
  if (v == null || v < 0) return '—'
  return `${v} ms`
}

function fmtNum(v?: number) {
  if (v == null) return '—'
  return String(v)
}

function fmtTenantId(v?: string) {
  return v && v !== '0' ? v : '—'
}

function MetaField({ label, value }: { label: string; value?: React.ReactNode }) {
  if (value == null || value === '' || value === '—') return null
  return (
    <div className="min-w-0 rounded-md border border-gray-200 bg-gray-50/80 p-2.5 text-sm">
      <div className="mb-0.5 text-[11px] text-gray-500">{label}</div>
      <div className="break-words font-medium">{value}</div>
    </div>
  )
}

function InvocationDetail({ detail, platform }: { detail: AIInvocationLog; platform: boolean }) {
  const comp = (detail.component || '').toLowerCase()
  const isNLU = comp === 'nlu'
  const isLLM = comp === 'llm'
  const isASR = comp === 'asr'
  const isTTS = comp === 'tts'

  const statusColor = (s?: string) => ((s || '').toLowerCase() === 'ok' ? 'green' : 'red')

  const fmtTime = (s?: string) => (s ? dayjs(s).format('YYYY-MM-DD HH:mm:ss') : '—')

  let metaJson: Record<string, unknown> | undefined
  if (detail.meta_json) {
    try {
      metaJson = JSON.parse(detail.meta_json)
    } catch {
      // ignore
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Tag color={statusColor(detail.status)} className="!rounded-full">
          {detail.status}
        </Tag>
        <Tag color={componentColor(comp)} className="!rounded-full">
          {comp.toUpperCase()}
        </Tag>
      </div>

      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
        <MetaField label="时间" value={fmtTime(detail.created_at)} />
        <MetaField label="类型" value={detail.component} />
        <MetaField label="提供商" value={detail.provider} />
        <MetaField label="模型" value={detail.model} />
        <MetaField label="来源" value={formatSourceLabel(detail.source)} />
        {platform ? <MetaField label="租户 ID" value={fmtTenantId(detail.tenant_id)} /> : null}
        {platform && detail.user_id && detail.user_id !== '0' ? (
          <MetaField label="用户 ID" value={detail.user_id} />
        ) : null}
        {detail.call_id ? <MetaField label="会话 ID" value={<span className="break-all font-mono text-xs">{detail.call_id}</span>} /> : null}
      </div>

      <section>
        <h4 className="mb-2 text-xs font-semibold text-gray-500 uppercase tracking-wide">性能指标</h4>
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
          <MetaField label="延迟" value={fmtMs(detail.latency_ms)} />
          {isLLM ? (
            <>
              <MetaField label="首 token" value={fmtMs(detail.first_token_ms)} />
              <MetaField label="Prompt tokens" value={fmtNum(detail.prompt_tokens)} />
              <MetaField label="Completion tokens" value={fmtNum(detail.completion_tokens)} />
              <MetaField label="总 tokens" value={fmtNum(detail.total_tokens)} />
            </>
          ) : null}
          {(isASR || isTTS) ? (
            <>
              <MetaField label="音频时长" value={fmtMs(detail.audio_ms)} />
              <MetaField label="音频字节" value={fmtNum(detail.audio_bytes)} />
            </>
          ) : null}
          {isNLU ? (
            <>
              <MetaField label="意图" value={detail.intent_name} />
              <MetaField label="置信度" value={detail.confidence != null && detail.confidence > 0 ? detail.confidence.toFixed(4) : '—'} />
            </>
          ) : null}
          {detail.input_chars != null ? <MetaField label="输入字符" value={fmtNum(detail.input_chars)} /> : null}
          {detail.output_chars != null ? <MetaField label="输出字符" value={fmtNum(detail.output_chars)} /> : null}
        </div>
      </section>

      {detail.error_msg ? (
        <section className="rounded-md border border-red-200 bg-red-50 p-3">
          <h4 className="mb-1 text-xs font-semibold text-red-600 uppercase tracking-wide">错误信息</h4>
          <p className="mb-0 text-sm text-red-700">{detail.error_msg}</p>
        </section>
      ) : null}

      {platform ? (
        <Collapse bordered={false} expandIconPosition="right" defaultActiveKey={[]}>
          {isASR ? (
            <Collapse.Item header="识别结果" name="response">
              <pre className="max-h-72 overflow-auto whitespace-pre-wrap break-words rounded border border-gray-200 bg-gray-50/80 p-2 text-xs font-sans mb-0">
                {detail.response_text || '—'}
              </pre>
            </Collapse.Item>
          ) : isTTS ? (
            <Collapse.Item header="合成文本" name="request">
              <pre className="max-h-72 overflow-auto whitespace-pre-wrap break-words rounded border border-gray-200 bg-gray-50/80 p-2 text-xs font-sans mb-0">
                {detail.request_text || '—'}
              </pre>
            </Collapse.Item>
          ) : (
            <>
              {detail.request_text?.trim() ? (
                <Collapse.Item header="请求内容" name="request">
                  <pre className="max-h-72 overflow-auto whitespace-pre-wrap break-words rounded border border-gray-200 bg-gray-50/80 p-2 text-xs font-sans mb-0">
                    {detail.request_text}
                  </pre>
                </Collapse.Item>
              ) : null}
              {detail.response_text?.trim() ? (
                <Collapse.Item header="响应内容" name="response">
                  <pre className="max-h-72 overflow-auto whitespace-pre-wrap break-words rounded border border-gray-200 bg-gray-50/80 p-2 text-xs font-sans mb-0">
                    {detail.response_text}
                  </pre>
                </Collapse.Item>
              ) : null}
            </>
          )}
        </Collapse>
      ) : null}

      {metaJson && Object.keys(metaJson).length > 0 ? (
        <section>
          <h4 className="mb-1.5 text-xs font-semibold text-gray-500 uppercase tracking-wide">元数据</h4>
          <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-gray-50 p-3 text-xs">
            {JSON.stringify(metaJson, null, 2)}
          </pre>
        </section>
      ) : null}
    </div>
  )
}

type Props = {
  embedded?: boolean
  platform?: boolean
}

export default function AIInvocationLogsPage({ embedded = false, platform = false }: Props) {
  const [list, setList] = useState<AIInvocationLog[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [component, setComponent] = useState('')
  const [status, setStatus] = useState('')
  const [source, setSource] = useState('')
  const [callId, setCallId] = useState('')
  const [tenantId, setTenantId] = useState('')
  const [detail, setDetail] = useState<AIInvocationLog | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  const fetchList = async () => {
    setLoading(true)
    try {
      const opts = {
        component: component || undefined,
        status: status || undefined,
        source: source || undefined,
        call_id: callId.trim() || undefined,
        tenant_id: platform ? tenantId.trim() || undefined : undefined,
      }
      const res = platform
        ? await listAdminAIInvocations(page, pageSize, opts)
        : await listTenantAIInvocations(page, pageSize, opts)
      setList(res.list || [])
      setTotal(res.total || 0)
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, '加载 AI 调用记录失败'), 'error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchList()
  }, [page, pageSize, component, status, source, platform])

  const openDetail = async (id: string) => {
    setDetailLoading(true)
    setDetail(null)
    try {
      const row = platform ? await getAdminAIInvocation(id) : await getTenantAIInvocation(id)
      setDetail(row)
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, '读取详情失败'), 'error')
    } finally {
      setDetailLoading(false)
    }
  }

  const columns: DataListColumn<Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'time',
      title: '时间',
      width: 160,
      render: (_, r) => {
        const ts = String(r.created_at || '')
        const formatted = ts ? dayjs(ts).format('MM-DD HH:mm:ss') : '—'
        return <span className="text-sm text-neutral-700">{formatted}</span>
      },
    },
    ...(platform
      ? [{
          key: 'tenant',
          title: '租户',
          width: 100,
          render: (_: unknown, r: Record<string, unknown>) => <span className="text-sm text-neutral-700">{fmtTenantId(r.tenant_id as string)}</span>,
        }]
      : []),
    {
      key: 'component',
      title: '类型',
      width: 70,
      render: (_, r) => <Tag color={componentColor(String(r.component))} className="!rounded-full !text-xs">{String(r.component || '—').toUpperCase()}</Tag>,
    },
    {
      key: 'status',
      title: '状态',
      width: 60,
      render: (_, r) => <Tag color={statusColor(String(r.status))} className="!rounded-full !text-xs">{String(r.status)}</Tag>,
    },
    {
      key: 'source',
      title: '来源',
      width: 110,
      render: (_, r) => (
        <span className="truncate text-xs text-neutral-600" title={String(r.source || '')}>
          {formatSourceLabel(r.source as string)}
        </span>
      ),
    },
    {
      key: 'summary',
      title: '摘要',
      render: (_, r) => {
        const c = String(r.component || '').toLowerCase()
        let text = String(r.call_id || '—')
        if (c === 'nlu') text = String(r.intent_name || '—')
        else if (c === 'llm') text = r.total_tokens ? `${r.total_tokens} tk` : '—'
        return <span className="truncate text-sm text-neutral-700">{text}</span>
      },
    },
    {
      key: 'latency',
      title: '延迟',
      width: 70,
      render: (_, r) => <span className="font-mono text-xs text-neutral-500">{fmtMs(r.latency_ms as number)}</span>,
    },
    {
      key: 'actions',
      title: '',
      width: 60,
      render: (_, r) => (
        <Button size="mini" icon={<Eye size={12} />} onClick={() => void openDetail(String(r.id))}>
          详情
        </Button>
      ),
    },
  ], [platform])

  if (embedded) {
    return (
      <>
        <DataList
          data={list as unknown as Record<string, unknown>[]}
          columns={columns}
          loading={loading}
          rowKey="id"
          emptyText="暂无 AI 调用记录"
          pagination={{ current: page, pageSize, total, onChange: (p: number) => setPage(p) }}
          header={
            <div className="flex flex-wrap items-end gap-3">
              <Select className="w-36" placeholder="类型" options={COMPONENT_OPTIONS} value={component} onChange={(v) => { setComponent(v); setPage(1) }} />
              <Select className="w-36" placeholder="状态" options={STATUS_OPTIONS} value={status} onChange={(v) => { setStatus(v); setPage(1) }} />
              <Select className="w-44" placeholder="来源" options={SOURCE_OPTIONS} value={source} onChange={(v) => { setSource(v); setPage(1) }} />
              <Input className="w-48" placeholder="会话 ID" value={callId} onChange={setCallId} onPressEnter={() => { setPage(1); void fetchList() }} />
              {platform ? <Input className="w-40" placeholder="租户 ID" value={tenantId} onChange={setTenantId} onPressEnter={() => { setPage(1); void fetchList() }} /> : null}
              <Button icon={<Search size={14} />} onClick={() => void fetchList()}>查询</Button>
              <Button type="outline" icon={<RefreshCw size={14} />} onClick={() => void fetchList()}>刷新</Button>
            </div>
          }
        />
        <Drawer width="min(640px, 100vw)" title="AI 调用详情" visible={Boolean(detail) || detailLoading} onCancel={() => { setDetail(null); setDetailLoading(false) }} footer={null}>
          {detailLoading ? <div className="py-8 text-center text-neutral-400">加载中…</div> : detail ? <InvocationDetail detail={detail} platform={platform} /> : null}
        </Drawer>
      </>
    )
  }

  return (
    <BaseLayout
      title="AI 调用记录"
      description={platform ? '平台全量 AI 调用记录，详情可查看请求与响应正文' : 'LLM / ASR / TTS / NLU 调用指标；含文本/语音调试与 JS 模版调用'}
    >
      <DataList
        data={list as unknown as Record<string, unknown>[]}
        columns={columns}
        loading={loading}
        rowKey="id"
        emptyText="暂无 AI 调用记录"
       
        pagination={{ current: page, pageSize, total, onChange: (p: number) => setPage(p) }}
        header={
          <div className="flex flex-wrap items-end gap-3">
            <Select className="w-36" placeholder="类型" options={COMPONENT_OPTIONS} value={component} onChange={(v) => { setComponent(v); setPage(1) }} />
            <Select className="w-36" placeholder="状态" options={STATUS_OPTIONS} value={status} onChange={(v) => { setStatus(v); setPage(1) }} />
            <Select className="w-44" placeholder="来源" options={SOURCE_OPTIONS} value={source} onChange={(v) => { setSource(v); setPage(1) }} />
            <Input className="w-48" placeholder="会话 ID" value={callId} onChange={setCallId} onPressEnter={() => { setPage(1); void fetchList() }} />
            {platform ? <Input className="w-40" placeholder="租户 ID" value={tenantId} onChange={setTenantId} onPressEnter={() => { setPage(1); void fetchList() }} /> : null}
            <Button icon={<Search size={14} />} onClick={() => void fetchList()}>查询</Button>
            <Button type="outline" icon={<RefreshCw size={14} />} onClick={() => void fetchList()}>刷新</Button>
          </div>
        }
      />
      <Drawer width="min(640px, 100vw)" title="AI 调用详情" visible={Boolean(detail) || detailLoading} onCancel={() => { setDetail(null); setDetailLoading(false) }} footer={null}>
        {detailLoading ? <div className="py-8 text-center text-neutral-400">加载中…</div> : detail ? <InvocationDetail detail={detail} platform={platform} /> : null}
      </Drawer>
    </BaseLayout>
  )
}

export { AIInvocationLogsPage as AIInvocationLogsPanel }
