import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { InputNumber, Space, Typography, Button, Tag } from '@arco-design/web-react'
import VoicePickerGrid from '@/components/voice/VoicePickerGrid'
import { Input, Checkbox, Select, Empty } from '@/components/ui'
import { listAssistantTools, mcpToolBindKey, type AssistantToolRow } from '@/api/assistantTools'
import {
  createDialogSkill,
  deleteDialogSkill,
  listDialogSkills,
  updateDialogSkill,
  uploadDialogSkillAssets,
  type TenantDialogSkillRow,
} from '@/api/dialog'
import ToolSchemaDetail from '@/components/mcp/ToolSchemaDetail'
import { httpToolSchema, mcpToolSchema, type JsonSchemaObject } from '@/components/mcp/toolSchema'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import { Drawer, Form, Modal, Switch } from '@arco-design/web-react'
import { isSnowflakeSet, snowflakeStr } from '@/api/voiceprint'
import type {
  AgentConfigDraft,
  AudioProcessConfigDraft,
  AudioTrackConfigDraft,
  HotWordDraft,
  InterruptionConfigDraft,
  McpServerDraft,
  QueryRewriterDraft,
  VadConfigDraft,
} from '@/constants/assistantAdvancedConfig'

const TextArea = Input.TextArea

export function AssistantDialogFields({
  welcome,
  setWelcome,
  description,
  setDescription,
  prompt,
  setPrompt,
  knowledgeNamespace,
  setKnowledgeNamespace,
  knowledgeOptions,
  nluModelId,
  setNluModelId,
  nluOptions,
  nluEnabled,
}: {
  welcome: string
  setWelcome: (v: string) => void
  description: string
  setDescription: (v: string) => void
  prompt: string
  setPrompt: (v: string) => void
  knowledgeNamespace: string
  setKnowledgeNamespace: (v: string) => void
  knowledgeOptions: { value: string; label: string }[]
  nluModelId: string
  setNluModelId: (v: string) => void
  nluOptions: { value: string; label: string }[]
  nluEnabled?: boolean
}) {
  return (
    <div className="space-y-4">
      <div className="rounded-xl border border-border bg-card p-5 space-y-3">
        <Typography.Text bold>开场白</Typography.Text>
        <TextArea
          placeholder="请输入欢迎语，例如：您好，请问有什么可以帮您？"
          value={welcome}
          autoSize={{ minRows: 3, maxRows: 8 }}
          onChange={setWelcome}
        />
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          会话开始后 AI 主动说出的第一句话。
        </Typography.Text>
      </div>

      <div className="rounded-xl border border-border bg-card p-5 space-y-3">
        <div className="flex items-center gap-2">
          <Typography.Text bold>提示词</Typography.Text>
          <Typography.Text type="error">*</Typography.Text>
        </div>
        <Input
          placeholder="一句话描述智能体角色，例如：旅游景区客服"
          value={description}
          onChange={setDescription}
        />
        <TextArea
          placeholder="请输入系统提示词（Prompt）"
          value={prompt}
          autoSize={{ minRows: 6, maxRows: 20 }}
          onChange={setPrompt}
        />
      </div>

      <div className="rounded-xl border border-border bg-card p-5 space-y-3">
        <Typography.Text bold>知识库</Typography.Text>
        <Select
          allowClear
          placeholder="可选，关联知识库 Namespace"
          value={knowledgeNamespace || undefined}
          options={knowledgeOptions}
          onChange={(v) => setKnowledgeNamespace(String(v || ''))}
          style={{ width: '100%' }}
        />
      </div>

      <div className="rounded-xl border border-border bg-card p-5 space-y-3">
        <Typography.Text bold>NLU 意图模型</Typography.Text>
        <Select
          allowClear
          disabled={nluEnabled === false}
          placeholder={nluEnabled === false ? '平台未启用 NLU（NLU_ENABLED）' : '可选，绑定已训练的意图模型'}
          value={nluModelId || undefined}
          options={nluOptions}
          onChange={(v) => setNluModelId(String(v || ''))}
          style={{ width: '100%' }}
        />
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          语音 ASR 出字后先跑 NLU：置信度 ≥ 0.85 才走意图固定话术，否则把意图上下文交给 LLM。建议模型 minConfidence ≥ 0.85。
        </Typography.Text>
      </div>
    </div>
  )
}

export function AssistantBehaviorFields({
  agent,
  setAgent,
}: {
  agent: AgentConfigDraft
  setAgent: (v: AgentConfigDraft) => void
}) {
  return (
    <Space direction="vertical" size={14} style={{ width: '100%' }}>
      <Space wrap>
        <Checkbox checked={!!agent.enableHangupTool} onChange={(v) => setAgent({ ...agent, enableHangupTool: v })}>
          启用结束会话工具
        </Checkbox>
        <Checkbox checked={!!agent.startCanBreak} onChange={(v) => setAgent({ ...agent, startCanBreak: v })}>
          开场白可打断
        </Checkbox>
        <Checkbox checked={!!agent.useFiller} onChange={(v) => setAgent({ ...agent, useFiller: v })}>
          启用垫词
        </Checkbox>
      </Space>

      {agent.useFiller && (
        <Input
          placeholder="垫词列表，逗号分隔"
          value={(agent.fillerWords || []).join(',')}
          onChange={(v) =>
            setAgent({
              ...agent,
              fillerWords: v.split(',').map((s) => s.trim()).filter(Boolean),
            })
          }
        />
      )}

      <Space wrap align="center">
        <span className="text-sm text-muted-foreground">沉默追问次数</span>
        <InputNumber
          min={0}
          max={10}
          value={agent.maxSilentAskTimes}
          onChange={(v) => setAgent({ ...agent, maxSilentAskTimes: Number(v) || 0 })}
        />
        <span className="text-sm text-muted-foreground">无法解答上限</span>
        <InputNumber
          min={0}
          max={10}
          value={agent.maxDialogueGapTimes}
          onChange={(v) => setAgent({ ...agent, maxDialogueGapTimes: Number(v) || 0 })}
        />
        <span className="text-sm text-muted-foreground">知识库 TopK</span>
        <InputNumber min={1} max={20} value={agent.topK} onChange={(v) => setAgent({ ...agent, topK: Number(v) || 3 })} />
        <span className="text-sm text-muted-foreground">切片时长(ms)</span>
        <InputNumber
          min={1000}
          max={30000}
          step={500}
          value={agent.sliceTime}
          onChange={(v) => setAgent({ ...agent, sliceTime: Number(v) || 5000 })}
        />
      </Space>
    </Space>
  )
}

export function AssistantVadFields({
  vad,
  setVad,
}: {
  vad: VadConfigDraft
  setVad: (v: VadConfigDraft) => void
}) {
  return (
    <Space direction="vertical" size={14} style={{ width: '100%' }}>
      <Space wrap align="center">
        <span className="text-sm text-muted-foreground">VAD 模式</span>
        <Select
          style={{ width: 160 }}
          value={String(vad.vadMode ?? 3)}
          options={[
            { value: '1', label: '普通模式' },
            { value: '2', label: '低码率模式' },
            { value: '3', label: '激进模式' },
            { value: '4', label: '高灵敏度' },
          ]}
          onChange={(v) => setVad({ ...vad, vadMode: Number(v) })}
        />
        <span className="text-sm text-muted-foreground">能量阈值</span>
        <InputNumber
          min={2000}
          max={12000}
          step={100}
          value={vad.energyThreshold}
          onChange={(v) => setVad({ ...vad, energyThreshold: Number(v) || 5500 })}
        />
      </Space>
      <Space wrap align="center">
        <span className="text-sm text-muted-foreground">正向语音阈值</span>
        <InputNumber
          min={0}
          max={1}
          step={0.05}
          value={vad.positiveSpeechThreshold}
          onChange={(v) => setVad({ ...vad, positiveSpeechThreshold: Number(v) || 0.5 })}
        />
        <span className="text-sm text-muted-foreground">负向语音阈值</span>
        <InputNumber
          min={0}
          max={1}
          step={0.05}
          value={vad.negativeSpeechThreshold}
          onChange={(v) => setVad({ ...vad, negativeSpeechThreshold: Number(v) || 0.4 })}
        />
      </Space>
      <Space wrap align="center">
        <span className="text-sm text-muted-foreground">最小语音帧</span>
        <InputNumber
          min={1}
          max={50}
          value={vad.minSpeechFrames}
          onChange={(v) => setVad({ ...vad, minSpeechFrames: Number(v) || 10 })}
        />
        <span className="text-sm text-muted-foreground">补偿帧</span>
        <InputNumber
          min={1}
          max={50}
          value={vad.redemptionFrames}
          onChange={(v) => setVad({ ...vad, redemptionFrames: Number(v) || 10 })}
        />
      </Space>
    </Space>
  )
}


export function AssistantInterruptionFields({ cfg, setCfg }: { cfg: InterruptionConfigDraft; setCfg: (v: InterruptionConfigDraft) => void }) {
  return (
    <Space direction="vertical" size={14} style={{ width: '100%' }}>
      <Space wrap align="center">
        <span className="text-sm text-muted-foreground">打断方式</span>
        <Select style={{ width: 220 }} value={cfg.method || 'vad+transcribing'} options={[
          { value: 'vad+transcribing', label: 'VAD + 转写' },
          { value: 'vad', label: '仅 VAD' },
          { value: 'transcribing', label: '仅转写' },
          { value: 'strict_interrupt', label: '禁止打断' },
        ]} onChange={(v) => setCfg({ ...cfg, method: String(v) })} />
      </Space>
      <Space wrap align="center">
        <span className="text-sm text-muted-foreground">播放后不可打断(ms)</span>
        <InputNumber min={0} max={10000} value={cfg.unInterruptableAfterPlayStart} onChange={(v) => setCfg({ ...cfg, unInterruptableAfterPlayStart: Number(v) || 0 })} />
        <span className="text-sm text-muted-foreground">抢话字数阈值</span>
        <InputNumber min={0} max={5000} value={cfg.talkOverThreshold} onChange={(v) => setCfg({ ...cfg, talkOverThreshold: Number(v) || 0 })} />
        <Checkbox checked={!!cfg.resumePlay} onChange={(v) => setCfg({ ...cfg, resumePlay: v })}>打断后恢复播放</Checkbox>
      </Space>
    </Space>
  )
}

export function AssistantHotWordsFields({ rows, setRows }: { rows: HotWordDraft[]; setRows: (v: HotWordDraft[]) => void }) {
  const update = (idx: number, patch: Partial<HotWordDraft>) => setRows(rows.map((r, i) => (i === idx ? { ...r, ...patch } : r)))
  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      {rows.map((row, idx) => (
        <div key={idx} className="rounded-lg border border-border p-3 space-y-2">
          <Space wrap>
            <Input placeholder="热词" value={row.word} onChange={(v) => update(idx, { word: v })} style={{ width: 160 }} />
            <InputNumber min={1} max={100} value={row.weight ?? 10} onChange={(v) => update(idx, { weight: Number(v) || 10 })} />
            <Checkbox checked={!!row.enableFuzzyMatch} onChange={(v) => update(idx, { enableFuzzyMatch: v })}>模糊匹配</Checkbox>
            <Button size="mini" type="text" status="danger" onClick={() => setRows(rows.filter((_, i) => i !== idx))}>删除</Button>
          </Space>
          <Input placeholder="同音/近音替换词，逗号分隔" value={(row.replacedWords || []).join(',')} onChange={(v) => update(idx, { replacedWords: v.split(',').map((s) => s.trim()).filter(Boolean) })} />
        </div>
      ))}
      <Button size="small" onClick={() => setRows([...rows, { word: '', weight: 10, replacedWords: [] }])}>添加热词</Button>
    </Space>
  )
}

export function AssistantAudioTrackFields({ cfg, setCfg }: { cfg: AudioTrackConfigDraft; setCfg: (v: AudioTrackConfigDraft) => void }) {
  const tracks = cfg.effectAudioTracks ?? []
  const setTracks = (next: typeof tracks) => setCfg({ ...cfg, effectAudioTracks: next })
  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <Space wrap>
        <span className="text-sm text-muted-foreground">主音量</span>
        <InputNumber min={0} max={2} step={0.1} value={cfg.masterVolume ?? 1} onChange={(v) => setCfg({ ...cfg, masterVolume: Number(v) || 1 })} />
        <span className="text-sm text-muted-foreground">主轨音量</span>
        <InputNumber min={0} max={2} step={0.1} value={cfg.mainTrackVolume ?? 0.8} onChange={(v) => setCfg({ ...cfg, mainTrackVolume: Number(v) || 0.8 })} />
      </Space>
      {tracks.map((row, idx) => (
        <Space key={idx} wrap>
          <Input placeholder="文件名" value={row.filename || ''} onChange={(v) => { const n = [...tracks]; n[idx] = { ...n[idx], filename: v }; setTracks(n) }} style={{ width: 180 }} />
          <Button size="mini" type="text" status="danger" onClick={() => setTracks(tracks.filter((_, i) => i !== idx))}>删除</Button>
        </Space>
      ))}
      <Button size="small" onClick={() => setTracks([...tracks, { filename: '', volume: 0.8, isActive: true, mode: 'loop' }])}>添加效果音轨</Button>
    </Space>
  )
}

export function AssistantAudioProcessFields({ cfg, setCfg }: { cfg: AudioProcessConfigDraft; setCfg: (v: AudioProcessConfigDraft) => void }) {
  return (
    <Space direction="vertical" size={14} style={{ width: '100%' }}>
      <Select style={{ width: 260 }} value={cfg.vadType || 'energy'} options={[
        { value: 'energy', label: '能量/RMS' },
        { value: 'webrtc', label: 'WebRTC VAD' },
        { value: 'silero', label: 'Silero VAD (CGO gosilero)' },
        { value: 'tinysilero', label: 'TinySilero (native levad)' },
        { value: 'ten', label: 'TinyTen (native levad)' },
      ]} onChange={(v) => setCfg({ ...cfg, vadType: String(v) })} />
      <Checkbox checked={cfg.noiseSuppressionEnabled !== false} onChange={(v) => setCfg({ ...cfg, noiseSuppressionEnabled: v })}>启用上行降噪</Checkbox>
      {cfg.noiseSuppressionEnabled !== false && (
        <Select style={{ width: 280 }} value={cfg.noiseSuppressionType || 'simple'} options={[
          { value: 'simple', label: 'Simple（占位 AEC/AGC，非真回声消除）' },
          { value: 'ledenoise', label: 'Ledenoise / RNNoise（native，推荐）' },
          { value: 'rnnoise', label: 'RNNoise（librnnoise，需 -tags rnnoise）' },
          { value: 'none', label: '关闭（passthrough）' },
        ]} onChange={(v) => setCfg({ ...cfg, noiseSuppressionType: String(v) })} />
      )}
    </Space>
  )
}

export function AssistantQueryRewriterFields({ cfg, setCfg }: { cfg: QueryRewriterDraft; setCfg: (v: QueryRewriterDraft) => void }) {
  return (
    <Space direction="vertical" size={14} style={{ width: '100%' }}>
      <Checkbox checked={!!cfg.useRewriter} onChange={(v) => setCfg({ ...cfg, useRewriter: v })}>启用查询改写</Checkbox>
      {cfg.useRewriter && <TextArea placeholder="改写提示词" autoSize={{ minRows: 3, maxRows: 8 }} value={cfg.rewritePrompt || ''} onChange={(v) => setCfg({ ...cfg, rewritePrompt: v })} />}
    </Space>
  )
}

export function AssistantCatalogToolBindFields({
  selectedIds,
  onChange,
}: {
  selectedIds: string[]
  onChange: (ids: string[]) => void
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [tools, setTools] = useState<AssistantToolRow[]>([])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listAssistantTools({ enabled: true })
      if (res.code === 200 && Array.isArray(res.data)) {
        setTools(res.data.filter((r) => r.enabled !== false))
      }
    } catch {
      /* optional */
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const selected = useMemo(() => new Set(selectedIds.map(String)), [selectedIds])

  const bindGroups = useMemo(() => {
    type BindItem = {
      key: string
      title: string
      description?: string
      schema?: JsonSchemaObject | null
    }
    type BindGroup = {
      id: string
      title: string
      subtitle: string
      kind: string
      /** Catalog-row id used for whole-server bind. */
      catalogKey: string
      needDiscover?: boolean
      items: BindItem[]
    }
    const groups: BindGroup[] = []
    for (const tool of tools) {
      const kind = tool.kind || 'http'
      if (kind === 'mcp_sse') {
        const discovered = tool.discoveredTools || []
        const items: BindItem[] = discovered.map((dt) => ({
          key: mcpToolBindKey(tool.id, dt.name),
          title: dt.name,
          description: dt.description,
          schema: mcpToolSchema(dt),
        }))
        groups.push({
          id: tool.id,
          title: tool.displayName || tool.name,
          subtitle: tool.mcpSseUrl || tool.name,
          kind: 'mcp_sse',
          catalogKey: tool.id,
          needDiscover: discovered.length === 0,
          items,
        })
        continue
      }
      if (kind !== 'http') continue
      groups.push({
        id: tool.id,
        title: tool.displayName || tool.name,
        subtitle: `${(tool.method || 'POST').toUpperCase()} ${tool.url || tool.name}`,
        kind: 'http',
        catalogKey: tool.id,
        items: [
          {
            key: tool.id,
            title: tool.displayName || tool.name,
            description: tool.description,
            schema: httpToolSchema(tool),
          },
        ],
      })
    }
    return groups
  }, [tools])

  const isItemChecked = useCallback(
    (catalogKey: string, itemKey: string) => selected.has(catalogKey) || selected.has(itemKey),
    [selected],
  )

  const groupSelection = useCallback(
    (catalogKey: string, itemKeys: string[]) => {
      if (itemKeys.length === 0) {
        const on = selected.has(catalogKey)
        return { all: on, some: false, checked: on }
      }
      if (selected.has(catalogKey)) {
        return { all: true, some: false, checked: true }
      }
      const n = itemKeys.filter((k) => selected.has(k)).length
      return { all: n === itemKeys.length && n > 0, some: n > 0 && n < itemKeys.length, checked: n === itemKeys.length && n > 0 }
    },
    [selected],
  )

  const setGroupAll = useCallback(
    (catalogKey: string, itemKeys: string[], checked: boolean) => {
      const next = new Set(selected)
      itemKeys.forEach((k) => next.delete(k))
      if (checked) next.add(catalogKey)
      else next.delete(catalogKey)
      onChange([...next])
    },
    [selected, onChange],
  )

  const toggleItem = useCallback(
    (catalogKey: string, itemKeys: string[], itemKey: string, checked: boolean) => {
      const next = new Set(selected)
      if (next.has(catalogKey)) {
        // Explode whole-server bind into individuals, then apply the toggle.
        next.delete(catalogKey)
        for (const k of itemKeys) {
          if (k !== itemKey) next.add(k)
        }
        if (checked) next.add(itemKey)
        onChange([...next])
        return
      }
      if (checked) {
        next.add(itemKey)
        if (itemKeys.length > 0 && itemKeys.every((k) => next.has(k))) {
          itemKeys.forEach((k) => next.delete(k))
          next.add(catalogKey)
        }
      } else {
        next.delete(itemKey)
      }
      onChange([...next])
    },
    [selected, onChange],
  )

  const bindItemCount = useMemo(
    () => bindGroups.reduce((n, g) => n + Math.max(g.items.length, g.kind === 'mcp_sse' ? 1 : 0), 0),
    [bindGroups],
  )

  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        {t('assistant.catalogToolsHint')}
      </Typography.Text>
      <div className="flex flex-wrap gap-2">
        <Button size="small" onClick={() => void load()} loading={loading}>{t('assistantTools.refresh')}</Button>
        <Button size="small" type="outline" onClick={() => navigate('/mcp')}>{t('assistant.manageCatalogTools')}</Button>
        <Button size="small" type="outline" onClick={() => navigate('/mcp-market')}>{t('nav.mcpMarket')}</Button>
      </div>
      {loading && !bindItemCount ? (
        <Typography.Text type="secondary">{t('common.loading')}</Typography.Text>
      ) : !bindItemCount ? (
        <Empty
          preset="no-data"
          description={t('assistant.catalogToolsEmpty')}
          className="py-4"
          imageClassName="h-14 w-14"
        >
          <Button type="primary" size="small" style={{ marginTop: 8 }} onClick={() => navigate('/mcp-market')}>
            {t('mcpMarket.goMarket')}
          </Button>
        </Empty>
      ) : (
        <div className="space-y-3">
          {bindGroups.map((group) => {
            const itemKeys = group.items.map((i) => i.key)
            const sel = groupSelection(group.catalogKey, itemKeys)
            return (
              <div key={group.id} className="rounded-lg border border-border overflow-hidden">
                <div className="px-3 py-2 bg-muted/30 border-b border-border">
                  <div className="flex items-center gap-2 min-w-0">
                    <Checkbox
                      size="sm"
                      checked={sel.checked}
                      indeterminate={sel.some}
                      onChange={(v) => setGroupAll(group.catalogKey, itemKeys, !!v)}
                      aria-label={t('assistant.selectAllTools')}
                    />
                    <span className="text-sm font-medium truncate">{group.title}</span>
                    <Tag size="small" color={group.kind === 'mcp_sse' ? 'purple' : 'arcoblue'}>
                      {group.kind === 'mcp_sse' ? 'MCP' : 'HTTP'}
                    </Tag>
                    {group.kind === 'mcp_sse' ? (
                      <Tag size="small">{t('assistant.mcpToolCount', { count: group.items.length })}</Tag>
                    ) : null}
                    <span className="text-xs text-muted-foreground ml-auto shrink-0">
                      {t('assistant.selectAllTools')}
                    </span>
                  </div>
                  <div className="text-xs text-muted-foreground font-mono truncate mt-0.5 pl-6">{group.subtitle}</div>
                </div>
                {group.needDiscover ? (
                  <div className="px-3 py-2.5 text-xs text-muted-foreground">
                    {t('assistant.catalogMcpNeedDiscover')}
                  </div>
                ) : (
                  <div className="divide-y divide-border">
                    {group.items.map((item) => (
                      <label
                        key={item.key}
                        className="flex items-start gap-2.5 px-3 py-2 cursor-pointer hover:bg-muted/20"
                      >
                        <Checkbox
                          size="sm"
                          className="mt-0.5"
                          checked={isItemChecked(group.catalogKey, item.key)}
                          onChange={(v) => toggleItem(group.catalogKey, itemKeys, item.key, !!v)}
                        />
                        <div className="min-w-0 flex-1">
                          <div className="text-sm font-medium leading-snug">{item.title}</div>
                          {item.description ? (
                            <div className="text-xs text-muted-foreground mt-0.5 line-clamp-2">{item.description}</div>
                          ) : null}
                          {item.schema ? <ToolSchemaDetail schema={item.schema} compact /> : null}
                          <div className="text-[11px] text-muted-foreground font-mono mt-1">
                            {t('assistantTools.bindKey')}: {item.key}
                          </div>
                        </div>
                      </label>
                    ))}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </Space>
  )
}

export function AssistantMcpServerFields({ rows, setRows }: { rows: McpServerDraft[]; setRows: (v: McpServerDraft[]) => void }) {
  const update = (idx: number, patch: Partial<McpServerDraft>) => setRows(rows.map((r, i) => (i === idx ? { ...r, ...patch } : r)))
  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        可选：本地进程型 MCP（stdio，需服务器可执行命令）。日常「查订单 / HTTP 回调」请用上方网络工具目录，勿在这里配。
      </Typography.Text>
      {rows.map((row, idx) => (
        <div key={idx} className="rounded-lg border border-border p-3 space-y-2">
          <Input placeholder="工具名" value={row.name || ''} onChange={(v) => update(idx, { name: v })} />
          <Input placeholder="命令" value={row.command || ''} onChange={(v) => update(idx, { command: v })} />
          <Button size="mini" type="text" status="danger" onClick={() => setRows(rows.filter((_, i) => i !== idx))}>删除</Button>
        </div>
      ))}
      <Button size="small" onClick={() => setRows([...rows, { name: '', command: '', args: [], envs: {} }])}>添加进程 MCP</Button>
    </Space>
  )
}

export function AssistantVoiceFields({
  voiceMode,
  provider,
  value,
  setValue,
  tenantId,
  includeCloneVoices,
  onCloneVoiceSelect,
}: {
  voiceMode: 'pipeline' | 'realtime'
  provider: string
  value: string
  setValue: (v: string) => void
  tenantId?: string
  includeCloneVoices?: boolean
  onCloneVoiceSelect?: (profile: import('@/api/voiceClone').VoiceCloneProfile | null, voiceId: string) => void
}) {
  return (
    <VoicePickerGrid
      voiceMode={voiceMode}
      provider={provider}
      value={value}
      setValue={setValue}
      tenantId={tenantId}
      includeCloneVoices={includeCloneVoices}
      onCloneVoiceSelect={onCloneVoiceSelect}
    />
  )
}

export function AssistantVoiceprintBindFields({
  assistantId,
  selectedIds,
  onChange,
}: {
  assistantId?: string
  selectedIds: string[]
  onChange: (ids: string[]) => void
}) {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [rows, setRows] = useState<import('@/api/voiceprint').VoiceprintProfile[]>([])
  const [assistantNames, setAssistantNames] = useState<Record<string, string>>({})
  const [enabled, setEnabled] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const { getVoiceprintConfig, listVoiceprints } = await import('@/api/voiceprint')
      const { listAssistants } = await import('@/api/assistants')
      const [cfgRes, listRes, asRes] = await Promise.all([
        getVoiceprintConfig(),
        listVoiceprints(),
        listAssistants(1, 200),
      ])
      if (cfgRes.code === 200) setEnabled(!!cfgRes.data?.enabled)
      if (listRes.code === 200 && Array.isArray(listRes.data)) setRows(listRes.data)
      const nameMap: Record<string, string> = {}
      if (asRes.code === 200 && Array.isArray(asRes.data?.list)) {
        for (const a of asRes.data.list) {
          nameMap[String(a.id)] = a.name || String(a.id)
        }
      }
      setAssistantNames(nameMap)
    } catch {
      /* optional */
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const aid = String(assistantId || '').trim()

  const options = useMemo(() => {
    return rows.map((r) => {
      const id = snowflakeStr(r.id)
      const ownerId = isSnowflakeSet(r.assistantId) ? snowflakeStr(r.assistantId) : ''
      let suffix = ''
      if (ownerId && ownerId === aid) suffix = ' · 本助手'
      else if (ownerId) {
        const ownerName = assistantNames[ownerId] || `助手#${ownerId}`
        suffix = ` · 现属「${ownerName}」`
      }
      return {
        value: id,
        label: `${r.name || '未命名'}${suffix}`,
        extra: r.featureId,
      }
    })
  }, [rows, aid, assistantNames])

  if (!enabled) {
    return (
      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        平台未启用声纹。启用后可在此绑定说话人。
      </Typography.Text>
    )
  }

  if (loading) {
    return <Typography.Text type="secondary">加载中…</Typography.Text>
  }

  if (rows.length === 0) {
    return (
      <Empty
        description={
          <span>
            暂无声纹，请先到{' '}
            <Button type="text" onClick={() => navigate('/voiceprint-manager')}>
              声纹识别
            </Button>{' '}
            注册。
          </span>
        }
      />
    )
  }

  return (
    <div className="space-y-2">
      <Select
        mode="multiple"
        allowClear
        showSearch
        disabled={!aid}
        placeholder={aid ? '选择要绑定到本助手的说话人' : '请先保存智能体后再绑定声纹'}
        value={selectedIds}
        onChange={(v) => onChange((v as string[]) || [])}
        options={options}
        style={{ width: '100%' }}
        filterOption={(input, option) => {
          const label = String(option?.label ?? '')
          const extra = String((option as { extra?: string })?.extra ?? '')
          const q = input.toLowerCase()
          return label.toLowerCase().includes(q) || extra.toLowerCase().includes(q)
        }}
      />
      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        保存后生效。若选项显示「现属某助手」，保存时会改绑到当前助手。已选 {selectedIds.length} 人。
        <Button type="text" size="mini" onClick={() => navigate('/voiceprint-manager')}>
          管理声纹
        </Button>
      </Typography.Text>
    </div>
  )
}

const SkillFormItem = Form.Item

/** Bind tenant dialog skills (agentConfig.dialogSkills) on the assistant form. */
export function AssistantDialogSkillBindFields({
  selectedCodes,
  onChange,
}: {
  selectedCodes: string[]
  onChange: (codes: string[]) => void
}) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [skills, setSkills] = useState<TenantDialogSkillRow[]>([])
  const [manageOpen, setManageOpen] = useState(false)
  const [editing, setEditing] = useState<TenantDialogSkillRow | null>(null)
  const [saving, setSaving] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [form] = Form.useForm()
  const kindWatch = Form.useWatch('kind', form) as string

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listDialogSkills({ enabled: false })
      if (res.code === 200 && Array.isArray(res.data)) {
        setSkills(res.data)
      }
    } catch {
      /* optional */
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const selected = useMemo(() => new Set(selectedCodes.map(String)), [selectedCodes])

  const toggle = (code: string, checked: boolean) => {
    const next = new Set(selected)
    if (checked) next.add(code)
    else next.delete(code)
    onChange([...next])
  }

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({
      enabled: true,
      kind: 'prompt',
      body: '',
      scriptContent: '',
      entryFile: '',
    })
    setManageOpen(true)
  }

  const openEdit = (row: TenantDialogSkillRow) => {
    setEditing(row)
    form.setFieldsValue({
      code: row.code,
      name: row.name,
      description: row.description || '',
      kind: row.kind || 'prompt',
      body: row.body,
      scriptContent: row.scriptContent || '',
      entryFile: row.entryFile || '',
      enabled: row.enabled !== false,
    })
    setManageOpen(true)
  }

  const handleSave = async () => {
    try {
      const values = await form.validate()
      setSaving(true)
      const payload = {
        code: values.code,
        name: values.name,
        description: values.description,
        kind: values.kind || 'prompt',
        body: values.body,
        scriptContent: values.scriptContent,
        entryFile: values.entryFile,
        enabled: values.enabled !== false,
      }
      if (editing) {
        const res = await updateDialogSkill(editing.id, payload)
        if (res.code !== 200) {
          showAlert(res.msg || t('assistant.dialogSkillsSaveFailed'), 'error')
          return
        }
        setEditing(res.data)
      } else {
        const res = await createDialogSkill(payload)
        if (res.code !== 200) {
          showAlert(res.msg || t('assistant.dialogSkillsSaveFailed'), 'error')
          return
        }
        setEditing(res.data)
        form.setFieldsValue({ code: res.data?.code })
      }
      showAlert(t('assistant.dialogSkillsSaved'), 'success')
      await load()
      if ((values.kind || 'prompt') === 'prompt') {
        setManageOpen(false)
      }
    } catch (e) {
      showAlert(extractApiErrorMessage(e, t('assistant.dialogSkillsSaveFailed')), 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleUploadZip = async (file: File) => {
    if (!editing?.id) {
      showAlert(t('assistant.dialogSkillsUploadNeedSave'), 'warning')
      return
    }
    setUploading(true)
    try {
      const res = await uploadDialogSkillAssets(editing.id, file)
      if (res.code !== 200) {
        showAlert(res.msg || t('assistant.dialogSkillsUploadFailed'), 'error')
        return
      }
      setEditing(res.data)
      showAlert(t('assistant.dialogSkillsUploaded'), 'success')
      await load()
    } catch (e) {
      showAlert(extractApiErrorMessage(e, t('assistant.dialogSkillsUploadFailed')), 'error')
    } finally {
      setUploading(false)
    }
  }

  const handleDelete = (row: TenantDialogSkillRow) => {
    Modal.confirm({
      title: t('assistant.dialogSkillsDeleteConfirm'),
      onOk: async () => {
        try {
          const res = await deleteDialogSkill(row.id)
          if (res.code !== 200) {
            showAlert(res.msg || t('assistant.dialogSkillsDeleteFailed'), 'error')
            return
          }
          onChange(selectedCodes.filter((c) => c !== row.code))
          showAlert(t('assistant.dialogSkillsDeleted'), 'success')
          await load()
        } catch (e) {
          showAlert(extractApiErrorMessage(e, t('assistant.dialogSkillsDeleteFailed')), 'error')
        }
      },
    })
  }

  const isScriptKind = kindWatch === 'python' || kindWatch === 'node'

  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
        {t('assistant.dialogSkillsHint')}
      </Typography.Text>
      <div className="flex flex-wrap gap-2">
        <Button size="small" onClick={() => void load()} loading={loading}>{t('common.refresh')}</Button>
        <Button size="small" type="outline" onClick={openCreate}>{t('assistant.dialogSkillsManage')}</Button>
      </div>
      {loading && skills.length === 0 ? (
        <Typography.Text type="secondary">{t('common.loading')}</Typography.Text>
      ) : skills.length === 0 ? (
        <Empty
          preset="no-data"
          description={t('assistant.dialogSkillsEmpty')}
          className="py-4"
          imageClassName="h-14 w-14"
        >
          <Button type="primary" size="small" style={{ marginTop: 8 }} onClick={openCreate}>
            {t('assistant.dialogSkillsCreate')}
          </Button>
        </Empty>
      ) : (
        <div className="space-y-2">
          {skills.map((skill) => {
            const on = skill.enabled !== false
            return (
              <div
                key={skill.id}
                className="flex items-start gap-3 rounded-lg border border-border px-3 py-2"
              >
                <Checkbox
                  checked={selected.has(skill.code)}
                  disabled={!on}
                  onChange={(checked) => toggle(skill.code, !!checked)}
                />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <Typography.Text bold>{skill.name}</Typography.Text>
                    <Tag size="small" color="arcoblue">{skill.code}</Tag>
                    <Tag size="small">{skill.kind || 'prompt'}</Tag>
                    {skill.hasAssets ? <Tag size="small" color="green">zip</Tag> : null}
                    {!on ? <Tag size="small">{t('common.disabled')}</Tag> : null}
                  </div>
                  {skill.description ? (
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      {skill.description}
                    </Typography.Text>
                  ) : null}
                </div>
                <Button size="mini" type="text" onClick={() => openEdit(skill)}>{t('common.edit')}</Button>
                <Button size="mini" type="text" status="danger" onClick={() => handleDelete(skill)}>
                  {t('common.delete')}
                </Button>
              </div>
            )
          })}
        </div>
      )}

      <Drawer
        width={560}
        title={editing ? t('assistant.dialogSkillsEdit') : t('assistant.dialogSkillsCreate')}
        visible={manageOpen}
        onCancel={() => setManageOpen(false)}
        footer={
          <div className="flex justify-end gap-2">
            <Button onClick={() => setManageOpen(false)}>{t('common.cancel')}</Button>
            <Button type="primary" loading={saving} onClick={() => void handleSave()}>
              {t('common.save')}
            </Button>
          </div>
        }
      >
        <Form form={form} layout="vertical">
          <SkillFormItem label={t('assistant.dialogSkillCode')} field="code" extra={t('assistant.dialogSkillCodeHint')}>
            <Input placeholder={t('assistant.dialogSkillCodePlaceholder')} disabled={!!editing} />
          </SkillFormItem>
          <SkillFormItem
            label={t('assistant.dialogSkillName')}
            field="name"
            rules={[{ required: true, message: t('assistant.dialogSkillNameRequired') }]}
          >
            <Input placeholder={t('assistant.dialogSkillName')} />
          </SkillFormItem>
          <SkillFormItem label={t('assistant.dialogSkillDescription')} field="description">
            <Input placeholder={t('assistant.dialogSkillDescription')} />
          </SkillFormItem>
          <SkillFormItem label={t('assistant.dialogSkillKind')} field="kind">
            <Select
              options={[
                { label: t('assistant.dialogSkillKindPrompt'), value: 'prompt' },
                { label: t('assistant.dialogSkillKindPython'), value: 'python' },
                { label: t('assistant.dialogSkillKindNode'), value: 'node' },
              ]}
            />
          </SkillFormItem>
          <SkillFormItem
            label={isScriptKind ? t('assistant.dialogSkillUsageHint') : t('assistant.dialogSkillBody')}
            field="body"
            rules={isScriptKind ? undefined : [{ required: true, message: t('assistant.dialogSkillBodyRequired') }]}
          >
            <TextArea
              autoSize={{ minRows: isScriptKind ? 3 : 8, maxRows: 16 }}
              placeholder={isScriptKind ? t('assistant.dialogSkillUsageHintPlaceholder') : t('assistant.dialogSkillBodyPlaceholder')}
            />
          </SkillFormItem>
          {isScriptKind ? (
            <>
              <SkillFormItem label={t('assistant.dialogSkillEntry')} field="entryFile">
                <Input placeholder={kindWatch === 'node' ? 'index.js' : 'main.py'} />
              </SkillFormItem>
              <SkillFormItem label={t('assistant.dialogSkillScript')} field="scriptContent">
                <TextArea
                  autoSize={{ minRows: 8, maxRows: 20 }}
                  placeholder={t('assistant.dialogSkillScriptPlaceholder')}
                  style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace', fontSize: 12 }}
                />
              </SkillFormItem>
              <div className="mb-4 space-y-2">
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  {t('assistant.dialogSkillsZipHint')}
                </Typography.Text>
                <input
                  type="file"
                  accept=".zip,application/zip"
                  disabled={!editing?.id || uploading}
                  onChange={(e) => {
                    const f = e.target.files?.[0]
                    e.target.value = ''
                    if (f) void handleUploadZip(f)
                  }}
                />
                {editing?.hasAssets ? (
                  <Tag color="green">{t('assistant.dialogSkillsHasAssets')}</Tag>
                ) : null}
              </div>
            </>
          ) : null}
          <SkillFormItem label={t('common.enable')} field="enabled" triggerPropName="checked">
            <Switch />
          </SkillFormItem>
        </Form>
      </Drawer>
    </Space>
  )
}
