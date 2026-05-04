import { get, put } from '@/utils/request'
import { getMainApiBaseURL } from '@/config/apiConfig'
import type { Role } from '@/services/roleApi'
import type { Permission } from '@/services/permissionApi'

const BASE = getMainApiBaseURL()

export interface UserAccessPayload {
  user: {
    id: number
    email: string
    displayName?: string
    legacyRole?: string
  }
  roles: Role[]
  extraPermissions: Permission[]
  effectivePermissionKeys: string[]
}

function unwrap<T>(res: { code: number; msg: string; data: T }, label: string): T {
  if (res.code !== 200) throw new Error(res.msg || label)
  return res.data
}

export async function getUserAccess(userId: number): Promise<UserAccessPayload> {
  const res = await get<UserAccessPayload>(`${BASE}/admin/users/${userId}/access`)
  return unwrap(res, '加载用户授权失败')
}

export async function setUserAccess(
  userId: number,
  body: { roleIds: number[]; permissionIds: number[] }
): Promise<void> {
  const res = await put(`${BASE}/admin/users/${userId}/access`, body)
  unwrap(res, '保存用户授权失败')
}
