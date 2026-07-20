import { useEffect, useState } from 'react'
import { Form, Modal, Switch } from '@arco-design/web-react'
import { Button, Input } from '@/components/ui'
import { showAlert } from '@/utils/notification'
import {
  changeMyEmail,
  requestAccountDeletion,
  sendChangeEmailCode,
  sendMePasswordEmailCode,
  updateMyPassword,
  disableTotp,
  enableTotp,
  setupTotp,
  revokeAllMySessions,
  updateSecurityPreferences,
  startGitHubBind,
  unbindGitHub,
} from '@/api/me'
import { useTranslation } from '@/i18n'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/stores/authStore'
import UserAccountVoiceprintSection from '@/components/profile/UserAccountVoiceprintSection'

const FormItem = Form.Item

type SecurityRowProps = {
  title: string
  description: string
  actionLabel: string
  actionTone?: 'outline' | 'dark'
  onAction: () => void
}

function SecurityRow({ title, description, actionLabel, actionTone = 'outline', onAction }: SecurityRowProps) {
  return (
    <div className="flex items-center justify-between gap-4 border-b border-neutral-100 py-5 last:border-b-0">
      <div className="min-w-0 flex-1">
        <div className="font-medium text-neutral-900">{title}</div>
        <div className="mt-0.5 text-sm text-neutral-500">{description}</div>
      </div>
      <Button
        type={actionTone === 'dark' ? 'primary' : 'outline'}
        className={actionTone === 'dark' ? '!border-neutral-900 !bg-neutral-900 !text-white hover:!bg-neutral-800' : undefined}
        onClick={onAction}
      >
        {actionLabel}
      </Button>
    </div>
  )
}

type Props = {
  me: any
  boundEmail?: string
  voiceprintEnabled?: boolean
  onReload: () => Promise<void>
  onTotpUpdated: (user: unknown) => void
}

export default function AccountSecurityPanel({ me, boundEmail, voiceprintEnabled = false, onReload, onTotpUpdated }: Props) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const clearUser = useAuthStore((s) => s.clearUser)

  const [pwdForm] = Form.useForm()
  const [emailForm] = Form.useForm()
  const [deleteForm] = Form.useForm()
  const [totpEnableForm] = Form.useForm()
  const [totpDisableForm] = Form.useForm()

  const [pwdMethod, setPwdMethod] = useState<'password' | 'email_code'>('password')
  const [pwdCodeCooldown, setPwdCodeCooldown] = useState(0)
  const [emailCodeCooldown, setEmailCodeCooldown] = useState(0)
  const [sendingPwdCode, setSendingPwdCode] = useState(false)
  const [sendingEmailCode, setSendingEmailCode] = useState(false)
  const [totpDraft, setTotpDraft] = useState<{ secret: string; qrDataUrl: string } | null>(null)

  const [emailModalOpen, setEmailModalOpen] = useState(false)
  const [pwdModalOpen, setPwdModalOpen] = useState(false)
  const [totpModalOpen, setTotpModalOpen] = useState(false)
  const [deleteModalOpen, setDeleteModalOpen] = useState(false)
  const [recoveryCodes, setRecoveryCodes] = useState<string[] | null>(null)
  const [prefSaving, setPrefSaving] = useState(false)
  const [revokingAll, setRevokingAll] = useState(false)
  const [idleDraft, setIdleDraft] = useState('12')
  const [maxDraft, setMaxDraft] = useState('48')

  const totpEnabled =
    me?.principal === 'platform' ? me?.platformAdmin?.totpEnabled : me?.user?.totpEnabled
  const isPlatform = me?.principal === 'platform'
  const receiveEmailNotify =
    me?.principal === 'platform' ? Boolean(me?.platformAdmin?.receiveEmailNotify) : Boolean(me?.user?.receiveEmailNotify)
  const requireDeviceVerify =
    me?.principal === 'platform'
      ? me?.platformAdmin?.requireDeviceVerify !== false
      : me?.user?.requireDeviceVerify === true
  const trustDeviceLoginEnabled =
    me?.principal === 'platform' ? me?.platformAdmin?.trustDeviceLoginEnabled !== false : me?.user?.trustDeviceLoginEnabled !== false
  const requireRemoteLoginVerify =
    me?.principal === 'platform'
      ? me?.platformAdmin?.requireRemoteLoginVerify !== false
      : me?.user?.requireRemoteLoginVerify === true
  const sessionIdleHours =
    me?.principal === 'platform' ? me?.platformAdmin?.sessionIdleTimeoutHours ?? 12 : me?.user?.sessionIdleTimeoutHours ?? 12
  const sessionMaxHours =
    me?.principal === 'platform' ? me?.platformAdmin?.sessionMaxLifetimeHours ?? 48 : me?.user?.sessionMaxLifetimeHours ?? 48
  const primaryLoginCity =
    me?.principal === 'platform' ? me?.platformAdmin?.primaryLoginCity : me?.user?.primaryLoginCity
  const githubLogin =
    me?.principal === 'platform' ? me?.platformAdmin?.githubLogin : me?.user?.githubLogin

  const downloadRecoveryCodes = () => {
    if (!recoveryCodes?.length) return
    const blob = new Blob([`${recoveryCodes.join('\n')}\n`], { type: 'text/plain;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'lingecho-recovery-codes.txt'
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  }

  useEffect(() => {
    setIdleDraft(String(sessionIdleHours))
    setMaxDraft(String(sessionMaxHours))
  }, [sessionIdleHours, sessionMaxHours])

  useEffect(() => {
    if (pwdCodeCooldown <= 0) return
    const tmr = window.setTimeout(() => setPwdCodeCooldown((s) => Math.max(0, s - 1)), 1000)
    return () => window.clearTimeout(tmr)
  }, [pwdCodeCooldown])

  useEffect(() => {
    if (emailCodeCooldown <= 0) return
    const tmr = window.setTimeout(() => setEmailCodeCooldown((s) => Math.max(0, s - 1)), 1000)
    return () => window.clearTimeout(tmr)
  }, [emailCodeCooldown])

  const sectionClass = 'rounded-xl border border-border bg-card px-5'

  const updatePreference = async (patch: Parameters<typeof updateSecurityPreferences>[0]) => {
    setPrefSaving(true)
    try {
      const res = await updateSecurityPreferences(patch)
      if (res.code !== 200) {
        showAlert(res.msg || t('profile.prefUpdateFailed'), 'error')
        return
      }
      showAlert(t('profile.prefUpdated'), 'success')
      await onReload()
    } catch (e: unknown) {
      showAlert(extractApiErrorMessage(e, t('profile.prefUpdateFailed')), 'error')
    } finally {
      setPrefSaving(false)
    }
  }

  const closeTotpModal = () => {
    setTotpModalOpen(false)
    setTotpDraft(null)
    totpEnableForm.resetFields()
    totpDisableForm.resetFields()
  }

  return (
    <>
      <div className={`${sectionClass} mb-4`}>
        <div className="pt-4 text-xs font-medium uppercase tracking-wide text-neutral-400">
          {t('profile.notificationPrefsTitle')}
        </div>
        <div className="flex items-center justify-between gap-4 border-b border-neutral-100 py-5">
          <div className="min-w-0 flex-1">
            <div className="font-medium text-neutral-900">{t('profile.receiveEmailNotify')}</div>
            <div className="mt-0.5 text-sm text-neutral-500">{t('profile.receiveEmailNotifyDesc')}</div>
          </div>
          <Switch
            checked={receiveEmailNotify}
            disabled={prefSaving}
            onChange={(v) => void updatePreference({ receiveEmailNotify: v })}
          />
        </div>
      </div>

      <div className={`${sectionClass} mb-4`}>
        <div className="pt-4 text-xs font-medium uppercase tracking-wide text-neutral-400">
          {t('profile.loginSecurityTitle')}
        </div>
        <div className="flex items-center justify-between gap-4 border-b border-neutral-100 py-5">
          <div className="min-w-0 flex-1">
            <div className="font-medium text-neutral-900">{t('profile.trustDeviceLogin')}</div>
            <div className="mt-0.5 text-sm text-neutral-500">{t('profile.trustDeviceLoginDesc')}</div>
          </div>
          <Switch
            checked={trustDeviceLoginEnabled}
            disabled={prefSaving}
            onChange={(v) => void updatePreference({ trustDeviceLoginEnabled: v })}
          />
        </div>
        <div className={`flex items-center justify-between gap-4 py-5 ${!isPlatform ? 'border-b border-neutral-100' : ''}`}>
          <div className="min-w-0 flex-1">
            <div className="font-medium text-neutral-900">{t('profile.requireRemoteLoginVerify')}</div>
            <div className="mt-0.5 text-sm text-neutral-500">
              {primaryLoginCity
                ? t('profile.requireRemoteLoginVerifyDescWithCity', { city: primaryLoginCity })
                : t('profile.requireRemoteLoginVerifyDesc')}
            </div>
          </div>
          <Switch
            checked={requireRemoteLoginVerify}
            disabled={prefSaving}
            onChange={(v) => void updatePreference({ requireRemoteLoginVerify: v })}
          />
        </div>
        {!isPlatform && (
          <div className="py-5">
            <div className="font-medium text-neutral-900">{t('profile.requireDeviceVerify')}</div>
            <div className="mt-0.5 text-sm text-neutral-500">{t('profile.requireDeviceVerifyDesc')}</div>
            <div className="mt-3">
              <Switch
                checked={requireDeviceVerify}
                disabled={prefSaving}
                onChange={(v) => void updatePreference({ requireDeviceVerify: v })}
              />
            </div>
          </div>
        )}
      </div>

      <div className={`${sectionClass} mb-4`}>
        <div className="pt-4 text-xs font-medium uppercase tracking-wide text-neutral-400">
          {t('profile.sessionManagementTitle')}
        </div>
        <div className="border-b border-neutral-100 py-5">
          <div className="font-medium text-neutral-900">{t('profile.revokeAllSessions')}</div>
          <div className="mt-0.5 text-sm text-neutral-500">{t('profile.revokeAllSessionsDesc')}</div>
          <Button
            type="outline"
            status="warning"
            className="mt-3"
            loading={revokingAll}
            onClick={async () => {
              setRevokingAll(true)
              try {
                const res = await revokeAllMySessions()
                if (res.code !== 200) {
                  showAlert(res.msg || t('profile.revokeAllSessionsFailed'), 'error')
                  return
                }
                showAlert(t('profile.revokeAllSessionsSuccess'), 'success')
                clearUser()
                navigate('/login', { replace: true, state: { reason: 'session_revoked' } })
              } catch (e: unknown) {
                showAlert(extractApiErrorMessage(e, t('profile.revokeAllSessionsFailed')), 'error')
              } finally {
                setRevokingAll(false)
              }
            }}
          >
            {t('profile.revokeAllSessionsBtn')}
          </Button>
        </div>
        <div className="grid gap-4 border-b border-neutral-100 py-5 sm:grid-cols-2">
          <div>
            <div className="font-medium text-neutral-900">{t('profile.sessionIdleTimeout')}</div>
            <div className="mt-0.5 text-sm text-neutral-500">{t('profile.sessionIdleTimeoutDesc')}</div>
            <Input
              className="mt-3"
              type="number"
              min={1}
              max={168}
              value={idleDraft}
              onChange={(v) => setIdleDraft(v.replace(/[^\d]/g, ''))}
              suffix={<span className="text-sm text-neutral-400">{t('profile.hoursUnit')}</span>}
            />
          </div>
          <div>
            <div className="font-medium text-neutral-900">{t('profile.sessionMaxLifetime')}</div>
            <div className="mt-0.5 text-sm text-neutral-500">{t('profile.sessionMaxLifetimeDesc')}</div>
            <Input
              className="mt-3"
              type="number"
              min={1}
              max={720}
              value={maxDraft}
              onChange={(v) => setMaxDraft(v.replace(/[^\d]/g, ''))}
              suffix={<span className="text-sm text-neutral-400">{t('profile.hoursUnit')}</span>}
            />
          </div>
        </div>
        <div className="flex justify-end py-4">
          <Button
            type="primary"
            loading={prefSaving}
            onClick={() => {
              const idle = Number(idleDraft)
              const max = Number(maxDraft)
              if (!idle || !max) {
                showAlert(t('profile.sessionTimeoutInvalid'), 'warning')
                return
              }
              void updatePreference({ sessionIdleTimeoutHours: idle, sessionMaxLifetimeHours: max })
            }}
          >
            {t('profile.saveSessionPrefs')}
          </Button>
        </div>
      </div>

      {!isPlatform && voiceprintEnabled ? (
        <UserAccountVoiceprintSection enabled={voiceprintEnabled} />
      ) : null}

      <div className={`${sectionClass} mb-4`}>
        <div className="pt-4 text-xs font-medium uppercase tracking-wide text-neutral-400">
          {t('profile.securityEmailTitle')}
        </div>
        <SecurityRow
          title={boundEmail || '-'}
          description={t('profile.securityEmailDesc')}
          actionLabel={t('profile.changeEmail')}
          onAction={() => setEmailModalOpen(true)}
        />
      </div>

      <div className={sectionClass}>
        <div className="pt-4 text-xs font-medium uppercase tracking-wide text-neutral-400">
          {t('profile.securitySectionTitle')}
        </div>
        <SecurityRow
          title={t('profile.changePassword')}
          description={t('profile.securityPasswordDesc')}
          actionLabel={t('profile.changePassword')}
          onAction={() => setPwdModalOpen(true)}
        />
        <SecurityRow
          title={t('profile.twoFactor')}
          description={totpEnabled ? t('profile.securityTotpOnDesc') : t('profile.securityTotpDesc')}
          actionLabel={totpEnabled ? t('profile.disableTotpBtn') : t('profile.enableTotp')}
          onAction={() => setTotpModalOpen(true)}
        />
        {totpEnabled && (
          <SecurityRow
            title={t('profile.recoveryCodes')}
            description={t('profile.recoveryCodesDesc')}
            actionLabel={t('profile.viewRecoveryHint')}
            onAction={() => showAlert(t('profile.recoveryCodesHint'), 'info')}
          />
        )}
        <SecurityRow
          title={t('profile.githubAccount')}
          description={
            githubLogin
              ? t('profile.githubBoundDesc', { login: githubLogin })
              : t('profile.githubUnboundDesc')
          }
          actionLabel={githubLogin ? t('profile.githubUnbind') : t('profile.githubBind')}
          onAction={async () => {
            if (githubLogin) {
              try {
                const res = await unbindGitHub()
                if (res.code !== 200) {
                  showAlert(res.msg || t('profile.githubUnbindFailed'), 'error')
                  return
                }
                showAlert(t('profile.githubUnbound'), 'success')
                await onReload()
              } catch (e: unknown) {
                showAlert(extractApiErrorMessage(e, t('profile.githubUnbindFailed')), 'error')
              }
              return
            }
            try {
              const res = await startGitHubBind()
              if (res.code !== 200 || !res.data?.authorizeUrl) {
                showAlert(res.msg || t('profile.githubBindFailed'), 'error')
                return
              }
              window.location.href = res.data.authorizeUrl
            } catch (e: unknown) {
              showAlert(extractApiErrorMessage(e, t('profile.githubBindFailed')), 'error')
            }
          }}
        />
        <SecurityRow
          title={t('profile.deleteAccount')}
          description={t('profile.deleteAccountDesc')}
          actionLabel={t('profile.goToDeletion')}
          actionTone="dark"
          onAction={() => setDeleteModalOpen(true)}
        />
      </div>

      <Modal
        title={t('profile.changeEmail')}
        visible={emailModalOpen}
        onCancel={() => {
          setEmailModalOpen(false)
          emailForm.resetFields()
        }}
        footer={null}
        unmountOnExit
      >
        <Form
          form={emailForm}
          layout="vertical"
          onSubmit={async (v) => {
            try {
              const res = await changeMyEmail({
                newEmail: String(v.newEmail || '').trim(),
                emailCode: String(v.emailCode || '').trim(),
              })
              if (res.code !== 200) {
                showAlert(res.msg || t('profile.changeEmailFailed'), 'error')
                return
              }
              showAlert(t('profile.changeEmailSuccess'), 'success')
              setEmailModalOpen(false)
              emailForm.resetFields()
              await onReload()
            } catch (e: unknown) {
              showAlert(extractApiErrorMessage(e, t('profile.changeEmailFailed')), 'error')
            }
          }}
        >
          <FormItem label={t('profile.newEmail')} field="newEmail" rules={[{ required: true, message: t('profile.newEmailRequired') }]}>
            <Input placeholder="name@example.com" />
          </FormItem>
          <FormItem label={t('profile.emailCode')} field="emailCode" rules={[{ required: true, message: t('profile.emailCodeRequired') }]}>
            <Input
              maxLength={6}
              suffix={
                <button
                  type="button"
                  disabled={emailCodeCooldown > 0 || sendingEmailCode}
                  className="text-sm text-neutral-600 disabled:text-neutral-300"
                  onClick={async () => {
                    const newEmail = String(emailForm.getFieldValue('newEmail') || '').trim()
                    if (!newEmail) {
                      showAlert(t('profile.newEmailRequired'), 'warning')
                      return
                    }
                    setSendingEmailCode(true)
                    try {
                      const res = await sendChangeEmailCode(newEmail)
                      if (res.code !== 200) {
                        showAlert(res.msg || t('profile.emailCodeSendFailed'), 'error')
                        return
                      }
                      setEmailCodeCooldown(60)
                      showAlert(t('profile.emailCodeSent'), 'success')
                    } catch (e: unknown) {
                      showAlert(extractApiErrorMessage(e, t('profile.emailCodeSendFailed')), 'error')
                    } finally {
                      setSendingEmailCode(false)
                    }
                  }}
                >
                  {emailCodeCooldown > 0
                    ? t('profile.emailCodeResendIn', { seconds: emailCodeCooldown })
                    : t('profile.sendEmailCode')}
                </button>
              }
            />
          </FormItem>
          <div className="flex justify-end gap-2">
            <Button type="outline" onClick={() => setEmailModalOpen(false)}>{t('common.cancel')}</Button>
            <Button type="primary" htmlType="submit">{t('profile.confirmChangeEmail')}</Button>
          </div>
        </Form>
      </Modal>

      <Modal
        title={t('profile.changePassword')}
        visible={pwdModalOpen}
        onCancel={() => {
          setPwdModalOpen(false)
          pwdForm.resetFields()
        }}
        footer={null}
        unmountOnExit
      >
        <div className="mb-4 flex rounded-xl bg-neutral-100 p-1">
          <button
            type="button"
            className={`flex-1 rounded-lg py-2 text-sm font-medium ${pwdMethod === 'password' ? 'bg-white shadow-sm' : 'text-neutral-500'}`}
            onClick={() => setPwdMethod('password')}
          >
            {t('profile.passwordMethodOld')}
          </button>
          <button
            type="button"
            className={`flex-1 rounded-lg py-2 text-sm font-medium ${pwdMethod === 'email_code' ? 'bg-white shadow-sm' : 'text-neutral-500'}`}
            onClick={() => setPwdMethod('email_code')}
          >
            {t('profile.passwordMethodEmail')}
          </button>
        </div>
        <Form
          form={pwdForm}
          layout="vertical"
          onSubmit={async (v) => {
            if (String(v.newPassword || '') !== String(v.confirmPassword || '')) {
              showAlert(t('profile.passwordMismatch'), 'error')
              return
            }
            try {
              const res = await updateMyPassword({
                method: pwdMethod,
                oldPassword: pwdMethod === 'password' ? String(v.oldPassword || '') : undefined,
                emailCode: pwdMethod === 'email_code' ? String(v.emailCode || '').trim() : undefined,
                newPassword: String(v.newPassword || ''),
              })
              if (res.code !== 200) {
                showAlert(res.msg || t('profile.changeFailed'), 'error')
                return
              }
              showAlert(t('profile.passwordChanged'), 'success')
              setPwdModalOpen(false)
              clearUser()
              navigate('/login', { replace: true })
            } catch (e: unknown) {
              showAlert(extractApiErrorMessage(e, t('profile.changeFailed')), 'error')
            }
          }}
        >
          {pwdMethod === 'password' ? (
            <FormItem label={t('profile.oldPassword')} field="oldPassword" rules={[{ required: true, message: t('profile.oldPasswordRequired') }]}>
              <Input.Password />
            </FormItem>
          ) : (
            <FormItem label={t('profile.emailCode')} field="emailCode" rules={[{ required: true, message: t('profile.emailCodeRequired') }]}>
              <Input
                maxLength={6}
                suffix={
                  <button
                    type="button"
                    disabled={pwdCodeCooldown > 0 || sendingPwdCode}
                    className="text-sm text-neutral-600 disabled:text-neutral-300"
                    onClick={async () => {
                      setSendingPwdCode(true)
                      try {
                        const res = await sendMePasswordEmailCode()
                        if (res.code !== 200) {
                          showAlert(res.msg || t('profile.emailCodeSendFailed'), 'error')
                          return
                        }
                        setPwdCodeCooldown(60)
                        showAlert(t('profile.emailCodeSent'), 'success')
                      } catch (e: unknown) {
                        showAlert(extractApiErrorMessage(e, t('profile.emailCodeSendFailed')), 'error')
                      } finally {
                        setSendingPwdCode(false)
                      }
                    }}
                  >
                    {pwdCodeCooldown > 0 ? t('profile.emailCodeResendIn', { seconds: pwdCodeCooldown }) : t('profile.sendEmailCode')}
                  </button>
                }
              />
            </FormItem>
          )}
          <FormItem label={t('profile.newPassword')} field="newPassword" rules={[{ required: true, message: t('profile.newPasswordRequired') }, { minLength: 8, message: t('profile.passwordMinLength') }]}>
            <Input.Password />
          </FormItem>
          <FormItem label={t('profile.confirmPassword')} field="confirmPassword" rules={[{ required: true, message: t('profile.confirmPasswordRequired') }]}>
            <Input.Password />
          </FormItem>
          <div className="flex justify-end gap-2">
            <Button type="outline" onClick={() => setPwdModalOpen(false)}>{t('common.cancel')}</Button>
            <Button type="primary" htmlType="submit">{t('profile.updatePassword')}</Button>
          </div>
        </Form>
      </Modal>

      <Modal
        title={t('profile.twoFactor')}
        visible={totpModalOpen}
        onCancel={closeTotpModal}
        footer={null}
        unmountOnExit
      >
        {totpEnabled ? (
          <Form
            form={totpDisableForm}
            layout="vertical"
            onSubmit={async (v) => {
              try {
                const res = await disableTotp({ password: String(v.password || ''), code: String(v.code || '').trim() })
                if (res.code !== 200) {
                  showAlert(res.msg || t('profile.disableFailed'), 'error')
                  return
                }
                showAlert(t('profile.totpDisabled'), 'success')
                onTotpUpdated(res.data)
                await onReload()
                closeTotpModal()
              } catch (e: unknown) {
                showAlert(extractApiErrorMessage(e, t('profile.disableFailed')), 'error')
              }
            }}
          >
            <p className="mb-3 text-sm text-neutral-500">{t('profile.disableTotpHint')}</p>
            <FormItem label={t('profile.loginPassword')} field="password" rules={[{ required: true }]}>
              <Input.Password />
            </FormItem>
            <FormItem label={t('profile.totpCodeOrRecovery')} field="code" rules={[{ required: true }]}>
              <Input maxLength={16} placeholder={t('profile.totpCodePlaceholder')} />
            </FormItem>
            <div className="flex justify-end gap-2">
              <Button type="outline" onClick={closeTotpModal}>{t('common.cancel')}</Button>
              <Button status="warning" htmlType="submit">{t('profile.disableTotpBtn')}</Button>
            </div>
          </Form>
        ) : (
          <>
            {!totpDraft && (
              <Button
                type="outline"
                onClick={async () => {
                  try {
                    const res = await setupTotp()
                    if (res.code !== 200 || !res.data) {
                      showAlert(res.msg || t('profile.generateFailed'), 'error')
                      return
                    }
                    setTotpDraft({ secret: res.data.secret, qrDataUrl: res.data.qrDataUrl })
                  } catch (e: unknown) {
                    showAlert(extractApiErrorMessage(e, t('profile.generateFailed')), 'error')
                  }
                }}
              >
                {t('profile.generateQrCode')}
              </Button>
            )}
            {totpDraft && (
              <Form
                form={totpEnableForm}
                layout="vertical"
                onSubmit={async (v) => {
                  try {
                    const res = await enableTotp({ secret: totpDraft.secret, code: String(v.code || '').trim() })
                    if (res.code !== 200 || !res.data) {
                      showAlert(res.msg || t('profile.enableFailed'), 'error')
                      return
                    }
                    showAlert(t('profile.totpEnabled'), 'success')
                    onTotpUpdated(res.data)
                    await onReload()
                    closeTotpModal()
                    if (Array.isArray(res.data.recoveryCodes) && res.data.recoveryCodes.length > 0) {
                      setRecoveryCodes(res.data.recoveryCodes)
                    }
                  } catch (e: unknown) {
                    showAlert(extractApiErrorMessage(e, t('profile.enableFailed')), 'error')
                  }
                }}
              >
                <img src={totpDraft.qrDataUrl} alt="qr" className="mx-auto mb-3 h-40 w-40" />
                <FormItem label={t('profile.totpCodeConfirm')} field="code" rules={[{ required: true }]}>
                  <Input maxLength={12} />
                </FormItem>
                <div className="flex justify-end gap-2">
                  <Button type="outline" onClick={closeTotpModal}>{t('common.cancel')}</Button>
                  <Button type="primary" htmlType="submit">{t('profile.confirmEnable')}</Button>
                </div>
              </Form>
            )}
          </>
        )}
      </Modal>

      <Modal
        title={t('profile.deleteAccount')}
        visible={deleteModalOpen}
        onCancel={() => {
          setDeleteModalOpen(false)
          deleteForm.resetFields()
        }}
        footer={null}
        unmountOnExit
      >
        <p className="mb-4 text-sm text-neutral-500">{t('profile.deleteAccountModalDesc')}</p>
        <Form
          form={deleteForm}
          layout="vertical"
          onSubmit={async (v) => {
            try {
              const res = await requestAccountDeletion({
                method: 'password',
                password: String(v.password || ''),
              })
              if (res.code !== 200) {
                showAlert(res.msg || t('profile.deletionRequestFailed'), 'error')
                return
              }
              const email = boundEmail || ''
              clearUser()
              setDeleteModalOpen(false)
              navigate(`/account/deletion/revoke?requested=1${email ? `&email=${encodeURIComponent(email)}` : ''}`, { replace: true })
            } catch (e: unknown) {
              showAlert(extractApiErrorMessage(e, t('profile.deletionRequestFailed')), 'error')
            }
          }}
        >
          <FormItem label={t('profile.loginPassword')} field="password" rules={[{ required: true }]}>
            <Input.Password />
          </FormItem>
          <div className="flex justify-end gap-2">
            <Button type="outline" onClick={() => setDeleteModalOpen(false)}>{t('common.cancel')}</Button>
            <Button status="danger" htmlType="submit">{t('profile.confirmDeletion')}</Button>
          </div>
        </Form>
      </Modal>

      <Modal
        title={t('profile.recoveryCodes')}
        visible={!!recoveryCodes}
        onCancel={() => setRecoveryCodes(null)}
        footer={
          <div className="flex w-full justify-end gap-2">
            <Button type="outline" onClick={downloadRecoveryCodes}>
              {t('profile.downloadRecoveryCodes')}
            </Button>
            <Button type="primary" onClick={() => setRecoveryCodes(null)}>
              {t('profile.recoveryCodesSaved')}
            </Button>
          </div>
        }
        unmountOnExit
      >
        <p className="mb-3 text-sm text-neutral-500">{t('profile.recoveryCodesSaveHint')}</p>
        <div className="rounded-lg bg-neutral-50 p-4 font-mono text-sm leading-7">
          {recoveryCodes?.map((code) => (
            <div key={code}>{code}</div>
          ))}
        </div>
      </Modal>
    </>
  )
}
