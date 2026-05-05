import { get, post, put, del } from '@/utils/request'
import { getMainApiBaseURL } from '@/config/apiConfig'

const BASE = getMainApiBaseURL()

export interface RoleLinkedPermission {
  id: number
  key: string
  name: string
  resource?: string
}

export interface Role {
  id: number
  name: string
  slug: string
  description?: string
  isSystem?: boolean
  permissions?: RoleLinkedPermission[]
  createdAt?: string
  updatedAt?: string
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

export async function listRoles(params?: {
  page?: number
  pageSize?: number
  search?: string
}): Promise<Paginated<Role>> {
  const res = await get<Paginated<Role>>(`${BASE}/admin/roles`, {
    params: { page: params?.page, pageSize: params?.pageSize, search: params?.search },
  })
  return unwrap(res, '加载角色失败')
}

export async function getRole(id: number): Promise<Role> {
  const res = await get<Role>(`${BASE}/admin/roles/${id}`)
  return unwrap(res, '加载角色失败')
}

export async function createRole(body: {
  name: string
  slug: string
  description?: string
}): Promise<Role> {
  const res = await post<Role>(`${BASE}/admin/roles`, body)
  return unwrap(res, '创建角色失败')
}

export async function updateRole(
  id: number,
  body: Partial<{ name: string; slug: string; description: string }>
): Promise<Role> {
  const res = await put<Role>(`${BASE}/admin/roles/${id}`, body)
  return unwrap(res, '更新角色失败')
}

export async function deleteRole(id: number): Promise<void> {
  const res = await del(`${BASE}/admin/roles/${id}`)
  unwrap(res, '删除角色失败')
}

/** 保存角色拥有的权限，返回更新后的角色（含 permissions） */
export async function setRolePermissions(roleId: number, permissionIds: number[]): Promise<Role> {
  const res = await put<Role>(`${BASE}/admin/roles/${roleId}/permissions`, { permissionIds })
  return unwrap(res, '保存角色权限失败')
}
