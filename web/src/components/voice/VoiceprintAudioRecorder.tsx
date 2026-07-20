import { useEffect, useRef, useState } from 'react'
import { Mic, Square } from 'lucide-react'
import { Button, Input } from '@/components/ui'
import { WavRecorder } from '@/utils/wavRecorder'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'

type Props = {
  /** record-only: login verify; record-or-url: account security enroll */
  mode?: 'record-only' | 'record-or-url'
  onAudioChange: (file: File | null) => void
  onUrlChange?: (url: string) => void
  disabled?: boolean
}

export default function VoiceprintAudioRecorder({
  mode = 'record-only',
  onAudioChange,
  onUrlChange,
  disabled = false,
}: Props) {
  const { t } = useTranslation()
  const recorderRef = useRef<WavRecorder | null>(null)
  const [recording, setRecording] = useState(false)
  const [audioFile, setAudioFile] = useState<File | null>(null)
  const [audioUrl, setAudioUrl] = useState('')

  useEffect(() => {
    return () => {
      recorderRef.current?.cancel()
      recorderRef.current = null
    }
  }, [])

  const syncFile = (file: File | null) => {
    setAudioFile(file)
    onAudioChange(file)
  }

  const startRecording = async () => {
    if (disabled) return
    try {
      recorderRef.current?.cancel()
      const rec = new WavRecorder()
      recorderRef.current = rec
      await rec.start()
      setRecording(true)
      setAudioUrl('')
      syncFile(null)
    } catch {
      showAlert(t('profile.voiceprintRecordFailed'), 'error')
      setRecording(false)
    }
  }

  const stopRecording = () => {
    const rec = recorderRef.current
    if (!rec) return
    const file = rec.stop()
    recorderRef.current = null
    setRecording(false)
    if (!file) {
      showAlert(t('profile.voiceprintRecordTooShort'), 'warning')
      return
    }
    syncFile(file)
  }

  const clearRecording = () => {
    recorderRef.current?.cancel()
    recorderRef.current = null
    setRecording(false)
    syncFile(null)
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        {recording ? (
          <Button type="primary" status="danger" icon={<Square size={16} />} onClick={stopRecording}>
            {t('profile.voiceprintStopRecord')}
          </Button>
        ) : (
          <Button
            type="outline"
            icon={<Mic size={16} />}
            disabled={disabled}
            onClick={() => void startRecording()}
          >
            {audioFile ? t('profile.voiceprintReRecord') : t('profile.voiceprintStartRecord')}
          </Button>
        )}
        {audioFile && !recording ? (
          <Button type="text" disabled={disabled} onClick={clearRecording}>
            {t('profile.voiceprintClearRecord')}
          </Button>
        ) : null}
        {recording ? (
          <span className="text-sm text-red-600">{t('profile.voiceprintRecording')}</span>
        ) : audioFile ? (
          <span className="text-sm text-green-700">{t('profile.voiceprintRecorded', { name: audioFile.name })}</span>
        ) : null}
      </div>
      {mode === 'record-or-url' ? (
        <Input
          placeholder={t('profile.voiceprintAudioUrlPlaceholder')}
          value={audioUrl}
          disabled={disabled || recording}
          onChange={(v) => {
            setAudioUrl(v)
            onUrlChange?.(v)
            if (v.trim()) syncFile(null)
          }}
        />
      ) : null}
    </div>
  )
}