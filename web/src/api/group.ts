import { get, post, put, del, ApiResponse } from '@/utils/request'

// 组织权限
export interface GroupPermission {
  permissions: string[]
}

// 组织信息
export interface Group {
  id: number
  createdAt: string
  updatedAt: string
  name: string
  type?: string
  extra?: string
  avatar?: string
  permission?: GroupPermission
  creatorId: number
  creator?: {
    id: number
    email: string
    displayName?: string
  }
  memberCount?: number
  myRole?: string
  members?: GroupMember[]
}

// 组织成员
export interface GroupMember {
  id: number
  createdAt: string
  userId: number
  user: {
    id: number
    email: string
    displayName?: string
  }
  groupId: number
  role: string
}

// 组织邀请
export interface GroupInvitation {
  id: number
  createdAt: string
  updatedAt: string
  groupId: number
  group: Group
  inviterId: number
  inviter: {
    id: number
    email: string
    displayName?: string
  }
  inviteeId: number
  invitee: {
    id: number
    email: string
    displayName?: string
  }
  status: 'pending' | 'accepted' | 'rejected' | 'expired'
  expiresAt?: string
}

// 创建组织请求
export interface CreateGroupRequest {
  name: string
  type?: string
  extra?: string
  permission?: GroupPermission
}

// 更新组织请求
export interface UpdateGroupRequest {
  name?: string
  type?: string
  extra?: string
  permission?: GroupPermission
}

// 邀请用户请求
export interface InviteUserRequest {
  userId: number
}

// 用户搜索结果
export interface UserSearchResult {
  id: number
  email: string
  displayName: string
  firstName: string
  lastName: string
  avatar?: string
  createdAt: string
}

// 创建组织
export const createGroup = async (data: CreateGroupRequest): Promise<ApiResponse<Group>> => {
  return post('/group', data)
}

// 获取组织列表
export const getGroupList = async (): Promise<ApiResponse<Group[]>> => {
  return get('/group')
}

// 获取组织详情
export const getGroup = async (id: number): Promise<ApiResponse<Group>> => {
  return get(`/group/${id}`)
}

// 更新组织
export const updateGroup = async (id: number, data: UpdateGroupRequest): Promise<ApiResponse<Group>> => {
  return put(`/group/${id}`, data)
}

// 删除组织
export const deleteGroup = async (id: number): Promise<ApiResponse<null>> => {
  return del(`/group/${id}`)
}

// 离开组织
export const leaveGroup = async (id: number): Promise<ApiResponse<null>> => {
  return post(`/group/${id}/leave`)
}

// 移除成员
export const removeMember = async (groupId: number, memberId: number): Promise<ApiResponse<null>> => {
  return del(`/group/${groupId}/members/${memberId}`)
}

// 邀请用户
export const inviteUser = async (groupId: number, data: InviteUserRequest): Promise<ApiResponse<GroupInvitation>> => {
  return post(`/group/${groupId}/invite`, data)
}

// 获取邀请列表
export const getInvitations = async (): Promise<ApiResponse<GroupInvitation[]>> => {
  return get('/group/invitations')
}

// 接受邀请
export const acceptInvitation = async (id: number): Promise<ApiResponse<null>> => {
  return post(`/group/invitations/${id}/accept`)
}

// 拒绝邀请
export const rejectInvitation = async (id: number): Promise<ApiResponse<null>> => {
  return post(`/group/invitations/${id}/reject`)
}

// 搜索用户（用于邀请）
export const searchUsers = async (keyword: string, limit: number = 20): Promise<ApiResponse<UserSearchResult[]>> => {
  return get('/group/search-users', { params: { keyword, limit } })
}

// 组织共享的资源
export interface GroupSharedResources {
  assistants: Array<{
    id: number
    name: string
    description: string
    icon: string
    createdAt: string
  }>
  knowledgeBases: Array<{
    id: number
    knowledge_key: string
    knowledge_name: string
    created_at: string
  }>
}

// 获取组织共享的资源
export const getGroupSharedResources = async (groupId: number): Promise<ApiResponse<GroupSharedResources>> => {
  return get(`/group/${groupId}/resources`)
}

// 上传组织头像
export const uploadGroupAvatar = async (groupId: number, file: File): Promise<ApiResponse<{ avatar: string }>> => {
  const formData = new FormData()
  formData.append('avatar', file)
  return post(`/group/${groupId}/avatar`, formData)
}

// 更新成员角色
export interface UpdateMemberRoleRequest {
  role: string
}

export const updateMemberRole = async (groupId: number, memberId: number, role: string): Promise<ApiResponse<null>> => {
  return put(`/group/${groupId}/members/${memberId}/role`, { role })
}

