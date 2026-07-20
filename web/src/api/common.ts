import { get, post, type ApiResponse } from '@/utils/request'
import type { CaptchaProof, CaptchaGenerateResult, CaptchaType } from '@/api/captcha'

const systemPrefix = '/system'

/** Fetches a captcha. Omit type (or pass random) for server-side random kind. */
export async function generateCaptcha(type?: CaptchaType | 'random'): Promise<ApiResponse<CaptchaGenerateResult>> {
  const resolved = !type || type === 'random' ? 'random' : type
  return get(`${systemPrefix}/captcha/generate?type=${encodeURIComponent(resolved)}`)
}

export async function sendLoginEmailCode(email: string, captcha: CaptchaProof): Promise<ApiResponse<unknown>> {
  return post(`${systemPrefix}/email-codes/login`, { email: email.trim(), ...captcha })
}

export async function sendLoginSMSCode(phone: string, captcha: CaptchaProof): Promise<ApiResponse<unknown>> {
  return post(`${systemPrefix}/sms-codes/login`, { phone: phone.trim(), ...captcha })
}

export async function sendDeviceVerifyEmailCode(body: {
  email: string
  loginMethod?: 'password' | 'email_code'
  password?: string
  emailCode?: string
  deviceId?: string
} & CaptchaProof): Promise<ApiResponse<unknown>> {
  return post(`${systemPrefix}/email-codes/login-device-verify`, {
    email: body.email.trim(),
    loginMethod: body.loginMethod,
    password: body.password,
    emailCode: body.emailCode,
    deviceId: body.deviceId,
    captchaId: body.captchaId,
    captchaType: body.captchaType,
    captchaValue: body.captchaValue,
  })
}

export async function sendForgotPasswordEmailCode(email: string, captcha: CaptchaProof): Promise<ApiResponse<unknown>> {
  return post(`${systemPrefix}/email-codes/forgot-password`, { email: email.trim(), ...captcha })
}

export async function sendRevokeDeletionEmailCode(email: string): Promise<ApiResponse<null>> {
  return post(`${systemPrefix}/email-codes/account-deletion-revoke`, { email })
}

export async function sendMePasswordEmailCode(): Promise<ApiResponse<unknown>> {
  return post(`${systemPrefix}/email-codes/me-password`)
}

export async function sendChangeEmailCode(newEmail: string): Promise<ApiResponse<unknown>> {
  return post(`${systemPrefix}/email-codes/me-email`, { newEmail: newEmail.trim() })
}
