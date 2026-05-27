import type {
  PlaygroundMessage,
  PlaygroundMetrics,
  PlaygroundProtocol,
  PlaygroundSettings,
} from './types'

function normalizeBase(baseUrl: string, protocol: PlaygroundProtocol): string {
  let b = baseUrl.trim().replace(/\/+$/, '')
  if (!b) return b
  if (protocol === 'openai') {
    if (!b.endsWith('/v1') && !b.endsWith('/v1/chat/completions')) {
      if (!b.includes('/v1')) {
        b = `${b}/v1`
      }
    }
    if (b.endsWith('/chat/completions')) {
      b = b.replace(/\/chat\/completions$/, '')
    }
  } else {
    if (!b.endsWith('/v1') && !b.endsWith('/v1/messages')) {
      if (!b.includes('/v1')) {
        b = `${b}/v1`
      }
    }
    if (b.endsWith('/messages')) {
      b = b.replace(/\/messages$/, '')
    }
  }
  return b
}

function parseExtra(extraJson: string): Record<string, unknown> {
  const t = extraJson.trim()
  if (!t) return {}
  return JSON.parse(t) as Record<string, unknown>
}

function openAIHeaders(apiKey: string): HeadersInit {
  return {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${apiKey}`,
  }
}

function anthropicHeaders(settings: PlaygroundSettings): HeadersInit {
  const h: Record<string, string> = {
    'Content-Type': 'application/json',
    'x-api-key': settings.apiKey,
    'anthropic-version': settings.anthropicVersion || '2023-06-01',
  }
  if (settings.anthropicBeta.trim()) {
    h['anthropic-beta'] = settings.anthropicBeta.trim()
  }
  return h
}

function buildOpenAIBody(settings: PlaygroundSettings, messages: PlaygroundMessage[]) {
  const msgs: Array<{ role: string; content: string }> = []
  if (settings.systemPrompt.trim()) {
    msgs.push({ role: 'system', content: settings.systemPrompt.trim() })
  }
  for (const m of messages) {
    if (m.role === 'system') {
      if (m.content.trim()) msgs.push({ role: 'system', content: m.content.trim() })
      continue
    }
    if (m.role === 'assistant' && !m.content.trim()) continue
    if (m.role === 'user' || m.role === 'assistant') {
      msgs.push({ role: m.role, content: m.content })
    }
  }

  const body: Record<string, unknown> = {
    model: settings.model,
    messages: msgs,
    stream: settings.stream,
  }
  if (settings.maxTokens > 0) body.max_tokens = settings.maxTokens
  if (settings.temperature !== 0) body.temperature = settings.temperature
  if (settings.topP > 0 && settings.topP !== 1) body.top_p = settings.topP
  if (settings.presencePenalty !== 0) body.presence_penalty = settings.presencePenalty
  if (settings.frequencyPenalty !== 0) body.frequency_penalty = settings.frequencyPenalty
  if (settings.n > 1) body.n = settings.n
  if (settings.seed.trim()) {
    const seedNum = Number(settings.seed.trim())
    if (!Number.isNaN(seedNum)) body.seed = seedNum
  }
  if (settings.user.trim()) body.user = settings.user.trim()
  if (settings.responseFormat === 'json_object') {
    body.response_format = { type: 'json_object' }
  }
  const stops = settings.stopSequences
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
  if (stops.length > 0) body.stop = stops.length === 1 ? stops[0] : stops
  if (settings.enableThinking) {
    body.enable_thinking = true
    if (settings.thinkingBudget > 0) body.thinking_budget = settings.thinkingBudget
  }
  if (settings.stream) {
    body.stream_options = { include_usage: true }
  }
  Object.assign(body, parseExtra(settings.extraJson))
  return body
}

function buildAnthropicBody(settings: PlaygroundSettings, messages: PlaygroundMessage[]) {
  const systemParts: string[] = []
  if (settings.systemPrompt.trim()) systemParts.push(settings.systemPrompt.trim())
  const convo: Array<{ role: string; content: string }> = []
  for (const m of messages) {
    if (m.role === 'system') {
      if (m.content.trim()) systemParts.push(m.content.trim())
      continue
    }
    if (m.role === 'assistant' || m.role === 'user') {
      convo.push({ role: m.role, content: m.content })
    }
  }
  const body: Record<string, unknown> = {
    model: settings.model,
    messages: convo,
    max_tokens: settings.maxTokens > 0 ? settings.maxTokens : 1024,
    stream: settings.stream,
  }
  if (systemParts.length > 0) body.system = systemParts.join('\n\n')
  if (settings.temperature !== 0) body.temperature = settings.temperature
  if (settings.topP > 0 && settings.topP !== 1) body.top_p = settings.topP
  const stops = settings.stopSequences
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
  if (stops.length > 0) body.stop_sequences = stops
  if (settings.anthropicThinkingEnabled) {
    body.thinking = {
      type: 'enabled',
      budget_tokens: settings.anthropicThinkingBudget > 0 ? settings.anthropicThinkingBudget : 1024,
    }
  }
  Object.assign(body, parseExtra(settings.extraJson))
  return body
}

export type StreamHandlers = {
  onDelta: (text: string) => void
  onThinkingDelta?: (text: string) => void
  onFirstToken?: () => void
  onUsage?: (usage: { input_tokens?: number; output_tokens?: number; total_tokens?: number }) => void
  onRaw?: (chunk: string) => void
}

function inferProvider(settings: PlaygroundSettings, headers: Headers): string {
  const fromHeader = headers.get('X-Provider')?.trim()
  if (fromHeader) return fromHeader
  if (settings.protocol === 'anthropic') return 'llm.anthropic'
  const base = settings.baseUrl.toLowerCase()
  if (base.includes('localhost') || base.includes('127.0.0.1') || base.includes('7072')) {
    return 'llm.openai'
  }
  return 'openai-compat'
}

function buildMetrics(
  settings: PlaygroundSettings,
  headers: Headers,
  usage: PlaygroundMetrics | undefined,
  latencyMs: number,
  ttftMs?: number,
): PlaygroundMetrics {
  const input = usage?.input_tokens ?? 0
  const output = usage?.output_tokens ?? 0
  const total = usage?.total_tokens ?? (input + output > 0 ? input + output : undefined)
  return {
    model: settings.model,
    provider: inferProvider(settings, headers),
    input_tokens: usage?.input_tokens,
    output_tokens: usage?.output_tokens,
    total_tokens: total,
    latency_ms: latencyMs,
    ttft_ms: ttftMs ?? latencyMs,
    duration_ms: latencyMs,
  }
}

async function consumeOpenAISSE(
  res: Response,
  handlers: StreamHandlers,
): Promise<{ content: string; usage?: StreamHandlers extends { onUsage?: infer U } ? U : never; raw: string }> {
  const reader = res.body?.getReader()
  if (!reader) throw new Error('无响应流')
  const decoder = new TextDecoder()
  let buffer = ''
  let content = ''
  let raw = ''
  let usage: StreamHandlers['onUsage'] extends (u: infer U) => void ? U : undefined

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''
    for (const line of lines) {
      const trimmed = line.trim()
      if (!trimmed.startsWith('data:')) continue
      const data = trimmed.slice(5).trim()
      if (!data || data === '[DONE]') continue
      raw += data + '\n'
      handlers.onRaw?.(data)
      try {
        const parsed = JSON.parse(data) as {
          choices?: Array<{
            delta?: { content?: string; reasoning_content?: string }
            finish_reason?: string
          }>
          usage?: { prompt_tokens?: number; completion_tokens?: number; total_tokens?: number }
        }
        const choiceDelta = parsed.choices?.[0]?.delta
        const reasoning = choiceDelta?.reasoning_content
        if (reasoning) {
          handlers.onThinkingDelta?.(reasoning)
        }
        const delta = choiceDelta?.content
        if (delta) {
          handlers.onFirstToken?.()
          content += delta
          handlers.onDelta(delta)
        }
        if (parsed.usage) {
          usage = {
            input_tokens: parsed.usage.prompt_tokens,
            output_tokens: parsed.usage.completion_tokens,
            total_tokens: parsed.usage.total_tokens,
          }
          handlers.onUsage?.(usage)
        }
      } catch {
        /* ignore partial json */
      }
    }
  }
  return { content, usage, raw }
}

async function consumeAnthropicSSE(
  res: Response,
  handlers: StreamHandlers,
): Promise<{ content: string; usage?: { input_tokens?: number; output_tokens?: number }; raw: string }> {
  const reader = res.body?.getReader()
  if (!reader) throw new Error('无响应流')
  const decoder = new TextDecoder()
  let buffer = ''
  let content = ''
  let raw = ''
  let usage: { input_tokens?: number; output_tokens?: number } | undefined

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const parts = buffer.split('\n\n')
    buffer = parts.pop() || ''
    for (const part of parts) {
      const lines = part.split('\n')
      let event = ''
      let data = ''
      for (const line of lines) {
        if (line.startsWith('event:')) event = line.slice(6).trim()
        if (line.startsWith('data:')) data += line.slice(5).trim()
      }
      if (!data) continue
      raw += `${event} ${data}\n`
      handlers.onRaw?.(data)
      try {
        const parsed = JSON.parse(data) as {
          type?: string
          delta?: { type?: string; text?: string }
          message?: { usage?: { input_tokens?: number; output_tokens?: number } }
          usage?: { input_tokens?: number; output_tokens?: number }
        }
        if (parsed.type === 'content_block_delta' && parsed.delta?.text) {
          handlers.onFirstToken?.()
          content += parsed.delta.text
          handlers.onDelta(parsed.delta.text)
        }
        if (parsed.message?.usage) {
          usage = parsed.message.usage
          handlers.onUsage?.({
            input_tokens: usage.input_tokens,
            output_tokens: usage.output_tokens,
            total_tokens: (usage.input_tokens || 0) + (usage.output_tokens || 0),
          })
        }
        if (parsed.usage) {
          usage = parsed.usage
          handlers.onUsage?.({
            input_tokens: usage.input_tokens,
            output_tokens: usage.output_tokens,
            total_tokens: (usage.input_tokens || 0) + (usage.output_tokens || 0),
          })
        }
      } catch {
        /* ignore */
      }
    }
  }
  return { content, usage, raw }
}

export async function runPlaygroundChat(
  settings: PlaygroundSettings,
  messages: PlaygroundMessage[],
  handlers?: StreamHandlers,
): Promise<{ content: string; usage?: PlaygroundMetrics; raw: string }> {
  if (!settings.apiKey.trim()) {
    throw new Error('请填写 API Key')
  }
  if (!settings.model.trim()) {
    throw new Error('请填写模型名称')
  }
  const base = normalizeBase(settings.baseUrl, settings.protocol)
  if (!base) throw new Error('请填写 Base URL')

  const startedAt = performance.now()
  let ttftMs: number | undefined
  let sawFirstToken = false
  const streamHandlers: StreamHandlers = {
    onDelta: (text) => {
      if (!sawFirstToken) {
        sawFirstToken = true
        ttftMs = Math.round(performance.now() - startedAt)
        handlers?.onFirstToken?.()
      }
      handlers?.onDelta(text)
    },
    onThinkingDelta: handlers?.onThinkingDelta,
    onFirstToken: () => {
      if (!sawFirstToken) {
        sawFirstToken = true
        ttftMs = Math.round(performance.now() - startedAt)
      }
      handlers?.onFirstToken?.()
    },
    onUsage: handlers?.onUsage,
    onRaw: handlers?.onRaw,
  }

  if (settings.protocol === 'openai') {
    const url = `${base}/chat/completions`
    const body = buildOpenAIBody(settings, messages)
    const res = await fetch(url, {
      method: 'POST',
      headers: openAIHeaders(settings.apiKey),
      body: JSON.stringify(body),
    })
    if (!res.ok) {
      const errText = await res.text()
      throw new Error(errText || `HTTP ${res.status}`)
    }
    if (settings.stream) {
      const out = await consumeOpenAISSE(res, streamHandlers)
      const latencyMs = Math.round(performance.now() - startedAt)
      return {
        content: out.content,
        usage: buildMetrics(settings, res.headers, out.usage, latencyMs, ttftMs),
        raw: out.raw,
      }
    }
    const json = (await res.json()) as {
      choices?: Array<{ message?: { content?: string } }>
      usage?: { prompt_tokens?: number; completion_tokens?: number; total_tokens?: number }
    }
    const content = json.choices?.[0]?.message?.content || ''
    const raw = JSON.stringify(json, null, 2)
    const latencyMs = Math.round(performance.now() - startedAt)
    const tokenUsage = json.usage
      ? {
          input_tokens: json.usage.prompt_tokens,
          output_tokens: json.usage.completion_tokens,
          total_tokens: json.usage.total_tokens,
        }
      : undefined
    return {
      content,
      usage: buildMetrics(settings, res.headers, tokenUsage, latencyMs, latencyMs),
      raw,
    }
  }

  const url = `${base}/messages`
  const body = buildAnthropicBody(settings, messages)
  const res = await fetch(url, {
    method: 'POST',
    headers: anthropicHeaders(settings),
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const errText = await res.text()
    throw new Error(errText || `HTTP ${res.status}`)
  }
  if (settings.stream) {
    const out = await consumeAnthropicSSE(res, streamHandlers)
    const latencyMs = Math.round(performance.now() - startedAt)
    return {
      content: out.content,
      usage: buildMetrics(settings, res.headers, out.usage, latencyMs, ttftMs),
      raw: out.raw,
    }
  }
  const json = (await res.json()) as {
    content?: Array<{ type?: string; text?: string }>
    usage?: { input_tokens?: number; output_tokens?: number }
  }
  let content = ''
  for (const block of json.content || []) {
    if (block.type === 'text' && block.text) content += block.text
  }
  const raw = JSON.stringify(json, null, 2)
  const latencyMs = Math.round(performance.now() - startedAt)
  const tokenUsage = json.usage
    ? {
        input_tokens: json.usage.input_tokens,
        output_tokens: json.usage.output_tokens,
        total_tokens: (json.usage.input_tokens || 0) + (json.usage.output_tokens || 0),
      }
    : undefined
  return {
    content,
    usage: buildMetrics(settings, res.headers, tokenUsage, latencyMs, latencyMs),
    raw,
  }
}

export function getDefaultPlaygroundBaseUrl(): string {
  const api = (import.meta.env.VITE_API_BASE_URL as string | undefined) || 'http://localhost:7072/api'
  const trimmed = api.replace(/\/+$/, '').replace(/\/api\/?$/, '')
  return `${trimmed}/v1`
}
