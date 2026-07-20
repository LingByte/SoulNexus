import { del, get, post, put, type ApiResponse } from '@/utils/request'

/** Snowflake IDs (>2^53) must stay strings in JavaScript. */
export type SnowflakeId = string

export interface OrgPermission {
  id: SnowflakeId
  code: string
  name: string
  description?: string
  /** module | menu | button | api | data */
  kind?: string
  /** Parent permission code (catalog tree); empty for top-level modules */
  parentCode?: string
  resource?: string
  action?: string
}

export interface OrgGroup {
  id: SnowflakeId
  name: string
  isDefault?: boolean
}

export interface OrgRole {
  id: SnowflakeId
  name: string
  description?: string
  isSystem?: boolean
}

export interface OrgRoleDetail extends OrgRole {
  permissionIds: SnowflakeId[]
}

export async function listOrgPermissions(): Promise<ApiResponse<{ list: OrgPermission[] }>> {
  return get('/tenant-org/permissions')
}

export async function listOrgGroups(): Promise<ApiResponse<{ list: OrgGroup[] }>> {
  return get('/tenant-org/groups')
}

export async function createOrgGroup(body: { name: string; isDefault?: boolean }): Promise<ApiResponse<OrgGroup>> {
  return post('/tenant-org/groups', body)
}

export async function updateOrgGroup(
  id: SnowflakeId,
  body: { name: string; isDefault?: boolean },
): Promise<ApiResponse<OrgGroup>> {
  return put(`/tenant-org/groups/${id}`, body)
}

export async function deleteOrgGroup(id: SnowflakeId): Promise<ApiResponse<{ id: SnowflakeId }>> {
  return del(`/tenant-org/groups/${id}`)
}

export async function listOrgRoles(): Promise<ApiResponse<{ list: OrgRole[] }>> {
  return get('/tenant-org/roles')
}

export async function getOrgRole(id: SnowflakeId): Promise<ApiResponse<OrgRoleDetail>> {
  return get(`/tenant-org/roles/${id}`)
}

export async function createOrgRole(body: { name: string; description?: string }): Promise<ApiResponse<OrgRole>> {
  return post('/tenant-org/roles', body)
}

export async function updateOrgRole(
  id: SnowflakeId,
  body: { name: string; description?: string },
): Promise<ApiResponse<{ id: SnowflakeId }>> {
  return put(`/tenant-org/roles/${id}`, body)
}

export async function deleteOrgRole(id: SnowflakeId): Promise<ApiResponse<{ id: SnowflakeId }>> {
  return del(`/tenant-org/roles/${id}`)
}

export async function putOrgRolePermissions(
  roleId: SnowflakeId,
  body: { permissionIds: SnowflakeId[] },
): Promise<ApiResponse<{ roleId: SnowflakeId }>> {
  return put(`/tenant-org/roles/${roleId}/permissions`, body)
}

export async function putOrgTenantUserRoles(
  userId: SnowflakeId,
  body: { roleIds: SnowflakeId[] },
): Promise<ApiResponse<Record<string, unknown>>> {
  return put(`/tenant-org/users/${userId}/roles`, body)
}

export async function putOrgTenantUserGroups(
  userId: SnowflakeId,
  body: { groupIds: SnowflakeId[] },
): Promise<ApiResponse<Record<string, unknown>>> {
  return put(`/tenant-org/users/${userId}/groups`, body)
}
