import { authGet, authPost } from '@/api/client'
import { buildEncryptedPassword } from '@/utils/passwordEncrypt'

export interface CaptchaData {
  id: string
  type: 'image' | 'click'
  image?: string
  data?: {
    image?: string
    count?: number
    tolerance?: number
    words?: string[]
  }
}

export interface LoginPasswordRequest {
  email: string
  password: string
  captchaId: string
  captchaType: string
  captchaCode?: string
  captchaData?: string
  timezone?: string
  remember?: boolean
  twoFactorCode?: string
}

export interface DeviceVerifyRequest {
  email: string
  deviceId: string
  verifyCode: string
}

export const getCaptcha = async (): Promise<CaptchaData> => {
  const res = await authGet<CaptchaData>('/auth/captcha')
  return res.data
}

export const loginByPassword = async (payload: LoginPasswordRequest) => {
  // 安全加固：永远不向后端发送明文密码，统一在此入口加密。
  const encrypted = await buildEncryptedPassword(payload.password)
  const res = await authPost('/auth/login/password', { ...payload, password: encrypted })
  return res
}

export const sendDeviceVerificationCode = async (payload: { email: string; deviceId: string }) => {
  const res = await authPost('/auth/devices/send-verification', payload)
  return res
}

export const verifyDeviceForLogin = async (payload: DeviceVerifyRequest) => {
  const res = await authPost('/auth/devices/verify', payload)
  return res
}

export const getCurrentAuthUser = async () => {
  const res = await authGet('/auth/info')
  return res
}

