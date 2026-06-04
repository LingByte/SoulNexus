import { useCallback, useEffect, useRef, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import {
  Beaker,
  Bot,
  ChevronDown,
  Loader2,
  RotateCcw,
  Send,
  Settings2,
} from 'lucide-react'
import { cn } from '@/utils/cn'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Textarea from '@/components/UI/Textarea'
import Switch from '@/components/UI/Switch'
import MarkdownPreview from '@/components/UI/MarkdownPreview'
import CollapsibleSectionHeader from '@/components/UI/CollapsibleSectionHeader'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/UI/Select'
import { showAlert } from '@/utils/notification'
import { useAuthStore } from '@/stores/authStore'
import {
  DEFAULT_PLAYGROUND_SETTINGS,
  PLAYGROUND_STORAGE_KEY,
  type PlaygroundMessage,
  type PlaygroundMetrics,
  type PlaygroundSettings,
} from '@/lib/playground/types'
import { getDefaultPlaygroundBaseUrl, runPlaygroundChat } from '@/lib/playground/client'
import { formatApiError } from '@/lib/playground/formatError'

function newId() {
  return `m_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`
}

function loadSettings(): PlaygroundSettings {
  try {
    const raw = localStorage.getItem(PLAYGROUND_STORAGE_KEY)
    if (!raw) {
      return { ...DEFAULT_PLAYGROUND_SETTINGS, baseUrl: getDefaultPlaygroundBaseUrl() }
    }
    const parsed = JSON.parse(raw) as Partial<PlaygroundSettings>
    return {
      ...DEFAULT_PLAYGROUND_SETTINGS,
      baseUrl: getDefaultPlaygroundBaseUrl(),
      ...parsed,
    }
  } catch {
    return { ...DEFAULT_PLAYGROUND_SETTINGS, baseUrl: getDefaultPlaygroundBaseUrl() }
  }
}

function SettingsSection({
  title,
  children,
  defaultOpen = false,
}: {
  title: string
  children: React.ReactNode
  defaultOpen?: boolean
}) {
  const [open, setOpen] = useState(defaultOpen)
  return (
    <div className="border-b border-border/60 last:border-0">
      <CollapsibleSectionHeader
        title={title}
        expanded={open}
        onToggle={() => setOpen((v) => !v)}
        compact
        titleSize="sm"
        className="px-3 py-2 hover:bg-accent/40 rounded-none"
      />
      <AnimatePresence initial={false}>
        {open && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.22, ease: [0.16, 1, 0.3, 1] }}
            className="overflow-hidden"
          >
            <div className="space-y-2.5 px-3 pb-3">{children}</div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

function MetricsDetailPanel({ metrics }: { metrics: PlaygroundMetrics }) {
  const total =
    metrics.total_tokens ??
    (metrics.input_tokens != null || metrics.output_tokens != null
      ? (metrics.input_tokens ?? 0) + (metrics.output_tokens ?? 0)
      : undefined)

  return (
    <div className="space-y-2 text-xs">
      <div className="grid grid-cols-2 gap-1.5">
        <MetricCell label="总 Tokens" value={total != null ? String(total) : '—'} />
        <MetricCell
          label="总耗时"
          value={metrics.latency_ms != null ? `${metrics.latency_ms} ms` : '—'}
        />
        <MetricCell
          label="首包"
          value={metrics.ttft_ms != null ? `${metrics.ttft_ms} ms` : '—'}
        />
        <MetricCell
          label="正文首字"
          value={metrics.ttfc_ms != null ? `${metrics.ttfc_ms} ms` : '—'}
        />
        <MetricCell label="Prompt" value={metrics.input_tokens != null ? String(metrics.input_tokens) : '—'} />
        <MetricCell
          label="Completion"
          value={metrics.output_tokens != null ? String(metrics.output_tokens) : '—'}
        />
      </div>
      <div className="space-y-1 border-t border-border/60 pt-2 text-[11px]">
        <MetricRow label="模型" value={metrics.model || '—'} mono />
        {metrics.ttft_ms != null &&
        metrics.ttfc_ms != null &&
        metrics.ttfc_ms - metrics.ttft_ms > 500 ? (
          <p className="text-[10px] text-muted-foreground">
            首包与正文间隔较大时，多为模型思考（thinking）阶段，属上游行为而非网关额外阻塞。
          </p>
        ) : null}
      </div>
    </div>
  )
}

function MetricCell({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-border/80 bg-muted/30 p-1.5 dark:bg-neutral-900/40">
      <div className="text-[10px] text-muted-foreground">{label}</div>
      <div className="font-semibold tabular-nums text-foreground">{value}</div>
    </div>
  )
}

function PlaygroundErrorBlock({ raw }: { raw: string }) {
  const { title, detail } = formatApiError(raw)
  return (
    <div className="min-w-0 max-w-full space-y-1.5">
      {title ? (
        <p className="text-[11px] font-medium text-red-600 dark:text-red-400 break-words">{title}</p>
      ) : null}
      <div className="max-h-36 overflow-auto rounded-md border border-red-200/80 bg-red-50/90 p-2 dark:border-red-900/50 dark:bg-red-950/40">
        <p className="text-xs leading-relaxed text-red-700 dark:text-red-300 break-words whitespace-pre-wrap">
          {detail}
        </p>
      </div>
    </div>
  )
}

function MetricRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex justify-between gap-2">
      <span className="text-muted-foreground">{label}</span>
      <span className={cn('text-right break-all', mono && 'font-mono')}>{value}</span>
    </div>
  )
}

const Playground = () => {
  const { user } = useAuthStore()
  const [settings, setSettings] = useState<PlaygroundSettings>(loadSettings)
  const [messages, setMessages] = useState<PlaygroundMessage[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [lastRaw, setLastRaw] = useState('')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [expandedMetrics, setExpandedMetrics] = useState<Set<string>>(new Set())
  const bottomRef = useRef<HTMLDivElement>(null)

  const userAvatar =
    user?.avatar ||
    `https://ui-avatars.com/api/?name=${encodeURIComponent(user?.displayName || user?.email || 'User')}&background=6366f1&color=fff&size=64`

  useEffect(() => {
    localStorage.setItem(PLAYGROUND_STORAGE_KEY, JSON.stringify(settings))
  }, [settings])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, loading])

  const patchSettings = (patch: Partial<PlaygroundSettings>) => {
    setSettings((s) => ({ ...s, ...patch }))
  }

  const toggleMetrics = (id: string) => {
    setExpandedMetrics((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const clearChat = () => {
    setMessages([])
    setLastRaw('')
    setExpandedMetrics(new Set())
  }

  const sendMessage = useCallback(async () => {
    const text = input.trim()
    if (!text || loading) return

    const userMsg: PlaygroundMessage = {
      id: newId(),
      role: 'user',
      content: text,
      createdAt: Date.now(),
    }
    const assistantId = newId()
    const history = [...messages, userMsg]
    const thinkingOn =
      settings.protocol === 'openai'
        ? settings.enableThinking
        : settings.anthropicThinkingEnabled

    setMessages([
      ...history,
      {
        id: assistantId,
        role: 'assistant',
        content: '',
        createdAt: Date.now(),
        streaming: settings.stream,
        thinking: thinkingOn && settings.stream,
      },
    ])
    setInput('')
    setLoading(true)

    try {
      const result = await runPlaygroundChat(settings, history, {
        onDelta: (delta) => {
          setMessages((prev) =>
            prev.map((m) =>
              m.id === assistantId
                ? { ...m, content: m.content + delta, streaming: true, thinking: false }
                : m,
            ),
          )
        },
        onThinkingDelta: () => {
          setMessages((prev) =>
            prev.map((m) => (m.id === assistantId ? { ...m, thinking: true, streaming: true } : m)),
          )
        },
        onFirstToken: () => {
          setMessages((prev) =>
            prev.map((m) => (m.id === assistantId ? { ...m, thinking: false } : m)),
          )
        },
        onRaw: (chunk) => {
          if (settings.showRaw) setLastRaw((r) => r + chunk + '\n')
        },
      })
      setMessages((prev) =>
        prev.map((m) =>
          m.id === assistantId
            ? {
                ...m,
                content: result.content || m.content,
                streaming: false,
                thinking: false,
                usage: result.usage,
                rawResponse: result.raw,
              }
            : m,
        ),
      )
      if (settings.showRaw) setLastRaw(result.raw)
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e)
      setMessages((prev) =>
        prev.map((m) =>
          m.id === assistantId
            ? { ...m, content: '', streaming: false, thinking: false, error: msg }
            : m,
        ),
      )
      showAlert(msg, 'error')
    } finally {
      setLoading(false)
    }
  }, [input, loading, messages, settings])

  const usePlatformGateway = () => {
    patchSettings({ baseUrl: getDefaultPlaygroundBaseUrl() })
    showAlert('已填入平台 /v1 网关地址', 'success')
  }

  return (
    <div className="flex h-[calc(100vh-3.5rem)] lg:h-[calc(100vh-0px)] min-h-0 bg-background">
      <AnimatePresence initial={false}>
        {settingsOpen && (
          <motion.aside
            key="playground-settings"
            initial={{ width: 0, opacity: 0 }}
            animate={{ width: 'min(100%, 22rem)', opacity: 1 }}
            exit={{ width: 0, opacity: 0 }}
            transition={{ duration: 0.28, ease: [0.16, 1, 0.3, 1] }}
            className="flex shrink-0 flex-col overflow-hidden border-r border-border bg-muted/20"
          >
            <div className="flex shrink-0 items-center gap-2 border-b border-border px-3 py-2.5">
              <Beaker className="h-4 w-4 text-violet-500" />
              <span className="text-sm font-semibold">演练场</span>
            </div>
            <div className="flex-1 overflow-y-auto custom-scrollbar">
              <SettingsSection title="连接" defaultOpen>
                <Select
                  value={settings.protocol}
                  onValueChange={(v) =>
                    patchSettings({ protocol: v as PlaygroundSettings['protocol'] })
                  }
                >
                  <SelectTrigger className="h-9 w-full text-xs">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="openai">OpenAI 兼容 (/v1/chat/completions)</SelectItem>
                    <SelectItem value="anthropic">Anthropic (/v1/messages)</SelectItem>
                  </SelectContent>
                </Select>
                <Input
                  size="sm"
                  label="Base URL"
                  value={settings.baseUrl}
                  onValueChange={(v) => patchSettings({ baseUrl: v })}
                  placeholder="https://api.example.com/v1"
                />
                <Button variant="ghost" size="sm" className="w-full text-xs" onClick={usePlatformGateway}>
                  使用本平台 /v1 网关
                </Button>
                <Input
                  size="sm"
                  type="password"
                  label="API Key"
                  value={settings.apiKey}
                  onValueChange={(v) => patchSettings({ apiKey: v })}
                  placeholder="sk-..."
                  autoComplete="off"
                />
                {settings.protocol === 'anthropic' && (
                  <>
                    <Input
                      size="sm"
                      label="anthropic-version"
                      value={settings.anthropicVersion}
                      onValueChange={(v) => patchSettings({ anthropicVersion: v })}
                    />
                    <Input
                      size="sm"
                      label="anthropic-beta（可选）"
                      value={settings.anthropicBeta}
                      onValueChange={(v) => patchSettings({ anthropicBeta: v })}
                      placeholder="extended-thinking-..."
                    />
                  </>
                )}
              </SettingsSection>

              <SettingsSection title="模型">
                <Input
                  size="sm"
                  label="model"
                  value={settings.model}
                  onValueChange={(v) => patchSettings({ model: v })}
                />
                <Textarea
                  size="sm"
                  label="system（系统提示）"
                  rows={3}
                  value={settings.systemPrompt}
                  onValueChange={(v) => patchSettings({ systemPrompt: v })}
                />
              </SettingsSection>

              <SettingsSection title="采样与长度">
                <div className="grid grid-cols-2 gap-2">
                  <Input
                    size="sm"
                    type="number"
                    label="max_tokens"
                    value={String(settings.maxTokens)}
                    onValueChange={(v) => patchSettings({ maxTokens: Number(v) || 0 })}
                  />
                  <Input
                    size="sm"
                    type="number"
                    label="temperature"
                    value={String(settings.temperature)}
                    onValueChange={(v) => patchSettings({ temperature: Number(v) })}
                  />
                  <Input
                    size="sm"
                    type="number"
                    label="top_p"
                    value={String(settings.topP)}
                    onValueChange={(v) => patchSettings({ topP: Number(v) })}
                  />
                  <Input
                    size="sm"
                    type="number"
                    label="n"
                    value={String(settings.n)}
                    onValueChange={(v) => patchSettings({ n: Math.max(1, Number(v) || 1) })}
                  />
                  {settings.protocol === 'openai' && (
                    <>
                      <Input
                        size="sm"
                        type="number"
                        label="presence_penalty"
                        value={String(settings.presencePenalty)}
                        onValueChange={(v) => patchSettings({ presencePenalty: Number(v) })}
                      />
                      <Input
                        size="sm"
                        type="number"
                        label="frequency_penalty"
                        value={String(settings.frequencyPenalty)}
                        onValueChange={(v) => patchSettings({ frequencyPenalty: Number(v) })}
                      />
                    </>
                  )}
                </div>
                <Input
                  size="sm"
                  label="stop（逗号分隔）"
                  value={settings.stopSequences}
                  onValueChange={(v) => patchSettings({ stopSequences: v })}
                />
                {settings.protocol === 'openai' && (
                  <>
                    <Input
                      size="sm"
                      label="seed"
                      value={settings.seed}
                      onValueChange={(v) => patchSettings({ seed: v })}
                    />
                    <div>
                      <p className="mb-1.5 text-sm font-medium text-foreground">response_format</p>
                      <Select
                        value={settings.responseFormat || 'default'}
                        onValueChange={(v) =>
                          patchSettings({
                            responseFormat:
                              v === 'default' ? '' : (v as PlaygroundSettings['responseFormat']),
                          })
                        }
                      >
                        <SelectTrigger className="h-9 w-full text-xs">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="default">默认 text</SelectItem>
                          <SelectItem value="json_object">json_object</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <Input
                      size="sm"
                      label="user（可选）"
                      value={settings.user}
                      onValueChange={(v) => patchSettings({ user: v })}
                    />
                  </>
                )}
              </SettingsSection>

              <SettingsSection title="Thinking / 推理">
                {settings.protocol === 'openai' ? (
                  <>
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-xs text-foreground">enable_thinking（Qwen 等）</span>
                      <Switch
                        size="sm"
                        checked={settings.enableThinking}
                        onCheckedChange={(v) => patchSettings({ enableThinking: v })}
                      />
                    </div>
                    <Input
                      size="sm"
                      type="number"
                      label="thinking_budget"
                      value={String(settings.thinkingBudget)}
                      onValueChange={(v) => patchSettings({ thinkingBudget: Number(v) || 0 })}
                    />
                  </>
                ) : (
                  <>
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-xs text-foreground">thinking.enabled</span>
                      <Switch
                        size="sm"
                        checked={settings.anthropicThinkingEnabled}
                        onCheckedChange={(v) => patchSettings({ anthropicThinkingEnabled: v })}
                      />
                    </div>
                    <Input
                      size="sm"
                      type="number"
                      label="budget_tokens"
                      value={String(settings.anthropicThinkingBudget)}
                      onValueChange={(v) =>
                        patchSettings({ anthropicThinkingBudget: Number(v) || 0 })
                      }
                    />
                  </>
                )}
              </SettingsSection>

              <SettingsSection title="高级">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs text-foreground">stream</span>
                  <Switch
                    size="sm"
                    checked={settings.stream}
                    onCheckedChange={(v) => patchSettings({ stream: v })}
                  />
                </div>
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs text-foreground">显示原始响应</span>
                  <Switch
                    size="sm"
                    checked={settings.showRaw}
                    onCheckedChange={(v) => patchSettings({ showRaw: v })}
                  />
                </div>
                <Textarea
                  size="sm"
                  label="extra JSON（合并进请求体）"
                  rows={4}
                  value={settings.extraJson}
                  onValueChange={(v) => patchSettings({ extraJson: v })}
                  placeholder='{"top_k": 20}'
                  className="font-mono text-[11px]"
                />
              </SettingsSection>
            </div>
          </motion.aside>
        )}
      </AnimatePresence>

      <div className="flex min-w-0 flex-1 flex-col">
        <motion.header
          layout
          className="flex shrink-0 items-center gap-2 border-b border-border px-4 py-2"
        >
          <Button
            variant="ghost"
            size="sm"
            className="!p-1.5"
            onClick={() => setSettingsOpen((v) => !v)}
            aria-label="切换参数面板"
            aria-pressed={settingsOpen}
          >
            <Settings2 className={cn('h-4 w-4 transition-transform', settingsOpen && 'rotate-90')} />
          </Button>
          <div className="min-w-0 flex-1">
            <h1 className="truncate text-sm font-semibold">演练场</h1>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={clearChat}
            leftIcon={<RotateCcw className="h-3.5 w-3.5" />}
          >
            清空
          </Button>
        </motion.header>

        <div className="flex-1 overflow-y-auto custom-scrollbar px-4 py-4">
          {messages.length === 0 && (
            <motion.div
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              className="flex h-full min-h-[12rem] flex-col items-center justify-center text-center text-muted-foreground"
            >
              <Beaker className="mb-3 h-10 w-10 opacity-30" />
              <p className="mt-1 max-w-md text-xs">
                点击左上角齿轮打开参数面板进行配置
              </p>
            </motion.div>
          )}

          <div className="mx-auto max-w-3xl space-y-4">
            <AnimatePresence initial={false}>
              {messages.map((m) => (
                <motion.div
                  key={m.id}
                  initial={{ opacity: 0, y: 12 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -8 }}
                  transition={{ duration: 0.28, ease: [0.16, 1, 0.3, 1] }}
                  className={cn(
                    'flex min-w-0 gap-3',
                    m.role === 'user' ? 'justify-end' : 'justify-start',
                  )}
                >
                  {m.role === 'assistant' && (
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-violet-500 to-indigo-600 text-white shadow-sm">
                      {m.thinking && !m.content ? (
                        <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                      ) : (
                        <Bot className="h-4 w-4" aria-hidden />
                      )}
                    </div>
                  )}
                  <div
                    className={cn(
                      'min-w-0 max-w-[min(85%,42rem)] rounded-2xl px-4 py-2.5 text-sm shadow-sm',
                      m.role === 'user'
                        ? 'bg-primary text-primary-foreground'
                        : 'border border-border bg-card text-foreground',
                      m.error && 'border-red-300 dark:border-red-800',
                    )}
                  >
                    {m.error ? (
                      <PlaygroundErrorBlock raw={m.error} />
                    ) : m.thinking && !m.content ? (
                      <span className="inline-flex items-center gap-2 text-muted-foreground">
                        <Loader2 className="h-4 w-4 animate-spin text-violet-500" />
                        <span className="text-xs">思考中…</span>
                      </span>
                    ) : m.content ? (
                      <MarkdownPreview content={m.content} className="prose-sm max-w-none" />
                    ) : m.streaming ? (
                      <span className="inline-flex items-center gap-2 text-muted-foreground">
                        <Loader2 className="h-4 w-4 animate-spin" />
                        <span className="text-xs">生成中…</span>
                      </span>
                    ) : null}

                    {m.role === 'assistant' && m.usage && !m.streaming && !m.error && (
                      <div className="mt-2 border-t border-border/60 pt-2">
                        <div className="flex items-center justify-between gap-2">
                          <p className="text-[10px] tabular-nums text-muted-foreground">
                            tokens in {m.usage.input_tokens ?? '—'} / out {m.usage.output_tokens ?? '—'}
                            {m.usage.total_tokens != null ? ` · Σ ${m.usage.total_tokens}` : ''}
                          </p>
                          <button
                            type="button"
                            onClick={() => toggleMetrics(m.id)}
                            className="inline-flex items-center gap-0.5 rounded-md px-1.5 py-0.5 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-accent"
                          >
                            {expandedMetrics.has(m.id) ? '收起' : '详情'}
                            <ChevronDown
                              className={cn(
                                'h-3.5 w-3.5 transition-transform duration-200',
                                expandedMetrics.has(m.id) && 'rotate-180',
                              )}
                            />
                          </button>
                        </div>
                        <AnimatePresence initial={false}>
                          {expandedMetrics.has(m.id) && (
                            <motion.div
                              initial={{ height: 0, opacity: 0 }}
                              animate={{ height: 'auto', opacity: 1 }}
                              exit={{ height: 0, opacity: 0 }}
                              transition={{ duration: 0.28, ease: [0.16, 1, 0.3, 1] }}
                              className="overflow-hidden"
                            >
                              <div className="mt-2 rounded-lg bg-muted/40 p-2 dark:bg-neutral-900/50">
                                <MetricsDetailPanel metrics={m.usage} />
                              </div>
                            </motion.div>
                          )}
                        </AnimatePresence>
                      </div>
                    )}
                  </div>
                  {m.role === 'user' && (
                    <img
                      src={userAvatar}
                      alt=""
                      className="h-9 w-9 shrink-0 rounded-full object-cover ring-1 ring-border"
                    />
                  )}
                </motion.div>
              ))}
            </AnimatePresence>
            <div ref={bottomRef} />
          </div>
        </div>

        <AnimatePresence>
          {settings.showRaw && lastRaw && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: 'auto', opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              className="max-h-32 shrink-0 overflow-auto border-t border-border bg-muted/30 px-4 py-2 font-mono text-[10px] text-muted-foreground"
            >
              {lastRaw}
            </motion.div>
          )}
        </AnimatePresence>

        <motion.div
          layout
          className="shrink-0 border-t border-border bg-background/80 p-3 backdrop-blur"
        >
          <div className="mx-auto flex max-w-3xl gap-2">
            <Textarea
              size="sm"
              rows={1}
              value={input}
              onValueChange={setInput}
              onChange={(v: string) => {
                // Handle onKeyDown separately
              }}
              onKeyDown={(e: any) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  void sendMessage()
                }
              }}
              placeholder="输入消息，Enter 发送，Shift+Enter 换行"
              wrapperClassName="flex-1"
              className="max-h-32 min-h-[2.75rem] resize-none rounded-xl"
            />
            <Button
              variant="primary"
              size="md"
              disabled={loading || !input.trim()}
              onClick={() => void sendMessage()}
              leftIcon={
                loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />
              }
            >
              发送
            </Button>
          </div>
        </motion.div>
      </div>
    </div>
  )
}

export default Playground
