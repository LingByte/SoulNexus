import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Spin, Tag, Typography } from '@arco-design/web-react'
import { IconPause, IconPlayArrow, IconUser } from '@arco-design/web-react/icon'
import { listVoices, previewVoice, type VoiceOption } from '@/api/voices'
import { listVoiceClones, previewVoiceClone, type VoiceCloneProfile } from '@/api/voiceClone'
import { Input } from '@/components/ui'
import { defaultVoiceProvider, voiceProviderLabel } from '@/constants/tenantAiConfigRules'
import { playVoicePreviewPcm, playVoicePreviewUrl, stopVoicePreview } from '@/utils/voicePreview'
import { showAlert } from '@/utils/notification'
import { cn } from '@/utils/cn'

const PREVIEW_TEXT = '您好，欢迎致电，我是您的智能客服助手。'

type PickerVoice = VoiceOption & {
  cloneProfileId?: number
  cloneProvider?: string
}

function voiceMetaLine(v: VoiceOption) {
  const parts: string[] = []
  if (v.locale) parts.push(v.locale)
  if (v.gender === 'female') parts.push('女声')
  else if (v.gender === 'male') parts.push('男声')
  if (v.category) parts.push(v.category)
  return parts.join(' · ')
}

function cloneVoiceID(p: VoiceCloneProfile): string {
  return (p.assetId || p.speakerId || String(p.id)).trim()
}

function VoiceAvatar({ voice }: { voice: VoiceOption }) {
  const isFemale = voice.gender === 'female'
  const isClone = voice.category === '我的克隆音色'
  return (
    <div
      className={cn(
        'flex h-14 w-14 shrink-0 items-center justify-center rounded-xl border',
        isClone
          ? 'border-violet-200 bg-violet-50 text-violet-500'
          : isFemale
            ? 'border-pink-200 bg-pink-50 text-pink-500'
            : 'border-sky-200 bg-sky-50 text-sky-500',
      )}
    >
      <IconUser style={{ fontSize: 28 }} />
    </div>
  )
}

export type VoicePickerGridProps = {
  voiceMode: 'pipeline' | 'realtime'
  provider: string
  value: string
  setValue: (v: string) => void
  tenantId?: string
  /** When true, prepend tenant-trained clone voices (assistant editor). */
  includeCloneVoices?: boolean
  onCloneVoiceSelect?: (profile: VoiceCloneProfile | null, voiceId: string) => void
}

export default function VoicePickerGrid({
  voiceMode,
  provider,
  value,
  setValue,
  tenantId,
  includeCloneVoices = false,
  onCloneVoiceSelect,
}: VoicePickerGridProps) {
  const mode = voiceMode === 'realtime' ? 'realtime' : 'tts'
  const catalogProvider = provider.trim() || defaultVoiceProvider(mode)
  const providerLabel = voiceProviderLabel(mode, catalogProvider)
  const isCloneProvider = catalogProvider === 'volcengine_clone' || catalogProvider === 'xunfei_clone'

  const [options, setOptions] = useState<VoiceOption[]>([])
  const [cloneVoices, setCloneVoices] = useState<VoiceCloneProfile[]>([])
  const [docUrl, setDocUrl] = useState('')
  const [loading, setLoading] = useState(false)
  const [catalogMissing, setCatalogMissing] = useState(false)
  const [playingId, setPlayingId] = useState<string | null>(null)
  const [previewingId, setPreviewingId] = useState<string | null>(null)
  const mountedRef = useRef(true)

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
      stopVoicePreview()
    }
  }, [])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setCatalogMissing(false)
    void listVoices(catalogProvider, mode)
      .then((res) => {
        if (cancelled) return
        if (res.code === 200 && res.data) {
          setOptions(res.data.voices ?? [])
          setDocUrl(res.data.docUrl ?? '')
          setCatalogMissing((res.data.voices ?? []).length === 0)
        } else {
          setOptions([])
          setDocUrl('')
          setCatalogMissing(true)
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [catalogProvider, mode])

  useEffect(() => {
    const shouldLoad = includeCloneVoices || isCloneProvider
    if (!shouldLoad) {
      setCloneVoices([])
      return
    }
    let cancelled = false
    void listVoiceClones('success')
      .then((res) => {
        if (!cancelled && res.code === 200 && res.data) setCloneVoices(res.data)
      })
      .catch(() => {
        if (!cancelled) setCloneVoices([])
      })
    return () => {
      cancelled = true
    }
  }, [includeCloneVoices, isCloneProvider])

  const cloneProfileByVoiceId = useMemo(() => {
    const map = new Map<string, VoiceCloneProfile>()
    for (const c of cloneVoices) {
      const id = cloneVoiceID(c)
      if (id) map.set(id, c)
    }
    return map
  }, [cloneVoices])

  const cloneOptions: PickerVoice[] = cloneVoices.map((c) => ({
    id: cloneVoiceID(c),
    label: c.name,
    category: '我的克隆音色',
    cloneProfileId: c.id,
    cloneProvider: c.provider,
  }))

  const catalogOptions: PickerVoice[] =
    isCloneProvider && !includeCloneVoices ? [] : options

  const mergedOptions: PickerVoice[] = [...cloneOptions, ...catalogOptions]

  const [clonePreviewUrls, setClonePreviewUrls] = useState<Record<string, string>>({})

  const handleSelect = (voice: PickerVoice) => {
    const isClone = !!voice.cloneProfileId
    if (includeCloneVoices && onCloneVoiceSelect) {
      const profile = isClone ? cloneProfileByVoiceId.get(voice.id) ?? null : null
      onCloneVoiceSelect(profile, voice.id)
    }
    if (!isClone) {
      setValue(voice.id)
    }
  }

  const handlePreview = useCallback(
    async (voice: PickerVoice, e: React.MouseEvent) => {
      e.stopPropagation()
      if (previewingId === voice.id) return
      if (playingId === voice.id) {
        stopVoicePreview()
        setPlayingId(null)
        return
      }
      stopVoicePreview()
      setPreviewingId(voice.id)
      try {
        if (voice.cloneProfileId) {
          const cachedUrl = clonePreviewUrls[voice.id]?.trim() || voice.previewUrl?.trim()
          if (cachedUrl) {
            setPlayingId(voice.id)
            await playVoicePreviewUrl(cachedUrl)
            if (mountedRef.current) setPlayingId(null)
            return
          }
          const res = await previewVoiceClone(voice.cloneProfileId, PREVIEW_TEXT)
          if (!mountedRef.current) return
          if (res.code !== 200 || !res.data) {
            showAlert(res.msg || '试听失败', 'error')
            return
          }
          setPlayingId(voice.id)
          if (res.data.audioUrl) {
            setClonePreviewUrls((prev) => ({ ...prev, [voice.id]: res.data!.audioUrl! }))
            await playVoicePreviewUrl(res.data.audioUrl)
          } else if (res.data.pcmBase64) {
            await playVoicePreviewPcm(res.data.pcmBase64, res.data.sampleRate ?? 16000)
          } else {
            showAlert('试听失败', 'error')
            setPlayingId(null)
            return
          }
          if (mountedRef.current) setPlayingId(null)
          return
        }
        const cachedUrl = voice.previewUrl?.trim()
        if (cachedUrl) {
          setPlayingId(voice.id)
          await playVoicePreviewUrl(cachedUrl)
          if (mountedRef.current) setPlayingId(null)
          return
        }
        const res = await previewVoice({
          provider: catalogProvider,
          mode,
          voiceId: voice.id,
          text: PREVIEW_TEXT,
          tenantId,
        })
        if (!mountedRef.current) return
        if (res.code !== 200 || !res.data) {
          showAlert(res.msg || '试听失败', 'error')
          return
        }
        setPlayingId(voice.id)
        if (res.data.audioUrl) {
          await playVoicePreviewUrl(res.data.audioUrl)
          if (!res.data.cached) {
            setOptions((prev) =>
              prev.map((v) => (v.id === voice.id ? { ...v, previewUrl: res.data!.audioUrl } : v)),
            )
          }
        } else if (res.data.pcmBase64) {
          await playVoicePreviewPcm(res.data.pcmBase64, res.data.sampleRate ?? 16000)
        } else {
          showAlert('试听失败', 'error')
          setPlayingId(null)
          return
        }
        if (mountedRef.current) setPlayingId(null)
      } catch (err: unknown) {
        if (mountedRef.current) {
          showAlert((err as { msg?: string })?.msg || '试听失败', 'error')
          setPlayingId(null)
        }
      } finally {
        if (mountedRef.current) setPreviewingId(null)
      }
    },
    [catalogProvider, mode, tenantId, playingId, previewingId, clonePreviewUrls],
  )

  if (catalogMissing && !includeCloneVoices && !isCloneProvider && mergedOptions.length === 0) {
    return (
      <div className="space-y-2">
        <Typography.Text type="secondary" style={{ fontSize: 13 }}>
          {voiceMode === 'realtime' ? 'Realtime 厂商' : 'TTS 厂商'}：<strong>{providerLabel}</strong>
        </Typography.Text>
        <Input
          placeholder="音色 ID（该厂商暂无预置列表，请手动填写）"
          value={value}
          onChange={setValue}
          style={{ maxWidth: 480 }}
        />
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          可在租户 AI 配置中切换厂商后刷新本页。
        </Typography.Text>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <Typography.Text type="secondary" style={{ fontSize: 13 }}>
        {voiceMode === 'realtime' ? 'Realtime 厂商' : 'TTS 厂商'}：<strong>{providerLabel}</strong>
        {docUrl ? (
          <>
            {' '}
            ·{' '}
            <a href={docUrl} target="_blank" rel="noreferrer">
              官方文档
            </a>
          </>
        ) : null}
      </Typography.Text>

      {includeCloneVoices && cloneOptions.length > 0 ? (
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          选择「我的克隆音色」时，会话将自动使用 Pipeline 及对应克隆 TTS；也可继续选择下方厂商预置音色。
        </Typography.Text>
      ) : null}

      {loading ? (
        <div className="flex items-center gap-2 py-8 text-sm text-muted-foreground">
          <Spin size={16} /> 加载音色列表…
        </div>
      ) : mergedOptions.length === 0 ? (
        <Typography.Text type="secondary">暂无可用音色。</Typography.Text>
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
          {mergedOptions.map((voice) => {
            const selected = value === voice.id
            const meta = voiceMetaLine(voice)
            const isPlaying = playingId === voice.id
            const isLoading = previewingId === voice.id
            return (
              <button
                key={`${voice.category || 'v'}-${voice.id}`}
                type="button"
                onClick={() => handleSelect(voice)}
                className={cn(
                  'group flex w-full items-start gap-3 rounded-xl border bg-card p-3 text-left transition-colors',
                  selected
                    ? 'border-primary ring-1 ring-primary/30'
                    : 'border-border hover:border-primary/40 hover:bg-muted/30',
                )}
              >
                <VoiceAvatar voice={voice} />
                <div className="min-w-0 flex-1 space-y-1.5">
                  <div className="truncate text-sm font-semibold text-foreground">
                    {voice.label || voice.name || voice.id}
                  </div>
                  {meta ? (
                    <div className="truncate text-xs text-muted-foreground">{meta}</div>
                  ) : null}
                  {voice.category ? (
                    <Tag
                      size="small"
                      color={voice.category === '我的克隆音色' ? 'purple' : 'arcoblue'}
                      style={{ marginTop: 2, maxWidth: '100%' }}
                      title={voice.category}
                    >
                      <span className="block max-w-full truncate">{voice.category}</span>
                    </Tag>
                  ) : null}
                  <div className="pt-1">
                    <span
                      role="button"
                      tabIndex={0}
                      className={cn(
                        'inline-flex h-8 w-8 items-center justify-center rounded-full border transition-colors',
                        isPlaying
                          ? 'border-primary bg-primary text-primary-foreground'
                          : 'border-border bg-background text-muted-foreground hover:border-primary hover:text-primary',
                      )}
                      onClick={(e) => void handlePreview(voice, e)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          void handlePreview(voice, e as unknown as React.MouseEvent)
                        }
                      }}
                      aria-label={isPlaying ? '停止试听' : '试听音色'}
                    >
                      {isLoading ? (
                        <Spin size={14} />
                      ) : isPlaying ? (
                        <IconPause style={{ fontSize: 14 }} />
                      ) : (
                        <IconPlayArrow style={{ fontSize: 14 }} />
                      )}
                    </span>
                  </div>
                </div>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}
