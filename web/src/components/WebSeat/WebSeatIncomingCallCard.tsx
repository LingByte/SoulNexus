import { useEffect, useRef, useState } from 'react'
import { Phone, PhoneOff, Volume2 } from 'lucide-react'
import { cn } from '@/utils/cn.ts'

const RING_SRC = `${import.meta.env.BASE_URL}ringing.wav`

interface WebSeatIncomingCallCardProps {
  callId: string
  onAnswer: () => void
  onReject: () => void
  className?: string
}

export function WebSeatIncomingCallCard({
  callId,
  onAnswer,
  onReject,
  className,
}: WebSeatIncomingCallCardProps) {
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const [ringNeedsGesture, setRingNeedsGesture] = useState(false)

  useEffect(() => {
    let cancelled = false
    const a = new Audio(RING_SRC)
    a.preload = 'auto'
    a.volume = 0.55
    audioRef.current = a

    const tryPlay = () => {
      if (cancelled) return
      a.loop = true
      void a
        .play()
        .then(() => {
          if (!cancelled) setRingNeedsGesture(false)
        })
        .catch(() => {
          if (!cancelled) setRingNeedsGesture(true)
        })
    }

    const onReady = () => {
      if (cancelled) return
      tryPlay()
    }

    if (a.readyState >= HTMLMediaElement.HAVE_ENOUGH_DATA) {
      tryPlay()
    } else {
      a.addEventListener('canplaythrough', onReady, { once: true })
      a.addEventListener('error', () => {
        if (!cancelled) setRingNeedsGesture(true)
      }, { once: true })
      a.load()
    }

    return () => {
      cancelled = true
      a.removeEventListener('canplaythrough', onReady)
      a.pause()
      a.removeAttribute('src')
      a.load()
      audioRef.current = null
    }
  }, [callId])

  const unlockRing = () => {
    const a = audioRef.current
    if (!a) return
    a.currentTime = 0
    void a.play().then(() => setRingNeedsGesture(false)).catch(() => {})
  }

  return (
    <div
      className={cn(
        'w-[min(calc(100vw-2rem),20rem)] overflow-hidden rounded-2xl border border-border bg-card shadow-2xl ring-1 ring-black/5 dark:ring-white/10',
        'animate-in fade-in slide-in-from-top-2 duration-200',
        className
      )}
      role="alertdialog"
      aria-labelledby="webseat-incoming-title"
    >
      <div className="flex items-center gap-3 border-b border-border/80 bg-muted/40 px-4 py-3">
        <div className="relative flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-primary/15 text-primary">
          <span className="absolute inset-0 rounded-full bg-primary/20 animate-ping opacity-75 [animation-duration:1.4s]" />
          <Phone className="relative h-6 w-6" strokeWidth={2} />
        </div>
        <div className="min-w-0 flex-1">
          <p id="webseat-incoming-title" className="text-sm font-semibold text-foreground">
            Web 坐席来电
          </p>
          <p className="mt-0.5 truncate font-mono text-xs text-muted-foreground" title={callId}>
            {callId}
          </p>
          {ringNeedsGesture && (
            <button
              type="button"
              className="mt-2 inline-flex items-center gap-1 rounded-md border border-border bg-background px-2 py-1 text-xs text-foreground hover:bg-muted"
              onClick={() => unlockRing()}
            >
              <Volume2 className="h-3.5 w-3.5" />
              点击开启铃声
            </button>
          )}
        </div>
      </div>
      <div className="flex gap-2 p-3">
        <button
          type="button"
          className="flex flex-1 items-center justify-center gap-1.5 rounded-xl bg-emerald-600 px-3 py-2.5 text-sm font-medium text-white hover:bg-emerald-700"
          onClick={() => {
            audioRef.current?.pause()
            onAnswer()
          }}
        >
          <Phone className="h-4 w-4" />
          接听
        </button>
        <button
          type="button"
          className="flex flex-1 items-center justify-center gap-1.5 rounded-xl border border-border bg-background px-3 py-2.5 text-sm font-medium text-foreground hover:bg-muted"
          onClick={() => {
            audioRef.current?.pause()
            onReject()
          }}
        >
          <PhoneOff className="h-4 w-4" />
          拒接
        </button>
      </div>
    </div>
  )
}
