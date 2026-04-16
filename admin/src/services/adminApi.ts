import { get, post, put, del, patch } from '@/utils/request'
import { getMainApiBaseURL, getAuthApiBaseURL } from '@/config/apiConfig'
import { handleConfigCacheUpdate } from '@/utils/siteConfigCache'

// 主业务服务 API（不含认证）
const BACKEND_BASE = getMainApiBaseURL()

// ==================== Users API ====================
export interface User {
  id: number
  email: string
  displayName?: string
  avatar?: string
  firstName?: string
  lastName?: string
  role?: string
  enabled: boolean
  phone?: string
  locale?: string
  timezone?: string
  city?: string
  region?: string
  gender?: string
  emailNotifications?: boolean
  pushNotifications?: boolean
  systemNotifications?: boolean
  lastLogin?: string
  loginCount?: number
  permissions?: { [key: string]: string[] }
  createdAt: string
  updatedAt?: string
}

export interface UserListResponse {
  users: User[]
  total: number
  page: number
  pageSize: number
}

export interface ListUsersParams {
  page?: number
  pageSize?: number
  search?: string
  role?: string
  enabled?: string
  hasPhone?: string
}

export interface CreateUserRequest {
  email: string
  password?: string
  displayName?: string
  firstName?: string
  lastName?: string
  role?: string
  enabled?: boolean
  phone?: string
  locale?: string
  timezone?: string
  city?: string
  region?: string
  gender?: string
  avatar?: string
  emailNotifications?: boolean
  pushNotifications?: boolean
  systemNotifications?: boolean
  permissions?: { [key: string]: string[] }
}

export interface UpdateUserRequest {
  email?: string
  password?: string
  displayName?: string
  firstName?: string
  lastName?: string
  role?: string
  enabled?: boolean
  phone?: string
  locale?: string
  timezone?: string
  city?: string
  region?: string
  gender?: string
  avatar?: string
  emailNotifications?: boolean
  pushNotifications?: boolean
  systemNotifications?: boolean
  permissions?: { [key: string]: string[] }
}

// User Management API
export const listUsers = async (params?: ListUsersParams): Promise<UserListResponse> => {
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.pageSize) queryParams.pageSize = params.pageSize
  if (params?.search) queryParams.search = params.search
  if (params?.role) queryParams.role = params.role
  if (params?.enabled) queryParams.enabled = params.enabled
  if (params?.hasPhone) queryParams.hasPhone = params.hasPhone
  
  const res = await get<UserListResponse>(`${BACKEND_BASE}/users`, { params: queryParams })
  return res.data
}

export const getUser = async (id: number): Promise<User> => {
  const res = await get<User>(`${BACKEND_BASE}/users/${id}`)
  return res.data
}

export const createUser = async (data: CreateUserRequest): Promise<User> => {
  const res = await post<User>(`${BACKEND_BASE}/users`, data)
  return res.data
}

export const updateUser = async (id: number, data: UpdateUserRequest): Promise<User> => {
  const res = await put<User>(`${BACKEND_BASE}/users/${id}`, data)
  return res.data
}

export const deleteUser = async (id: number): Promise<void> => {
  await del(`${BACKEND_BASE}/users/${id}`)
}

// Legacy function for backward compatibility
export const getUsers = async (params?: {
  page?: number
  pageSize?: number
  search?: string
  status?: string
  role?: string
  isSuperUser?: boolean
}) => {
  const listParams: ListUsersParams = {
    page: params?.page,
    pageSize: params?.pageSize,
    search: params?.search,
    role: params?.role,
    enabled: params?.status === 'enabled' || params?.status === 'active' ? 'true' : params?.status === 'disabled' ? 'false' : undefined,
  }
  const result = await listUsers(listParams)
  return {
    list: result.users,
    total: result.total,
    page: result.page,
    pageSize: result.pageSize,
  }
}

// ==================== Notifications API ====================
export interface Notification {
  id: number
  title: string
  content: string
  type?: string
  read: boolean
  created_at: string
}

export interface NotificationListResponse {
  list: Notification[]
  total: number
  page: number
  pageSize: number
}

export const getNotifications = async (params?: {
  page?: number
  pageSize?: number
  search?: string
  filter?: 'all' | 'read' | 'unread'
  userId?: string
}) => {
  // 管理后台专用的notification接口
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.pageSize) queryParams.pageSize = params.pageSize
  if (params?.filter === 'read') queryParams.read = true
  if (params?.filter === 'unread') queryParams.read = false
  const res = await get<NotificationListResponse>(`${BACKEND_BASE}/notification`, { params: queryParams })
  // 转换server返回格式
  if (Array.isArray(res.data)) {
    return {
      list: res.data,
      total: res.data.length,
      page: params?.page || 1,
      pageSize: params?.pageSize || 10,
    }
  }
  return res.data
}

export const markAllNotificationsRead = async (userId?: string) => {
  const res = await post(`${BACKEND_BASE}/notification/readAll`, userId ? { userId } : {})
  return res.data
}

export const deleteNotification = async (id: number) => {
  const res = await del(`${BACKEND_BASE}/notification/${id}`)
  return res.data
}

// ==================== Profile API ====================
// 管理后台专用的认证接口
const ADMIN_AUTH_BASE = `${getAuthApiBaseURL()}/auth`

export interface ProfileUpdateRequest {
  displayName?: string
  email?: string
  phone?: string
  timezone?: string
  gender?: string
  city?: string
  region?: string
  extra?: string
  avatar?: string
  emailNotifications?: boolean
  pushNotifications?: boolean
  systemNotifications?: boolean
}

export interface ChangePasswordRequest {
  oldPassword: string
  newPassword: string
}

// 获取当前管理员信息
// ==================== API Keys Management API ====================
export interface APIKey {
  id: number
  name: string
  apiKey: string
  apiSecret?: string // 仅在创建时返回
  createdAt: string
  lastUsedAt?: string
  isActive: boolean
  isBanned: boolean
  expiresAt?: string
}

export interface CreateAPIKeyRequest {
  name: string
  expiresAt?: string
}

export interface CreateAPIKeyResponse {
  id: number
  name: string
  apiKey: string
  apiSecret: string // 仅在创建时返回
  createdAt: string
  expiresAt?: string
}

export const getAPIKeys = async (): Promise<APIKey[]> => {
  const res = await get<{ keys: APIKey[] }>(`${BACKEND_BASE}/auth/api-keys`)
  return res.data.keys
}

export const createAPIKey = async (data: CreateAPIKeyRequest): Promise<CreateAPIKeyResponse> => {
  const res = await post<CreateAPIKeyResponse>(`${BACKEND_BASE}/auth/api-keys`, data)
  return res.data
}

export const deleteAPIKey = async (id: number): Promise<void> => {
  await del(`${BACKEND_BASE}/auth/api-keys/${id}`)
}

export const banAPIKey = async (id: number): Promise<void> => {
  await post(`${BACKEND_BASE}/auth/api-keys/${id}/ban`, {})
}

export const unbanAPIKey = async (id: number): Promise<void> => {
  await post(`${BACKEND_BASE}/auth/api-keys/${id}/unban`, {})
}

// ==================== Security API ====================
// Operation Logs
export interface OperationLog {
  id: number
  user_id: number
  username: string
  action: string
  target: string
  details: string
  ip_address: string
  user_agent: string
  referer: string
  device: string
  browser: string
  operating_system: string
  location: string
  request_method: string
  created_at: string
}

export interface OperationLogListResponse {
  logs: OperationLog[]
  total: number
  page: number
  page_size: number
}

export const getOperationLogs = async (params?: {
  page?: number
  page_size?: number
  user_id?: number
  action?: string
  target?: string
}): Promise<OperationLogListResponse> => {
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.page_size) queryParams.page_size = params.page_size
  if (params?.user_id) queryParams.user_id = params.user_id
  if (params?.action) queryParams.action = params.action
  if (params?.target) queryParams.target = params.target
  const res = await get<OperationLogListResponse>(`${BACKEND_BASE}/security/operation-logs`, { params: queryParams })
  return res.data
}

export const getOperationLog = async (id: number): Promise<{ log: OperationLog }> => {
  const res = await get<{ log: OperationLog }>(`${BACKEND_BASE}/security/operation-logs/${id}`)
  return res.data
}

// User Devices
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

export const getUserDevices = async (userId?: number): Promise<{ devices: UserDevice[] }> => {
  const queryParams: any = {}
  if (userId) queryParams.user_id = userId
  const res = await get<{ devices: UserDevice[] }>(`${BACKEND_BASE}/auth/devices`, { params: queryParams })
  return res.data
}

export const deleteUserDevice = async (deviceId: string): Promise<void> => {
  await del(`${BACKEND_BASE}/auth/devices/${deviceId}`)
}

export const trustUserDevice = async (deviceId: string): Promise<void> => {
  await post(`${BACKEND_BASE}/auth/devices/${deviceId}/trust`, {})
}

export const getUserDevice = async (deviceId: string): Promise<{ device: UserDevice }> => {
  const res = await get<{ device: UserDevice }>(`${BACKEND_BASE}/auth/devices/${deviceId}`)
  return res.data
}

export const updateUserDevice = async (deviceId: string, data: { deviceName?: string }): Promise<void> => {
  await put(`${BACKEND_BASE}/auth/devices/${deviceId}`, data)
}

// Login History
export interface LoginHistory {
  id: number
  userId: number
  email: string
  ipAddress: string
  location: string
  country: string
  city: string
  userAgent: string
  deviceId: string
  loginType: string
  success: boolean
  failureReason: string
  isSuspicious: boolean
  createdAt: string
}

export interface LoginHistoryListResponse {
  histories: LoginHistory[]
  total: number
  page: number
  page_size: number
}

export const getLoginHistory = async (params?: {
  page?: number
  page_size?: number
  user_id?: number
  success?: boolean
  is_suspicious?: boolean
}): Promise<LoginHistoryListResponse> => {
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.page_size) queryParams.page_size = params.page_size
  if (params?.user_id) queryParams.user_id = params.user_id
  if (params?.success !== undefined) queryParams.success = params.success.toString()
  if (params?.is_suspicious !== undefined) queryParams.is_suspicious = params.is_suspicious.toString()
  const res = await get<LoginHistoryListResponse>(`${BACKEND_BASE}/security/login-history`, { params: queryParams })
  return res.data
}

export const getLoginHistoryDetail = async (id: number): Promise<{ history: LoginHistory }> => {
  const res = await get<{ history: LoginHistory }>(`${BACKEND_BASE}/security/login-history/${id}`)
  return res.data
}

// Account Locks
export interface AccountLock {
  id: number
  userId: number
  email: string
  ipAddress: string
  lockedAt: string
  unlockAt: string
  reason: string
  failedAttempts: number
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export interface AccountLockListResponse {
  locks: AccountLock[]
  total: number
  page: number
  page_size: number
}

export const getAccountLocks = async (params?: {
  page?: number
  page_size?: number
  user_id?: number
  email?: string
  is_active?: boolean
}): Promise<AccountLockListResponse> => {
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.page_size) queryParams.page_size = params.page_size
  if (params?.user_id) queryParams.user_id = params.user_id
  if (params?.email) queryParams.email = params.email
  if (params?.is_active !== undefined) queryParams.is_active = params.is_active.toString()
  const res = await get<AccountLockListResponse>(`${BACKEND_BASE}/security/account-locks`, { params: queryParams })
  return res.data
}

export const unlockAccount = async (id: number): Promise<void> => {
  await post(`${BACKEND_BASE}/security/account-locks/${id}/unlock`, {})
}

// ==================== User API ====================
export const getCurrentUser = async () => {
  const res = await get(`${ADMIN_AUTH_BASE}/info`)
  // 后端返回格式: {code: 200, msg: "...", data: {...}}
  if (res.code === 200) {
    return res.data
  }
  throw new Error(res.msg || '获取用户信息失败')
}

// 更新管理员信息
export const updateProfile = async (data: ProfileUpdateRequest) => {
  const res = await put(`${ADMIN_AUTH_BASE}/update`, data)
  // 后端返回格式: {code: 200, msg: "...", data: {...}}
  if (res.code === 200) {
    return res.data
  }
  throw new Error(res.msg || '更新用户信息失败')
}

// 修改管理员密码
export const changePassword = async (data: ChangePasswordRequest) => {
  const res = await post(`${ADMIN_AUTH_BASE}/change-password`, data)
  // 后端返回格式: {code: 200, msg: "...", data: {...}}
  if (res.code === 200) {
    return res.data
  }
  throw new Error(res.msg || '修改密码失败')
}

export const getUserActivity = async () => {
  const res = await get(`${ADMIN_AUTH_BASE}/activity`)
  if (res.code === 200) {
    return res.data
  }
  throw new Error(res.msg || '获取用户活动失败')
}

export const updateNotificationSettings = async (settings: {
  email_notifications?: boolean
  push_notifications?: boolean
  system_notifications?: boolean
}) => {
  const res = await put(`${ADMIN_AUTH_BASE}/notification-settings`, settings)
  if (res.code !== 200) {
    throw new Error(res.msg || '更新通知设置失败')
  }
}

// ==================== Two-Factor Authentication API ====================
export interface TwoFactorStatus {
  enabled: boolean
  hasSecret: boolean
}

export interface TwoFactorSetup {
  secret: string
  qrCode: string
  url: string
}

// 获取2FA状态
export const getTwoFactorStatus = async (): Promise<TwoFactorStatus> => {
  const res = await get<TwoFactorStatus>(`${ADMIN_AUTH_BASE}/two-factor/status`)
  if (res.code === 200) {
    return res.data
  }
  throw new Error(res.msg || '获取2FA状态失败')
}

// 设置2FA（生成密钥和QR码）
export const setupTwoFactor = async (): Promise<TwoFactorSetup> => {
  const res = await post<TwoFactorSetup>(`${ADMIN_AUTH_BASE}/two-factor/setup`, {})
  if (res.code === 200) {
    return res.data
  }
  throw new Error(res.msg || '设置2FA失败')
}

// 启用2FA
export const enableTwoFactor = async (code: string): Promise<void> => {
  const res = await post(`${ADMIN_AUTH_BASE}/two-factor/enable`, { code })
  if (res.code !== 200) {
    throw new Error(res.msg || '启用2FA失败')
  }
}

// 禁用2FA
export const disableTwoFactor = async (code: string): Promise<void> => {
  const res = await post(`${ADMIN_AUTH_BASE}/two-factor/disable`, { code })
  if (res.code !== 200) {
    throw new Error(res.msg || '禁用2FA失败')
  }
}

// 上传管理员头像
// uploadAvatar 已移除 - 不再支持头像上传

// ==================== Storage Management API ====================
const STORAGE_API_BASE = `${BACKEND_BASE}/storage`

// 存储配置信息
export interface StorageInfo {
  storageKind: string
  supported: string[]
}

export const getStorageInfo = async () => {
  const res = await get<StorageInfo>(`${STORAGE_API_BASE}/info`)
  return res.data
}

export const switchStorageType = async (storageKind: string) => {
  const res = await post(`${STORAGE_API_BASE}/switch`, { storageKind })
  return res.data
}

// 存储桶管理
export const listBuckets = async (params?: { tagCondition?: string; shared?: boolean }) => {
  const queryParams: any = {}
  if (params?.tagCondition) queryParams.tagCondition = params.tagCondition
  if (params?.shared !== undefined) queryParams.shared = params.shared
  const res = await get<{ buckets: string[] }>(`${STORAGE_API_BASE}/buckets`, { params: queryParams })
  return res.data.buckets || []
}

export const createBucket = async (bucketName: string, region?: string) => {
  const res = await post(`${STORAGE_API_BASE}/buckets`, { bucketName, region })
  return res.data
}

export const deleteBucket = async (bucketName: string) => {
  const res = await del(`${STORAGE_API_BASE}/buckets/${bucketName}`)
  return res.data
}

export const getBucketDomains = async (bucketName: string) => {
  const res = await get<{ domains: string[] }>(`${STORAGE_API_BASE}/buckets/${bucketName}/domains`)
  return res.data.domains || []
}

export const setBucketPrivate = async (bucketName: string, isPrivate: boolean) => {
  const res = await put(`${STORAGE_API_BASE}/buckets/${bucketName}/private`, { isPrivate })
  return res.data
}

// 文件管理
export interface FileInfo {
  key: string
  size: number
  putTime: number
  mimeType: string
  hash: string
  endUser: string
  type: number
  status: number
  createdAt: string
  updatedAt: string
  publicURL: string
}

export interface ListFilesRequest {
  prefix?: string
  marker?: string
  limit?: number
  delimiter?: string
}

export interface ListFilesResponse {
  files: FileInfo[]
  marker: string
  isTruncated: boolean
  commonPrefixes: string[]
}

export const listFiles = async (bucketName: string, params?: ListFilesRequest) => {
  const queryParams: any = {}
  if (params?.prefix) queryParams.prefix = params.prefix
  if (params?.marker) queryParams.marker = params.marker
  if (params?.limit) queryParams.limit = params.limit
  if (params?.delimiter) queryParams.delimiter = params.delimiter
  const res = await get<ListFilesResponse>(`${STORAGE_API_BASE}/buckets/${bucketName}/files`, { params: queryParams })
  return res.data
}

export const getFileInfo = async (bucketName: string, key: string) => {
  const res = await get<FileInfo>(`${STORAGE_API_BASE}/buckets/${bucketName}/file/info`, {
    params: { key },
  })
  return res.data
}

export interface UploadFileOptions {
  compress?: boolean // 是否压缩（仅对图片）
  quality?: number  // 压缩质量 1-100（默认100，即不压缩）
}

export const uploadFile = async (bucketName: string, key: string, file: File, options?: UploadFileOptions) => {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('key', key)
  if (options?.compress) {
    formData.append('compress', 'true')
    formData.append('quality', (options.quality || 100).toString())
  }
  const res = await post(`${STORAGE_API_BASE}/buckets/${bucketName}/files`, formData, {
    headers: {} as any, // FormData会自动设置Content-Type
  })
  return res.data
}

export const deleteFile = async (bucketName: string, key: string) => {
  const res = await del(`${STORAGE_API_BASE}/buckets/${bucketName}/file?key=${encodeURIComponent(key)}`)
  return res.data
}

export const copyFile = async (bucketName: string, key: string, destBucket: string, destKey: string) => {
  const res = await post(
    `${STORAGE_API_BASE}/buckets/${bucketName}/file/copy?key=${encodeURIComponent(key)}`,
    {
      destBucket,
      destKey,
    }
  )
      return res.data
}

export const moveFile = async (bucketName: string, key: string, destBucket: string, destKey: string) => {
  const res = await post(
    `${STORAGE_API_BASE}/buckets/${bucketName}/file/move?key=${encodeURIComponent(key)}`,
    {
      destBucket,
      destKey,
    }
  )
  return res.data
}

export const getFileURL = async (bucketName: string, key: string, expires?: string) => {
  const queryParams: any = { key }
  if (expires) queryParams.expires = expires
  const res = await get<{ url: string; expires: string }>(
    `${STORAGE_API_BASE}/buckets/${bucketName}/file/url`,
    { params: queryParams }
  )
  return res.data
}

// 图片处理
export interface ProcessImageRequest {
  operation?: string // 单个操作（向后兼容）
  params?: Record<string, any> // 单个操作参数（向后兼容）
  operations?: Array<{ operation: string; params: Record<string, any> }> // 批量操作数组
  destKey?: string // 目标文件路径（可选，不提供则覆盖原文件）
  preview?: boolean // 是否只预览（不保存）
}

export interface ProcessImageResponse {
  key?: string
  fileInfo?: FileInfo
  preview?: string // base64 预览图（当 preview=true 时返回）
  format?: string
  size?: number
}

export const processImage = async (
  bucketName: string,
  key: string,
  request: ProcessImageRequest
): Promise<ProcessImageResponse> => {
  const res = await post<ProcessImageResponse>(
    `${STORAGE_API_BASE}/buckets/${bucketName}/file/process-image?key=${encodeURIComponent(key)}`,
    request
  )
  return res.data
}

// ==================== Configs API ====================
const CONFIGS_API_BASE = `${BACKEND_BASE}/configs`

export interface Config {
  id: number
  key: string
  desc: string
  autoload: boolean
  public: boolean
  format: 'json' | 'yaml' | 'int' | 'float' | 'bool' | 'text'
  value: string
  Value?: string // Backend may return uppercase Value
  createdAt?: string
  updatedAt?: string
}

export interface ListConfigsResponse {
  configs: Config[]
  total: number
  page: number
  size: number
}

export interface ListConfigsParams {
  page?: number
  page_size?: number
  autoload?: boolean
  public?: boolean
  search?: string
}

export const listConfigs = async (params?: ListConfigsParams) => {
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.page_size) queryParams.page_size = params.page_size
  if (params?.autoload !== undefined) queryParams.autoload = params.autoload.toString()
  if (params?.public !== undefined) queryParams.public = params.public.toString()
  if (params?.search) queryParams.search = params.search
  const res = await get<ListConfigsResponse>(CONFIGS_API_BASE, { params: queryParams })
  return res.data
}

// ==================== Public Site Config API ====================
export interface SiteConfig {
  SITE_NAME: string
  SITE_DESCRIPTION: string
  SITE_TERMS_URL: string
  SITE_URL: string
  SITE_LOGO_URL: string
  SHOULD_UPGRADE_DB: boolean
  CENSOR_ENABLED: boolean
}

export const getSiteConfig = async (): Promise<SiteConfig> => {
  const res = await get(`${BACKEND_BASE}/public/site-config`)
  return res.data
}

// ==================== Config Management API ====================
export const getConfig = async (key: string) => {
  const res = await get<{ config: Config }>(`${CONFIGS_API_BASE}/${key}`)
  return res.data.config
}

export const createConfig = async (data: {
  key: string
  desc?: string
  value: string
  format?: 'json' | 'yaml' | 'int' | 'float' | 'bool' | 'text'
  autoload?: boolean
  public?: boolean
}) => {
  const res = await post<{ config: Config }>(CONFIGS_API_BASE, data)
  
  // 如果创建的是站点配置，清除缓存
  handleConfigCacheUpdate(data.key, 'create')
  
  return res.data.config
}

export const updateConfig = async (key: string, data: {
  desc?: string
  value?: string
  format?: 'json' | 'yaml' | 'int' | 'float' | 'bool' | 'text'
  autoload?: boolean
  public?: boolean
}) => {
  const res = await put<{ config: Config }>(`${CONFIGS_API_BASE}/${key}`, data)
  
  // 如果更新的是站点配置，清除缓存
  handleConfigCacheUpdate(key, 'update')
  
  return res.data.config
}

export const deleteConfig = async (key: string) => {
  const res = await del(`${CONFIGS_API_BASE}/${key}`)
  
  // 如果删除的是站点配置，清除缓存
  handleConfigCacheUpdate(key, 'delete')
  
  return res.data
}

export interface BatchUpdateConfig {
  key: string
  desc?: string
  value?: string
  format?: 'json' | 'yaml' | 'int' | 'float' | 'bool' | 'text'
  autoload?: boolean
  public?: boolean
}

export const batchUpdateConfigs = async (configs: BatchUpdateConfig[]) => {
  const res = await put<{ updated: number; failed: number; errors: string[] }>(
    `${CONFIGS_API_BASE}/batch`,
    { configs }
  )
  
  // 检查是否有站点配置被更新，如果有则清除缓存
  configs.forEach(config => {
    handleConfigCacheUpdate(config.key, 'update')
  })
  
  return res.data
}

// ==================== OAuth Clients API ====================
const OAUTH_CLIENTS_API_BASE = `${BACKEND_BASE}/oauth-clients`

export interface OAuthClient {
  id: number
  clientId: string
  clientSecret: string
  name: string
  redirectUri: string
  status: number
  createdAt: string
}

export interface OAuthClientListResponse {
  clients: OAuthClient[]
  total: number
  page: number
  pageSize: number
}

export interface ListOAuthClientsParams {
  page?: number
  pageSize?: number
}

export interface UpsertOAuthClientRequest {
  clientId?: string
  clientSecret?: string
  name?: string
  redirectUri?: string
  status?: number
}

export const listOAuthClients = async (params?: ListOAuthClientsParams): Promise<OAuthClientListResponse> => {
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.pageSize) queryParams.pageSize = params.pageSize
  const res = await get<OAuthClientListResponse>(OAUTH_CLIENTS_API_BASE, { params: queryParams })
  return res.data
}

export const getOAuthClient = async (id: number): Promise<OAuthClient> => {
  const res = await get<OAuthClient>(`${OAUTH_CLIENTS_API_BASE}/${id}`)
  return res.data
}

export const createOAuthClient = async (data: UpsertOAuthClientRequest): Promise<OAuthClient> => {
  const res = await post<OAuthClient>(OAUTH_CLIENTS_API_BASE, data)
  return res.data
}

export const updateOAuthClient = async (id: number, data: UpsertOAuthClientRequest): Promise<OAuthClient> => {
  const res = await put<OAuthClient>(`${OAUTH_CLIENTS_API_BASE}/${id}`, data)
  return res.data
}

export const deleteOAuthClient = async (id: number): Promise<void> => {
  await del(`${OAUTH_CLIENTS_API_BASE}/${id}`)
}

// ==================== Admin Groups API ====================
const ADMIN_GROUPS_API_BASE = `${BACKEND_BASE}/admin/groups`

export interface AdminGroup {
  id: number
  name: string
  type: string
  extra?: string
  isArchived: boolean
  createdAt: string
  updatedAt: string
  creatorId: number
  memberCount?: number
}

export interface AdminGroupListResponse {
  groups: AdminGroup[]
  total: number
  page: number
  pageSize: number
}

export const listAdminGroups = async (params?: {
  page?: number
  pageSize?: number
  search?: string
  type?: string
  isArchived?: boolean
}): Promise<AdminGroupListResponse> => {
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.pageSize) queryParams.pageSize = params.pageSize
  if (params?.search) queryParams.search = params.search
  if (params?.type) queryParams.type = params.type
  if (params?.isArchived !== undefined) queryParams.isArchived = params.isArchived.toString()
  const res = await get<AdminGroupListResponse>(ADMIN_GROUPS_API_BASE, { params: queryParams })
  return res.data
}

export const updateAdminGroup = async (id: number, data: Partial<AdminGroup>): Promise<AdminGroup> => {
  const res = await put<AdminGroup>(`${ADMIN_GROUPS_API_BASE}/${id}`, data)
  return res.data
}

export const deleteAdminGroup = async (id: number): Promise<void> => {
  await del(`${ADMIN_GROUPS_API_BASE}/${id}`)
}

// ==================== Admin Credentials API ====================
const ADMIN_CREDENTIALS_API_BASE = `${BACKEND_BASE}/admin/credentials`

export interface AdminCredential {
  id: number
  userId: number
  name: string
  apiKey: string
  status: 'active' | 'banned' | 'suspended'
  llmProvider?: string
  usageCount: number
  lastUsedAt?: string
  createdAt: string
  updatedAt: string
}

export interface AdminCredentialListResponse {
  credentials: AdminCredential[]
  total: number
  page: number
  pageSize: number
}

export const listAdminCredentials = async (params?: {
  page?: number
  pageSize?: number
  search?: string
  status?: string
  user_id?: number
}): Promise<AdminCredentialListResponse> => {
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.pageSize) queryParams.pageSize = params.pageSize
  if (params?.search) queryParams.search = params.search
  if (params?.status) queryParams.status = params.status
  if (params?.user_id) queryParams.user_id = params.user_id
  const res = await get<AdminCredentialListResponse>(ADMIN_CREDENTIALS_API_BASE, { params: queryParams })
  return res.data
}

export const updateAdminCredentialStatus = async (
  id: number,
  data: { status: 'active' | 'banned' | 'suspended'; bannedReason?: string }
): Promise<AdminCredential> => {
  const res = await patch<AdminCredential>(`${ADMIN_CREDENTIALS_API_BASE}/${id}/status`, data)
  return res.data
}

export const deleteAdminCredential = async (id: number): Promise<void> => {
  await del(`${ADMIN_CREDENTIALS_API_BASE}/${id}`)
}

// ==================== Admin JS Templates API ====================
const ADMIN_JS_TEMPLATES_API_BASE = `${BACKEND_BASE}/admin/js-templates`

export interface AdminJSTemplate {
  id: string
  jsSourceId: string
  name: string
  type: string
  status: string
  user_id: number
  group_id?: number
  version: number
  created_at: string
  updated_at: string
}

export interface AdminJSTemplateListResponse {
  templates: AdminJSTemplate[]
  total: number
  page: number
  pageSize: number
}

export const listAdminJSTemplates = async (params?: {
  page?: number
  pageSize?: number
  search?: string
  status?: string
  type?: string
  user_id?: number
}): Promise<AdminJSTemplateListResponse> => {
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.pageSize) queryParams.pageSize = params.pageSize
  if (params?.search) queryParams.search = params.search
  if (params?.status) queryParams.status = params.status
  if (params?.type) queryParams.type = params.type
  if (params?.user_id) queryParams.user_id = params.user_id
  const res = await get<AdminJSTemplateListResponse>(ADMIN_JS_TEMPLATES_API_BASE, { params: queryParams })
  return res.data
}

export const updateAdminJSTemplate = async (id: string, data: Partial<AdminJSTemplate>) => {
  const res = await put(`${ADMIN_JS_TEMPLATES_API_BASE}/${id}`, data)
  return res.data
}

export const deleteAdminJSTemplate = async (id: string): Promise<void> => {
  await del(`${ADMIN_JS_TEMPLATES_API_BASE}/${id}`)
}

// ==================== Admin Bills API ====================
const ADMIN_BILLS_API_BASE = `${BACKEND_BASE}/admin/bills`

export interface AdminBill {
  id: number
  userId: number
  groupId?: number
  credentialId?: number
  billNo: string
  title: string
  status: string
  startTime: string
  endTime: string
  totalLLMCalls: number
  totalLLMTokens: number
  totalAPICalls: number
  createdAt: string
}

export interface AdminBillListResponse {
  bills: AdminBill[]
  total: number
  page: number
  pageSize: number
}

export const listAdminBills = async (params?: {
  page?: number
  pageSize?: number
  search?: string
  status?: string
  user_id?: number
  group_id?: number
}): Promise<AdminBillListResponse> => {
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.pageSize) queryParams.pageSize = params.pageSize
  if (params?.search) queryParams.search = params.search
  if (params?.status) queryParams.status = params.status
  if (params?.user_id) queryParams.user_id = params.user_id
  if (params?.group_id) queryParams.group_id = params.group_id
  const res = await get<AdminBillListResponse>(ADMIN_BILLS_API_BASE, { params: queryParams })
  return res.data
}

export const updateAdminBill = async (id: number, data: { title?: string; notes?: string; status?: string }) => {
  const res = await put(`${ADMIN_BILLS_API_BASE}/${id}`, data)
  return res.data
}

export const deleteAdminBill = async (id: number): Promise<void> => {
  await del(`${ADMIN_BILLS_API_BASE}/${id}`)
}

// ==================== Admin Assistants API ====================
const ADMIN_ASSISTANTS_API_BASE = `${BACKEND_BASE}/admin/assistants`

export interface AdminAssistant {
  id: number
  userId: number
  groupId?: number
  name: string
  description?: string
  systemPrompt?: string
  temperature?: number
  maxTokens?: number
  language?: string
  speaker?: string
  ttsProvider?: string
  llmModel?: string
  enableGraphMemory?: boolean
  enableJSONOutput?: boolean
  createdAt: string
  updatedAt?: string
}

export interface AdminAssistantListResponse {
  assistants: AdminAssistant[]
  total: number
  page: number
  pageSize: number
}

export const listAdminAssistants = async (params?: { page?: number; pageSize?: number; search?: string }) => {
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.pageSize) queryParams.pageSize = params.pageSize
  if (params?.search) queryParams.search = params.search
  const res = await get<AdminAssistantListResponse>(ADMIN_ASSISTANTS_API_BASE, { params: queryParams })
  return res.data
}

export const getAdminAssistant = async (id: number): Promise<AdminAssistant> => {
  const res = await get<AdminAssistant>(`${ADMIN_ASSISTANTS_API_BASE}/${id}`)
  return res.data
}

export const updateAdminAssistant = async (id: number, data: Partial<AdminAssistant>) => {
  const res = await put(`${ADMIN_ASSISTANTS_API_BASE}/${id}`, data)
  return res.data
}

export const deleteAdminAssistant = async (id: number): Promise<void> => {
  await del(`${ADMIN_ASSISTANTS_API_BASE}/${id}`)
}



// ==================== File Censor Records API ====================
export interface FileCensorRecord {
  id: number
  userId: number
  fileId: number
  fileName: string
  fileUrl: string
  censorType: string
  provider: string
  status: string
  result: string
  suggestion: string
  label: string
  score: number
  details?: any
  errorMessage?: string
  taskId: string
  requestId: string
  processTime: number
  submittedAt: string
  completedAt?: string
  remark?: string
  createdAt: string
  updatedAt: string
}

export interface FileCensorRecordListResponse {
  records: FileCensorRecord[]
  total: number
  page: number
  page_size: number
}

export const getFileCensorRecords = async (params?: {
  page?: number
  page_size?: number
  user_id?: number
  censor_type?: string
  provider?: string
  status?: string
  result?: string
}): Promise<FileCensorRecordListResponse> => {
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.page_size) queryParams.page_size = params.page_size
  if (params?.user_id) queryParams.user_id = params.user_id
  if (params?.censor_type) queryParams.censor_type = params.censor_type
  if (params?.provider) queryParams.provider = params.provider
  if (params?.status) queryParams.status = params.status
  if (params?.result) queryParams.result = params.result
  const res = await get<FileCensorRecordListResponse>(`${BACKEND_BASE}/security/censor-records`, { params: queryParams })
  return res.data
}

export const getFileCensorRecord = async (id: number): Promise<{ record: FileCensorRecord }> => {
  const res = await get<{ record: FileCensorRecord }>(`${BACKEND_BASE}/security/censor-records/${id}`)
  return res.data
}

export interface CensorStatistics {
  date: string
  censor_type: string
  provider: string
  total_count: number
  pass_count: number
  review_count: number
  block_count: number
  failed_count: number
  average_score: number
  average_process_time: number
}

export const getCensorStatistics = async (params?: {
  start_date?: string
  end_date?: string
  censor_type?: string
  provider?: string
}): Promise<{ statistics: CensorStatistics[] }> => {
  const queryParams: any = {}
  if (params?.start_date) queryParams.start_date = params.start_date
  if (params?.end_date) queryParams.end_date = params.end_date
  if (params?.censor_type) queryParams.censor_type = params.censor_type
  if (params?.provider) queryParams.provider = params.provider
  const res = await get<{ statistics: CensorStatistics[] }>(`${BACKEND_BASE}/security/censor-statistics`, { params: queryParams })
  return res.data
}

export const updateCensorConfig = async (data: { key: string; value: string }) => {
  const res = await put(`${BACKEND_BASE}/configs/${data.key}`, { value: data.value })
  return res.data
}

export const getCensorConfig = async (key: string) => {
  const res = await get(`${BACKEND_BASE}/configs/${key}`)
  return res.data
}

// ==================== Generic Admin Modules API ====================
export interface AdminModuleListResponse {
  items: Record<string, any>[]
  total: number
  page: number
  pageSize: number
}

export const listAdminVoiceTrainingTasks = async (params?: { page?: number; pageSize?: number; search?: string }) => {
  const res = await get<AdminModuleListResponse>(`${BACKEND_BASE}/admin/voice-training/tasks`, { params })
  return res.data
}
export const getAdminVoiceTrainingTask = async (id: number) => (await get(`${BACKEND_BASE}/admin/voice-training/tasks/${id}`)).data
export const deleteAdminVoiceTrainingTask = async (id: number) => del(`${BACKEND_BASE}/admin/voice-training/tasks/${id}`)

export const listAdminMCPServers = async (params?: { page?: number; pageSize?: number; search?: string }) => {
  const res = await get<AdminModuleListResponse>(`${BACKEND_BASE}/admin/mcp-servers`, { params })
  return res.data
}
export const getAdminMCPServer = async (id: number) => (await get(`${BACKEND_BASE}/admin/mcp-servers/${id}`)).data
export const deleteAdminMCPServer = async (id: number) => del(`${BACKEND_BASE}/admin/mcp-servers/${id}`)

export const listAdminMCPMarketplace = async (params?: { page?: number; pageSize?: number; search?: string }) => {
  const res = await get<AdminModuleListResponse>(`${BACKEND_BASE}/admin/mcp-marketplace`, { params })
  return res.data
}
export const getAdminMCPMarketplace = async (id: number) => (await get(`${BACKEND_BASE}/admin/mcp-marketplace/${id}`)).data
export const deleteAdminMCPMarketplace = async (id: number) => del(`${BACKEND_BASE}/admin/mcp-marketplace/${id}`)

export const listAdminWorkflows = async (params?: { page?: number; pageSize?: number; search?: string }) => {
  const res = await get<AdminModuleListResponse>(`${BACKEND_BASE}/admin/workflows`, { params })
  return res.data
}
export const getAdminWorkflow = async (id: number) => (await get(`${BACKEND_BASE}/admin/workflows/${id}`)).data
export const deleteAdminWorkflow = async (id: number) => del(`${BACKEND_BASE}/admin/workflows/${id}`)

export const listAdminWorkflowPlugins = async (params?: { page?: number; pageSize?: number; search?: string }) => {
  const res = await get<AdminModuleListResponse>(`${BACKEND_BASE}/admin/workflow-plugins`, { params })
  return res.data
}
export const getAdminWorkflowPlugin = async (id: number) => (await get(`${BACKEND_BASE}/admin/workflow-plugins/${id}`)).data
export const deleteAdminWorkflowPlugin = async (id: number) => del(`${BACKEND_BASE}/admin/workflow-plugins/${id}`)

export const listAdminNodePlugins = async (params?: { page?: number; pageSize?: number; search?: string }) => {
  const res = await get<AdminModuleListResponse>(`${BACKEND_BASE}/admin/node-plugins`, { params })
  return res.data
}
export const getAdminNodePlugin = async (id: number) => (await get(`${BACKEND_BASE}/admin/node-plugins/${id}`)).data
export const deleteAdminNodePlugin = async (id: number) => del(`${BACKEND_BASE}/admin/node-plugins/${id}`)

export const listAdminAlerts = async (params?: { page?: number; pageSize?: number; search?: string }) => {
  const res = await get<AdminModuleListResponse>(`${BACKEND_BASE}/admin/alerts`, { params })
  return res.data
}
export const getAdminAlert = async (id: number) => (await get(`${BACKEND_BASE}/admin/alerts/${id}`)).data
export const deleteAdminAlert = async (id: number) => del(`${BACKEND_BASE}/admin/alerts/${id}`)

export const listAdminNotifications = async (params?: { page?: number; pageSize?: number; search?: string }) => {
  const res = await get<AdminModuleListResponse>(`${BACKEND_BASE}/admin/notifications`, { params })
  return res.data
}
export const getAdminNotification = async (id: number) => (await get(`${BACKEND_BASE}/admin/notifications/${id}`)).data
export const deleteAdminNotification = async (id: number) => del(`${BACKEND_BASE}/admin/notifications/${id}`)

export const listAdminKnowledgeBases = async (params?: { page?: number; pageSize?: number; search?: string }) => {
  const res = await get<AdminModuleListResponse>(`${BACKEND_BASE}/admin/knowledge-bases`, { params })
  return res.data
}
export const getAdminKnowledgeBase = async (id: number) => (await get(`${BACKEND_BASE}/admin/knowledge-bases/${id}`)).data
export const deleteAdminKnowledgeBase = async (id: number) => del(`${BACKEND_BASE}/admin/knowledge-bases/${id}`)

export const listAdminDevices = async (params?: { page?: number; pageSize?: number; search?: string }) => {
  const res = await get<AdminModuleListResponse>(`${BACKEND_BASE}/admin/devices`, { params })
  return res.data
}
export const getAdminDevice = async (id: string) => (await get(`${BACKEND_BASE}/admin/devices/${encodeURIComponent(id)}`)).data
export const deleteAdminDevice = async (id: string) => del(`${BACKEND_BASE}/admin/devices/${encodeURIComponent(id)}`)
