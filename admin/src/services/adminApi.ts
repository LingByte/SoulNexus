import { get, post, put, del } from '@/utils/request'
import { getApiBaseURL } from '@/config/apiConfig'
import { handleConfigCacheUpdate } from '@/utils/siteConfigCache'

// Use the API base URL directly (already includes /api prefix)
const BACKEND_BASE = getApiBaseURL()

// ==================== Users API ====================
export interface User {
  id: number
  email: string
  displayName?: string
  firstName?: string
  lastName?: string
  role?: string
  isStaff: boolean
  enabled: boolean
  activated?: boolean
  phone?: string
  locale?: string
  timezone?: string
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
  isStaff?: string
}

export interface CreateUserRequest {
  email: string
  password?: string
  displayName?: string
  firstName?: string
  lastName?: string
  role?: string
  isStaff?: boolean
  enabled?: boolean
  activated?: boolean
  phone?: string
  locale?: string
  timezone?: string
  permissions?: { [key: string]: string[] }
}

export interface UpdateUserRequest {
  email?: string
  password?: string
  displayName?: string
  firstName?: string
  lastName?: string
  role?: string
  isStaff?: boolean
  enabled?: boolean
  activated?: boolean
  phone?: string
  locale?: string
  timezone?: string
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
  if (params?.isStaff) queryParams.isStaff = params.isStaff
  
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
  isStaff?: boolean
  isSuperUser?: boolean
}) => {
  const listParams: ListUsersParams = {
    page: params?.page,
    pageSize: params?.pageSize,
    search: params?.search,
    role: params?.role,
    enabled: params?.status === 'enabled' || params?.status === 'active' ? 'true' : params?.status === 'disabled' ? 'false' : undefined,
    isStaff: params?.isStaff !== undefined ? (params.isStaff ? 'true' : 'false') : undefined,
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
const ADMIN_AUTH_BASE = `${BACKEND_BASE}/auth`

export interface ProfileUpdateRequest {
  displayName?: string
  email?: string
  phone?: string
  timezone?: string
  gender?: string
  extra?: string
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
  const res = await get<LoginHistoryListResponse>(`${BACKEND_BASE}/auth/login-history`, { params: queryParams })
  return res.data
}

export const getLoginHistoryDetail = async (id: number): Promise<{ history: LoginHistory }> => {
  const res = await get<{ history: LoginHistory }>(`${BACKEND_BASE}/auth/login-history/${id}`)
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
