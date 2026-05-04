import { get, post, put, del } from '@/utils/request'
import { getMainApiBaseURL } from '@/config/apiConfig'

const BASE = getMainApiBaseURL()

/** withRoles=1 时后端返回的关联角色摘要 */
export interface PermissionLinkedRole {
  id: number
  slug: string
  name: string
}

export interface Permission {
  id: number
  key: string
  name: string
  description?: string
  resource?: string
  createdAt?: string
  updatedAt?: string
  roles?: PermissionLinkedRole[]
}

export interface PermissionListItem extends Permission {
  roleCount: number
  directUserGrantCount: number
}

export interface Paginated<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}

function unwrap<T>(res: { code: number; msg: string; data: T }, label: string): T {
  if (res.code !== 200) throw new Error(res.msg || label)
  return res.data
}

export async function listPermissions(params?: {
  page?: number
  pageSize?: number
  search?: string
  /** 为 true 时附带关联的角色列表 */
  withRoles?: boolean
}): Promise<Paginated<PermissionListItem>> {
  const res = await get<Paginated<PermissionListItem>>(`${BASE}/admin/permissions`, {
    params: {
      page: params?.page,
      pageSize: params?.pageSize,
      search: params?.search,
      withRoles: params?.withRoles ? '1' : undefined,
    },
  })
  return unwrap(res, '加载权限失败')
}

export async function createPermission(body: {
  key: string
  name: string
  description?: string
  resource?: string
}): Promise<Permission> {
  const res = await post<Permission>(`${BASE}/admin/permissions`, body)
  return unwrap(res, '创建权限失败')
}

export async function updatePermission(
  id: number,
  body: Partial<{ key: string; name: string; description: string; resource: string }>
): Promise<Permission> {
  const res = await put<Permission>(`${BASE}/admin/permissions/${id}`, body)
  return unwrap(res, '更新权限失败')
}

export async function deletePermission(id: number): Promise<void> {
  const res = await del(`${BASE}/admin/permissions/${id}`)
  unwrap(res, '删除权限失败')
}
