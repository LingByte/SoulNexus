import { get, post, put, ApiResponse } from '@/utils/request'

// 设备信息
export interface Device {
  id: string
  userId: number
  macAddress: string
  board?: string
  appVersion?: string
  autoUpdate: number
  assistantId?: number
  alias?: string
  lastConnected?: string
  createdAt: string
  updatedAt: string
}

// 绑定设备请求
export interface BindDeviceRequest {
  agentId: string
  deviceCode: string
}

// 解绑设备请求
export interface UnbindDeviceRequest {
  deviceId: string
}

// 更新设备信息请求
export interface UpdateDeviceRequest {
  alias?: string
  autoUpdate?: number
}

// 手动添加设备请求
export interface ManualAddDeviceRequest {
  agentId: string
  board: string
  appVersion?: string
  macAddress: string
}

// 绑定设备（激活设备）
export const bindDevice = async (agentId: string, deviceCode: string): Promise<ApiResponse<null>> => {
  return post(`/device/bind/${agentId}/${deviceCode}`, {})
}

// 获取已绑定设备列表
export const getUserDevices = async (agentId: string): Promise<ApiResponse<Device[]>> => {
  return get(`/device/bind/${agentId}`)
}

// 解绑设备
export const unbindDevice = async (data: UnbindDeviceRequest): Promise<ApiResponse<null>> => {
  return post('/device/unbind', data)
}

// 更新设备信息
export const updateDevice = async (deviceId: string, data: UpdateDeviceRequest): Promise<ApiResponse<Device>> => {
  return put(`/device/update/${deviceId}`, data)
}

// 手动添加设备
export const manualAddDevice = async (data: ManualAddDeviceRequest): Promise<ApiResponse<Device>> => {
  return post('/device/manual-add', data)
}

