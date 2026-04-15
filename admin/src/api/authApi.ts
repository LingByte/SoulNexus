import { authGet, authPost } from '@/api/client'

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

export const getCaptcha = async (): Promise<CaptchaData> => {
  const res = await authGet<CaptchaData>('/auth/captcha')
  return res.data
}

export const loginByPassword = async (payload: LoginPasswordRequest) => {
  const res = await authPost('/auth/login/password', payload)
  return res
}

export const getCurrentAuthUser = async () => {
  const res = await authGet('/auth/info')
  return res
}

