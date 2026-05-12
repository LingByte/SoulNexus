import { get, post, put, del, patch } from '@/utils/request'
import { getMainApiBaseURL, getAuthApiBaseURL } from '@/config/apiConfig'
import { handleConfigCacheUpdate } from '@/utils/siteConfigCache'
import { normalizeAuthUser } from '@/utils/authUserProfile'

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
  status?: string
  source?: string
  phone?: string
  locale?: string
  timezone?: string
  city?: string
  region?: string
  gender?: string
  emailNotifications?: boolean
  pushNotifications?: boolean
  lastLogin?: string
  loginCount?: number
  permissions?: { [key: string]: string[] }
  createdAt: string
  updatedAt?: string
  profile?: Record<string, unknown>
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
  /** 账号状态，如 active、banned */
  status?: string
  /** @deprecated 使用 status；后端仍支持 true/false 映射 */
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
  status?: string
  phone?: string
  locale?: string
  timezone?: string
  city?: string
  region?: string
  gender?: string
  avatar?: string
  emailNotifications?: boolean
  pushNotifications?: boolean
  permissions?: { [key: string]: string[] }
}

export interface UpdateUserRequest {
  email?: string
  password?: string
  displayName?: string
  firstName?: string
  lastName?: string
  role?: string
  status?: string
  phone?: string
  locale?: string
  timezone?: string
  city?: string
  region?: string
  gender?: string
  avatar?: string
  emailNotifications?: boolean
  pushNotifications?: boolean
  permissions?: { [key: string]: string[] }
}

// User Management API
export const listUsers = async (params?: ListUsersParams): Promise<UserListResponse> => {
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.pageSize) queryParams.pageSize = params.pageSize
  if (params?.search) queryParams.search = params.search
  if (params?.role) queryParams.role = params.role
  if (params?.status) queryParams.status = params.status
  if (params?.enabled) queryParams.enabled = params.enabled
  if (params?.hasPhone) queryParams.hasPhone = params.hasPhone
  
  const res = await get<UserListResponse>(`${BACKEND_BASE}/users`, { params: queryParams })
  const body = res.data
  return {
    ...body,
    users: body.users.map((u) => normalizeAuthUser(u as unknown as Record<string, unknown>) as User),
  }
}

export const getUser = async (id: number): Promise<User> => {
  const res = await get<User>(`${BACKEND_BASE}/users/${id}`)
  return normalizeAuthUser(res.data as unknown as Record<string, unknown>) as User
}

export const createUser = async (data: CreateUserRequest): Promise<User> => {
  const res = await post<User>(`${BACKEND_BASE}/users`, data)
  return normalizeAuthUser(res.data as unknown as Record<string, unknown>) as User
}

export const updateUser = async (id: number, data: UpdateUserRequest): Promise<User> => {
  const res = await put<User>(`${BACKEND_BASE}/users/${id}`, data)
  return normalizeAuthUser(res.data as unknown as Record<string, unknown>) as User
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
  let statusFilter: string | undefined
  let enabledLegacy: string | undefined
  switch (params?.status) {
    case 'enabled':
    case 'active':
      statusFilter = 'active'
      break
    case 'disabled':
      enabledLegacy = 'false'
      break
    default:
      if (params?.status) {
        statusFilter = params.status
      }
  }
  const listParams: ListUsersParams = {
    page: params?.page,
    pageSize: params?.pageSize,
    search: params?.search,
    role: params?.role,
    status: statusFilter,
    enabled: enabledLegacy,
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
    return normalizeAuthUser(res.data as Record<string, unknown>)
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
  expiresAt?: string
  tokenQuota?: number
  tokenUsed?: number
  requestQuota?: number
  useNativeQuota?: boolean
  unlimitedQuota?: boolean
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
  data: {
    status: 'active' | 'banned' | 'suspended'
    bannedReason?: string
    expiresAt?: string
    tokenQuota?: number
    requestQuota?: number
    useNativeQuota?: boolean
    unlimitedQuota?: boolean
  }
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

// ==================== Admin Agents API ====================
const ADMIN_AGENTS_API_BASE = `${BACKEND_BASE}/admin/agents`

export interface AdminAssistant {
  id: number
  userId: number
  groupId?: number
  name: string
  description?: string
  systemPrompt?: string
  temperature?: number
  maxTokens?: number
  speaker?: string
  ttsProvider?: string
  llmModel?: string
  enableGraphMemory?: boolean
  enableJSONOutput?: boolean
  createdAt: string
  updatedAt?: string
}

export interface AdminAssistantListResponse {
  agents: AdminAssistant[]
  total: number
  page: number
  pageSize: number
}

export const listAdminAssistants = async (params?: { page?: number; pageSize?: number; search?: string }) => {
  const queryParams: any = {}
  if (params?.page) queryParams.page = params.page
  if (params?.pageSize) queryParams.pageSize = params.pageSize
  if (params?.search) queryParams.search = params.search
  const res = await get<AdminAssistantListResponse>(ADMIN_AGENTS_API_BASE, { params: queryParams })
  return res.data
}

export const getAdminAssistant = async (id: number): Promise<AdminAssistant> => {
  const res = await get<AdminAssistant>(`${ADMIN_ASSISTANTS_API_BASE}/${id}`)
  return res.data
}

export const updateAdminAssistant = async (id: number, data: Partial<AdminAssistant>) => {
  const res = await put(`${ADMIN_AGENTS_API_BASE}/${id}`, data)
  return res.data
}

export const deleteAdminAssistant = async (id: number): Promise<void> => {
  await del(`${ADMIN_AGENTS_API_BASE}/${id}`)
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

export interface AdminAnnouncement {
  id: number
  title: string
  summary?: string
  content: string
  status: 'draft' | 'published' | 'offline'
  pinned?: boolean
  publishAt?: string
  expireAt?: string
  createBy?: string
  updateBy?: string
  createdAt: string
  updatedAt?: string
}

export const listAdminAnnouncements = async (params?: { page?: number; pageSize?: number; search?: string; status?: string }) => {
  const res = await get<{ items: AdminAnnouncement[]; total: number; page: number; pageSize: number }>(`${BACKEND_BASE}/admin/announcements`, { params })
  return res.data
}
export const getAdminAnnouncement = async (id: number) => (await get(`${BACKEND_BASE}/admin/announcements/${id}`)).data
export const createAdminAnnouncement = async (data: {
  title: string
  summary?: string
  content: string
  status?: 'draft' | 'published' | 'offline'
  pinned?: boolean
  publishAt?: string
  expireAt?: string
}) => (await post(`${BACKEND_BASE}/admin/announcements`, data)).data
export const updateAdminAnnouncement = async (id: number, data: {
  title?: string
  summary?: string
  content?: string
  status?: 'draft' | 'published' | 'offline'
  pinned?: boolean
  publishAt?: string
  expireAt?: string
}) => (await put(`${BACKEND_BASE}/admin/announcements/${id}`, data)).data
export const publishAdminAnnouncement = async (id: number) => post(`${BACKEND_BASE}/admin/announcements/${id}/publish`)
export const offlineAdminAnnouncement = async (id: number) => post(`${BACKEND_BASE}/admin/announcements/${id}/offline`)
export const deleteAdminAnnouncement = async (id: number) => del(`${BACKEND_BASE}/admin/announcements/${id}`)

// ==================== Admin Chat / Usage APIs ====================
export interface AdminChatSession {
  id: string
  user_id: string
  agent_id?: number
  agentId?: number
  title: string
  provider: string
  model: string
  status: string
  created_at: string
  updated_at: string
}

export interface AdminChatMessage {
  id: string
  session_id: string
  role: string
  content: string
  token_count: number
  model: string
  provider: string
  request_id: string
  created_at: string
}

export interface AdminLLMUsage {
  id: string
  request_id: string
  session_id: string
  provider: string
  model: string
  request_type: string
  input_tokens: number
  output_tokens: number
  total_tokens: number
  latency_ms: number
  request_content: string
  response_content: string
  user_agent: string
  ip_address: string
  success: boolean
  error_message?: string
  requested_at: string
}

export const listAdminChatSessions = async (params?: { page?: number; pageSize?: number; search?: string }) => {
  const res = await get<{ items: AdminChatSession[]; total: number; page: number; pageSize: number }>(`${BACKEND_BASE}/admin/chat-sessions`, { params })
  return res.data
}

export const listAdminChatMessages = async (params?: { page?: number; pageSize?: number; search?: string; session_id?: string }) => {
  const res = await get<{ items: AdminChatMessage[]; total: number; page: number; pageSize: number }>(`${BACKEND_BASE}/admin/chat-messages`, { params })
  return res.data
}

export const listAdminLLMUsage = async (params?: { page?: number; pageSize?: number; search?: string; session_id?: string; success?: string }) => {
  const res = await get<{ items: AdminLLMUsage[]; total: number; page: number; pageSize: number }>(`${BACKEND_BASE}/admin/llm-usage`, { params })
  return res.data
}

// ==================== Speech Usage (ASR/TTS) ====================
export interface AdminSpeechUsage {
  id: string
  request_id: string
  kind: 'asr' | 'tts' | string
  user_id: number
  token_id: number
  group_id: number
  group_key: string
  channel_id: number
  provider: string
  model: string
  status: number
  success: boolean
  error_msg?: string
  audio_bytes: number
  text_chars: number
  duration_ms: number
  client_ip: string
  user_agent?: string
  req_snap?: string
  resp_snap?: string
  created_at: string
}

export interface AdminSpeechUsageStatsRow {
  kind: string
  provider: string
  total: number
  success: number
  failed: number
}

export const listAdminSpeechUsage = async (params?: {
  page?: number
  pageSize?: number
  search?: string
  kind?: 'asr' | 'tts' | ''
  user_id?: number
  token_id?: number
  group?: string
  channel_id?: number
  provider?: string
  status?: number
  success?: string
  from?: string
  to?: string
}) => {
  const res = await get<{ items: AdminSpeechUsage[]; total: number; page: number; pageSize: number }>(
    `${BACKEND_BASE}/admin/speech-usage`,
    { params },
  )
  return res.data
}

export const getAdminSpeechUsage = async (id: string) =>
  (await get<{ item: AdminSpeechUsage }>(`${BACKEND_BASE}/admin/speech-usage/${encodeURIComponent(id)}`)).data

export const listAdminSpeechUsageStats = async () =>
  (await get<{ items: AdminSpeechUsageStatsRow[] }>(`${BACKEND_BASE}/admin/speech-usage/stats`)).data

// ==================== LLM Channels API ====================
export interface LLMChannelInfo {
  is_multi_key?: boolean
  multi_key_size?: number
  multi_key_status_list?: Record<number, number>
  multi_key_disabled_reason?: Record<number, string>
  multi_key_disabled_time?: Record<number, number>
  multi_key_polling_index?: number
  multi_key_mode?: 'random' | 'polling'
}

export interface LLMChannel {
  id: number
  protocol: string
  type: number
  key: string
  openai_organization?: string | null
  test_model?: string | null
  status: number
  name: string
  weight?: number | null
  created_time?: number
  test_time?: number
  response_time?: number
  base_url?: string | null
  balance?: number
  balance_updated_time?: number
  models: string
  group: string
  used_quota?: number
  model_mapping?: string | null
  status_code_mapping?: string | null
  priority?: number | null
  auto_ban?: number | null
  tag?: string | null
  channel_info?: LLMChannelInfo
}

export interface ListLLMChannelsParams {
  page?: number
  pageSize?: number
  search?: string
  group?: string
  protocol?: string
  status?: string
  mask_key?: string
}

export const listLLMChannels = async (params?: ListLLMChannelsParams) => {
  const res = await get<{ channels: LLMChannel[]; total: number; page: number; pageSize: number }>(`${BACKEND_BASE}/admin/llm-channels`, { params })
  return res.data
}

export const getLLMChannel = async (id: number) => {
  const res = await get<{ channel: LLMChannel }>(`${BACKEND_BASE}/admin/llm-channels/${id}`)
  return res.data.channel
}

export const createLLMChannel = async (body: Partial<LLMChannel>) => {
  const res = await post<{ channel: LLMChannel }>(`${BACKEND_BASE}/admin/llm-channels`, body)
  return res.data.channel
}

export const updateLLMChannel = async (id: number, body: Partial<LLMChannel>) => {
  const res = await put<{ channel: LLMChannel }>(`${BACKEND_BASE}/admin/llm-channels/${id}`, body)
  return res.data.channel
}

export const deleteLLMChannel = async (id: number) => {
  await del(`${BACKEND_BASE}/admin/llm-channels/${id}`)
}

export const syncLLMChannelAbilities = async (id: number) => {
  await post(`${BACKEND_BASE}/admin/llm-channels/${id}/sync-abilities`, {})
}

// ==================== LLM Abilities API ====================
export interface LLMAbility {
  group: string
  model: string
  channel_id: number
  model_meta_id?: number | null
  enabled: boolean
  priority: number
  weight: number
  tag?: string | null
}

export const listLLMAbilities = async (params?: { page?: number; pageSize?: number; group?: string; model?: string; channel_id?: number; enabled?: string }) => {
  const res = await get<{ abilities: LLMAbility[]; total: number; page: number; pageSize: number }>(`${BACKEND_BASE}/admin/llm-abilities`, { params })
  return res.data
}

// ==================== LLM Model Metas API ====================
export interface LLMModelMeta {
  id: number
  model_name: string
  description?: string
  tags?: string
  status: number
  icon_url?: string
  vendor?: string
  sort_order: number
  context_length?: number | null
  max_output_tokens?: number | null
  quota_billing_mode?: string
  quota_model_ratio: number
  quota_prompt_ratio: number
  quota_completion_ratio: number
  quota_cache_read_ratio: number
  created_time?: number
  updated_time?: number
}

export const listLLMModelMetas = async (params?: { page?: number; pageSize?: number; search?: string; vendor?: string; status?: string }) => {
  const res = await get<{ items: LLMModelMeta[]; total: number; page: number; pageSize: number }>(`${BACKEND_BASE}/admin/llm-model-metas`, { params })
  return res.data
}

export const upsertLLMModelMeta = async (body: Partial<LLMModelMeta>) => {
  if (body.id) {
    const res = await put<{ meta: LLMModelMeta }>(`${BACKEND_BASE}/admin/llm-model-metas/${body.id}`, body)
    return res.data.meta
  }
  const res = await post<{ meta: LLMModelMeta }>(`${BACKEND_BASE}/admin/llm-model-metas`, body)
  return res.data.meta
}

export const deleteLLMModelMeta = async (id: number) => {
  await del(`${BACKEND_BASE}/admin/llm-model-metas/${id}`)
}

// ==================== LLM Tokens API ====================
export interface LLMToken {
  id: number
  user_id: number
  name: string
  api_key: string
  type: 'llm' | 'asr' | 'tts'
  status: 'active' | 'disabled' | 'expired'
  group: string
  model_whitelist: string
  unlimited_quota: boolean
  token_quota: number
  token_used: number
  quota_used: number
  request_quota: number
  request_used: number
  expires_at?: string | null
  last_used_at?: string | null
  created_at: string
  updated_at: string
}

export interface ListLLMTokensParams {
  page?: number
  pageSize?: number
  search?: string
  user_id?: number
  group?: string
  status?: string
  show_key?: string
}

export interface LLMTokenWriteReq {
  user_id?: number
  name?: string
  type?: 'llm' | 'asr' | 'tts'
  status?: string
  group?: string
  model_whitelist?: string
  unlimited_quota?: boolean
  token_quota?: number
  request_quota?: number
  expires_at?: string | null
}

// ==================== Speech Channels (ASR / TTS) ====================
export interface SpeechChannel {
  id: number
  provider: string
  name: string
  enabled: boolean
  group: string
  sort_order: number
  models: string
  config_json?: string
  created_at: string
  updated_at: string
}

export interface SpeechChannelWriteReq {
  provider: string
  name: string
  enabled?: boolean
  group?: string
  sort_order?: number
  models?: string
  config_json?: string
}

export interface ListSpeechChannelsParams {
  page?: number
  pageSize?: number
  search?: string
  group?: string
  provider?: string
}

const speechCRUD = (kind: 'asr' | 'tts') => {
  const base = `${BACKEND_BASE}/admin/${kind}-channels`
  return {
    list: async (params?: ListSpeechChannelsParams) => {
      const res = await get<{ channels: SpeechChannel[]; total: number; page: number; pageSize: number }>(base, { params })
      return res.data
    },
    get: async (id: number) => {
      const res = await get<{ channel: SpeechChannel }>(`${base}/${id}`)
      return res.data.channel
    },
    create: async (body: SpeechChannelWriteReq) => {
      const res = await post<{ channel: SpeechChannel }>(base, body)
      return res.data.channel
    },
    update: async (id: number, body: SpeechChannelWriteReq) => {
      const res = await put<{ channel: SpeechChannel }>(`${base}/${id}`, body)
      return res.data.channel
    },
    remove: async (id: number) => {
      await del(`${base}/${id}`)
    },
  }
}

export const asrChannelsApi = speechCRUD('asr')
export const ttsChannelsApi = speechCRUD('tts')

export const listLLMTokens = async (params?: ListLLMTokensParams) => {
  const res = await get<{ tokens: LLMToken[]; total: number; page: number; pageSize: number }>(`${BACKEND_BASE}/admin/llm-tokens`, { params })
  return res.data
}

export const getLLMToken = async (id: number) => {
  const res = await get<{ token: LLMToken }>(`${BACKEND_BASE}/admin/llm-tokens/${id}`)
  return res.data.token
}

export const createLLMToken = async (body: LLMTokenWriteReq) => {
  const res = await post<{ token: LLMToken; raw_api_key: string }>(`${BACKEND_BASE}/admin/llm-tokens`, body)
  return res.data
}

export const updateLLMToken = async (id: number, body: LLMTokenWriteReq) => {
  const res = await put<{ token: LLMToken }>(`${BACKEND_BASE}/admin/llm-tokens/${id}`, body)
  return res.data.token
}

export const regenerateLLMToken = async (id: number) => {
  const res = await post<{ token: LLMToken; raw_api_key: string }>(`${BACKEND_BASE}/admin/llm-tokens/${id}/regenerate`, {})
  return res.data
}

export const resetLLMTokenUsage = async (id: number) => {
  await post(`${BACKEND_BASE}/admin/llm-tokens/${id}/reset-usage`, {})
}

export const deleteLLMToken = async (id: number) => {
  await del(`${BACKEND_BASE}/admin/llm-tokens/${id}`)
}
