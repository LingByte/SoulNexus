import { useEffect, useRef, useState } from 'react'
import { Alert, Form, Modal } from '@arco-design/web-react'
import { IconUser } from '@arco-design/web-react/icon'
import { Building2, Github, Lock, Mail } from 'lucide-react'
import { useNavigate, Link, useSearchParams } from 'react-router-dom'
import { PLATFORM_HOME_PATH, TENANT_HOME_PATH } from '@/constants/appPaths'
import AuthSplitLayout, { type AuthPageMode } from '@/components/Auth/AuthSplitLayout'
import {
  registerTenant,
  forgotPassword,
  sendForgotPasswordEmailCode,
  sendLoginEmailCode,
  sendLoginSMSCode,
  sendDeviceVerifyEmailCode,
  tenantLogin,
  exchangeGitHubOAuth,
  getGitHubLoginUrl,
  type TenantAuthPayload,
} from '@/api/tenantAuth'
import { resolveCompanyName } from '@/config/brandConfig'
import { getApiBaseURL } from '@/config/apiConfig'
import { useAuthStore } from '@/stores/authStore'
import { useSiteConfig } from '@/contexts/siteConfig'
import { useTranslation } from '@/i18n'
import { Button, Input } from '@/components/ui'
import CaptchaVerifyModal from '@/components/Captcha/CaptchaVerifyModal'
import type { CaptchaProof } from '@/api/captcha'
import { getOrCreateDeviceId } from '@/utils/deviceId'
import { showAlert } from '@/utils/notification'
import VoiceprintAudioRecorder from '@/components/voice/VoiceprintAudioRecorder'

const FormItem = Form.Item

type TenantAuthProps = {
  initialMode?: AuthPageMode
}

type LoginMethod = 'password' | 'email_code' | 'phone_code'

const submitBtnClass =
  '!h-11 !rounded-xl !border-neutral-900 !bg-neutral-900 !text-white hover:!bg-neutral-800'

const CODE_RESEND_SECONDS = 60

async function fileToBase64(file: File): Promise<string> {
  const buf = await file.arrayBuffer()
  const bytes = new Uint8Array(buf)
  let binary = ''
  for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i])
  return btoa(binary)
}

export default function TenantAuth({ initialMode = 'login' }: TenantAuthProps) {
  const { t } = useTranslation()
  const { config } = useSiteConfig()
  const [loginForm] = Form.useForm()
  const [registerForm] = Form.useForm()
  const [forgotForm] = Form.useForm()
  const [totpForm] = Form.useForm()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const sessionRevokedShown = useRef(false)
  const login = useAuthStore((s) => s.login)
  const clearUser = useAuthStore((s) => s.clearUser)

  const [pageMode, setPageMode] = useState<AuthPageMode>(initialMode)
  const [loginMethod, setLoginMethod] = useState<LoginMethod>('password')
  const [needsTotp, setNeedsTotp] = useState(false)
  const [totpModalOpen, setTotpModalOpen] = useState(false)
  const [oauthTicket, setOauthTicket] = useState<string | null>(null)
  const [githubOAuthEnabled, setGithubOAuthEnabled] = useState(false)
  const tenantSelfRegisterEnabled = Boolean(config.tenantSelfRegisterEnabled)
  const smsLoginEnabled = Boolean(config.smsLoginEnabled)
  const [needsDeviceVerify, setNeedsDeviceVerify] = useState(false)
  const [needsRemoteVerify, setNeedsRemoteVerify] = useState(false)
  const [voiceprintAvailable, setVoiceprintAvailable] = useState(false)
  const [deviceVerifyMethod, setDeviceVerifyMethod] = useState<'email' | 'voiceprint'>('email')
  const [voiceprintLoginFile, setVoiceprintLoginFile] = useState<File | null>(null)
  const voiceprintEnabled = Boolean(config.VOICEPRINT_PROVIDER?.trim())
  const [trustDeviceFor7Days, setTrustDeviceFor7Days] = useState(true)
  const [deviceCodeCooldown, setDeviceCodeCooldown] = useState(0)
  const [sendingDeviceCode, setSendingDeviceCode] = useState(false)
  const [codeCooldown, setCodeCooldown] = useState(0)
  const [sendingCode, setSendingCode] = useState(false)
  const [showForgotPassword, setShowForgotPassword] = useState(false)
  const [forgotCodeCooldown, setForgotCodeCooldown] = useState(0)
  const [sendingForgotCode, setSendingForgotCode] = useState(false)
  const [deletionRevokeVisible, setDeletionRevokeVisible] = useState(false)
  const [sessionRevokedBanner, setSessionRevokedBanner] = useState(false)
  const [captchaModalOpen, setCaptchaModalOpen] = useState(false)
  const [formSubmitting, setFormSubmitting] = useState(false)
  const pendingCaptchaAction = useRef<((proof: CaptchaProof) => void | Promise<void>) | null>(null)
  const pendingFormSubmit = useRef(false)

  const openCaptchaGate = (
    action: (proof: CaptchaProof) => void | Promise<void>,
    opts?: { formSubmit?: boolean },
  ) => {
    pendingCaptchaAction.current = action
    pendingFormSubmit.current = Boolean(opts?.formSubmit)
    setCaptchaModalOpen(true)
  }

  const handleCaptchaVerified = (proof: CaptchaProof) => {
    setCaptchaModalOpen(false)
    const action = pendingCaptchaAction.current
    const showFormLoading = pendingFormSubmit.current
    pendingCaptchaAction.current = null
    pendingFormSubmit.current = false
    if (!action) return
    if (showFormLoading) setFormSubmitting(true)
    Promise.resolve(action(proof)).finally(() => {
      if (showFormLoading) setFormSubmitting(false)
    })
  }

  const companyName = resolveCompanyName(config.SITE_NAME, t('layout.siteName'))

  type DeviceVerifyExtra = {
    needsDeviceVerify?: boolean
    needsRemoteVerify?: boolean
    voiceprintAvailable?: boolean
    invalidVoiceprint?: boolean
  }

  const applyDeviceVerifyChallenge = (extra: DeviceVerifyExtra | undefined, failMsg?: string) => {
    if (!extra?.needsDeviceVerify && !extra?.needsRemoteVerify) return false
    setNeedsDeviceVerify(Boolean(extra.needsDeviceVerify))
    setNeedsRemoteVerify(Boolean(extra.needsRemoteVerify))

    const keepVoiceprintTab =
      Boolean(extra.invalidVoiceprint) ||
      deviceVerifyMethod === 'voiceprint' ||
      extra.voiceprintAvailable === true
    if (extra.voiceprintAvailable !== undefined) {
      setVoiceprintAvailable(Boolean(extra.voiceprintAvailable) && voiceprintEnabled)
    } else if (keepVoiceprintTab && voiceprintEnabled) {
      setVoiceprintAvailable(true)
    }

    if (extra.invalidVoiceprint) {
      setDeviceVerifyMethod('voiceprint')
      setVoiceprintLoginFile(null)
      showAlert((failMsg || '').trim() || t('auth.invalidVoiceprint'), 'error')
    } else {
      showAlert(
        extra.needsDeviceVerify
          ? t('auth.newDeviceDetected')
          : extra.needsRemoteVerify
            ? t('auth.remoteLoginDetected')
            : t('auth.newDeviceDetected'),
        'warning',
      )
    }
    return true
  }

  useEffect(() => {
    if (config.githubOAuthEnabled) {
      setGithubOAuthEnabled(true)
      return
    }
    const base = getApiBaseURL().replace(/\/$/, '')
    fetch(`${base}/system/init?_t=${Date.now()}`)
      .then((r) => r.json())
      .then((body: { data?: { githubOAuthEnabled?: boolean } }) => {
        setGithubOAuthEnabled(Boolean(body?.data?.githubOAuthEnabled))
      })
      .catch(() => {
        /* optional */
      })
  }, [config.githubOAuthEnabled])

  useEffect(() => {
    setPageMode(initialMode)
  }, [initialMode])

  useEffect(() => {
    if (pageMode !== 'login' || sessionRevokedShown.current) return
    let revoked = searchParams.get('reason') === 'session_revoked'
    try {
      revoked = revoked || sessionStorage.getItem('lingecho.auth.sessionRevoked') === '1'
    } catch {
      /* ignore */
    }
    if (!revoked) return
    sessionRevokedShown.current = true
    try {
      sessionStorage.removeItem('lingecho.auth.sessionRevoked')
    } catch {
      /* ignore */
    }
    setSessionRevokedBanner(true)
    showAlert(t('auth.sessionRevoked'), 'warning', undefined, { duration: 8000 })
    if (searchParams.get('reason') === 'session_revoked') {
      window.history.replaceState(null, '', '/login')
    }
  }, [pageMode, searchParams, t])

  useEffect(() => {
    if (codeCooldown <= 0) return
    const timer = window.setTimeout(() => setCodeCooldown((s) => Math.max(0, s - 1)), 1000)
    return () => window.clearTimeout(timer)
  }, [codeCooldown])

  useEffect(() => {
    if (deviceCodeCooldown <= 0) return
    const timer = window.setTimeout(() => setDeviceCodeCooldown((s) => Math.max(0, s - 1)), 1000)
    return () => window.clearTimeout(timer)
  }, [deviceCodeCooldown])

  useEffect(() => {
    if (forgotCodeCooldown <= 0) return
    const timer = window.setTimeout(() => setForgotCodeCooldown((s) => Math.max(0, s - 1)), 1000)
    return () => window.clearTimeout(timer)
  }, [forgotCodeCooldown])

  useEffect(() => {
    if (pageMode !== 'login') return
    const ticket = searchParams.get('oauth_ticket')
    const oauthError = searchParams.get('oauth_error')
    if (oauthError) {
      showAlert(t('auth.oauthLoginFailed'), 'error')
      window.history.replaceState(null, '', '/login')
      return
    }
    if (!ticket) return
    setOauthTicket(ticket)
    if (searchParams.get('oauth_needs_totp') === '1') {
      setNeedsTotp(true)
      setTotpModalOpen(true)
    } else {
      void completeOAuthLogin(ticket)
    }
    window.history.replaceState(null, '', '/login')
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pageMode, searchParams])

  const switchMode = (mode: AuthPageMode) => {
    setPageMode(mode)
    navigate(mode === 'register' ? '/register' : '/login', { replace: true })
  }

  const finishAuth = async (d: TenantAuthPayload) => {
    clearUser()
    if (d.principal === 'platform' && d.platformAdmin) {
      const a = d.platformAdmin
      await login(d.token, {
        id: a.id,
        email: a.email,
        displayName: a.displayName,
        isPlatformAdmin: true,
        principal: 'platform' as const,
      })
    } else if (d.principal === 'tenant' && d.user && d.tenant) {
      const { token, user, tenant } = d
      await login(token, {
        ...user,
        tenantSlug: tenant.slug,
        tenantName: tenant.name,
        principal: 'tenant' as const,
        permissionCodes: d.permissionCodes ?? [],
      })
    } else {
      showAlert(t('auth.invalidResponse'), 'error')
      return false
    }
    showAlert(t('auth.loginSuccess'), 'success')
    navigate(d.principal === 'platform' ? PLATFORM_HOME_PATH : TENANT_HOME_PATH, { replace: true })
    return true
  }

  const completeOAuthLogin = async (ticket: string, totpCode?: string) => {
    try {
      const res = await exchangeGitHubOAuth({ ticket, totpCode: totpCode?.trim() || undefined })
      if (res.code !== 200 || !res.data?.token) {
        const extra = res.data as { needsTotp?: boolean } | undefined
        if (extra?.needsTotp) {
          setNeedsTotp(true)
          setTotpModalOpen(true)
          return
        }
        showAlert(res.msg || t('auth.loginFailed'), 'error')
        return
      }
      setTotpModalOpen(false)
      setNeedsTotp(false)
      setOauthTicket(null)
      totpForm.resetFields()
      await finishAuth(res.data)
    } catch (e: unknown) {
      const errObj = e as { msg?: string; data?: { needsTotp?: boolean } | null }
      if (errObj?.data?.needsTotp) {
        setNeedsTotp(true)
        setTotpModalOpen(true)
        return
      }
      const msg =
        typeof e === 'object' && e && 'msg' in e ? String((e as { msg?: string }).msg) : t('auth.loginFailed')
      showAlert(msg, 'error')
    }
  }

  const handleTotpConfirm = async (v: Record<string, unknown>) => {
    const totpCode = String(v.totpCode || '').trim()
    if (!totpCode) {
      showAlert(t('auth.totpCodeRequired'), 'warning')
      return
    }
    if (oauthTicket) {
      await completeOAuthLogin(oauthTicket, totpCode)
      return
    }
    const loginValues = loginForm.getFieldsValue()
    await handleLogin({ ...loginValues, totpCode }, undefined)
  }

  const handleLogin = async (v: Record<string, unknown>, captcha?: CaptchaProof) => {
    setDeletionRevokeVisible(false)
    const totpCode = String(v.totpCode || '').trim() || undefined
    if (!captcha && !totpCode) return
    try {
      let voiceprintAudioBase64: string | undefined
      if ((needsDeviceVerify || needsRemoteVerify) && deviceVerifyMethod === 'voiceprint') {
        if (!voiceprintLoginFile) {
          showAlert(t('profile.voiceprintAudioRequired'), 'warning')
          return
        }
        voiceprintAudioBase64 = await fileToBase64(voiceprintLoginFile)
      }
      const res = await tenantLogin({
        email: loginMethod === 'phone_code' ? undefined : String(v.email || '').trim(),
        phone: loginMethod === 'phone_code' ? String(v.phone || '').trim() : undefined,
        loginMethod,
        password: loginMethod === 'password' ? String(v.password || '') : undefined,
        emailCode: loginMethod === 'email_code' ? String(v.emailCode || '').trim() : undefined,
        smsCode: loginMethod === 'phone_code' ? String(v.smsCode || '').trim() : undefined,
        totpCode,
        deviceId: getOrCreateDeviceId(),
        deviceVerifyCode:
          (needsDeviceVerify || needsRemoteVerify) && deviceVerifyMethod === 'email'
            ? String(v.deviceVerifyCode || '').trim()
            : undefined,
        voiceprintAudioBase64,
        trustDeviceFor7Days: (needsDeviceVerify || needsRemoteVerify) && trustDeviceFor7Days,
        ...(captcha ?? {}),
      })
      if (res.code !== 200 || !res.data?.token) {
        const extra = res.data as DeviceVerifyExtra & {
          needsTotp?: boolean
          revokePath?: string
        } | undefined
        if (extra?.needsTotp) {
          setNeedsTotp(true)
          setTotpModalOpen(true)
          return
        }
        if (applyDeviceVerifyChallenge(extra, res.msg)) return
        if (extra?.revokePath) setDeletionRevokeVisible(true)
        showAlert(res.msg || t('auth.loginFailed'), 'error')
        return
      }
      const d = res.data
      setTotpModalOpen(false)
      setNeedsTotp(false)
      totpForm.resetFields()
      await finishAuth(d)
    } catch (e: unknown) {
      const errObj = e as {
        msg?: string
        data?: (DeviceVerifyExtra & {
          revokePath?: string
          needsTotp?: boolean
        }) | null
      }
      if (errObj?.data?.needsTotp) {
        setNeedsTotp(true)
        setTotpModalOpen(true)
        return
      }
      if (errObj?.data?.revokePath) setDeletionRevokeVisible(true)
      if (applyDeviceVerifyChallenge(errObj?.data ?? undefined, errObj.msg)) return
      const msg =
        typeof e === 'object' && e && 'msg' in e ? String((e as { msg?: string }).msg) : t('auth.loginFailed')
      showAlert(msg, 'error')
    }
  }

  const handleSendDeviceVerifyCode = async (captcha: CaptchaProof) => {
    const email = String(loginForm.getFieldValue('email') || '').trim()
    if (!email) {
      showAlert(t('auth.usernameOrEmailRequired'), 'warning')
      return
    }
    if (loginMethod === 'password') {
      const password = String(loginForm.getFieldValue('password') || '')
      if (!password) {
        showAlert(t('auth.passwordRequired'), 'warning')
        return
      }
    } else {
      const emailCode = String(loginForm.getFieldValue('emailCode') || '').trim()
      if (!emailCode) {
        showAlert(t('auth.emailCodeRequired'), 'warning')
        return
      }
    }
    if (deviceCodeCooldown > 0 || sendingDeviceCode) return
    setSendingDeviceCode(true)
    try {
      const res = await sendDeviceVerifyEmailCode({
        email,
        loginMethod,
        password: loginMethod === 'password' ? String(loginForm.getFieldValue('password') || '') : undefined,
        emailCode: loginMethod === 'email_code' ? String(loginForm.getFieldValue('emailCode') || '').trim() : undefined,
        deviceId: getOrCreateDeviceId(),
        ...captcha,
      })
      if (res.code !== 200) {
        showAlert(res.msg || t('auth.emailCodeSendFailed'), 'error')
        return
      }
      setDeviceCodeCooldown(CODE_RESEND_SECONDS)
      showAlert(t('auth.emailCodeSent'), 'success')
    } catch (e: unknown) {
      const msg =
        typeof e === 'object' && e && 'msg' in e ? String((e as { msg?: string }).msg) : t('auth.emailCodeSendFailed')
      showAlert(msg, 'error')
    } finally {
      setSendingDeviceCode(false)
    }
  }

  const handleSendEmailCode = async (captcha: CaptchaProof) => {
    setDeletionRevokeVisible(false)
    const email = String(loginForm.getFieldValue('email') || '').trim()
    if (!email) {
      showAlert(t('auth.usernameOrEmailRequired'), 'warning')
      return
    }
    if (codeCooldown > 0 || sendingCode) return
    setSendingCode(true)
    try {
      const res = await sendLoginEmailCode(email, captcha)
      if (res.code !== 200) {
        const extra = res.data as { revokePath?: string } | undefined
        if (extra?.revokePath) setDeletionRevokeVisible(true)
        showAlert(res.msg || t('auth.emailCodeSendFailed'), 'error')
        return
      }
      setCodeCooldown(CODE_RESEND_SECONDS)
      showAlert(t('auth.emailCodeSent'), 'success')
    } catch (e: unknown) {
      const errObj = e as { msg?: string; data?: { revokePath?: string } | null }
      if (errObj?.data?.revokePath) setDeletionRevokeVisible(true)
      const msg =
        typeof e === 'object' && e && 'msg' in e ? String((e as { msg?: string }).msg) : t('auth.emailCodeSendFailed')
      showAlert(msg, 'error')
    } finally {
      setSendingCode(false)
    }
  }

  const handleSendSMSCode = async (captcha: CaptchaProof) => {
    const phone = String(loginForm.getFieldValue('phone') || '').trim()
    if (!phone) {
      showAlert(t('auth.phoneRequired'), 'warning')
      return
    }
    if (codeCooldown > 0 || sendingCode) return
    setSendingCode(true)
    try {
      const res = await sendLoginSMSCode(phone, captcha)
      if (res.code !== 200) {
        showAlert(res.msg || t('auth.smsCodeSendFailed'), 'error')
        return
      }
      setCodeCooldown(CODE_RESEND_SECONDS)
      showAlert(t('auth.smsCodeSent'), 'success')
    } catch (e: unknown) {
      const msg =
        typeof e === 'object' && e && 'msg' in e ? String((e as { msg?: string }).msg) : t('auth.smsCodeSendFailed')
      showAlert(msg, 'error')
    } finally {
      setSendingCode(false)
    }
  }

  const handleSendForgotEmailCode = async (captcha: CaptchaProof) => {
    const email = String(forgotForm.getFieldValue('email') || '').trim()
    if (!email) {
      showAlert(t('auth.usernameOrEmailRequired'), 'warning')
      return
    }
    if (forgotCodeCooldown > 0 || sendingForgotCode) return
    setSendingForgotCode(true)
    try {
      const res = await sendForgotPasswordEmailCode(email, captcha)
      if (res.code !== 200) {
        showAlert(res.msg || t('auth.emailCodeSendFailed'), 'error')
        return
      }
      setForgotCodeCooldown(CODE_RESEND_SECONDS)
      showAlert(t('auth.emailCodeSent'), 'success')
    } catch (e: unknown) {
      const msg =
        typeof e === 'object' && e && 'msg' in e ? String((e as { msg?: string }).msg) : t('auth.emailCodeSendFailed')
      showAlert(msg, 'error')
    } finally {
      setSendingForgotCode(false)
    }
  }

  const handleForgotPassword = async (v: Record<string, unknown>, captcha: CaptchaProof) => {
    const newPassword = String(v.newPassword || '')
    const confirmPassword = String(v.confirmPassword || '')
    if (newPassword !== confirmPassword) {
      showAlert(t('profile.passwordMismatch'), 'error')
      return
    }
    try {
      const res = await forgotPassword({
        email: String(v.email || '').trim(),
        emailCode: String(v.emailCode || '').trim(),
        newPassword,
        ...captcha,
      })
      if (res.code !== 200) {
        showAlert(res.msg || t('auth.resetPasswordFailed'), 'error')
        return
      }
      showAlert(t('auth.resetPasswordSuccess'), 'success')
      setShowForgotPassword(false)
      forgotForm.resetFields()
      navigate('/login', { replace: true })
    } catch (e: unknown) {
      const msg =
        typeof e === 'object' && e && 'msg' in e ? String((e as { msg?: string }).msg) : t('auth.resetPasswordFailed')
      showAlert(msg, 'error')
    }
  }

  const handleRegister = async (v: Record<string, unknown>, captcha: CaptchaProof) => {
    try {
      const res = await registerTenant({
        companyName: String(v.companyName || '').trim(),
        adminEmail: String(v.adminEmail || '').trim(),
        adminPassword: String(v.adminPassword || ''),
        adminDisplayName: String(v.companyName || '').trim(),
        deviceId: getOrCreateDeviceId(),
        ...captcha,
      })
      if (res.code !== 200 || !res.data?.token) {
        showAlert(res.msg || t('auth.registerFailed'), 'error')
        return
      }
      const d = res.data
      if (!d.token || d.principal !== 'tenant' || !d.user || !d.tenant) {
        showAlert(t('auth.registerInvalidResponse'), 'error')
        return
      }
      const { token, user, tenant } = d
      await login(token, {
        ...user,
        tenantSlug: tenant.slug,
        tenantName: tenant.name,
        principal: 'tenant' as const,
        permissionCodes: d.permissionCodes ?? [],
      })
      showAlert(t('auth.registerSuccess'), 'success')
      navigate(TENANT_HOME_PATH, { replace: true })
    } catch (e: unknown) {
      const msg =
        typeof e === 'object' && e && 'msg' in e ? String((e as { msg?: string }).msg) : t('auth.registerFailed')
      showAlert(msg, 'error')
    }
  }

  return (
    <>
    <AuthSplitLayout mode={pageMode} onSwitchMode={switchMode}>
      <div className="mb-4 flex justify-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-full bg-neutral-100 text-neutral-400">
          <IconUser style={{ fontSize: 24 }} />
        </div>
      </div>

      {pageMode === 'login' ? (
        showForgotPassword ? (
          <>
            <h1 className="mb-2 text-center text-2xl font-semibold tracking-tight text-neutral-900">
              {t('auth.forgotPasswordTitle')}
            </h1>
            <p className="mb-6 text-center text-sm text-neutral-500">{t('auth.forgotPasswordSubtitle')}</p>
            <Form
              form={forgotForm}
              layout="vertical"
              requiredSymbol={false}
              onSubmit={(v) => openCaptchaGate((proof) => handleForgotPassword(v, proof), { formSubmit: true })}
            >
              <FormItem
                label={<span className="text-sm text-neutral-600">{t('auth.email')}</span>}
                field="email"
                rules={[{ required: true, message: t('auth.usernameOrEmailRequired') }]}
                style={{ marginBottom: 18 }}
              >
                <Input
                  size="lg"
                  variant="filled"
                  placeholder={t('auth.emailInputPlaceholder')}
                  prefix={<Mail className="h-4 w-4 text-neutral-400" strokeWidth={1.75} />}
                />
              </FormItem>
              <FormItem
                label={<span className="text-sm text-neutral-600">{t('auth.emailCode')}</span>}
                field="emailCode"
                rules={[{ required: true, message: t('auth.emailCodeRequired') }]}
                style={{ marginBottom: 18 }}
              >
                <Input
                  size="lg"
                  variant="filled"
                  maxLength={6}
                  placeholder={t('auth.emailCodePlaceholder')}
                  suffix={
                    <button
                      type="button"
                      disabled={forgotCodeCooldown > 0 || sendingForgotCode}
                      className="whitespace-nowrap text-sm text-neutral-600 transition-colors hover:text-neutral-900 disabled:cursor-not-allowed disabled:text-neutral-300"
                      onClick={() => openCaptchaGate((proof) => void handleSendForgotEmailCode(proof))}
                    >
                      {forgotCodeCooldown > 0
                        ? t('auth.emailCodeResendIn', { seconds: forgotCodeCooldown })
                        : t('auth.sendEmailCode')}
                    </button>
                  }
                />
              </FormItem>
              <FormItem
                label={<span className="text-sm text-neutral-600">{t('auth.password')}</span>}
                field="newPassword"
                rules={[
                  { required: true, message: t('auth.passwordMin8') },
                  { minLength: 8, message: t('auth.passwordMin8Short') },
                ]}
                style={{ marginBottom: 18 }}
              >
                <Input.Password
                  size="lg"
                  variant="filled"
                  placeholder={t('auth.passwordPlaceholder')}
                  prefix={<Lock className="h-4 w-4 text-neutral-400" strokeWidth={1.75} />}
                />
              </FormItem>
              <FormItem
                label={<span className="text-sm text-neutral-600">{t('profile.confirmPassword')}</span>}
                field="confirmPassword"
                rules={[{ required: true, message: t('profile.confirmPasswordRequired') }]}
                style={{ marginBottom: 24 }}
              >
                <Input.Password
                  size="lg"
                  variant="filled"
                  placeholder={t('auth.passwordPlaceholder')}
                  prefix={<Lock className="h-4 w-4 text-neutral-400" strokeWidth={1.75} />}
                />
              </FormItem>
              <Button type="primary" htmlType="submit" long size="lg" className={submitBtnClass} loading={formSubmitting}>
                {t('auth.resetPassword')}
              </Button>
              <div className="mt-4 text-center">
                <button
                  type="button"
                  className="text-sm text-neutral-500 transition-colors hover:text-neutral-800"
                  onClick={() => setShowForgotPassword(false)}
                >
                  {t('auth.goLogin')}
                </button>
              </div>
            </Form>
          </>
        ) : (
        <>
          {sessionRevokedBanner ? (
            <Alert
              type="warning"
              showIcon
              closable
              className="mb-4"
              content={t('auth.sessionRevoked')}
              onClose={() => setSessionRevokedBanner(false)}
            />
          ) : null}
          <h1 className="mb-4 text-center text-2xl font-semibold tracking-tight text-neutral-900">
            {t('auth.loginTo', { name: companyName })}
          </h1>

          <div className="mb-4 flex rounded-xl bg-neutral-100 p-1">
            <button
              type="button"
              className={`flex-1 rounded-lg py-2 text-sm font-medium transition-colors ${
                loginMethod === 'password'
                  ? 'bg-white text-neutral-900 shadow-sm'
                  : 'text-neutral-500 hover:text-neutral-700'
              }`}
              onClick={() => {
                setLoginMethod('password')
                setNeedsTotp(false)
                setTotpModalOpen(false)
              }}
            >
              {t('auth.passwordLogin')}
            </button>
            <button
              type="button"
              className={`flex-1 rounded-lg py-2 text-sm font-medium transition-colors ${
                loginMethod === 'email_code'
                  ? 'bg-white text-neutral-900 shadow-sm'
                  : 'text-neutral-500 hover:text-neutral-700'
              }`}
              onClick={() => {
                setLoginMethod('email_code')
                setNeedsTotp(false)
                setTotpModalOpen(false)
              }}
            >
              {t('auth.emailCodeLogin')}
            </button>
            {smsLoginEnabled ? (
              <button
                type="button"
                className={`flex-1 rounded-lg py-2 text-sm font-medium transition-colors ${
                  loginMethod === 'phone_code'
                    ? 'bg-white text-neutral-900 shadow-sm'
                    : 'text-neutral-500 hover:text-neutral-700'
                }`}
                onClick={() => {
                  setLoginMethod('phone_code')
                  setNeedsTotp(false)
                  setTotpModalOpen(false)
                }}
              >
                {t('auth.phoneLogin')}
              </button>
            ) : null}
          </div>

          <Form
            form={loginForm}
            layout="vertical"
            requiredSymbol={false}
            onValuesChange={(patch) => {
              if (
                patch.email !== undefined ||
                patch.phone !== undefined ||
                patch.password !== undefined ||
                patch.emailCode !== undefined ||
                patch.smsCode !== undefined
              ) {
                setNeedsTotp(false)
                setTotpModalOpen(false)
                setNeedsDeviceVerify(false)
              }
            }}
            onSubmit={(v) => openCaptchaGate((proof) => handleLogin(v, proof), { formSubmit: true })}
          >
            {loginMethod === 'phone_code' ? (
              <FormItem
                label={<span className="text-sm text-neutral-600">{t('auth.phone')}</span>}
                field="phone"
                rules={[{ required: true, message: t('auth.phoneRequired') }]}
                style={{ marginBottom: 18 }}
              >
                <Input
                  size="lg"
                  variant="filled"
                  placeholder={t('auth.phonePlaceholder')}
                  prefix={<IconUser className="text-neutral-400" />}
                />
              </FormItem>
            ) : (
              <FormItem
                label={<span className="text-sm text-neutral-600">{t('auth.usernameOrEmail')}</span>}
                field="email"
                rules={[{ required: true, message: t('auth.usernameOrEmailRequired') }]}
                style={{ marginBottom: 18 }}
              >
                <Input
                  size="lg"
                  variant="filled"
                  placeholder={t('auth.emailInputPlaceholder')}
                  prefix={<Mail className="h-4 w-4 text-neutral-400" strokeWidth={1.75} />}
                />
              </FormItem>
            )}
            {loginMethod === 'password' ? (
              <>
                <FormItem
                  label={<span className="text-sm text-neutral-600">{t('auth.password')}</span>}
                  field="password"
                  rules={[{ required: true, message: t('auth.passwordRequired') }]}
                  style={{ marginBottom: 8 }}
                >
                  <Input.Password
                    size="lg"
                    variant="filled"
                    placeholder={t('auth.passwordInputPlaceholder')}
                    prefix={<Lock className="h-4 w-4 text-neutral-400" strokeWidth={1.75} />}
                  />
                </FormItem>
                <div className="mb-4 flex items-center justify-between">
                  <span className="text-sm text-neutral-500">
                    {t('auth.noAccountYet')}{' '}
                    {tenantSelfRegisterEnabled ? (
                      <button
                        type="button"
                        className="text-sm text-blue-600 transition-colors hover:text-blue-800"
                        onClick={() => setPageMode('register')}
                      >
                        {t('auth.registerAccount')}
                      </button>
                    ) : (
                      <span className="text-sm text-neutral-400">{t('auth.registerDisabled')}</span>
                    )}
                  </span>
                  <button
                    type="button"
                    className="text-sm text-neutral-500 transition-colors hover:text-neutral-800"
                    onClick={() => setShowForgotPassword(true)}
                  >
                    {t('auth.forgotPassword')}
                  </button>
                </div>
              </>
            ) : loginMethod === 'email_code' ? (
              <FormItem
                label={<span className="text-sm text-neutral-600">{t('auth.emailCode')}</span>}
                field="emailCode"
                rules={[{ required: true, message: t('auth.emailCodeRequired') }]}
                style={{ marginBottom: 18 }}
              >
                <Input
                  size="lg"
                  variant="filled"
                  maxLength={6}
                  placeholder={t('auth.emailCodePlaceholder')}
                  suffix={
                    <button
                      type="button"
                      disabled={codeCooldown > 0 || sendingCode}
                      className="whitespace-nowrap text-sm text-neutral-600 transition-colors hover:text-neutral-900 disabled:cursor-not-allowed disabled:text-neutral-300"
                      onClick={() => openCaptchaGate((proof) => void handleSendEmailCode(proof))}
                    >
                      {codeCooldown > 0
                        ? t('auth.emailCodeResendIn', { seconds: codeCooldown })
                        : t('auth.sendEmailCode')}
                    </button>
                  }
                />
              </FormItem>
            ) : (
              <FormItem
                label={<span className="text-sm text-neutral-600">{t('auth.smsCode')}</span>}
                field="smsCode"
                rules={[{ required: true, message: t('auth.smsCodeRequired') }]}
                style={{ marginBottom: 18 }}
              >
                <Input
                  size="lg"
                  variant="filled"
                  maxLength={6}
                  placeholder={t('auth.smsCodePlaceholder')}
                  suffix={
                    <button
                      type="button"
                      disabled={codeCooldown > 0 || sendingCode}
                      className="whitespace-nowrap text-sm text-neutral-600 transition-colors hover:text-neutral-900 disabled:cursor-not-allowed disabled:text-neutral-300"
                      onClick={() => openCaptchaGate((proof) => void handleSendSMSCode(proof))}
                    >
                      {codeCooldown > 0
                        ? t('auth.emailCodeResendIn', { seconds: codeCooldown })
                        : t('auth.sendSMSCode')}
                    </button>
                  }
                />
              </FormItem>
            )}
            {(loginMethod === 'email_code' || loginMethod === 'phone_code') && (
              <div className="mb-4">
                <span className="text-sm text-neutral-500">
                  {t('auth.noAccountYet')}{' '}
                  {tenantSelfRegisterEnabled ? (
                    <button
                      type="button"
                      className="text-sm text-blue-600 transition-colors hover:text-blue-800"
                      onClick={() => setPageMode('register')}
                    >
                      {t('auth.registerAccount')}
                    </button>
                  ) : (
                    <span className="text-sm text-neutral-400">{t('auth.registerDisabled')}</span>
                  )}
                </span>
              </div>
            )}
            {(needsDeviceVerify || needsRemoteVerify) && (
              <div className="mb-5 rounded-xl border border-amber-200 bg-amber-50 px-4 py-4">
                <p className="text-sm font-medium text-amber-900">
                  {needsRemoteVerify && !needsDeviceVerify ? t('auth.remoteLoginTitle') : t('auth.newDeviceTitle')}
                </p>
                <p className="mt-1 text-sm leading-relaxed text-amber-800">
                  {needsRemoteVerify && !needsDeviceVerify ? t('auth.remoteLoginDesc') : t('auth.newDeviceDesc')}
                </p>
                {voiceprintAvailable ? (
                  <div className="mt-3 flex gap-2">
                    <button
                      type="button"
                      className={`rounded-lg px-3 py-1.5 text-sm ${deviceVerifyMethod === 'email' ? 'bg-white font-medium shadow-sm' : 'text-amber-800'}`}
                      onClick={() => setDeviceVerifyMethod('email')}
                    >
                      {t('auth.voiceprintVerifyMethodEmail')}
                    </button>
                    <button
                      type="button"
                      className={`rounded-lg px-3 py-1.5 text-sm ${deviceVerifyMethod === 'voiceprint' ? 'bg-white font-medium shadow-sm' : 'text-amber-800'}`}
                      onClick={() => setDeviceVerifyMethod('voiceprint')}
                    >
                      {t('auth.voiceprintVerifyMethodVoice')}
                    </button>
                  </div>
                ) : null}
                {deviceVerifyMethod === 'email' || !voiceprintAvailable ? (
                  <>
                    <p className="mt-3 text-xs font-medium uppercase tracking-wide text-amber-700">
                      {t('auth.verifyMethodEmail')}
                    </p>
                    <FormItem
                      label={<span className="text-sm text-neutral-600">{t('auth.deviceVerifyCode')}</span>}
                      field="deviceVerifyCode"
                      rules={
                        deviceVerifyMethod === 'email' || !voiceprintAvailable
                          ? [{ required: true, message: t('auth.deviceVerifyCodeRequired') }]
                          : []
                      }
                      style={{ marginBottom: 0, marginTop: 12 }}
                    >
                      <Input
                        maxLength={6}
                        size="lg"
                        variant="filled"
                        placeholder={t('auth.deviceVerifyCodePlaceholder')}
                        suffix={
                          <button
                            type="button"
                            disabled={deviceCodeCooldown > 0 || sendingDeviceCode}
                            className="whitespace-nowrap text-sm text-neutral-600 transition-colors hover:text-neutral-900 disabled:cursor-not-allowed disabled:text-neutral-300"
                            onClick={() => openCaptchaGate((proof) => void handleSendDeviceVerifyCode(proof))}
                          >
                            {deviceCodeCooldown > 0
                              ? t('auth.emailCodeResendIn', { seconds: deviceCodeCooldown })
                              : t('auth.sendDeviceVerifyCode')}
                          </button>
                        }
                      />
                    </FormItem>
                  </>
                ) : (
                  <div className="mt-3 space-y-2">
                    <p className="text-xs font-medium uppercase tracking-wide text-amber-700">
                      {t('auth.voiceprintVerifyTitle')}
                    </p>
                    <p className="text-sm text-amber-800">{t('auth.voiceprintVerifyDesc')}</p>
                    <VoiceprintAudioRecorder
                      mode="record-only"
                      onAudioChange={setVoiceprintLoginFile}
                    />
                  </div>
                )}
                <label className="mt-3 flex cursor-pointer items-center gap-2 text-sm text-amber-900">
                  <input
                    type="checkbox"
                    checked={trustDeviceFor7Days}
                    onChange={(e) => setTrustDeviceFor7Days(e.target.checked)}
                    className="h-4 w-4 rounded border-amber-300"
                  />
                  {t('auth.trustDeviceFor7Days')}
                </label>
              </div>
            )}
            {needsTotp && !totpModalOpen && (
              <p className="mb-4 rounded-xl bg-blue-50 px-4 py-3 text-sm text-blue-800">{t('auth.totpRequiredHint')}</p>
            )}
            {deletionRevokeVisible && (
              <p className="mb-4 rounded-xl bg-amber-50 px-4 py-3 text-sm text-amber-800">
                {t('auth.deletionRevokeHint')}{' '}
                <Link to="/account/deletion/revoke" className="font-medium underline">
                  {t('auth.goRevokeDeletion')}
                </Link>
              </p>
            )}
            <Button type="primary" htmlType="submit" long size="lg" className={submitBtnClass} loading={formSubmitting}>
              {t('auth.login')}
            </Button>
            {githubOAuthEnabled && (
              <Button
                type="outline"
                long
                size="lg"
                className="mt-3 !h-11 !rounded-xl"
                onClick={() => {
                  window.location.href = getGitHubLoginUrl(getOrCreateDeviceId())
                }}
              >
                <span className="inline-flex items-center justify-center gap-2">
                  <Github className="h-4 w-4" strokeWidth={1.75} />
                  {t('auth.loginWithGitHub')}
                </span>
              </Button>
            )}
          </Form>
        </>
        )
      ) : !tenantSelfRegisterEnabled ? (
        <>
          <h1 className="mb-2 text-center text-2xl font-semibold tracking-tight text-neutral-900">
            {t('auth.registerAccount')}
          </h1>
          <Alert type="warning" showIcon className="mb-4" content={t('auth.registerDisabled')} />
          <div className="text-center">
            <button
              type="button"
              className="text-sm text-blue-600 transition-colors hover:text-blue-800"
              onClick={() => setPageMode('login')}
            >
              {t('auth.goLogin')}
            </button>
          </div>
        </>
      ) : (
        <>
          <h1 className="mb-2 text-center text-2xl font-semibold tracking-tight text-neutral-900">
            {t('auth.registerAccount')}
          </h1>
          <p className="mb-4 text-center text-sm leading-relaxed text-neutral-500">{t('auth.registerSubtitle')}</p>
          <Form
            form={registerForm}
            layout="vertical"
            requiredSymbol={false}
            onSubmit={(v) => openCaptchaGate((proof) => handleRegister(v, proof), { formSubmit: true })}
          >
            <FormItem
              label={<span className="text-sm text-neutral-600">{t('auth.companyNameLabel')}</span>}
              field="companyName"
              rules={[{ required: true, message: t('auth.companyNameRequired') }]}
              style={{ marginBottom: 14 }}
            >
              <Input
                size="lg"
                variant="filled"
                placeholder={t('auth.companyNamePlaceholder')}
                autoComplete="organization"
                prefix={<Building2 className="h-4 w-4 text-neutral-400" strokeWidth={1.75} />}
              />
            </FormItem>
            <FormItem
              label={<span className="text-sm text-neutral-600">{t('auth.adminEmail')}</span>}
              field="adminEmail"
              rules={[{ required: true, message: t('auth.email') }]}
              style={{ marginBottom: 14 }}
            >
              <Input
                size="lg"
                variant="filled"
                placeholder={t('auth.adminEmailPlaceholder')}
                autoComplete="email"
                prefix={<Mail className="h-4 w-4 text-neutral-400" strokeWidth={1.75} />}
              />
            </FormItem>
            <FormItem
              label={<span className="text-sm text-neutral-600">{t('auth.adminPasswordLabel')}</span>}
              field="adminPassword"
              rules={[
                { required: true, message: t('auth.passwordMin8') },
                { minLength: 8, message: t('auth.passwordMin8Short') },
              ]}
              style={{ marginBottom: 20 }}
            >
              <Input.Password
                size="lg"
                variant="filled"
                placeholder={t('auth.passwordPlaceholder')}
                autoComplete="new-password"
                prefix={<Lock className="h-4 w-4 text-neutral-400" strokeWidth={1.75} />}
              />
            </FormItem>
            <Button type="primary" htmlType="submit" long size="lg" className={submitBtnClass} loading={formSubmitting}>
              {t('auth.registerSubmit')}
            </Button>
            <div className="mt-3 text-center">
              <span className="text-sm text-neutral-500">{t('auth.hasAccount')}{' '}</span>
              <button
                type="button"
                className="text-sm text-blue-600 transition-colors hover:text-blue-800"
                onClick={() => setPageMode('login')}
              >
                {t('auth.goLogin')}
              </button>
            </div>
          </Form>
        </>
      )}
    </AuthSplitLayout>

      <CaptchaVerifyModal
        open={captchaModalOpen}
        onClose={() => {
          setCaptchaModalOpen(false)
          pendingCaptchaAction.current = null
          pendingFormSubmit.current = false
        }}
        onVerified={handleCaptchaVerified}
      />

      <Modal
        title={t('auth.totpModalTitle')}
        visible={totpModalOpen}
        onCancel={() => {
          setTotpModalOpen(false)
          totpForm.resetFields()
        }}
        footer={null}
        unmountOnExit
      >
        <p className="mb-4 text-sm text-neutral-500">{t('auth.totpModalDesc')}</p>
        <Form form={totpForm} layout="vertical" onSubmit={handleTotpConfirm}>
          <FormItem
            label={t('auth.totpCodeOrRecovery')}
            field="totpCode"
            rules={[{ required: true, message: t('auth.totpCodeRequired') }]}
          >
            <Input maxLength={16} size="lg" variant="filled" placeholder={t('auth.totpCodeOrRecoveryPlaceholder')} />
          </FormItem>
          <div className="flex justify-end gap-2">
            <Button type="outline" onClick={() => setTotpModalOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button type="primary" htmlType="submit">
              {t('auth.confirmLogin')}
            </Button>
          </div>
        </Form>
      </Modal>
    </>
  )
}
