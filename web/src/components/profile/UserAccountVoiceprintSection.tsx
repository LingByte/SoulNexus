import { useCallback, useEffect, useState } from 'react'
import { Button } from '@/components/ui'
import { deleteMyVoiceprint, enrollMyVoiceprint, fetchMyVoiceprint } from '@/api/me'
import VoiceprintAudioRecorder from '@/components/voice/VoiceprintAudioRecorder'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useTranslation } from '@/i18n'

type Props = {
  enabled: boolean
}

export default function UserAccountVoiceprintSection({ enabled }: Props) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(true)
  const [enrolled, setEnrolled] = useState(false)
  const [profileName, setProfileName] = useState('')
  const [audioFile, setAudioFile] = useState<File | null>(null)
  const [audioUrl, setAudioUrl] = useState('')
  const [saving, setSaving] = useState(false)

  const reload = useCallback(async () => {
    if (!enabled) {
      setLoading(false)
      return
    }
    setLoading(true)
    try {
      const res = await fetchMyVoiceprint()
      if (res.code === 200 && res.data) {
        setEnrolled(Boolean(res.data.enrolled))
        setProfileName(res.data.profile?.name || '')
      }
    } finally {
      setLoading(false)
    }
  }, [enabled])

  useEffect(() => {
    void reload()
  }, [reload])

  if (!enabled) return null

  const handleEnroll = async () => {
    const url = audioUrl.trim()
    if (!audioFile && !url) {
      showAlert(t('profile.voiceprintAudioRequired'), 'warning')
      return
    }
    setSaving(true)
    try {
      const res = await enrollMyVoiceprint({
        audio: audioFile || undefined,
        audioUrl: url || undefined,
        name: t('profile.voiceprintDefaultName'),
      })
      if (res.code !== 200) {
        showAlert(res.msg || t('profile.voiceprintEnrollFailed'), 'error')
        return
      }
      showAlert(t('profile.voiceprintEnrollSuccess'), 'success')
      setAudioFile(null)
      setAudioUrl('')
      await reload()
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('profile.voiceprintEnrollFailed')), 'error')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    setSaving(true)
    try {
      const res = await deleteMyVoiceprint()
      if (res.code !== 200) {
        showAlert(res.msg || t('profile.voiceprintDeleteFailed'), 'error')
        return
      }
      showAlert(t('profile.voiceprintDeleteSuccess'), 'success')
      await reload()
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('profile.voiceprintDeleteFailed')), 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="rounded-xl border border-border bg-card px-5 mb-4">
      <div className="pt-4 text-xs font-medium uppercase tracking-wide text-neutral-400">
        {t('profile.voiceprintSectionTitle')}
      </div>
      <div className="py-5 border-b border-neutral-100 last:border-b-0">
        <div className="font-medium text-neutral-900">{t('profile.voiceprintTitle')}</div>
        <div className="mt-0.5 text-sm text-neutral-500">{t('profile.voiceprintDesc')}</div>
        {loading ? (
          <p className="mt-3 text-sm text-neutral-400">{t('common.loading')}</p>
        ) : enrolled ? (
          <div className="mt-3 space-y-3">
            <p className="text-sm text-green-700">{t('profile.voiceprintEnrolled', { name: profileName || '-' })}</p>
            <Button type="outline" status="warning" loading={saving} onClick={() => void handleDelete()}>
              {t('profile.voiceprintReEnroll')}
            </Button>
          </div>
        ) : (
          <div className="mt-3 space-y-3">
            <VoiceprintAudioRecorder
              mode="record-or-url"
              disabled={saving}
              onAudioChange={setAudioFile}
              onUrlChange={setAudioUrl}
            />
            <Button type="primary" loading={saving} onClick={() => void handleEnroll()}>
              {t('profile.voiceprintEnrollBtn')}
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}
