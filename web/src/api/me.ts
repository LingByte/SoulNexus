import { get, post, put, del, type ApiResponse } from '@/utils/request'

export interface MeTenant {
  id: number
  name: string
  slug: string
  status?: string
  voiceMode?: 'pipeline' | 'realtime'
}

export interface MeTenantGroup {
  id: number
  name: string
}

export interface MeUser {
  id: number
  tenantId: number
  email: string
  phone?: string
  username?: string
  displayName?: string
  status?: string
  avatarUrl?: string
  lastLogin?: string
  lastLoginIp?: string
  source?: string
  loginCount?: number
  totpEnabled?: boolean
  receiveEmailNotify?: boolean
  requireDeviceVerify?: boolean
  trustDeviceLoginEnabled?: boolean
  requireRemoteLoginVerify?: boolean
  primaryLoginCity?: string
  sessionIdleTimeoutHours?: number
  sessionMaxLifetimeHours?: number
  tenantGroup?: MeTenantGroup | null
  githubLogin?: string
  voiceprintId?: string
  voiceprintEnrolled?: boolean
  createdAt?: string
}

export interface PlatformAdminMe {
  id: number
  email: string
  displayName?: string
  status?: string
  totpEnabled?: boolean
  receiveEmailNotify?: boolean
  requireDeviceVerify?: boolean
  trustDeviceLoginEnabled?: boolean
  requireRemoteLoginVerify?: boolean
  primaryLoginCity?: string
  sessionIdleTimeoutHours?: number
  sessionMaxLifetimeHours?: number
  githubLogin?: string
}

export type MePayload =
  | {
      principal: 'tenant'
      user: MeUser
      tenant: MeTenant
      /** 菜单与接口权限码（侧边栏过滤）；不在「个人资料」中展示 */
      permissionCodes?: string[]
      platformAdmin?: undefined
    }
  | { principal: 'platform'; platformAdmin: PlatformAdminMe; user?: undefined; tenant?: undefined }

export async function fetchMe(): Promise<ApiResponse<MePayload>> {
  return get<MePayload>('/me')
}

export async function updateMe(body: {
  displayName?: string
  phone?: string
  username?: string
}): Promise<ApiResponse<MeUser | PlatformAdminMe>> {
  return put<MeUser | PlatformAdminMe>('/me', body)
}

export async function updateMyPassword(body: {
  oldPassword?: string
  newPassword: string
  emailCode?: string
  method?: 'password' | 'email_code'
}): Promise<ApiResponse<{ id: number }>> {
  return put<{ id: number }>('/me/password', body)
}

export { sendChangeEmailCode, sendMePasswordEmailCode } from '@/api/common'

export async function changeMyEmail(body: { newEmail: string; emailCode: string }): Promise<ApiResponse<unknown>> {
  return put('/me/email', body)
}

export async function fetchAccountDeletionStatus(): Promise<
  ApiResponse<{ pending: boolean; requestedAt?: string; scheduledDeleteAt?: string; coolingDays: number }>
> {
  return get('/me/account/deletion')
}

export async function requestAccountDeletion(body: {
  method?: 'password' | 'email_code'
  password?: string
  emailCode?: string
}): Promise<ApiResponse<{ pending: boolean; requestedAt?: string; scheduledDeleteAt?: string; coolingDays: number }>> {
  return post('/me/account/deletion/request', body)
}

export async function cancelAccountDeletion(): Promise<ApiResponse<{ pending: boolean; coolingDays: number }>> {
  return post('/me/account/deletion/cancel')
}

export interface UserDeviceItem {
  id: string
  deviceKey: string
  category: string
  limitCategory: string
  displayName: string
  lastIp?: string
  lastLoginCity?: string
  isTrusted: boolean
  sessionActive: boolean
  lastLoginAt?: string
  trustedAt?: string
  isCurrent: boolean
}

export async function fetchMyDevices(page = 1, size = 10): Promise<ApiResponse<{ list: UserDeviceItem[]; total: number; page: number; size: number }>> {
  return get(`/me/devices?page=${page}&size=${size}`)
}

export async function trustMyDevice(id: string): Promise<ApiResponse<{ trusted: boolean }>> {
  return post(`/me/devices/${id}/trust`, {})
}

export async function revokeMyDevice(id: string): Promise<ApiResponse<{ revoked: boolean }>> {
  return post(`/me/devices/${id}/revoke`, {})
}

export async function deleteMyDevice(id: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return del(`/me/devices/${id}`)
}

export async function updateSecurityPreferences(body: {
  receiveEmailNotify?: boolean
  requireDeviceVerify?: boolean
  trustDeviceLoginEnabled?: boolean
  requireRemoteLoginVerify?: boolean
  sessionIdleTimeoutHours?: number
  sessionMaxLifetimeHours?: number
}): Promise<ApiResponse<MeUser | PlatformAdminMe>> {
  return put<MeUser | PlatformAdminMe>('/me/security-preferences', body)
}

export async function revokeAllMySessions(): Promise<ApiResponse<{ revokedAll: boolean }>> {
  return post<{ revokedAll: boolean }>('/me/sessions/revoke-all')
}

export async function uploadMyAvatar(file: File): Promise<ApiResponse<{ avatarUrl: string; user: MeUser }>> {
  const fd = new FormData()
  fd.append('file', file)
  return post<{ avatarUrl: string; user: MeUser }>('/me/avatar', fd)
}

export async function setupTotp(): Promise<
  ApiResponse<{ secret: string; url: string; qrDataUrl: string }>
> {
  return post<{ secret: string; url: string; qrDataUrl: string }>('/me/totp/setup')
}

export async function enableTotp(body: { secret: string; code: string }): Promise<
  ApiResponse<(MeUser | PlatformAdminMe) & { recoveryCodes?: string[] }>
> {
  return post<(MeUser | PlatformAdminMe) & { recoveryCodes?: string[] }>('/me/totp/enable', body)
}

export async function disableTotp(body: { password: string; code: string }): Promise<ApiResponse<MeUser>> {
  return post<MeUser>('/me/totp/disable', body)
}

export async function logoutApi(): Promise<ApiResponse<{ loggedOut: boolean }>> {
  return post<{ loggedOut: boolean }>('/auth/logout')
}

export async function startGitHubBind(): Promise<ApiResponse<{ authorizeUrl: string }>> {
  return get<{ authorizeUrl: string }>('/me/oauth/github/bind')
}

export async function unbindGitHub(): Promise<ApiResponse<{ unbound: boolean }>> {
  return del<{ unbound: boolean }>('/me/oauth/github')
}

export interface MyVoiceprintStatus {
  enrolled: boolean
  profile?: {
    id: number
    name: string
    featureId: string
    status: string
    createdAt?: string
  }
}

export async function fetchMyVoiceprint(): Promise<ApiResponse<MyVoiceprintStatus>> {
  return get<MyVoiceprintStatus>('/me/voiceprint')
}

export async function enrollMyVoiceprint(body: {
  audio?: File
  audioUrl?: string
  name?: string
}): Promise<ApiResponse<unknown>> {
  const fd = new FormData()
  if (body.audio) fd.append('audio', body.audio)
  if (body.audioUrl?.trim()) fd.append('audioUrl', body.audioUrl.trim())
  if (body.name?.trim()) fd.append('name', body.name.trim())
  return post('/me/voiceprint', fd)
}

export async function deleteMyVoiceprint(): Promise<ApiResponse<{ deleted: boolean }>> {
  return del<{ deleted: boolean }>('/me/voiceprint')
}
