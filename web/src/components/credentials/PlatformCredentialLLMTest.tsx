import { useState } from 'react'
import { Typography } from '@arco-design/web-react'
import { Button, Input } from '@/components/ui'
import { useTranslation } from '@/i18n'
import { testCredentialLLMStream, type CredentialLLMTestResult } from '@/api/credentials'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

/** LLM stream test for platform_bundle keys (config comes from 号池). */
export default function PlatformCredentialLLMTest({ credentialId }: { credentialId: string }) {
  const { t } = useTranslation()
  const [testing, setTesting] = useState(false)
  const [prompt, setPrompt] = useState('请用一句话自我介绍。')
  const [result, setResult] = useState<CredentialLLMTestResult | null>(null)

  const onTest = async () => {
    if (!credentialId) return
    setTesting(true)
    setResult(null)
    try {
      const res = await testCredentialLLMStream({
        prompt: prompt.trim() || undefined,
        credentialId,
      })
      if (res.code !== 200 || !res.data) {
        showAlert(res.msg || t('credentialAi.llmTestFailed'), 'error')
        return
      }
      setResult(res.data)
      showAlert(
        t('credentialAi.llmTestSuccess', {
          first: res.data.firstTokenMs ?? '—',
          wall: res.data.wallMs ?? '—',
        }),
        'success',
      )
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('credentialAi.llmTestFailed')), 'error')
    } finally {
      setTesting(false)
    }
  }

  return (
    <div className="rounded-xl border border-border bg-muted/20 p-4">
      <Typography.Text style={{ display: 'block', marginBottom: 8 }}>
        <strong>{t('credentialAi.llmTestTitle')}</strong>
      </Typography.Text>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 10, fontSize: 12 }}>
        {t('credentialAi.llmTestPlatformHint')}
      </Typography.Paragraph>
      <Input.TextArea
        value={prompt}
        onChange={setPrompt}
        autoSize={{ minRows: 2, maxRows: 4 }}
        placeholder={t('credentialAi.llmTestPrompt')}
        style={{ marginBottom: 10 }}
      />
      <Button type="outline" loading={testing} onClick={() => void onTest()}>
        {t('credentialAi.llmTestRun')}
      </Button>
      {result ? (
        <div className="mt-3 space-y-1 text-sm">
          <div>
            {t('credentialAi.llmTestFirstToken')}: <strong>{result.firstTokenMs ?? '—'} ms</strong>
            <span className="mx-2 text-neutral-300">·</span>
            {t('credentialAi.llmTestWall')}: <strong>{result.wallMs ?? '—'} ms</strong>
            <span className="mx-2 text-neutral-300">·</span>
            {result.provider}/{result.model}
          </div>
          <div className="whitespace-pre-wrap rounded-lg bg-white px-3 py-2 dark:bg-neutral-900">
            {result.reply || '—'}
          </div>
        </div>
      ) : null}
    </div>
  )
}
