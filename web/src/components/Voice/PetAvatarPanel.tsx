import { useCallback, useEffect, useRef, useState } from 'react'
import { Mic, PhoneOff } from 'lucide-react'
import { getApiBaseURL } from '@/config/apiConfig'
import { cn } from '@/utils/cn'

interface PetAvatarPanelProps {
  templateJsSourceId: string
  className?: string
}

declare global {
  interface Window {
    __PET_SPRITE__?: { stop?: () => void }
    __SOUL_PET_VOICE__?: {
      toggleCall?: () => void | Promise<void>
      isCalling?: () => boolean
      isConnecting?: () => boolean
    }
    __SOUL_PET_VOICE_HOST__?: {
      toggleCall?: () => void | Promise<void>
      isCalling?: () => boolean
      isConnecting?: () => boolean
    }
  }
}

function destroySpritePet(): void {
  try {
    window.__PET_SPRITE__?.stop?.()
  } catch {
    /* ignore */
  }
  window.__PET_SPRITE__ = undefined
}

/** Loads bound sprite pet via public embed loader.js */
export default function PetAvatarPanel({ templateJsSourceId, className = '' }: PetAvatarPanelProps) {
  const mountRef = useRef<HTMLDivElement>(null)
  const loadGenRef = useRef(0)
  const [voicePhase, setVoicePhase] = useState<'idle' | 'connecting' | 'calling'>('idle')

  const syncVoicePhase = useCallback(() => {
    const host = window.__SOUL_PET_VOICE_HOST__
    const bridge = window.__SOUL_PET_VOICE__
    if (host?.isConnecting?.()) {
      setVoicePhase('connecting')
      return
    }
    if (host?.isCalling?.()) {
      setVoicePhase('calling')
      return
    }
    if (bridge?.isConnecting?.()) {
      setVoicePhase('connecting')
      return
    }
    if (bridge?.isCalling?.()) {
      setVoicePhase('calling')
      return
    }
    setVoicePhase('idle')
  }, [])

  useEffect(() => {
    const timer = window.setInterval(syncVoicePhase, 400)
    return () => window.clearInterval(timer)
  }, [syncVoicePhase])

  const handleVoiceClick = () => {
    const host = window.__SOUL_PET_VOICE_HOST__
    if (host?.toggleCall) {
      void host.toggleCall()
      window.setTimeout(syncVoicePhase, 50)
      return
    }
    window.__SOUL_PET_VOICE__?.toggleCall?.()
    window.setTimeout(syncVoicePhase, 50)
  }

  useEffect(() => {
    const id = templateJsSourceId?.trim()
    if (!id || !mountRef.current) return

    const host = mountRef.current
    const loadGen = ++loadGenRef.current
    let cancelled = false
    let loadTimer = 0

    destroySpritePet()
    host.innerHTML = ''

    const mount = document.createElement('div')
    mount.id = 'app'
    mount.className = 'absolute inset-0 w-full h-full overflow-hidden soul-pet-mount'
    host.appendChild(mount)

    loadTimer = window.setTimeout(() => {
      if (cancelled || loadGen !== loadGenRef.current) return

      const script = document.createElement('script')
      script.async = true
      script.src = `${getApiBaseURL()}/js-templates/embed/${encodeURIComponent(id)}/loader.js?_=${loadGen}`
      script.onerror = () => {
        if (cancelled) return
        mount.innerHTML =
          '<p class="text-xs text-red-300 p-4 text-center whitespace-pre-wrap">桌宠加载失败\n请确认项目已保存且 manifest 正确</p>'
      }
      script.onload = () => {
        window.setTimeout(() => {
          if (cancelled || loadGen !== loadGenRef.current) return
          if (!window.__PET_SPRITE__) {
            mount.innerHTML =
              '<p class="text-xs text-amber-200 p-4 text-center whitespace-pre-wrap">脚本已加载但桌宠未就绪\n请确认 pet.js 与 manifest.json 完整</p>'
          }
        }, 8000)
      }
      host.appendChild(script)
    }, 120)

    return () => {
      cancelled = true
      window.clearTimeout(loadTimer)
      destroySpritePet()
      host.innerHTML = ''
    }
  }, [templateJsSourceId])

  const inCall = voicePhase === 'calling' || voicePhase === 'connecting'

  return (
    <div
      className={cn(
        'relative overflow-hidden rounded-xl border border-gray-200/80 dark:border-neutral-700 bg-gradient-to-b from-indigo-950/40 to-slate-900/60',
        className,
      )}
      aria-label="桌宠"
    >
      <div ref={mountRef} className="absolute inset-0" />
      <button
        type="button"
        onClick={handleVoiceClick}
        title={inCall ? '结束对话' : '开始对话'}
        aria-label={inCall ? '结束对话' : '开始对话'}
        className={cn(
          'absolute bottom-3 right-3 z-30 flex flex-col items-center gap-1 rounded-full border-2 border-white/40 p-0 shadow-lg transition-transform hover:scale-105 active:scale-95',
          inCall
            ? 'bg-gradient-to-br from-red-500 to-red-600 text-white'
            : 'bg-gradient-to-br from-indigo-500 to-violet-600 text-white',
          voicePhase === 'connecting' && 'animate-pulse',
        )}
      >
        <span className="flex h-11 w-11 items-center justify-center rounded-full bg-white/15">
          {inCall ? <PhoneOff className="h-5 w-5" /> : <Mic className="h-5 w-5" />}
        </span>
        <span className="pb-1.5 text-[10px] font-semibold leading-none">
          {voicePhase === 'connecting' ? '连接中' : inCall ? '结束' : '聊天'}
        </span>
      </button>
    </div>
  )
}
