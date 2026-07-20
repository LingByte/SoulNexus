import {
  Alert,
  Card,
  Input,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react'
import { Play, Zap } from 'lucide-react'
import { Button } from '@/components/ui'
import { NluLineListTextArea } from '@/components/nlu/NluLineListTextArea'
import { useTranslation } from '@/i18n'
import type { TenantNluIntentDef, TenantNluModel, TenantNluSpec } from '@/api/nluModels'

const statusColor: Record<string, string> = {
  draft: 'gray',
  training: 'blue',
  ready: 'green',
  failed: 'red',
}

type Props = {
  model: TenantNluModel
  spec: TenantNluSpec
  nluMode: string
  maxIntents: number
  parseText: string
  parseResult: string
  parseLatencyMs?: number | null
  trainLoading: boolean
  parseLoading: boolean
  saveLoading?: boolean
  onSpecChange: (spec: TenantNluSpec) => void
  onParseTextChange: (text: string) => void
  onSave: () => void
  onTrain: () => void
  onParse: () => void
}

export default function NluModelEditorForm({
  model,
  spec,
  nluMode,
  maxIntents,
  parseText,
  parseResult,
  parseLatencyMs,
  trainLoading,
  parseLoading,
  saveLoading,
  onSpecChange,
  onParseTextChange,
  onSave,
  onTrain,
  onParse,
}: Props) {
  const { t } = useTranslation()

  const updateIntent = (idx: number, patch: Partial<TenantNluIntentDef>) => {
    const intents = spec.intents.map((it, i) => (i === idx ? { ...it, ...patch } : it))
    onSpecChange({ ...spec, intents })
  }

  const addIntent = () => {
    if (spec.intents.length >= maxIntents) {
      return
    }
    onSpecChange({
      ...spec,
      intents: [
        ...spec.intents,
        { name: t('nluLab.newIntentName'), reply: t('nluLab.newIntentReply'), keywords: [], samples: [] },
      ],
    })
  }

  const removeIntent = (idx: number) => {
    onSpecChange({ ...spec, intents: spec.intents.filter((_, i) => i !== idx) })
  }

  return (
    <div className="grid gap-4 lg:h-full lg:grid-cols-[minmax(280px,360px)_1fr] lg:items-stretch">
      <div className="flex min-h-0 flex-col gap-4">
        <Card className="shrink-0">
          <div className="flex flex-wrap items-center gap-2">
            <Typography.Title heading={5} style={{ margin: 0 }}>
              {model.name}
            </Typography.Title>
            <Tag color={statusColor[model.status] || 'gray'}>{model.status}</Tag>
          </div>
          {model.trainError ? (
            <Typography.Text type="error" style={{ fontSize: 12, display: 'block', marginTop: 8 }}>
              {model.trainError}
            </Typography.Text>
          ) : null}
          <Alert
            className="mt-3"
            type="info"
            content={
              nluMode === 'embedding'
                ? t('nluLab.embeddingIntentHint', { n: maxIntents })
                : t('nluLab.intentCountHint', { n: model.numClasses || '?' })
            }
          />
          <div className="mt-3 flex flex-wrap gap-2">
            <Button loading={saveLoading} onClick={onSave}>
              {t('common.save')}
            </Button>
            <Button variant="outline" leftIcon={<Zap className="h-4 w-4" />} loading={trainLoading} onClick={onTrain}>
              {t('nluLab.train')}
            </Button>
          </div>
        </Card>

        <Card title={t('nluLab.parseTitle')} className="ling-card-fill flex-1">
          <Input.TextArea
            className="shrink-0"
            value={parseText}
            onChange={onParseTextChange}
            autoSize={{ minRows: 3, maxRows: 6 }}
            onKeyDown={(e) => {
              if (e.key === 'Enter') e.stopPropagation()
            }}
          />
          <Space className="mt-2 shrink-0" align="center">
            <Button leftIcon={<Play className="h-4 w-4" />} loading={parseLoading} onClick={onParse}>
              {t('nluLab.parse')}
            </Button>
            {typeof parseLatencyMs === 'number' ? (
              <Tag color="arcoblue">{t('nluLab.parseLatency', { ms: parseLatencyMs })}</Tag>
            ) : null}
          </Space>
          {parseResult ? (
            <pre className="mt-3 min-h-0 flex-1 overflow-auto rounded-lg bg-neutral-50 p-3 text-xs dark:bg-neutral-900">
              {parseResult}
            </pre>
          ) : null}
        </Card>
      </div>

      <Card title={t('nluLab.intentsEditor')} className="ling-card-fill min-h-[480px] lg:min-h-0">
        <div className="mb-3 flex shrink-0 justify-end">
          <Button size="sm" variant="outline" onClick={addIntent} disabled={spec.intents.length >= maxIntents}>
            {t('nluLab.addIntent')}
          </Button>
        </div>
        <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
          {spec.intents.map((intent, idx) => (
            <Card key={`${intent.name}-${idx}`} size="small">
              <div className="mb-2 grid gap-2 sm:grid-cols-2">
                <Input
                  addBefore={t('nluLab.intent')}
                  value={intent.name}
                  onChange={(v) => updateIntent(idx, { name: v })}
                />
                <Input
                  addBefore={t('nluLab.cannedReply')}
                  value={intent.reply}
                  onChange={(v) => updateIntent(idx, { reply: v })}
                />
              </div>
              <NluLineListTextArea
                placeholder={t('nluLab.keywordsPlaceholder')}
                value={intent.keywords ?? []}
                onChange={(lines) => updateIntent(idx, { keywords: lines })}
                minRows={2}
                maxRows={4}
              />
              <NluLineListTextArea
                className="mt-2"
                placeholder={t('nluLab.samplesPlaceholder')}
                value={intent.samples ?? []}
                onChange={(lines) => updateIntent(idx, { samples: lines })}
                minRows={2}
                maxRows={6}
              />
              <Button className="mt-2" size="sm" variant="outline" status="danger" onClick={() => removeIntent(idx)}>
                {t('nluLab.removeIntent')}
              </Button>
            </Card>
          ))}
          {!spec.intents.length ? (
            <Typography.Text type="secondary">{t('nluLab.noIntentsYet')}</Typography.Text>
          ) : null}
        </div>
      </Card>
    </div>
  )
}
