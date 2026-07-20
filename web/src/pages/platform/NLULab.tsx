import { useCallback, useEffect, useState } from 'react'
import { Alert, Card, Input, Space, Tag, Typography } from '@arco-design/web-react'
import { BrainCircuit, Play, RefreshCw, RotateCcw } from 'lucide-react'
import BaseLayout from '@/components/Layout/BaseLayout'
import { Button } from '@/components/ui'
import { useTranslation } from '@/i18n'
import {
  fetchNluStatus,
  parseNluText,
  reloadNluEngine,
  type NluParseResult,
  type NluStatus,
} from '@/api/nlu'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

const { TextArea } = Input

const SAMPLE_TEXTS = ['查订单', '帮我查一下订单', '我要投诉', '你好']

export default function NLULabPage() {
  const { t } = useTranslation()
  const [status, setStatus] = useState<NluStatus | null>(null)
  const [statusLoading, setStatusLoading] = useState(false)
  const [text, setText] = useState('查订单')
  const [parseLoading, setParseLoading] = useState(false)
  const [reloadLoading, setReloadLoading] = useState(false)
  const [result, setResult] = useState<NluParseResult | null>(null)

  const loadStatus = useCallback(async () => {
    setStatusLoading(true)
    try {
      const s = await fetchNluStatus()
      setStatus(s)
    } catch (e: unknown) {
      showAlert(t('nluLab.statusFailed'), 'error', extractApiErrorMessage(e))
    } finally {
      setStatusLoading(false)
    }
  }, [t])

  useEffect(() => {
    void loadStatus()
  }, [loadStatus])

  const handleParse = async () => {
    const q = text.trim()
    if (!q) {
      showAlert(t('nluLab.textRequired'), 'warning')
      return
    }
    setParseLoading(true)
    setResult(null)
    try {
      const res = await parseNluText(q)
      setResult(res)
    } catch (e: unknown) {
      showAlert(t('nluLab.parseFailed'), 'error', extractApiErrorMessage(e))
    } finally {
      setParseLoading(false)
    }
  }

  const handleReload = async () => {
    setReloadLoading(true)
    try {
      const res = await reloadNluEngine()
      showAlert(res.message || t('nluLab.reloadSuccess'), 'success')
      await loadStatus()
    } catch (e: unknown) {
      showAlert(t('nluLab.reloadFailed'), 'error', extractApiErrorMessage(e))
    } finally {
      setReloadLoading(false)
    }
  }

  const intentOk =
    result &&
    result.prediction?.intent_name &&
    (status?.minConfidence ?? 0.22) <= (result.prediction.confidence ?? 0)

  return (
    <BaseLayout title={t('nluLab.title')} description={t('nluLab.description')}>
      <div className="space-y-4">
        {!status?.deployEnabled ? (
          <Alert type="warning" showIcon content={t('nluLab.envDisabledHint')} />
        ) : !status?.ready ? (
          <Alert type="warning" showIcon content={t('nluLab.configMissingHint')} />
        ) : !status.engineOnline ? (
          <Alert type="error" showIcon content={status.engineError || t('nluLab.engineOffline')} />
        ) : null}

        <Card className="shadow-sm">
          <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
            <Typography.Title heading={6} style={{ margin: 0 }}>
              {t('nluLab.statusTitle')}
            </Typography.Title>
            <Button
              variant="outline"
              size="sm"
              leftIcon={<RefreshCw className="h-4 w-4" />}
              loading={statusLoading}
              onClick={() => void loadStatus()}
            >
              {t('common.refresh')}
            </Button>
          </div>
          <div className="grid gap-2 text-sm sm:grid-cols-2">
            <div>
              <span className="text-neutral-500">{t('nluLab.deployEnabled')}: </span>
              <Tag color={status?.deployEnabled ? 'green' : 'gray'}>
                {status?.deployEnabled ? t('common.yes') : t('common.no')}
              </Tag>
            </div>
            <div>
              <span className="text-neutral-500">{t('nluLab.engineOnline')}: </span>
              <Tag color={status?.engineOnline ? 'green' : 'red'}>
                {status?.engineOnline ? t('nluLab.online') : t('nluLab.offline')}
              </Tag>
            </div>
            <div className="sm:col-span-2">
              <span className="text-neutral-500">{t('nluLab.modelPath')}: </span>
              <code className="text-xs">{status?.modelPath || '—'}</code>
            </div>
            <div className="sm:col-span-2">
              <span className="text-neutral-500">{t('nluLab.tokenizerPath')}: </span>
              <code className="text-xs">{status?.tokenizerPath || '—'}</code>
            </div>
            {status?.intents?.length ? (
              <div className="sm:col-span-2">
                <span className="text-neutral-500">{t('nluLab.intentLabels')}: </span>
                <Space wrap size="small">
                  {status.intents.map((name) => (
                    <Tag key={name}>{name}</Tag>
                  ))}
                </Space>
              </div>
            ) : null}
          </div>
        </Card>

        <Card className="shadow-sm">
          <Typography.Title heading={6} style={{ marginTop: 0 }}>
            {t('nluLab.parseTitle')}
          </Typography.Title>
          <p className="mb-3 text-sm text-neutral-500">{t('nluLab.parseHint')}</p>
          <TextArea
            value={text}
            onChange={setText}
            autoSize={{ minRows: 2, maxRows: 6 }}
            placeholder={t('nluLab.textPlaceholder')}
          />
          <div className="mt-3 flex flex-wrap gap-2">
            {SAMPLE_TEXTS.map((s) => (
              <button
                key={s}
                type="button"
                className="rounded-lg border border-neutral-200 px-2.5 py-1 text-xs text-neutral-600 hover:bg-neutral-50"
                onClick={() => setText(s)}
              >
                {s}
              </button>
            ))}
          </div>
          <div className="mt-4 flex flex-wrap gap-2">
            <Button
              leftIcon={<Play className="h-4 w-4" />}
              loading={parseLoading}
              disabled={!status?.ready}
              onClick={() => void handleParse()}
            >
              {t('nluLab.parse')}
            </Button>
            <Button
              variant="outline"
              leftIcon={<RotateCcw className="h-4 w-4" />}
              loading={reloadLoading}
              disabled={!status?.ready}
              onClick={() => void handleReload()}
            >
              {t('nluLab.reload')}
            </Button>
          </div>

          {result ? (
            <div className="mt-5 rounded-xl border border-neutral-100 bg-neutral-50 p-4">
              <div className="mb-2 flex items-center gap-2">
                <BrainCircuit className="h-4 w-4 text-neutral-500" />
                <span className="font-medium">{t('nluLab.result')}</span>
              </div>
              <div className="space-y-2 text-sm">
                <div>
                  <span className="text-neutral-500">{t('nluLab.channel')}: </span>
                  <Tag>{result.channel}</Tag>
                </div>
                <div>
                  <span className="text-neutral-500">{t('nluLab.intent')}: </span>
                  <Tag color={intentOk ? 'green' : 'orangered'}>
                    {result.prediction?.intent_name || '—'}
                  </Tag>
                  <span className="ml-2 text-neutral-500">
                    {(result.prediction?.confidence ?? 0).toFixed(3)}
                  </span>
                </div>
                {result.reply ? (
                  <div>
                    <span className="text-neutral-500">{t('nluLab.cannedReply')}: </span>
                    {result.reply}
                  </div>
                ) : null}
                <pre className="mt-2 max-h-48 overflow-auto rounded-lg bg-white p-3 text-xs">
                  {JSON.stringify(result, null, 2)}
                </pre>
              </div>
            </div>
          ) : null}
        </Card>

        <Card className="shadow-sm">
          <Typography.Title heading={6} style={{ marginTop: 0 }}>
            {t('nluLab.trainTitle')}
          </Typography.Title>
          <p className="text-sm leading-relaxed text-neutral-500">{t('nluLab.trainHint')}</p>
        </Card>
      </div>
    </BaseLayout>
  )
}
