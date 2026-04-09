import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type MutableRefObject,
  type ReactNode,
} from 'react'
import { useLocation } from 'react-router-dom'
import { useAuthStore } from '@/stores/authStore'
import { formatUserSeatBaseName } from '@/utils/userDisplayName'
import { useI18nStore } from '@/stores/i18nStore'
import {
  clearWebSeatAcdPoolAnchor,
  ensureWebSeatAcdPoolRowOnline,
  postWebSeatAcdHeartbeat,
  setWebSeatAcdPoolRowOffline,
} from '@/api/sipContactCenter'
import { showAlert } from '@/utils/notification'
import { WebSeatContext, type WebSeatContextValue, type WebSeatWsState } from './WebSeatContext'
import { getUserMediaAudioOnly } from './getUserMediaCompat'
import { buildWebSeatWebSocketURL, webSeatHttpBase, webSeatWsBase, webSeatWsToken } from './webseatEnv'
import { WebSeatIncomingCallCard } from './WebSeatIncomingCallCard.tsx'
import { WebSeatWsIndicator } from './WebSeatWsIndicator.tsx'

const MAX_SIGNAL_LINES = 400
const MAX_RX_LINES = 250
/** Should be < server `WebSeatStaleAfter` (90s) so the row stays eligible for transfer pick. */
const WEBSEAT_ACD_HEARTBEAT_MS = 30_000

function appendLog(prev: string, line: string, maxLines: number): string {
  const next = prev + line + '\n'
  const parts = next.split('\n')
  if (parts.length > maxLines) {
    return parts.slice(-maxLines).join('\n')
  }
  return next
}

interface WebSeatProviderProps {
  children: ReactNode
}

function waitForWebSocketOpen(wsRef: MutableRefObject<WebSocket | null>, timeoutMs: number): Promise<void> {
  const start = Date.now()
  return new Promise((resolve, reject) => {
    const id = window.setInterval(() => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        window.clearInterval(id)
        resolve()
        return
      }
      if (Date.now() - start >= timeoutMs) {
        window.clearInterval(id)
        reject(new Error('WebSocket open timeout'))
      }
    }, 80)
  })
}

export function WebSeatProvider({ children }: WebSeatProviderProps) {
  const location = useLocation()
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const user = useAuthStore((s) => s.user)
  const { t } = useI18nStore()

  const httpBase = useMemo(() => webSeatHttpBase(), [])
  const wsBase = useMemo(() => webSeatWsBase(), [])
  const wsToken = useMemo(() => webSeatWsToken(), [])
  const webSeatAcdDisplayLabel = useMemo(() => {
    const base = formatUserSeatBaseName(user)
    const root = base || t('webseat.acdFallbackName')
    return `${root}-Web`
  }, [user, t])
  const configured = httpBase.length > 0

  const [wsState, setWsState] = useState<WebSeatWsState>(configured ? 'idle' : 'disabled')
  const [wsStatusText, setWsStatusText] = useState(
    configured ? t('webseat.wsNotLoggedIn') : t('webseat.wsNotConfigured')
  )
  const [presenceWsClients, setPresenceWsClients] = useState(0)
  const [presenceOnline, setPresenceOnline] = useState(false)
  const [signalLog, setSignalLog] = useState('')
  const [rxLog, setRxLog] = useState('')
  const [inCall, setInCall] = useState(false)
  const [hangupDisabled, setHangupDisabled] = useState(true)
  const [pendingIncomingCallId, setPendingIncomingCallId] = useState<string | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  /** When closing WS intentionally as "go offline", onclose should show user-offline copy. */
  const wsCloseIntentRef = useRef<'user-offline' | null>(null)
  const acdHeartbeatTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const stopAcdHeartbeat = useCallback(() => {
    if (acdHeartbeatTimerRef.current != null) {
      clearInterval(acdHeartbeatTimerRef.current)
      acdHeartbeatTimerRef.current = null
    }
  }, [])

  const pcRef = useRef<RTCPeerConnection | null>(null)
  const localStreamRef = useRef<MediaStream | null>(null)
  const activeCallIdRef = useRef<string | null>(null)
  const inWebSeatCallRef = useRef(false)
  const sessionEndingRef = useRef(false)
  const statusPollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const rxAudioCtxRef = useRef<AudioContext | null>(null)
  const rxMonitorStopRef = useRef<(() => void) | null>(null)
  /** Plays remote WebRTC audio to speakers (rx monitor alone uses gain=0 and is inaudible). */
  const remoteAudioRef = useRef<HTMLAudioElement | null>(null)

  const log = useCallback((...args: unknown[]) => {
    const line = args.map((a) => (typeof a === 'string' ? a : JSON.stringify(a))).join(' ')
    const ts = new Date().toISOString().slice(11, 23)
    setSignalLog((p) => appendLog(p, `[${ts}] ${line}`, MAX_SIGNAL_LINES))
    console.log('[webseat]', ...args)
  }, [])

  const closeWsConnection = useCallback(
    (intent?: 'user-offline') => {
      if (intent === 'user-offline') {
        wsCloseIntentRef.current = 'user-offline'
      } else {
        wsCloseIntentRef.current = null
      }
      const had = !!wsRef.current
      if (wsRef.current) {
        try {
          wsRef.current.close()
        } catch {
          /* ignore */
        }
        wsRef.current = null
      }
      setPresenceWsClients(0)
      setPresenceOnline(false)
      if (!had && intent === 'user-offline' && configured) {
        wsCloseIntentRef.current = null
        setWsState('closed')
        setWsStatusText(t('webseat.wsUserOffline'))
      }
    },
    [configured, t]
  )

  const connectWebSocket = useCallback(() => {
    if (!configured) return
    closeWsConnection()
    const url = buildWebSeatWebSocketURL(httpBase, wsToken, wsBase)
    setWsState('connecting')
    setWsStatusText(t('webseat.wsConnecting'))
    try {
      const ws = new WebSocket(url)
      wsRef.current = ws
      ws.onopen = () => {
        setWsState('open')
        setWsStatusText(t('webseat.wsWaitingCall'))
        log('WebSocket 已连接', url.replace(/token=[^&]+/, 'token=***'))
      }
      ws.onclose = () => {
        wsRef.current = null
        stopAcdHeartbeat()
        setWsState('closed')
        const intent = wsCloseIntentRef.current
        wsCloseIntentRef.current = null
        setWsStatusText(
          intent === 'user-offline' ? t('webseat.wsUserOffline') : t('webseat.wsDisconnected')
        )
        if (intent !== 'user-offline') {
          void setWebSeatAcdPoolRowOffline()
            .then(() => window.dispatchEvent(new CustomEvent('soulnexus-acd-refresh')))
            .catch(() => {})
        }
      }
      ws.onerror = () => {
        log('WebSocket error')
      }
      ws.onmessage = (ev) => {
        try {
          const data = JSON.parse(ev.data as string) as {
            type?: string
            call_id?: string
            ws_clients?: number
            online?: boolean
          }
          if (data?.type === 'presence') {
            setPresenceWsClients(typeof data.ws_clients === 'number' ? data.ws_clients : 0)
            setPresenceOnline(Boolean(data.online))
            return
          }
          if (data?.type === 'incoming' && data.call_id) {
            const cid = String(data.call_id)
            log('WS 推送来电 call_id=', cid)
            setPendingIncomingCallId(cid)
          }
        } catch {
          /* ignore */
        }
      }
    } catch (e: unknown) {
      setWsState('closed')
      setWsStatusText(t('webseat.wsCreateFailed'))
      log('WebSocket 错误:', e instanceof Error ? e.message : String(e))
    }
  }, [closeWsConnection, configured, httpBase, log, stopAcdHeartbeat, t, wsBase, wsToken])

  const reconnectWebSocket = useCallback(() => {
    if (isAuthenticated && configured) connectWebSocket()
  }, [configured, connectWebSocket, isAuthenticated])

  const goOnline = useCallback(async () => {
    if (!configured || !isAuthenticated) return
    stopAcdHeartbeat()
    try {
      connectWebSocket()
      await waitForWebSocketOpen(wsRef, 15_000)
      const operatorKey =
        (user?.email && String(user.email).trim()) || (user?.id != null ? String(user.id) : '')
      const tid = await ensureWebSeatAcdPoolRowOnline({
        displayLabel: webSeatAcdDisplayLabel,
        operatorKey,
      })
      void postWebSeatAcdHeartbeat(tid).catch(() => {})
      // @ts-ignore
        acdHeartbeatTimerRef.current = window.setInterval(() => {
        void postWebSeatAcdHeartbeat(tid).catch(() => {})
      }, WEBSEAT_ACD_HEARTBEAT_MS)
      window.dispatchEvent(new CustomEvent('soulnexus-acd-refresh'))
      log(t('webseat.logOnlineOk'))
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : t('common.failed')
      showAlert(msg, 'error')
      stopAcdHeartbeat()
      closeWsConnection()
      if (configured) {
        setWsState('closed')
        setWsStatusText(t('webseat.wsDisconnected'))
      }
    }
  }, [
    closeWsConnection,
    configured,
    connectWebSocket,
    isAuthenticated,
    log,
    stopAcdHeartbeat,
    t,
    user?.email,
    user?.id,
    webSeatAcdDisplayLabel,
  ])

  const goOffline = useCallback(async () => {
    stopAcdHeartbeat()
    try {
      await setWebSeatAcdPoolRowOffline()
      window.dispatchEvent(new CustomEvent('soulnexus-acd-refresh'))
    } catch (e: unknown) {
      showAlert(e instanceof Error ? e.message : t('common.failed'), 'error')
      return
    }
    closeWsConnection('user-offline')
    log(t('webseat.logOfflineOk'))
  }, [closeWsConnection, log, stopAcdHeartbeat, t])

  useEffect(() => {
    if (!configured) {
      closeWsConnection()
      setWsState('disabled')
      setWsStatusText(t('webseat.wsNotConfigured'))
      return
    }
    if (!isAuthenticated) {
      stopAcdHeartbeat()
      void (async () => {
        try {
          await setWebSeatAcdPoolRowOffline()
        } catch {
          /* ignore */
        }
        clearWebSeatAcdPoolAnchor()
      })()
      closeWsConnection()
      setWsState('idle')
      setWsStatusText(t('webseat.wsNotLoggedIn'))
      setPendingIncomingCallId(null)
      return
    }
    closeWsConnection()
    setWsState('closed')
    setWsStatusText(t('webseat.wsClickOnline'))
    return () => {
      stopAcdHeartbeat()
      closeWsConnection()
      if (configured) {
        setWsState('closed')
        setWsStatusText(t('webseat.wsDisconnected'))
      }
    }
  }, [closeWsConnection, configured, isAuthenticated, stopAcdHeartbeat, t])

  const stopRxAudioMonitor = useCallback(() => {
    if (rxMonitorStopRef.current) {
      try {
        rxMonitorStopRef.current()
      } catch {
        /* ignore */
      }
      rxMonitorStopRef.current = null
    }
    if (rxAudioCtxRef.current) {
      try {
        void rxAudioCtxRef.current.close()
      } catch {
        /* ignore */
      }
      rxAudioCtxRef.current = null
    }
  }, [])

  const startRxAudioMonitor = useCallback(
    (stream: MediaStream) => {
      stopRxAudioMonitor()
      const ctx = new AudioContext()
      rxAudioCtxRef.current = ctx
      const src = ctx.createMediaStreamSource(stream)
      const analyser = ctx.createAnalyser()
      analyser.fftSize = 512
      analyser.smoothingTimeConstant = 0.15
      const silentOut = ctx.createGain()
      silentOut.gain.value = 0
      src.connect(analyser)
      analyser.connect(silentOut)
      silentOut.connect(ctx.destination)
      const buf = new Float32Array(analyser.fftSize)
      const spec = new Uint8Array(analyser.frequencyBinCount)
      const nPreview = 16
      void ctx.resume()
      const timer = window.setInterval(() => {
        if (!rxAudioCtxRef.current || rxAudioCtxRef.current.state === 'closed') return
        analyser.getFloatTimeDomainData(buf)
        let sum = 0
        let peak = 0
        for (let i = 0; i < buf.length; i++) {
          const v = buf[i]!
          sum += v * v
          const a = Math.abs(v)
          if (a > peak) peak = a
        }
        const rms = Math.sqrt(sum / buf.length)
        analyser.getByteFrequencyData(spec)
        let specMax = 0
        for (let i = 0; i < spec.length; i++) {
          if (spec[i]! > specMax) specMax = spec[i]!
        }
        const preview: string[] = []
        for (let i = 0; i < nPreview; i++) preview.push(buf[i]!.toFixed(4))
        const ts = new Date().toISOString().slice(11, 23)
        let tag = ''
        if (rms < 0.00008 && specMax < 8) tag = ' [≈静音/无能量]'
        else if (rms >= 0.003 || specMax >= 40) tag = ' [有音]'
        const line = `[${ts}] rms=${rms.toFixed(5)} peak=${peak.toFixed(5)} specMax=${specMax} | f[0..${nPreview - 1}]= ${preview.join(' ')}${tag}`
        setRxLog((p) => appendLog(p, line, MAX_RX_LINES))
      }, 250)
      rxMonitorStopRef.current = () => window.clearInterval(timer)
    },
    [stopRxAudioMonitor]
  )

  const stopSessionWatch = useCallback(() => {
    if (statusPollTimerRef.current != null) {
      clearInterval(statusPollTimerRef.current)
      statusPollTimerRef.current = null
    }
  }, [])

  const endSession = useCallback(
    async (opts?: { operatorHangup?: boolean; reason?: string }) => {
      const operatorHangup = !!opts?.operatorHangup
      const reason = opts?.reason
      if (sessionEndingRef.current) return
      const hasLocal = !!(
        pcRef.current ||
        localStreamRef.current ||
        activeCallIdRef.current
      )
      if (!operatorHangup && !inWebSeatCallRef.current && !hasLocal) return

      sessionEndingRef.current = true
      try {
        stopSessionWatch()
        const cid = activeCallIdRef.current
        if (operatorHangup && cid) {
          try {
            const res = await fetch(`${httpBase}/webseat/v1/hangup`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ call_id: cid }),
            })
            if (res.ok) log('已通知网关挂断 SIP 客户')
            else if (res.status === 404) log('网关已无此 call（可能客户已先挂机）')
            else log('挂断信令 HTTP ' + String(res.status))
          } catch (e: unknown) {
            log('挂断信令失败（仍将关闭本机 WebRTC）:', e instanceof Error ? e.message : String(e))
          }
        } else if (reason) {
          log(reason)
        }
        inWebSeatCallRef.current = false
        activeCallIdRef.current = null
        setInCall(false)
        setHangupDisabled(true)
        stopRxAudioMonitor()
        const ra = remoteAudioRef.current
        if (ra) {
          try {
            ra.pause()
            ra.removeAttribute('src')
            ra.srcObject = null
          } catch {
            /* ignore */
          }
          remoteAudioRef.current = null
        }
        const pc = pcRef.current
        if (pc) {
          try {
            pc.getSenders().forEach((s) => {
              s.track?.stop()
            })
            pc.close()
          } catch {
            /* ignore */
          }
          pcRef.current = null
        }
        const ls = localStreamRef.current
        if (ls) {
          ls.getTracks().forEach((t) => t.stop())
          localStreamRef.current = null
        }
        log(operatorHangup ? '坐席侧已结束会话' : '页面已随会话结束（本机 WebRTC 已关闭）')
      } finally {
        sessionEndingRef.current = false
      }
    },
    [httpBase, log, stopRxAudioMonitor, stopSessionWatch]
  )

  const startSessionWatch = useCallback(() => {
    stopSessionWatch()
    statusPollTimerRef.current = setInterval(async () => {
      const cid = activeCallIdRef.current
      const pc = pcRef.current
      if (!cid || !pc) return
      try {
        const res = await fetch(`${httpBase}/webseat/v1/status/${encodeURIComponent(cid)}`)
        if (!res.ok) return
        const data = (await res.json()) as { pending_or_active?: boolean }
        if (data && data.pending_or_active === false) {
          await endSession({
            operatorHangup: false,
            reason: '网关已无 Web 坐席会话（客户已挂断或会话已清除），自动关闭连接',
          })
        }
      } catch {
        /* ignore */
      }
    }, 2000)
  }, [endSession, httpBase, stopSessionWatch])

  const joinWithCallId = useCallback(
    async (callId: string, clearLog: boolean) => {
      if (!httpBase) return
      const cid = String(callId || '').trim()
      if (!cid) return
      setPendingIncomingCallId(null)
      if (clearLog) {
        setSignalLog('')
        setRxLog('')
      }
      stopRxAudioMonitor()
      {
        const ra = remoteAudioRef.current
        if (ra) {
          try {
            ra.pause()
            ra.removeAttribute('src')
            ra.srcObject = null
          } catch {
            /* ignore */
          }
          remoteAudioRef.current = null
        }
      }
      log('开始… call_id=', cid)

      try {
        const localStream = await getUserMediaAudioOnly()
        localStreamRef.current = localStream
        log('getUserMedia OK')

        const iceServers: RTCIceServer[] = [{ urls: 'stun:stun.l.google.com:19302' }]
        const pc = new RTCPeerConnection({ iceServers })
        pcRef.current = pc

        pc.ontrack = (ev) => {
          if (ev.track.kind !== 'audio') return
          const stream = ev.streams[0] ?? new MediaStream([ev.track])
          const t = ev.track
          log(
            'ontrack: 远端音频 track，muted=',
            t.muted,
            'enabled=',
            t.enabled,
            'readyState=',
            t.readyState,
            'id=',
            t.id
          )
          t.addEventListener('unmute', () => log('远端音频 track: unmute（开始有媒体）'))
          t.addEventListener('mute', () => log('远端音频 track: mute'))

          let audio = remoteAudioRef.current
          if (!audio) {
            audio = new Audio()
            audio.setAttribute('playsinline', 'true')
            audio.autoplay = true
            remoteAudioRef.current = audio
          }
          audio.srcObject = stream
          void audio.play().catch((err) => {
            log(
              '远端音频 play() 失败（部分浏览器需用户手势；接听后一般会成功）:',
              err instanceof Error ? err.message : String(err)
            )
          })

          startRxAudioMonitor(stream)
        }

        localStream.getTracks().forEach((track) => pc.addTrack(track, localStream))

        const offer = await pc.createOffer()
        await pc.setLocalDescription(offer)
        log('setLocalDescription(offer)')

        await new Promise<void>((resolve) => {
          if (pc.iceGatheringState === 'complete') {
            resolve()
            return
          }
          const t = setTimeout(() => resolve(), 8000)
          pc.addEventListener(
            'icegatheringstatechange',
            () => {
              if (pc.iceGatheringState === 'complete') {
                clearTimeout(t)
                resolve()
              }
            },
            { once: true }
          )
        })

        const ld = pc.localDescription
        if (!ld) throw new Error('no localDescription')

        const body = {
          call_id: cid,
          sdp: ld.sdp,
          type: ld.type,
          candidates: [] as unknown[],
        }

        const url = `${httpBase}/webseat/v1/join`
        log('POST', url)
        const res = await fetch(url, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        })
        const text = await res.text()
        if (!res.ok) throw new Error(`HTTP ${res.status}: ${text}`)
        const ans = JSON.parse(text) as { sdp?: string; type?: RTCSdpType }
        if (!ans.sdp || !ans.type) throw new Error(`bad answer json: ${text}`)

        await pc.setRemoteDescription({ type: ans.type, sdp: ans.sdp })
        log('setRemoteDescription(answer), 应能听到客户侧声音')
        activeCallIdRef.current = cid
        inWebSeatCallRef.current = true
        setInCall(true)
        setHangupDisabled(false)

        pc.onconnectionstatechange = () => {
          const s = pc.connectionState
          log('pc.connectionState=', s)
          if (s === 'failed' || s === 'closed') {
            void endSession({
              operatorHangup: false,
              reason: s === 'failed' ? 'WebRTC 连接失败，已结束会话' : 'WebRTC 已关闭',
            })
          }
        }
        pc.oniceconnectionstatechange = () => log('pc.iceConnectionState=', pc.iceConnectionState)

        startSessionWatch()
      } catch (e: unknown) {
        log('错误:', e instanceof Error ? e.message : String(e))
        await endSession({ operatorHangup: false })
      }
    },
    [endSession, httpBase, log, startRxAudioMonitor, startSessionWatch, stopRxAudioMonitor]
  )

  const hangup = useCallback(() => {
    void endSession({ operatorHangup: true })
  }, [endSession])

  const answerIncoming = useCallback(() => {
    const cid = pendingIncomingCallId
    if (!cid) return
    void joinWithCallId(cid, false)
  }, [joinWithCallId, pendingIncomingCallId])

  const rejectIncoming = useCallback(async () => {
    const cid = pendingIncomingCallId
    if (!cid) return
    setPendingIncomingCallId(null)
    try {
      const res = await fetch(`${httpBase}/webseat/v1/reject`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ call_id: cid }),
      })
      if (res.ok) log('已拒接，网关已拆线（对客户 BYE）')
      else if (res.status === 404) log('拒接：网关已无此会话')
      else log('拒接 HTTP ' + String(res.status))
    } catch (e: unknown) {
      log('拒接请求失败:', e instanceof Error ? e.message : String(e))
    }
  }, [httpBase, log, pendingIncomingCallId])

  const ctxValue: WebSeatContextValue = useMemo(
    () => ({
      configured,
      wsState,
      wsStatusText,
      presenceWsClients,
      presenceOnline,
      signalLog,
      rxLog,
      inCall,
      hangupDisabled,
      pendingIncomingCallId,
      hangup,
      reconnectWebSocket,
      goOnline,
      goOffline,
    }),
    [
      configured,
      goOffline,
      goOnline,
      hangup,
      hangupDisabled,
      inCall,
      pendingIncomingCallId,
      presenceOnline,
      presenceWsClients,
      reconnectWebSocket,
      rxLog,
      signalLog,
      wsState,
      wsStatusText,
    ]
  )
  const showOverlay = location.pathname.startsWith('/contact-center')

  return (
    <WebSeatContext.Provider value={ctxValue}>
      {children}
      {isAuthenticated && configured && showOverlay && (
        <div className="pointer-events-none fixed right-2 z-[200] flex flex-col items-end gap-3 top-[66px] lg:top-2">
          {pendingIncomingCallId && (
            <WebSeatIncomingCallCard
              className="pointer-events-auto"
              callId={pendingIncomingCallId}
              onAnswer={() => answerIncoming()}
              onReject={() => void rejectIncoming()}
            />
          )}
          <WebSeatWsIndicator
            className="pointer-events-auto"
            wsState={wsState}
            wsStatusText={wsStatusText}
            presenceWsClients={presenceWsClients}
            onGoOnline={() => void goOnline()}
            onGoOffline={() => void goOffline()}
            onReconnect={reconnectWebSocket}
          />
        </div>
      )}
    </WebSeatContext.Provider>
  )
}
