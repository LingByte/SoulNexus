import { get, put, post, del, ApiResponse } from '@/utils/request'

// 用户资料更新表单 - 对应后端 UpdateUserRequest
export interface UpdateProfileForm {
  email?: string
  phone?: string
  displayName?: string
  firstName?: string
  lastName?: string
  locale?: string
  timezone?: string
  gender?: string
  city?: string
  region?: string
  extra?: string
  avatar?: string
}

// 用户基本信息更新表单 - 对应后端 UserBasicInfoUpdate
export interface UpdateBasicInfoForm {
  fatherCallName?: string
  motherCallName?: string
  wifiName?: string
  wifiPassword?: string
}

// 用户偏好设置表单
export interface UpdatePreferencesForm {
  emailNotifications?: boolean
  pushNotifications?: boolean
  systemNotifications?: boolean
  autoCleanUnreadEmails?: boolean
}

// 密码修改表单
export interface ChangePasswordForm {
  currentPassword: string
  newPassword: string
  confirmPassword: string
}

// 头像上传响应
export interface AvatarUploadResponse {
  avatar: string
  url: string
}

// 获取用户资料
export const getProfile = async (): Promise<ApiResponse<any>> => {
  return get('/auth/info')
}

// 更新用户资料 - 对应 PUT /auth/update
export const updateProfile = async (data: UpdateProfileForm): Promise<ApiResponse<null>> => {
  return put('/auth/update', data)
}

// 更新用户基本信息 - 对应 POST /auth/update/basic/info
export const updateBasicInfo = async (data: UpdateBasicInfoForm): Promise<ApiResponse<null>> => {
  return post('/auth/update/basic/info', data)
}

// 更新用户偏好设置 - 对应 PUT /auth/update/preferences
export const updatePreferences = async (data: UpdatePreferencesForm): Promise<ApiResponse<null>> => {
  return put('/auth/update/preferences', data)
}

// 修改密码
export const changePassword = async (data: ChangePasswordForm): Promise<ApiResponse<null>> => {
  return post('/auth/change-password', data)
}

// 通过邮箱验证码修改密码
export interface ChangePasswordByEmailForm {
  emailCode: string
  newPassword: string
  confirmPassword: string
}

export const changePasswordByEmail = async (data: ChangePasswordByEmailForm): Promise<ApiResponse<null>> => {
  return post('/auth/change-password/email', data)
}

// 上传头像
export const uploadAvatar = async (file: File): Promise<ApiResponse<AvatarUploadResponse>> => {
  const formData = new FormData()
  formData.append('avatar', file)
  return post('/auth/avatar/upload', formData)
}

// 删除头像

// 两步验证相关接口
export interface TwoFactorSetupResponse {
  secret: string
  qrCode: string
  url: string
}

export interface TwoFactorStatusResponse {
  enabled: boolean
  hasSecret: boolean
}

export interface TwoFactorCodeRequest {
  code: string
}

// 设置两步验证（生成密钥和QR码）
export const setupTwoFactor = async (): Promise<ApiResponse<TwoFactorSetupResponse>> => {
  return post('/auth/two-factor/setup', {})
}

// 启用两步验证
export const enableTwoFactor = async (code: string): Promise<ApiResponse<null>> => {
  return post('/auth/two-factor/enable', { code })
}

// 禁用两步验证
export const disableTwoFactor = async (code: string): Promise<ApiResponse<null>> => {
  return post('/auth/two-factor/disable', { code })
}

// 获取两步验证状态
export const getTwoFactorStatus = async (): Promise<ApiResponse<TwoFactorStatusResponse>> => {
  return get('/auth/two-factor/status')
}

// 活动记录相关接口
export interface ActivityLog {
  id: number
  action: string
  target: string
  details: string
  ipAddress: string
  userAgent: string
  device: string
  browser: string
  os: string
  location: string
  createdAt: string
}

export interface ActivityLogResponse {
  activities: ActivityLog[]
  pagination: {
    page: number
    limit: number
    total: number
    totalPages: number
  }
}

// 获取用户活动记录
export const getUserActivity = async (params?: {
  page?: number
  limit?: number
  action?: string
}): Promise<ApiResponse<ActivityLogResponse>> => {
  const queryParams = new URLSearchParams()
  if (params?.page) queryParams.append('page', params.page.toString())
  if (params?.limit) queryParams.append('limit', params.limit.toString())
  if (params?.action) queryParams.append('action', params.action)
  
  const queryString = queryParams.toString()
  const url = queryString ? `/auth/activity?${queryString}` : '/auth/activity'
  return get(url)
}

// 设备管理相关接口
export interface UserDevice {
  id: number
  userId: number
  deviceId: string
  deviceName: string
  deviceType: string
  os: string
  browser: string
  userAgent: string
  ipAddress: string
  location: string
  isTrusted: boolean
  isActive: boolean
  lastUsedAt: string
  createdAt: string
  updatedAt: string
}

export interface UserDevicesResponse {
  devices: UserDevice[]
}

// 获取用户设备列表
export const getUserDevices = async (): Promise<ApiResponse<UserDevicesResponse>> => {
  return get('/auth/devices')
}

// 删除用户设备
export const deleteUserDevice = async (deviceId: string): Promise<ApiResponse<null>> => {
  return del(`/auth/devices/${deviceId}`)
}

// 信任用户设备
export const trustUserDevice = async (deviceId: string): Promise<ApiResponse<null>> => {
  return post('/auth/devices/trust', { deviceId })
}

// 取消信任用户设备
export const untrustUserDevice = async (deviceId: string): Promise<ApiResponse<null>> => {
  return post('/auth/devices/untrust', { deviceId })
}
