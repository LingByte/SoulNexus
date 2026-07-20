import { useCallback, useEffect, useState } from 'react'
import { Alert, Spin } from '@arco-design/web-react'
import { ArrowLeft } from 'lucide-react'
import { useNavigate, useParams } from 'react-router-dom'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button, Empty } from '@/components/ui'
import NluModelEditorForm from '@/components/nlu/NluModelEditorForm'
import { useTranslation } from '@/i18n'
import {
  fetchNluConfig,
  getNluModel,
  parseNluModel,
  trainNluModel,
  updateNluModel,
  type TenantNluModel,
  type TenantNluSpec,
} from '@/api/nluModels'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

export default function NluModelDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id = '' } = useParams<{ id: string }>()

  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [saveLoading, setSaveLoading] = useState(false)
  const [trainLoading, setTrainLoading] = useState(false)
  const [parseLoading, setParseLoading] = useState(false)
  const [deployEnabled, setDeployEnabled] = useState(false)
  const [cfgOk, setCfgOk] = useState(false)
  const [nluMode, setNluMode] = useState('embedding')
  const [maxIntents, setMaxIntents] = useState(64)
  const [model, setModel] = useState<TenantNluModel | null>(null)
  const [spec, setSpec] = useState<TenantNluSpec | null>(null)
  const [parseText, setParseText] = useState('查订单')
  const [parseResult, setParseResult] = useState('')
  const [parseLatencyMs, setParseLatencyMs] = useState<number | null>(null)

  const load = useCallback(async () => {
    const sid = String(id).trim()
    if (!sid) {
      setLoadError(t('nluLab.loadFailed'))
      setLoading(false)
      return
    }
    setLoading(true)
    setLoadError('')
    try {
      const c = await fetchNluConfig()
      setDeployEnabled(Boolean(c.deployEnabled))
      setCfgOk(Boolean(c.deployEnabled && c.platformReady))
      setNluMode(c.mode || 'embedding')
      setMaxIntents(c.maxIntents ?? 64)
      const row = await getNluModel(sid)
      setModel(row)
      setSpec(JSON.parse(JSON.stringify(row.spec)) as TenantNluSpec)
    } catch (e: unknown) {
      setLoadError(extractApiErrorMessage(e) || t('nluLab.loadFailed'))
      setModel(null)
      setSpec(null)
    } finally {
      setLoading(false)
    }
  }, [id, t])

  useEffect(() => {
    void load()
  }, [load])

  const handleSave = async () => {
    if (!model || !spec) return
    setSaveLoading(true)
    try {
      const updated = await updateNluModel(model.id, { spec })
      setModel(updated)
      setSpec(JSON.parse(JSON.stringify(updated.spec)) as TenantNluSpec)
      showAlert(t('common.saveSuccess'), 'success')
    } catch (e: unknown) {
      showAlert(t('common.saveFailed'), 'error', extractApiErrorMessage(e))
    } finally {
      setSaveLoading(false)
    }
  }

  const handleTrain = async () => {
    if (!model || !spec) return
    setTrainLoading(true)
    try {
      await updateNluModel(model.id, { spec })
      const res = await trainNluModel(model.id)
      showAlert(res.message || t('nluLab.trainSuccess'), 'success')
      await load()
    } catch (e: unknown) {
      showAlert(t('nluLab.trainFailed'), 'error', extractApiErrorMessage(e))
    } finally {
      setTrainLoading(false)
    }
  }

  const handleParse = async () => {
    if (!model) return
    setParseLoading(true)
    setParseResult('')
    setParseLatencyMs(null)
    try {
      const res = await parseNluModel(model.id, parseText.trim())
      setParseResult(JSON.stringify(res, null, 2))
      setParseLatencyMs(typeof res.latencyMs === 'number' ? res.latencyMs : null)
    } catch (e: unknown) {
      showAlert(t('nluLab.parseFailed'), 'error', extractApiErrorMessage(e))
    } finally {
      setParseLoading(false)
    }
  }

  const backBtn = (
    <Button variant="outline" leftIcon={<ArrowLeft className="h-4 w-4" />} onClick={() => navigate('/nlu-models')}>
      {t('common.back')}
    </Button>
  )

  if (loading) {
    return (
      <BaseLayout title={t('nluLab.title')} actions={backBtn}>
        <div className="flex justify-center py-24">
          <Spin />
        </div>
      </BaseLayout>
    )
  }

  if (loadError || !model || !spec) {
    return (
      <BaseLayout title={t('nluLab.title')} actions={backBtn}>
        <Empty preset="404" description={loadError || t('nluLab.loadFailed')}>
          <Button variant="outline" onClick={() => void load()}>
            {t('common.refresh')}
          </Button>
        </Empty>
      </BaseLayout>
    )
  }

  return (
    <BaseLayout title={model.name} actions={backBtn}>
      <div className="flex flex-col gap-4 lg:h-[calc(100vh-96px)]">
        {!deployEnabled ? (
          <Alert type="warning" showIcon content={t('nluLab.envDisabledHint')} className="shrink-0" />
        ) : !cfgOk ? (
          <Alert type="warning" showIcon content={t('nluLab.platformNotReady')} className="shrink-0" />
        ) : null}

        <div className="min-h-0 flex-1">
          <NluModelEditorForm
            model={model}
            spec={spec}
            nluMode={nluMode}
            maxIntents={maxIntents}
            parseText={parseText}
            parseResult={parseResult}
            parseLatencyMs={parseLatencyMs}
            trainLoading={trainLoading}
            parseLoading={parseLoading}
            saveLoading={saveLoading}
            onSpecChange={setSpec}
            onParseTextChange={setParseText}
            onSave={() => void handleSave()}
            onTrain={() => void handleTrain()}
            onParse={() => void handleParse()}
          />
        </div>
      </div>
    </BaseLayout>
  )
}
