import { post, type ApiResponse } from '@/utils/request'
import { getApiBaseURL } from '@/config/apiConfig'
import type { CaptchaProof } from '@/api/captcha'

export {
  sendDeviceVerifyEmailCode,
  sendForgotPasswordEmailCode,
  sendLoginEmailCode,
  sendLoginSMSCode,
} from '@/api/common'

type CaptchaPayload = CaptchaProof

export interface TenantAuthTenant {
  id: number
  name: string
  slug: string
  status?: string
}

export interface TenantAuthUser {
  id: number
  tenantId: number
  email: string
  displayName?: string
  status?: string
  avatarUrl?: string
  totpEnabled?: boolean
}

export interface PlatformAdminAuth {
  id: number
  email: string
  displayName?: string
  status?: string
}

export type LoginPrincipal = 'tenant' | 'platform'

/** Same endpoint `/login`; shape depends on principal. */
export type TenantAuthPayload =
  | {
      principal: 'tenant'
      token: string
      expiresIn: number
      tenant: TenantAuthTenant
      user: TenantAuthUser
      permissionCodes?: string[]
      platformAdmin?: undefined
    }
  | {
      principal: 'platform'
      token: string
      expiresIn: number
      platformAdmin: PlatformAdminAuth
      tenant?: undefined
      user?: undefined
    }

export interface TenantRegisterBody extends CaptchaPayload {
  companyName: string
  adminEmail: string
  adminPassword: string
  adminDisplayName?: string
  tenantDescription?: string
  deviceId?: string
}

export interface TenantLoginBody extends CaptchaPayload {
  email?: string
  phone?: string
  password?: string
  emailCode?: string
  smsCode?: string
  totpCode?: string
  loginMethod?: 'password' | 'email_code' | 'phone_code'
  deviceId?: string
  deviceVerifyCode?: string
  voiceprintAudioBase64?: string
  trustDeviceFor7Days?: boolean
}

export async function forgotPassword(body: {
  email: string
  emailCode: string
  newPassword: string
} & CaptchaPayload): Promise<ApiResponse<{ email: string }>> {
  return post<{ email: string }>('/forgot-password', body)
}

export async function registerTenant(body: TenantRegisterBody): Promise<ApiResponse<TenantAuthPayload>> {
  return post<TenantAuthPayload>('/register', body)
}

export async function tenantLogin(body: TenantLoginBody): Promise<ApiResponse<TenantAuthPayload>> {
  return post<TenantAuthPayload>('/login', body)
}

export function getGitHubLoginUrl(deviceId?: string): string {
  const base = getApiBaseURL().replace(/\/$/, '')
  const params = new URLSearchParams()
  if (deviceId) params.set('deviceId', deviceId)
  const q = params.toString()
  return `${base}/oauth/github/login${q ? `?${q}` : ''}`
}

export async function exchangeGitHubOAuth(body: {
  ticket: string
  totpCode?: string
}): Promise<ApiResponse<TenantAuthPayload & { needsTotp?: boolean }>> {
  return post<TenantAuthPayload & { needsTotp?: boolean }>('/oauth/github/exchange', body)
}
