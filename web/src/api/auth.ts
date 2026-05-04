import { post, get, ApiResponse } from '@/utils/request'
import { getUserServiceBaseURL } from '@/config/apiConfig'

const userServiceConfig = {
  baseURL: getUserServiceBaseURL(),
}

// 用户注册表单类型
export interface RegisterUserForm {
  email: string
  password: string
  displayName?: string
  firstName?: string
  lastName?: string
  locale?: string
  timezone?: string
  source?: string
  captchaId?: string
  captchaCode?: string
  captchaType?: 'image' | 'click'
  captchaData?: string
  mouseTrack?: string       // 鼠标轨迹数据（JSON字符串）
  formFillTime?: number     // 表单填写时间（毫秒）
  keystrokePattern?: string // 按键模式数据（JSON字符串）
}

// 邮箱验证码注册表单类型
export interface EmailRegisterForm {
  email: string
  password: string
  userName: string
  displayName: string
  code: string
  firstName?: string
  lastName?: string
  locale?: string
  timezone?: string
  source?: string
  captchaId?: string
  captchaCode?: string
  captchaType?: 'image' | 'click'
  captchaData?: string
  // 智能风控字段
  mouseTrack?: string       // 鼠标轨迹数据（JSON字符串）
  formFillTime?: number     // 表单填写时间（毫秒）
  keystrokePattern?: string // 按键模式数据（JSON字符串）
}

// 验证码响应类型
export interface CaptchaResponse {
  id: string
  type?: 'image' | 'click'
  image: string
  count?: number
  tolerance?: number
  words?: string[]
}

// 发送邮箱验证码请求类型
export interface SendEmailCodeRequest {
  email: string
  clientIp?: string
  userAgent?: string
}

// 用户登录表单类型
export interface LoginForm {
  email: string
  password: string
  twoFactorCode?: string
}

// 密码登录表单类型
export interface PasswordLoginForm {
  email: string
  password: string
  timezone?: string
  remember?: boolean
  authToken?: boolean
  twoFactorCode?: string
  captchaId?: string
  captchaCode?: string
  captchaType?: 'image' | 'click'
  captchaData?: string
}

// 邮箱验证码登录表单类型
export interface EmailCodeLoginForm {
  email: string
  code: string
  timezone?: string
  remember?: boolean
  authToken?: boolean
  captchaId?: string
  captchaCode?: string
  captchaType?: 'image' | 'click'
  captchaData?: string
}

// 登录响应数据类型
export interface LoginResponseData {
  token?: string
  refreshToken?: string
  user?: {
    id?: number | string
    createdAt?: string
    updatedAt?: string
    displayName?: string
    DisplayName?: string
    email?: string
    emailNotifications?: boolean
    firstName?: string
    hasFilledDetails?: boolean
    lastLogin?: string
    lastName?: string
    timezone?: string
    token?: string
    authToken?: string
    AuthToken?: string
    requiresTwoFactor?: boolean
    [key: string]: any
  }
  createdAt?: string
  updatedAt?: string
  displayName?: string
  DisplayName?: string
  email?: string
  emailNotifications?: boolean
  firstName?: string
  hasFilledDetails?: boolean
  lastLogin?: string
  lastName?: string
  timezone?: string
  requiresTwoFactor?: boolean
  requiresDeviceVerification?: boolean
  deviceId?: string
  message?: string
  suspiciousLogin?: boolean
  accountDeletionPending?: boolean
  accountDeletionEffectiveAt?: string
  [key: string]: any
}

// 注册响应数据类型
export interface RegisterResponseData {
  createdAt?: string
  updatedAt?: string
  email: string
  emailNotifications?: boolean
  firstName?: string
  lastName?: string
  displayName?: string
  timezone?: string
  hasFilledDetails?: boolean
  activation?: boolean
  expired?: string
}

// 用户信息类型
export interface User {
  id?: string | number
  ID?: number
  email: string
  displayName?: string
  firstName?: string
  lastName?: string
  phone?: string
  gender?: string
  city?: string
  region?: string
  extra?: string
  locale?: string
  timezone: string
  avatar?: string
  role?: 'user' | 'admin'
  createdAt: string
  updatedAt: string
  lastLogin: string
  loginCount?: number
  lastPasswordChange?: string
  profileComplete?: number
  hasFilledDetails: boolean
  emailNotifications: boolean
  pushNotifications?: boolean
  twoFactorEnabled?: boolean
  emailVerified?: boolean
  wechatOpenId?: string
  githubId?: string
  githubLogin?: string
  /** 账号状态：active | pending_verification | suspended | banned */
  status?: string
  /** 注册来源：SYSTEM | ADMIN | WECHAT | GITHUB */
  source?: string
  /** 角色 slug（user_roles） */
  roleSlugs?: string[]
  /** 合并后的权限 key（角色 ∪ 用户附加权限） */
  permissionKeys?: string[]
  /** 注销冷静期预计完成永久注销的时间（RFC3339） */
  accountDeletionEffectiveAt?: string | null
  accountDeletionRequestedAt?: string | null
}

// 用户注册
export const registerUser = async (data: RegisterUserForm): Promise<ApiResponse<RegisterResponseData>> => {
  return post<RegisterResponseData>('/auth/register', data, userServiceConfig)
}

// 邮箱验证码注册
export const registerUserByEmail = async (data: EmailRegisterForm): Promise<ApiResponse<RegisterResponseData>> => {
  return post<RegisterResponseData>('/auth/register/email', data, userServiceConfig)
}

// 发送邮箱验证码
export const sendEmailCode = async (data: SendEmailCodeRequest): Promise<ApiResponse<null>> => {
  return post<null>('/auth/send/email', data, userServiceConfig)
}

// 用户登录
export const loginUser = async (data: LoginForm): Promise<ApiResponse<LoginResponseData>> => {
  return post<LoginResponseData>('/auth/login/password', data, userServiceConfig)
}

// 密码登录
export const loginWithPassword = async (data: PasswordLoginForm): Promise<ApiResponse<LoginResponseData>> => {
  return post<LoginResponseData>('/auth/login/password', data, userServiceConfig)
}

// 邮箱验证码登录
export const loginWithEmailCode = async (data: EmailCodeLoginForm): Promise<ApiResponse<LoginResponseData>> => {
  return post<LoginResponseData>('/auth/login/email', data, userServiceConfig)
}

// 发送设备验证码
export const sendDeviceVerificationCode = async (data: { email: string; deviceId: string }): Promise<ApiResponse<null>> => {
  return post('/auth/devices/send-verification', data, userServiceConfig)
}

// 验证设备
export const verifyDevice = async (data: { email: string; deviceId: string; verifyCode: string }): Promise<ApiResponse<null>> => {
  return post('/auth/devices/verify', data, userServiceConfig)
}

// 获取用户信息
export const getUserInfo = async (): Promise<ApiResponse<User>> => {
  return get<User>('/auth/info', userServiceConfig)
}

// 刷新token
export const refreshToken = async (): Promise<ApiResponse<{ token: string; refreshToken: string }>> => {
  return post<{ token: string; refreshToken: string }>('/auth/refresh', {
    refresh_token: localStorage.getItem('refresh_token') || '',
  }, userServiceConfig)
}

// 发送邮箱验证邮件
export const sendEmailVerification = async (): Promise<ApiResponse<null>> => {
  return post<null>('/auth/send-email-verification', undefined, userServiceConfig)
}

// 验证邮箱（通过URL中的token）
export const verifyEmail = async (token: string): Promise<ApiResponse<User>> => {
  return get<User>(`/auth/verify-email?token=${token}`, userServiceConfig)
}

// 登出 - 对应 GET /auth/logout
export const logoutUser = async (next?: string): Promise<ApiResponse<null>> => {
  const params = next ? { next } : undefined
  return get<null>('/auth/logout', { ...userServiceConfig, params })
}

// 获取图形验证码
export const getCaptcha = async (): Promise<ApiResponse<CaptchaResponse>> => {
  return get<CaptchaResponse>('/auth/captcha', userServiceConfig)
}

// 验证图形验证码
export const verifyCaptcha = async (
  id: string,
  code: string,
  type: 'image' | 'click' = 'image'
): Promise<ApiResponse<{ valid: boolean }>> => {
  return post<{ valid: boolean }>('/auth/captcha/verify', { id, code, type }, userServiceConfig)
}

// 忘记密码 - 发送重置密码邮件
export const forgotPassword = async (email: string): Promise<ApiResponse<null>> => {
  return post<null>('/auth/reset-password', { email }, userServiceConfig)
}

// 重置密码确认
export const resetPasswordConfirm = async (token: string, password: string): Promise<ApiResponse<null>> => {
  return post<null>('/auth/reset-password/confirm', { token, password }, userServiceConfig)
}
