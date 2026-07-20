import { useEffect, useState } from 'react'
import { Form } from '@arco-design/web-react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import AuthSplitLayout from '@/components/Auth/AuthSplitLayout'
import { sendRevokeDeletionEmailCode, revokeAccountDeletionPublic } from '@/api/accountDeletion'
import { Button, Input } from '@/components/ui'
import { useTranslation } from '@/i18n'
import { showAlert } from '@/utils/notification'
import { extractApiErrorMessage } from '@/utils/apiError'

const FormItem = Form.Item
const CODE_RESEND_SECONDS = 60

export default function AccountDeletionRevoke() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [form] = Form.useForm()
  const [codeCooldown, setCodeCooldown] = useState(0)
  const [sendingCode, setSendingCode] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const requested = searchParams.get('requested') === '1'

  useEffect(() => {
    const email = searchParams.get('email')
    if (email) form.setFieldValue('email', email)
  }, [form, searchParams])

  useEffect(() => {
    if (codeCooldown <= 0) return
    const timer = window.setTimeout(() => setCodeCooldown((s) => Math.max(0, s - 1)), 1000)
    return () => window.clearTimeout(timer)
  }, [codeCooldown])

  const handleSendCode = async () => {
    const email = String(form.getFieldValue('email') || '').trim()
    if (!email) {
      showAlert(t('accountDeletionRevoke.emailRequired'), 'warning')
      return
    }
    setSendingCode(true)
    try {
      const res = await sendRevokeDeletionEmailCode(email)
      if (res.code !== 200) {
        showAlert(res.msg || t('profile.emailCodeSendFailed'), 'error')
        return
      }
      setCodeCooldown(CODE_RESEND_SECONDS)
      showAlert(t('profile.emailCodeSent'), 'success')
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('profile.emailCodeSendFailed')), 'error')
    } finally {
      setSendingCode(false)
    }
  }

  const handleSubmit = async (values: Record<string, unknown>) => {
    setSubmitting(true)
    try {
      const res = await revokeAccountDeletionPublic({
        email: String(values.email || '').trim(),
        emailCode: String(values.emailCode || '').trim(),
      })
      if (res.code !== 200) {
        showAlert(res.msg || t('accountDeletionRevoke.revokeFailed'), 'error')
        return
      }
      showAlert(t('profile.deletionCancelled'), 'success')
      navigate('/login', { replace: true })
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('accountDeletionRevoke.revokeFailed')), 'error')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthSplitLayout subpage>
      <h1 className="mb-2 text-center text-2xl font-semibold tracking-tight text-neutral-900">
        {t('accountDeletionRevoke.title')}
      </h1>
      <p className="mb-6 text-center text-sm leading-relaxed text-neutral-500">{t('accountDeletionRevoke.desc')}</p>
      {requested && (
        <p className="mb-6 rounded-xl bg-amber-50 px-4 py-3 text-sm text-amber-800">
          {t('accountDeletionRevoke.requestedHint')}
        </p>
      )}

      <Form form={form} layout="vertical" onSubmit={handleSubmit}>
        <FormItem
          label={t('auth.email')}
          field="email"
          rules={[{ required: true, message: t('accountDeletionRevoke.emailRequired') }]}
        >
          <Input placeholder="name@example.com" size="lg" variant="filled" />
        </FormItem>
        <FormItem
          label={t('profile.emailCode')}
          field="emailCode"
          rules={[{ required: true, message: t('profile.emailCodeRequired') }]}
        >
          <Input
            maxLength={6}
            size="lg"
            variant="filled"
            suffix={
              <button
                type="button"
                disabled={codeCooldown > 0 || sendingCode}
                className="text-sm text-neutral-600 disabled:text-neutral-300"
                onClick={() => void handleSendCode()}
              >
                {codeCooldown > 0
                  ? t('profile.emailCodeResendIn', { seconds: codeCooldown })
                  : t('profile.sendEmailCode')}
              </button>
            }
          />
        </FormItem>
        <Button type="primary" htmlType="submit" long size="lg" loading={submitting} className="!mt-2">
          {t('accountDeletionRevoke.submit')}
        </Button>
      </Form>

      <p className="mt-6 text-center text-sm text-neutral-500">
        <Link to="/login" className="text-[hsl(var(--primary))] hover:underline">
          {t('auth.goLogin')}
        </Link>
      </p>
    </AuthSplitLayout>
  )
}
