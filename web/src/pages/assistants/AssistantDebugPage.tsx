import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Descriptions, Modal, Slider, Tag, Typography, Drawer } from '@arco-design/web-react'
import { Input } from '@/components/ui'
import { IconClose, IconHome, IconInfoCircle, IconMenu, IconPhone, IconRefresh, IconSend, IconUpload, IconUser } from '@arco-design/web-react/icon'
import { Bot } from 'lucide-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { useSidebar } from '@/contexts/SidebarContext'
import { getAssistant, type AssistantRow } from '@/api/assistants'
import { listAssistantTools, mcpToolBindKey, type AssistantToolRow } from '@/api/assistantTools'
import { listVoiceClones } from '@/api/voiceClone'
import {
  buildVoiceSessionWebSocketURL,
  createVoiceSession,
  decodeBase64PCM,
  decodeBinaryPCM,
  endVoiceSession,
  getAuthToken,
  pcm16ToBytes,
  postVoiceSessionWebRTCOffer,
  type VoiceSessionTurnMetrics,
  type VoiceSessionWireFrame,
} from '@/api/voiceSession'
import {
  createDialogConversation,
  endDialogConversation,
  postDialogMessage,
  postDialogMessageStream,
} from '@/api/dialog'
import { showAlert } from '@/utils/notification'
import { VoiceDebugAudio } from '@/pages/assistants/assistantDebugAudio'
import { Button } from '@/components/ui'
import { useAuthStore } from '@/stores/authStore'
import { assistantDebugVoiceMode, assistantUsesCloneVoice } from '@/utils/assistantVoiceMode'
import { agentConfigFromJSON } from '@/constants/assistantAdvancedConfig'
import { useTranslation } from '@/i18n'

type DebugTransport = 'text' | 'websocket' | 'webrtc'

type ChatMsg = {
  id: string
  role: 'user' | 'assistant' | 'system'
  text: string
  turnId?: string
  metrics?: VoiceSessionTurnMetrics
  /** text-mode pending assistant bubble */
  status?: 'thinking' | 'streaming' | 'done'
  stage?: string
  toolsJson?: string
  latencyMs?: number
}

const TEXT_STAGE_LABEL: Record<string, string> = {
  thinking: '思考中',
  tools: '调用工具',
  retrieving: '检索知识库',
  generating: '生成回复',
}

function parseToolsJson(raw?: string): { name: string; arguments?: string }[] {
  if (!raw?.trim()) return []
  try {
    const arr = JSON.parse(raw) as { name?: string; arguments?: string }[]
    if (!Array.isArray(arr)) return []
    return arr
      .map((x) => ({ name: String(x.name || '').trim(), arguments: x.arguments }))
      .filter((x) => x.name)
  } catch {
    return []
  }
}

let msgSeq = 0
function nextMsgId() {
  msgSeq += 1
  return `m-${msgSeq}`
}

function fmtMs(v?: number) {
  if (v == null || v <= 0) return '—'
  return `${v} ms`
}

function fmtTime(iso?: string) {
  if (!iso) return '—'
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}

function parseWireData(data: unknown): VoiceSessionWireFrame | null {
  try {
    if (typeof data === 'string') return JSON.parse(data) as VoiceSessionWireFrame
    if (data instanceof ArrayBuffer) {
      return JSON.parse(new TextDecoder().decode(data)) as VoiceSessionWireFrame
    }
    if (ArrayBuffer.isView(data)) {
      return JSON.parse(new TextDecoder().decode(data)) as VoiceSessionWireFrame
    }
    return JSON.parse(String(data)) as VoiceSessionWireFrame
  } catch {
    return null
  }
}

function hasLatencyMetrics(m?: VoiceSessionTurnMetrics) {
  if (!m) return false
  return (m.e2eFirstMs ?? 0) > 0 || (m.llmFirstMs ?? 0) > 0 || (m.llmWallMs ?? 0) > 0
}

function hasKnowledgeRetrievals(m?: VoiceSessionTurnMetrics) {
  return (m?.knowledgeRetrievals?.length ?? 0) > 0
}

function kbSummary(m?: VoiceSessionTurnMetrics): string | null {
  const recs = m?.knowledgeRetrievals
  if (!recs?.length) return null
  const totalHits = recs.reduce((n, r) => n + (r.hitCount ?? r.hits?.length ?? 0), 0)
  const recallMs = recs.reduce((n, r) => n + (r.recallMs ?? 0), 0)
  const embedMs = recs.reduce((n, r) => n + (r.embedMs ?? 0), 0)
  const parts = [`KB ${totalHits} hits`]
  if (recallMs > 0) parts.push(`${recallMs}ms`)
  if (embedMs > 0) parts.push(`embed ${embedMs}ms`)
  return parts.join(' · ')
}

const TRANSPORT_OPTIONS: { key: DebugTransport; label: string; hint: string }[] = [
  { key: 'websocket', label: 'WebSocket', hint: 'PCM 16kHz · 带转写' },
  { key: 'webrtc', label: 'WebRTC', hint: 'Opus · 低延迟 · 带转写' },
  { key: 'text', label: '文本', hint: '纯文字 LLM 对话' },
]

export type AssistantDebugPanelProps = {
  assistantId: string
  embedded?: boolean
  onClose?: () => void
  assistantName?: string
  assistantAvatar?: string
  fileUploadEnabled?: boolean
  maxFiles?: number
}

export function AssistantDebugPanel({
  assistantId,
  embedded = false,
  onClose,
  assistantName: assistantNameProp,
  assistantAvatar: assistantAvatarProp,
  fileUploadEnabled = false,
  maxFiles = 8,
}: AssistantDebugPanelProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { setMobileOpen } = useSidebar()
  const tenantVoiceMode = useAuthStore((s) => s.user?.tenantVoiceMode)
  const [isLg, setIsLg] = useState(() =>
    typeof window !== 'undefined' ? window.matchMedia('(min-width: 1024px)').matches : false,
  )

  useEffect(() => {
    const q = window.matchMedia('(min-width: 1024px)')
    const sync = () => setIsLg(q.matches)
    sync()
    q.addEventListener('change', sync)
    return () => q.removeEventListener('change', sync)
  }, [])

  const [assistant, setAssistant] = useState<AssistantRow | null>(null)
  const [loading, setLoading] = useState(true)
  const [transport, setTransport] = useState<DebugTransport>('text')
  const [connected, setConnected] = useState(false)
  const [connecting, setConnecting] = useState(false)
  const [sending, setSending] = useState(false)
  const [messages, setMessages] = useState<ChatMsg[]>([])
  const [status, setStatus] = useState('未连接')
  const [draft, setDraft] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [attachedFiles, setAttachedFiles] = useState<File[]>([])
  const [metricsModal, setMetricsModal] = useState<ChatMsg | null>(null)
  const [initialized, setInitialized] = useState(false)
  const [gatewayMode, setGatewayMode] = useState(false)
  const [realtimeTemperature, setRealtimeTemperature] = useState(0.6)
  const [cloneVoiceIds, setCloneVoiceIds] = useState<Set<string>>(() => new Set())
  const [catalogTools, setCatalogTools] = useState<AssistantToolRow[]>([])

  const wsRef = useRef<WebSocket | null>(null)
  const sessionIdRef = useRef('')
  /** Dialog conversation id when transport=text (persisted text core). */
  const dialogConvIdRef = useRef('')
  const sampleRateRef = useRef(16000)
  const audioRef = useRef<VoiceDebugAudio | null>(null)
  const pcRef = useRef<RTCPeerConnection | null>(null)
  const dcRef = useRef<RTCDataChannel | null>(null)
  const localStreamRef = useRef<MediaStream | null>(null)
  const remoteAudioRef = useRef<HTMLAudioElement | null>(null)
  const listRef = useRef<HTMLDivElement | null>(null)

  const upsertTranscript = useCallback((role: 'user' | 'assistant', text: string, turnId?: string) => {
    const t = text.trim()
    if (!t) return
    if (!turnId) {
      setMessages((prev) => [...prev, { id: nextMsgId(), role, text: t }])
      return
    }
    setMessages((prev) => {
      const idx = prev.findIndex((m) => m.turnId === turnId && m.role === role)
      if (idx >= 0) {
        const next = [...prev]
        next[idx] = { ...next[idx], text: t }
        return next
      }
      return [...prev, { id: nextMsgId(), role, text: t, turnId }]
    })
  }, [])

  const applyTurnMetrics = useCallback((metrics: VoiceSessionTurnMetrics) => {
    const turnId = metrics.turnId?.trim()
    if (!turnId || (!hasLatencyMetrics(metrics) && !hasKnowledgeRetrievals(metrics))) return
    setMessages((prev) =>
      prev.map((m) =>
        m.turnId === turnId && m.role === 'assistant' ? { ...m, metrics } : m,
      ),
    )
  }, [])

  const appendMessage = useCallback((role: ChatMsg['role'], text: string) => {
    const t = text.trim()
    if (!t) return
    setMessages((prev) => [...prev, { id: nextMsgId(), role, text: t }])
  }, [])

  useEffect(() => {
    if (!assistantId) return
    setLoading(true)
    getAssistant(assistantId)
      .then((res) => {
        if (res.data) setAssistant(res.data)
      })
      .catch((e: { msg?: string }) => showAlert(e?.msg || '加载智能体失败', 'error'))
      .finally(() => setLoading(false))
  }, [assistantId])

  useEffect(() => {
    let cancelled = false
    void listVoiceClones('success')
      .then((res) => {
        if (cancelled || res.code !== 200 || !res.data) return
        const ids = new Set<string>()
        for (const c of res.data) {
          const id = (c.assetId || c.speakerId || String(c.id)).trim()
          if (id) ids.add(id)
        }
        setCloneVoiceIds(ids)
      })
      .catch(() => {
        if (!cancelled) setCloneVoiceIds(new Set())
      })
    return () => {
      cancelled = true
    }
  }, [assistantId])

  useEffect(() => {
    void listAssistantTools({ enabled: true })
      .then((res) => {
        if (res.code === 200 && Array.isArray(res.data)) {
          setCatalogTools(res.data.filter((r) => r.enabled !== false))
        }
      })
      .catch(() => setCatalogTools([]))
  }, [assistantId])

  const [boundToolsOpen, setBoundToolsOpen] = useState(false)

  const boundToolLabels = useMemo(() => {
    const agent = agentConfigFromJSON(assistant?.agentConfig)
    const ids = (agent.customToolIds || []).map((x) => String(x).trim()).filter(Boolean)
    if (!ids.length) return []
    const labels: string[] = []
    for (const id of ids) {
      let found = false
      for (const tool of catalogTools) {
        if (id === tool.id) {
          labels.push(`${tool.displayName || tool.name} (全部)`)
          found = true
          break
        }
        for (const dt of tool.discoveredTools || []) {
          if (id === mcpToolBindKey(tool.id, dt.name)) {
            labels.push(`${tool.displayName || tool.name}/${dt.name}`)
            found = true
            break
          }
        }
        if (found) break
      }
      if (!found) labels.push(id)
    }
    return labels
  }, [assistant, catalogTools])

  // 自动连接：助手加载完成后，自动以文本模式创建 dialog 会话
  useEffect(() => {
    if (initialized || !assistant || loading) return
    setInitialized(true)
    const autoStart = async () => {
      setTransport('text')
      setConnecting(true)
      setMessages([])
      setDraft('')
      setStatus('创建会话…')
      try {
        const res = await createDialogConversation({
          assistantId,
          channel: 'debug',
        })
        if (!res.data?.id) throw new Error(res.msg || '创建会话失败')
        dialogConvIdRef.current = res.data.id
        sessionIdRef.current = res.data.id
        setConnected(true)
        setConnecting(false)
        setStatus('文本 · 已连接')
        const welcome = res.data.welcomeText?.trim()
        if (welcome) {
          setMessages([{ id: nextMsgId(), role: 'assistant', text: welcome }])
        }
      } catch (e: unknown) {
        const msg = e instanceof Error ? e.message : '连接失败'
        showAlert(msg, 'error')
        cleanup()
      }
    }
    void autoStart()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [assistant, loading])

  const cleanup = useCallback(() => {
    const sid = sessionIdRef.current
    const dialogId = dialogConvIdRef.current
    audioRef.current?.close()
    audioRef.current = null
    if (wsRef.current) {
      try {
        wsRef.current.send(JSON.stringify({ type: 'hangup' }))
      } catch {
        /* ignore */
      }
      wsRef.current.close()
      wsRef.current = null
    }
    dcRef.current?.close()
    dcRef.current = null
    localStreamRef.current?.getTracks().forEach((t) => t.stop())
    localStreamRef.current = null
    pcRef.current?.close()
    pcRef.current = null
    if (remoteAudioRef.current) {
      remoteAudioRef.current.srcObject = null
    }
    if (dialogId) {
      void endDialogConversation(dialogId).catch(() => {})
      dialogConvIdRef.current = ''
      sessionIdRef.current = ''
    } else if (sid) {
      void endVoiceSession(sid).catch(() => {})
      sessionIdRef.current = ''
    }
    setConnected(false)
    setConnecting(false)
    setSending(false)
    setInitialized(false)
    setStatus('已断开')
  }, [])

  useEffect(() => () => cleanup(), [cleanup])

  const handleWireFrame = useCallback(
    (fr: VoiceSessionWireFrame) => {
      switch (fr.type) {
        case 'hello':
          if (fr.sampleRateHz) sampleRateRef.current = fr.sampleRateHz
          setStatus(`已连接 · 引擎 ${sampleRateRef.current} Hz`)
          break
        case 'pcm':
          if (fr.data && audioRef.current) {
            const raw = decodeBase64PCM(fr.data)
            const pcm = new Int16Array(raw.buffer, raw.byteOffset, raw.byteLength / 2)
            audioRef.current.enqueue(pcm, fr.sampleRateHz || sampleRateRef.current)
          }
          break
        case 'transcript.user':
          upsertTranscript('user', fr.text || '', fr.turnId)
          break
        case 'transcript.assistant':
          upsertTranscript('assistant', fr.text || '', fr.turnId)
          break
        case 'turn.metrics':
          applyTurnMetrics(fr as VoiceSessionTurnMetrics)
          break
        case 'status':
          setStatus(fr.message || fr.state || 'ready')
          break
        case 'error':
          appendMessage('system', fr.message || 'error')
          showAlert(fr.message || '会话错误', 'error')
          break
        default:
          break
      }
    },
    [appendMessage, upsertTranscript, applyTurnMetrics],
  )

  const bindDataChannel = (dc: RTCDataChannel) => {
    dcRef.current = dc
    dc.binaryType = 'arraybuffer'
    dc.onopen = () => {
      setStatus('WebRTC · 已连接 · 转写通道就绪')
    }
    dc.onerror = () => {
      showAlert('WebRTC 转写通道错误', 'error')
    }
    dc.onmessage = (ev) => {
      const fr = parseWireData(ev.data)
      if (fr) handleWireFrame(fr)
    }
  }

  const connectWebSocket = async (sessionId: string, sampleRate: number) => {
    const token = await getAuthToken()
    const url = buildVoiceSessionWebSocketURL(sessionId, token)
    const audio = new VoiceDebugAudio(sampleRate)
    audioRef.current = audio
    await audio.resume()

    const ws = new WebSocket(url)
    ws.binaryType = 'arraybuffer'
    wsRef.current = ws
    ws.onopen = async () => {
      setConnected(true)
      setConnecting(false)
      await audio.startCapture((pcm) => {
        if (ws.readyState !== WebSocket.OPEN) return
        ws.send(pcm16ToBytes(pcm))
      })
      const ctxRate = audio.captureSampleRate()
      const rateHint = ctxRate !== sampleRate ? ` · 采集 ${ctxRate}→${sampleRate}` : ''
      setStatus(`WebSocket · ${sampleRate} Hz · 半双工${rateHint}`)
    }
    ws.onmessage = (ev) => {
      const payload = ev.data
      if (payload instanceof ArrayBuffer) {
        if (audioRef.current) {
          const pcm = decodeBinaryPCM(payload)
          audioRef.current.enqueue(pcm, sampleRateRef.current)
        }
        return
      }
      if (ArrayBuffer.isView(payload)) {
        if (audioRef.current) {
          const u8 = new Uint8Array(payload.buffer, payload.byteOffset, payload.byteLength)
          const pcm = decodeBinaryPCM(u8.slice().buffer as ArrayBuffer)
          audioRef.current.enqueue(pcm, sampleRateRef.current)
        }
        return
      }
      try {
        handleWireFrame(JSON.parse(String(payload)) as VoiceSessionWireFrame)
      } catch {
        /* ignore */
      }
    }
    ws.onerror = () => {
      showAlert('WebSocket 连接失败', 'error')
      cleanup()
    }
    ws.onclose = () => cleanup()
  }

  const connectWebRTC = async (sessionId: string) => {
    const pc = new RTCPeerConnection({
      iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
    })
    pcRef.current = pc

    pc.ondatachannel = (ev) => {
      if (ev.channel.label === 'dialog') bindDataChannel(ev.channel)
    }

    // Offerer must create the dialog data channel (transcripts + metrics).
    bindDataChannel(pc.createDataChannel('dialog', { ordered: true }))

    const audio = document.createElement('audio')
    audio.autoplay = true
    // @ts-expect-error playsInline not in standard HTMLAudioElement typing but supported in modern browsers
    audio.playsInline = true
    remoteAudioRef.current = audio
    pc.ontrack = (ev) => {
      audio.srcObject = ev.streams[0]
      void audio.play().catch(() => {})
    }
    pc.onconnectionstatechange = () => {
      const st = pc.connectionState
      if (st === 'connected') {
        setStatus('WebRTC · 已连接 · 带转写')
      } else if (st === 'failed') {
        showAlert(`WebRTC 连接失败: ${st}`, 'error')
        cleanup()
      }
    }

    const stream = await navigator.mediaDevices.getUserMedia({
      audio: { echoCancellation: true, noiseSuppression: true },
      video: false,
    })
    localStreamRef.current = stream
    stream.getTracks().forEach((t) => pc.addTrack(t, stream))

    const offer = await pc.createOffer({ offerToReceiveAudio: true })
    await pc.setLocalDescription(offer)
    await new Promise<void>((resolve) => {
      if (pc.iceGatheringState === 'complete') {
        resolve()
        return
      }
      const check = () => {
        if (pc.iceGatheringState === 'complete') {
          pc.removeEventListener('icegatheringstatechange', check)
          resolve()
        }
      }
      pc.addEventListener('icegatheringstatechange', check)
      setTimeout(resolve, 3000)
    })

    const answer = await postVoiceSessionWebRTCOffer({
      sessionId,
      sdp: pc.localDescription?.sdp || offer.sdp || '',
      type: 'offer',
    })
    await pc.setRemoteDescription({ type: 'answer', sdp: answer.sdp })
    setConnected(true)
    setConnecting(false)
    setStatus('WebRTC · 连接中…')
  }

  const connectText = async (welcomeText?: string) => {
    setConnected(true)
    setConnecting(false)
    setStatus('文本 · 已连接')
    if (welcomeText?.trim()) {
      appendMessage('assistant', welcomeText.trim())
    }
  }

  const connect = async () => {
    if (!assistantId || connecting || connected) return
    if (sessionIdRef.current || dialogConvIdRef.current || wsRef.current || pcRef.current) {
      cleanup()
    }
    setInitialized(true)
    setConnecting(true)
    setMessages([])
    setDraft('')
    setStatus('创建会话…')
    try {
      if (transport === 'text') {
        const res = await createDialogConversation({
          assistantId,
          channel: 'debug',
        })
        if (!res.data?.id) throw new Error(res.msg || '创建会话失败')
        dialogConvIdRef.current = res.data.id
        sessionIdRef.current = res.data.id
        await connectText(res.data.welcomeText)
        return
      }
      const res = await createVoiceSession({
        transport,
        assistantId,
        credentialId: assistant?.credentialId,
        sampleRateHz: 16000,
        dialogMode: transport === 'websocket' ? (customDialogWs ? 'gateway' : 'engine') : undefined,
      })
      if (!res.data?.sessionId) throw new Error(res.msg || '创建会话失败')
      setGatewayMode(res.data.dialogMode === 'gateway')
      sessionIdRef.current = res.data.sessionId
      sampleRateRef.current = res.data.sampleRateHz || 16000
      if (transport === 'websocket') {
        await connectWebSocket(res.data.sessionId, sampleRateRef.current)
      } else {
        await connectWebRTC(res.data.sessionId)
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '连接失败'
      showAlert(msg, 'error')
      cleanup()
    }
  }

  const sendText = async () => {
    const text = draft.trim()
    const files = attachedFiles
    if ((!text && files.length === 0) || !connected || sending || transport !== 'text') return
    const cid = dialogConvIdRef.current || sessionIdRef.current
    if (!cid) return
    if (files.length > 0) {
      showAlert('文本对话通道暂不支持文件上传，请改用语音调试或纯文本', 'warning')
      return
    }
    const displayText = text
    const userMsgId = nextMsgId()
    const assistantId = nextMsgId()
    setMessages((prev) => [
      ...prev,
      { id: userMsgId, role: 'user', text: displayText },
      { id: assistantId, role: 'assistant', text: '', status: 'thinking', stage: 'thinking' },
    ])
    setDraft('')
    setAttachedFiles([])
    setSending(true)
    const patchAssistant = (patch: Partial<ChatMsg>) => {
      setMessages((prev) => prev.map((m) => (m.id === assistantId ? { ...m, ...patch } : m)))
    }
    try {
      let streamed = false
      try {
        const result = await postDialogMessageStream(cid, text, {
          onStage: (stage) => {
            patchAssistant({
              stage,
              status: streamed ? 'streaming' : 'thinking',
            })
          },
          onDelta: (delta) => {
            streamed = true
            setMessages((prev) =>
              prev.map((m) =>
                m.id === assistantId
                  ? { ...m, text: m.text + delta, status: 'streaming', stage: 'generating' }
                  : m,
              ),
            )
          },
        })
        const reply = result.reply?.trim() ?? ''
        if (!reply && !streamed) {
          setMessages((prev) => prev.filter((m) => m.id !== assistantId))
          return
        }
        setMessages((prev) =>
          prev.map((m) =>
            m.id === assistantId
              ? {
                  ...m,
                  text: reply || m.text,
                  status: 'done',
                  stage: undefined,
                  toolsJson: result.toolsJson,
                  latencyMs: result.latencyMs,
                }
              : m,
          ),
        )
      } catch {
        const res = await postDialogMessage(cid, text)
        if (res.code !== 200 && res.code !== 0 && res.msg) {
          showAlert(res.msg, 'error')
          setMessages((prev) => prev.filter((m) => m.id !== assistantId))
          return
        }
        const reply = res.data?.reply?.trim()
        if (reply) {
          setMessages((prev) =>
            prev.map((m) =>
              m.id === assistantId
                ? {
                    ...m,
                    text: reply,
                    status: 'done',
                    stage: undefined,
                    toolsJson: res.data?.toolsJson,
                    latencyMs: res.data?.latencyMs,
                  }
                : m,
            ),
          )
        } else {
          setMessages((prev) => prev.filter((m) => m.id !== assistantId))
        }
      }
    } catch (e: unknown) {
      showAlert(e instanceof Error ? e.message : '发送失败', 'error')
      setMessages((prev) => prev.filter((m) => m.id !== assistantId))
    } finally {
      setSending(false)
    }
  }

  const resolvedFileUpload = useMemo(() => {
    if (fileUploadEnabled) {
      return { enabled: true, maxFiles }
    }
    const cfg = agentConfigFromJSON(assistant?.agentConfig).fileUpload
    const enabled = !!cfg?.enabled && (!!cfg.documentEnabled || !!cfg.imageEnabled)
    return { enabled, maxFiles: cfg?.maxFiles ?? 8 }
  }, [assistant?.agentConfig, fileUploadEnabled, maxFiles])
  const voiceMode = assistantDebugVoiceMode(
    tenantVoiceMode,
    assistant?.ttsVoice,
    cloneVoiceIds,
    transport,
  )
  const usesCloneVoice = assistantUsesCloneVoice(assistant?.ttsVoice, cloneVoiceIds)
  const voiceTransport = transport === 'websocket' || transport === 'webrtc'
  const assistantName = assistantNameProp || assistant?.name || (loading ? '加载中…' : '智能体')
  const assistantAvatar = assistantAvatarProp || assistant?.avatarUrl || assistant?.avatar
  const isTextMode = transport === 'text'
  const customDialogWs = assistant?.voiceDialogWsUrl?.trim() || ''
  const shellClass = embedded
    ? 'flex h-full min-h-[520px] flex-col bg-[#f7f7f8] dark:bg-[#1e1e1e]'
    : 'flex h-[calc(100vh-56px)] flex-col bg-[#f7f7f8] dark:bg-[#1e1e1e]'

  const panel = (
    <>
    <div className={shellClass}>
        {/* ---- 顶部栏 ---- */}
        <div className="flex items-center justify-between gap-2 border-b border-border/60 bg-white px-4 py-2.5 dark:bg-[#2d2d2d]" style={{ minHeight: embedded ? 44 : 52 }}>
          <div className="flex min-w-0 flex-1 items-center gap-1.5 text-sm text-muted-foreground">
            {!embedded && !isLg && (
              <button
                onClick={() => setMobileOpen(true)}
                className="shrink-0 rounded p-0.5 hover:text-foreground transition-colors mr-1"
              >
                <IconMenu style={{ fontSize: 15 }} />
              </button>
            )}
            {!embedded ? (
              <>
                <button onClick={() => navigate('/')} className="shrink-0 rounded p-0.5 hover:text-foreground transition-colors">
                  <IconHome style={{ fontSize: 15 }} />
                </button>
                <span className="text-border">/</span>
                <button onClick={() => navigate('/assistant-manager')} className="shrink-0 hover:text-foreground transition-colors">
                  智能小助手
                </button>
                <span className="text-border">/</span>
                <button onClick={() => navigate(`/assistant-manager/${assistantId}/edit`)} className="shrink-0 hover:text-foreground transition-colors truncate max-w-[160px]">
                  {assistantName}
                </button>
                <span className="text-border">/</span>
                <span className="font-medium text-foreground truncate">调试对话</span>
              </>
            ) : (
              <span className="font-medium text-foreground truncate">调试对话</span>
            )}
          </div>
          <div className="flex shrink-0 items-center gap-1.5">
            <Tag
              color={voiceMode === 'realtime' ? 'green' : 'arcoblue'}
              size="small"
              className="max-w-[5.5rem] truncate"
              title={voiceMode === 'realtime' ? 'Realtime' : 'Pipeline'}
            >
              {voiceMode === 'realtime' ? 'Realtime' : 'Pipeline'}
            </Tag>
            {usesCloneVoice ? (
              <Tag color="purple" size="small" className="max-w-[5rem] truncate" title="克隆音色">
                克隆
              </Tag>
            ) : null}
            {voiceTransport && tenantVoiceMode === 'realtime' ? (
              <Tag color="gray" size="small" className="hidden sm:inline-flex" title="WebSocket/WebRTC 调试固定走 Pipeline 管线">
                语音·Pipeline
              </Tag>
            ) : null}
            {embedded && onClose ? (
              <button
                type="button"
                onClick={onClose}
                className="flex h-7 w-7 items-center justify-center rounded-md border border-border text-muted-foreground hover:text-foreground"
                aria-label="关闭调试"
              >
                <IconClose style={{ fontSize: 14 }} />
              </button>
            ) : null}
            {!embedded ? (
              <>
                {customDialogWs ? (
                  <Tag color="orange" size="small" title={customDialogWs}>
                    外部对话 WS
                  </Tag>
                ) : null}
                {gatewayMode ? (
                  <Tag color="purple" size="small">
                    网关模拟
                  </Tag>
                ) : null}
                <button
                  type="button"
                  onClick={cleanup}
                  disabled={!connected}
                  className="flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:text-destructive hover:border-destructive/50 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                >
                  <IconRefresh style={{ fontSize: 13 }} />
                  <span>重新开始</span>
                </button>
              </>
            ) : (
              <button
                type="button"
                onClick={cleanup}
                disabled={!connected}
                className="rounded-md border border-border px-2 py-1 text-xs text-muted-foreground hover:text-destructive disabled:opacity-30"
              >
                重置
              </button>
            )}
          </div>
        </div>

        {/* ---- 传输模式选择 + 状态 ---- */}
        <div className="border-b border-border/40 bg-white px-4 py-2 dark:bg-[#2d2d2d]">
          <div className={`flex flex-col gap-2 ${embedded ? '' : 'mx-auto max-w-3xl'}`}>
            <div className="flex flex-wrap items-center gap-2">
            {TRANSPORT_OPTIONS.map((opt) => {
              const active = transport === opt.key
              return (
                <button
                  key={opt.key}
                  type="button"
                  disabled={connected || connecting}
                  onClick={() => setTransport(opt.key)}
                  className={`max-w-full truncate rounded-lg border px-2.5 py-1.5 text-xs font-medium transition-colors ${
                    active
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'border-border/60 text-muted-foreground hover:border-primary/30 hover:text-foreground'
                  } disabled:cursor-not-allowed disabled:opacity-40`}
                >
                  {opt.label}
                  {!embedded ? (
                    <span className="ml-1.5 hidden font-normal opacity-70 sm:inline">{opt.hint}</span>
                  ) : null}
                </button>
              )
            })}
            </div>
            {transport === 'text' && voiceMode === 'realtime' && (
              <div className="flex items-center gap-3 text-xs text-muted-foreground">
                <span className="shrink-0">温度 {realtimeTemperature.toFixed(1)}</span>
                <Slider
                  style={{ flex: 1, minWidth: 120 }}
                  min={0.6}
                  max={1.2}
                  step={0.1}
                  value={realtimeTemperature}
                  disabled={connected || connecting}
                  onChange={(v) => setRealtimeTemperature(Number(v))}
                />
                <span className="shrink-0 text-[11px]">0.6 稳定 · 1.2 发散</span>
              </div>
            )}
            <span className="text-[11px] text-muted-foreground">
              {status}
              {gatewayMode ? ' · 网关模拟（voicedialog）' : connected ? ' · 引擎模式（含已绑定 MCP/HTTP 工具）' : ''}
              {customDialogWs && !isTextMode ? ` · 对话端: ${customDialogWs}` : ''}
            </span>
            {boundToolLabels.length > 0 ? (
              <div className="flex flex-wrap items-center gap-1.5 text-[11px]">
                <span className="text-muted-foreground shrink-0">{t('assistant.debugBoundTools')}:</span>
                <span>{t('assistant.debugBoundToolsCount', { count: boundToolLabels.length })}</span>
                <button
                  type="button"
                  className="text-primary underline"
                  onClick={() => setBoundToolsOpen(true)}
                >
                  {t('assistant.debugBoundToolsDetail')}
                </button>
              </div>
            ) : (
              <span className="text-[11px] text-amber-600 dark:text-amber-400">
                {t('assistant.debugNoBoundTools')}
              </span>
            )}
          </div>
        </div>

        {/* ---- 消息列表 ---- */}
        <div ref={listRef} className="flex-1 overflow-y-auto">
          {messages.length === 0 && !loading && (
            <div className="flex h-full items-center justify-center">
              <div className="text-center text-muted-foreground">
                <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-muted/50">
                  <IconPhone style={{ fontSize: 20 }} />
                </div>
                <p className="text-sm font-medium">
                  {isTextMode ? '文本对话模式' : '语音对话模式'}
                </p>
                <p className="mt-1 text-xs text-muted-foreground/70">
                  {isTextMode ? '在下方输入消息开始对话' : '点击开始对话后对着麦克风说话'}
                </p>
              </div>
            </div>
          )}
          <div className={embedded ? 'py-3' : 'mx-auto max-w-3xl py-4'}>
            {messages.map((m) => {
              const hasMetrics =
                m.role === 'assistant' && (hasLatencyMetrics(m.metrics) || hasKnowledgeRetrievals(m.metrics))
              const kbLine = kbSummary(m.metrics)
              const isUser = m.role === 'user'
              const isSystem = m.role === 'system'
              const isThinking =
                m.role === 'assistant' &&
                (m.status === 'thinking' || (m.status === 'streaming' && !m.text.trim()))
              const isStreaming = m.role === 'assistant' && m.status === 'streaming' && !!m.text.trim()
              const toolRows = parseToolsJson(m.toolsJson)
              const stageLabel = m.stage ? TEXT_STAGE_LABEL[m.stage] || m.stage : ''
              const isRight = isUser
              return (
                <div key={m.id} className={`px-5 py-3 ${isRight ? '' : 'bg-white dark:bg-[#2d2d2d]'}`}>
                  <div className={`mx-auto flex max-w-3xl gap-3 ${isRight ? 'flex-row-reverse' : ''}`}>
                    {isUser ? (
                      assistantAvatar ? (
                        <img
                          src={assistantAvatar}
                          alt="avatar"
                          className="h-8 w-8 shrink-0 rounded-full object-cover"
                        />
                      ) : (
                        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
                          <IconUser style={{ fontSize: 16 }} />
                        </div>
                      )
                    ) : isSystem ? (
                      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
                        <IconInfoCircle style={{ fontSize: 16 }} />
                      </div>
                    ) : (
                      <div
                        className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#19c37d]/10 text-[#19c37d] ${
                          isThinking ? 'animate-pulse' : ''
                        }`}
                      >
                        <Bot size={18} />
                      </div>
                    )}
                    <div className={`min-w-0 flex-1 ${isRight ? 'flex flex-col items-end' : ''}`}>
                      {!isUser && (isThinking || isStreaming || toolRows.length > 0 || (m.latencyMs ?? 0) > 0) ? (
                        <div className="mb-1.5 flex flex-wrap items-center gap-1.5">
                          {isThinking || isStreaming ? (
                            <span className="inline-flex items-center gap-1.5 rounded-full bg-[#19c37d]/10 px-2 py-0.5 text-[11px] font-medium text-[#19c37d]">
                              <span className="relative flex h-1.5 w-1.5">
                                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-[#19c37d] opacity-60" />
                                <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-[#19c37d]" />
                              </span>
                              {stageLabel || (isStreaming ? '生成中' : '思考中')}
                            </span>
                          ) : null}
                          {toolRows.map((t) => (
                            <Tag key={t.name} size="small" color="purple">
                              {t.name}
                            </Tag>
                          ))}
                          {(m.latencyMs ?? 0) > 0 && m.status === 'done' ? (
                            <span className="text-[11px] text-muted-foreground">{m.latencyMs} ms</span>
                          ) : null}
                        </div>
                      ) : null}
                      <div
                        role={hasMetrics ? 'button' : undefined}
                        tabIndex={hasMetrics ? 0 : undefined}
                        onClick={hasMetrics ? () => setMetricsModal(m) : undefined}
                        onKeyDown={
                          hasMetrics
                            ? (e) => {
                                if (e.key === 'Enter' || e.key === ' ') setMetricsModal(m)
                              }
                            : undefined
                        }
                        className={`text-sm leading-relaxed whitespace-pre-wrap ${
                          hasMetrics ? 'cursor-pointer rounded-lg p-2 -mx-2 hover:bg-muted/30 transition-colors' : ''
                        }`}
                      >
                        {isThinking && !m.text.trim() ? (
                          <div className="flex items-center gap-2 py-0.5 text-muted-foreground">
                            <span className="flex items-center gap-1">
                              <span
                                className="h-1.5 w-1.5 animate-bounce rounded-full bg-muted-foreground/70"
                                style={{ animationDelay: '0ms' }}
                              />
                              <span
                                className="h-1.5 w-1.5 animate-bounce rounded-full bg-muted-foreground/70"
                                style={{ animationDelay: '150ms' }}
                              />
                              <span
                                className="h-1.5 w-1.5 animate-bounce rounded-full bg-muted-foreground/70"
                                style={{ animationDelay: '300ms' }}
                              />
                            </span>
                            <span className="text-xs">{stageLabel || '正在思考…'}</span>
                          </div>
                        ) : (
                          <>
                            {m.text}
                            {isStreaming ? (
                              <span className="ml-0.5 inline-block h-3.5 w-0.5 animate-pulse bg-foreground/70 align-middle" />
                            ) : null}
                          </>
                        )}
                        {hasMetrics && (
                          <span className="ml-2 inline-flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[11px] text-muted-foreground">
                            {hasLatencyMetrics(m.metrics) ? (
                              <span className="inline-flex items-center gap-1">
                                <IconInfoCircle style={{ fontSize: 12 }} />
                                E2E {fmtMs(m.metrics?.e2eFirstMs || m.metrics?.ttsFirstMs)}
                                {(m.metrics?.llmFirstMs ?? 0) > 0 ? (
                                  <span>· LLM {fmtMs(m.metrics?.llmFirstMs)}</span>
                                ) : null}
                              </span>
                            ) : null}
                            {kbLine ? <span className="inline-flex items-center gap-1">{kbLine}</span> : null}
                          </span>
                        )}
                      </div>
                      {hasKnowledgeRetrievals(m.metrics) ? (
                        <div className={`mt-2 max-w-full space-y-1.5 ${isRight ? 'text-right' : ''}`}>
                          {(m.metrics?.knowledgeRetrievals || []).map((rec, idx) => (
                            <div
                              key={idx}
                              className="rounded-lg border border-border/60 bg-muted/30 px-2.5 py-2 text-left text-[11px] text-muted-foreground"
                            >
                              <div className="font-medium text-foreground/80">
                                知识库检索
                                {rec.searchQuery || rec.query
                                  ? ` · ${(rec.searchQuery || rec.query || '').slice(0, 40)}`
                                  : ''}
                                {rec.hitCount != null ? ` · ${rec.hitCount} hits` : ''}
                                {rec.recallMs != null && rec.recallMs > 0 ? ` · ${rec.recallMs}ms` : ''}
                                {rec.embedMs != null && rec.embedMs > 0 ? ` · embed ${rec.embedMs}ms` : ''}
                                {rec.qdrantMs != null && rec.qdrantMs > 0 ? ` · qdrant ${rec.qdrantMs}ms` : ''}
                              </div>
                              {rec.strategy ? <div className="mt-0.5">策略: {rec.strategy}</div> : null}
                              {(rec.hits || []).slice(0, 3).map((hit, hi) => (
                                <div key={hi} className="mt-1 border-t border-border/40 pt-1 text-foreground/70">
                                  {hit.title ? <span className="font-medium">{hit.title}: </span> : null}
                                  <span className="whitespace-pre-wrap">{(hit.content || '—').slice(0, 180)}</span>
                                  {hit.quoted ? (
                                    <Tag size="small" color="green" className="ml-1">
                                      引用
                                    </Tag>
                                  ) : null}
                                </div>
                              ))}
                            </div>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  </div>
                </div>
              )
            })}
            {/* 语音模式底部加载点；文本模式思考态已内嵌在助手气泡，避免双条 */}
            {sending && !isTextMode && (
              <div className="bg-white px-5 py-3 dark:bg-[#2d2d2d]">
                <div className="mx-auto flex max-w-3xl gap-3">
                  <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#19c37d]/10 text-[#19c37d]">
                    <Bot size={18} />
                  </div>
                  <div className="flex items-center gap-1.5 py-1">
                    <span className="h-2 w-2 animate-bounce rounded-full bg-muted-foreground/50" style={{ animationDelay: '0ms' }} />
                    <span className="h-2 w-2 animate-bounce rounded-full bg-muted-foreground/50" style={{ animationDelay: '150ms' }} />
                    <span className="h-2 w-2 animate-bounce rounded-full bg-muted-foreground/50" style={{ animationDelay: '300ms' }} />
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* ---- 底部输入区 ---- */}
        <div className="border-t border-border/40 bg-white px-4 py-3 dark:bg-[#2d2d2d]">
          <div className={embedded ? '' : 'mx-auto max-w-3xl'}>
            {!connected ? (
              <div className="flex justify-center">
                <Button
                  type="primary"
                  size="large"
                  shape="round"
                  icon={<IconPhone />}
                  loading={connecting}
                  long
                  style={{ maxWidth: 280 }}
                  onClick={() => void connect()}
                >
                  {connecting ? '连接中…' : isTextMode ? '开始文本对话' : '开始语音对话'}
                </Button>
              </div>
            ) : isTextMode ? (
              <div className="space-y-2">
                {attachedFiles.length > 0 && (
                  <div className="flex flex-wrap gap-2">
                    {attachedFiles.map((f) => (
                      <Tag key={f.name + f.size} closable onClose={() => setAttachedFiles((prev) => prev.filter((x) => x !== f))}>
                        {f.name}
                      </Tag>
                    ))}
                  </div>
                )}
                <div className="flex items-center gap-2">
                <input
                  ref={fileInputRef}
                  type="file"
                  className="hidden"
                  multiple={resolvedFileUpload.maxFiles > 1}
                  onChange={(e) => {
                    const picked = Array.from(e.target.files || [])
                    if (!picked.length) return
                    setAttachedFiles((prev) => [...prev, ...picked].slice(0, resolvedFileUpload.maxFiles))
                    e.target.value = ''
                  }}
                />
                {resolvedFileUpload.enabled && (
                  <button
                    type="button"
                    className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-border text-muted-foreground hover:bg-muted/40"
                    disabled={sending || attachedFiles.length >= resolvedFileUpload.maxFiles}
                    onClick={() => fileInputRef.current?.click()}
                    title="上传文件"
                  >
                    <IconUpload />
                  </button>
                )}
                <Input
                  value={draft}
                  placeholder="输入消息…（Enter 发送）"
                  disabled={sending}
                  onChange={setDraft}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !e.shiftKey) {
                      e.preventDefault()
                      void sendText()
                    }
                  }}
                  style={{
                    borderRadius: 12,
                    fontSize: 14,
                    height: 44,
                    border: '1px solid var(--color-border)',
                  }}
                />
                <button
                  type="button"
                  disabled={(!draft.trim() && attachedFiles.length === 0) || sending}
                  onClick={() => void sendText()}
                  className="flex h-[44px] w-[44px] shrink-0 items-center justify-center rounded-xl bg-primary text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-30 disabled:cursor-not-allowed"
                  style={{ fontSize: 18 }}
                >
                  <IconSend />
                </button>
              </div>
              </div>
            ) : (
              <div className="flex justify-center">
                <Button status="danger" size="large" shape="round" icon={<IconRefresh />} long style={{ maxWidth: 280 }} onClick={cleanup}>
                  结束对话
                </Button>
              </div>
            )}
            <p className="mt-2 text-center text-[11px] text-muted-foreground/60">
              {connected && isTextMode
                ? 'AI 智能体调试助手 · 文本模式'
                : connected
                  ? 'AI 智能体调试助手 · 语音模式'
                  : '选择传输模式开始调试'}
            </p>
          </div>
        </div>
      </div>

      <Modal
        title="本轮延迟指标"
        visible={!!metricsModal}
        footer={null}
        onCancel={() => setMetricsModal(null)}
        style={{ width: 480 }}
      >
        {metricsModal?.metrics &&
          (hasLatencyMetrics(metricsModal.metrics) || hasKnowledgeRetrievals(metricsModal.metrics)) && (
          <>
            <Typography.Paragraph type="secondary" style={{ marginTop: 0, fontSize: 13 }}>
              {metricsModal.text}
            </Typography.Paragraph>
            <Descriptions
              column={1}
              border
              size="small"
              data={[
                { label: 'Turn ID', value: metricsModal.metrics.turnId || '—' },
                { label: '模式', value: metricsModal.metrics.mode || '—' },
                { label: '传输', value: metricsModal.metrics.transport || '—' },
                { label: 'LLM 首 token', value: fmtMs(metricsModal.metrics.llmFirstMs) },
                { label: 'LLM 总耗时', value: fmtMs(metricsModal.metrics.llmWallMs) },
                { label: 'TTS 首包（ASR定稿起）', value: fmtMs(metricsModal.metrics.ttsFirstMs) },
                { label: '管线总耗时', value: fmtMs(metricsModal.metrics.pipelineMs) },
                { label: 'E2E（说完→首包到客户端）', value: fmtMs(metricsModal.metrics.e2eFirstMs) },
                { label: '完成时间', value: fmtTime(metricsModal.metrics.completedAt) },
              ]}
            />
            {hasKnowledgeRetrievals(metricsModal.metrics) ? (
              <div className="mt-3 space-y-2">
                <Typography.Text bold style={{ fontSize: 13 }}>知识库检索</Typography.Text>
                {(metricsModal.metrics.knowledgeRetrievals || []).map((rec, idx) => (
                  <div key={idx} className="rounded border border-border/60 px-2 py-1.5 text-xs text-muted-foreground">
                    <div>
                      {(rec.searchQuery || rec.query || '—').slice(0, 80)}
                      {rec.recallMs != null && rec.recallMs > 0 ? ` · ${rec.recallMs}ms` : ''}
                      {rec.embedMs != null && rec.embedMs > 0 ? ` · embed ${rec.embedMs}ms` : ''}
                      {rec.qdrantMs != null && rec.qdrantMs > 0 ? ` · qdrant ${rec.qdrantMs}ms` : ''}
                    </div>
                    {(rec.hits || []).map((hit, hi) => (
                      <div key={hi} className="mt-1">
                        {hit.title ? <strong>{hit.title}: </strong> : null}
                        {(hit.content || '').slice(0, 240)}
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            ) : null}
            {metricsModal.metrics.transport !== 'text' && (
              <Typography.Text type="secondary" style={{ display: 'block', marginTop: 12, fontSize: 12 }}>
                语音模式下，计时起点为 ASR 最终识别完成（用户说完一句话）。ASR 识别过程本身的耗时暂未单独统计。
              </Typography.Text>
            )}
          </>
        )}
      </Modal>

      <Drawer
        title={t('assistant.debugBoundToolsTitle')}
        visible={boundToolsOpen}
        onCancel={() => setBoundToolsOpen(false)}
        footer={null}
        width={420}
        unmountOnExit
      >
        <div className="flex flex-wrap gap-1.5">
          {boundToolLabels.map((label) => (
            <Tag key={label} size="small" color="arcoblue">{label}</Tag>
          ))}
        </div>
      </Drawer>
    </>
  )

  if (embedded) return panel

  return <BaseLayout hideHeader>{panel}</BaseLayout>
}

export default function AssistantDebugPage() {
  const { id: assistantId = '' } = useParams<{ id: string }>()
  return <AssistantDebugPanel assistantId={assistantId} />
}
