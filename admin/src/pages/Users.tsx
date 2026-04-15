import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { Users as UsersIcon, Plus, Search, Edit, Trash2, RefreshCw, Save, UserCheck, UserX } from 'lucide-react'
import AdminLayout from '@/components/Layout/AdminLayout'
import Card from '@/components/UI/Card'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import Badge from '@/components/UI/Badge'
import EmptyState from '@/components/UI/EmptyState'
import Modal from '@/components/UI/Modal'
import ConfirmDialog from '@/components/UI/ConfirmDialog'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/UI/Select'
import { showAlert } from '@/utils/notification'
import {
  listUsers,
  getUser,
  createUser,
  updateUser,
  deleteUser,
  type User,
  type ListUsersParams,
  type CreateUserRequest,
  type UpdateUserRequest,
} from '@/services/adminApi'

const ROLES = [
  { value: 'user', label: '普通用户' },
  { value: 'admin', label: '管理员' },
  { value: 'superadmin', label: '超级管理员' },
]



const Users = () => {
  const [users, setUsers] = useState<User[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [search, setSearch] = useState('')
  const [roleFilter, setRoleFilter] = useState<string>('')
  const [enabledFilter, setEnabledFilter] = useState<string>('')
  const [hasPhoneFilter, setHasPhoneFilter] = useState<string>('')
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize] = useState(20)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showEditModal, setShowEditModal] = useState(false)
  const [showDetailModal, setShowDetailModal] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [editingUser, setEditingUser] = useState<User | null>(null)
  // Form states
  const [formData, setFormData] = useState<CreateUserRequest>({
    email: '',
    displayName: '',
    firstName: '',
    lastName: '',
    role: 'user',
    enabled: true,
    phone: '',
    locale: '',
    timezone: '',
    city: '',
    region: '',
    gender: '',
    emailNotifications: true,
    pushNotifications: true,
    systemNotifications: true,
  })

  // 获取用户列表
  const fetchUsers = async () => {
    try {
      setLoading(true)
      const params: ListUsersParams = {
        page: currentPage,
        pageSize: pageSize,
      }
      if (search) params.search = search
      if (roleFilter) params.role = roleFilter
      if (enabledFilter) params.enabled = enabledFilter
      if (hasPhoneFilter) params.hasPhone = hasPhoneFilter

      const response = await listUsers(params)
      setUsers(response.users || [])
      setTotal(response.total || 0)
    } catch (error: any) {
      showAlert('获取用户列表失败', 'error', error?.msg || error?.message)
    } finally {
      setLoading(false)
    }
  }

  // Reset to first page when filters change
  useEffect(() => {
    setCurrentPage(1)
  }, [search, roleFilter, enabledFilter, hasPhoneFilter])

  // Fetch users when page or filters change
  useEffect(() => {
    fetchUsers()
  }, [currentPage, search, roleFilter, enabledFilter, hasPhoneFilter])

  // 处理创建用户
  const handleCreate = () => {
    setFormData({
      email: '',
      displayName: '',
      firstName: '',
      lastName: '',
      role: 'user',
      enabled: true,
      phone: '',
      locale: '',
      timezone: '',
      city: '',
      region: '',
      gender: '',
      emailNotifications: true,
      pushNotifications: true,
      systemNotifications: true,
    })
    setShowCreateModal(true)
  }

  // 处理编辑用户
  const handleEdit = async (user: User) => {
    try {
      const fullUser = await getUser(user.id)
      setEditingUser(fullUser)
      setFormData({
        email: fullUser.email,
        displayName: fullUser.displayName || '',
        firstName: fullUser.firstName || '',
        lastName: fullUser.lastName || '',
        role: fullUser.role || 'user',
        enabled: fullUser.enabled,
        phone: fullUser.phone || '',
        locale: fullUser.locale || '',
        timezone: fullUser.timezone || '',
        city: (fullUser as any).city || '',
        region: (fullUser as any).region || '',
        gender: (fullUser as any).gender || '',
        emailNotifications: (fullUser as any).emailNotifications ?? true,
        pushNotifications: (fullUser as any).pushNotifications ?? true,
        systemNotifications: (fullUser as any).systemNotifications ?? true,
      })
      setShowEditModal(true)
    } catch (error: any) {
      showAlert('获取用户详情失败', 'error', error?.msg || error?.message)
    }
  }

  // 处理删除用户
  const handleDelete = (user: User) => {
    setSelectedUser(user)
    setShowDeleteConfirm(true)
  }

  const handleViewDetail = async (user: User) => {
    try {
      const fullUser = await getUser(user.id)
      setSelectedUser(fullUser)
      setShowDetailModal(true)
    } catch (error: any) {
      showAlert('获取用户详情失败', 'error', error?.msg || error?.message)
    }
  }

  // 提交创建
  const handleSubmitCreate = async () => {
    try {
      if (!formData.email) {
        showAlert('请输入邮箱', 'error')
        return
      }
      await createUser(formData)
      showAlert('创建用户成功', 'success')
      setShowCreateModal(false)
      fetchUsers()
    } catch (error: any) {
      showAlert('创建用户失败', 'error', error?.msg || error?.message)
    }
  }

  // 提交更新
  const handleSubmitUpdate = async () => {
    if (!editingUser) return
    try {
      const updateData: UpdateUserRequest = {
        email: formData.email,
        displayName: formData.displayName,
        firstName: formData.firstName,
        lastName: formData.lastName,
        role: formData.role,
        enabled: formData.enabled,
        phone: formData.phone,
        locale: formData.locale,
        timezone: formData.timezone,
        city: (formData as any).city,
        region: (formData as any).region,
        gender: (formData as any).gender,
        emailNotifications: (formData as any).emailNotifications,
        pushNotifications: (formData as any).pushNotifications,
        systemNotifications: (formData as any).systemNotifications,
      }
      // 只有在编辑模式下且密码不为空时才更新密码
      if (formData.password) {
        updateData.password = formData.password
      }
      await updateUser(editingUser.id, updateData)
      showAlert('更新用户成功', 'success')
      setShowEditModal(false)
      setEditingUser(null)
      fetchUsers()
    } catch (error: any) {
      showAlert('更新用户失败', 'error', error?.msg || error?.message)
    }
  }

  // 确认删除
  const handleConfirmDelete = async () => {
    if (!selectedUser) return
    try {
      await deleteUser(selectedUser.id)
      showAlert('删除用户成功', 'success')
      setShowDeleteConfirm(false)
      setSelectedUser(null)
      fetchUsers()
    } catch (error: any) {
      showAlert('删除用户失败', 'error', error?.msg || error?.message)
    }
  }

  const totalPages = Math.ceil(total / pageSize)

  return (
    <AdminLayout>
      <div className="space-y-6">
        {/* Toolbar */}
        <Card>
          <div className="flex flex-col sm:flex-row gap-4 items-start sm:items-center justify-between">
            <div className="flex flex-1 flex-wrap gap-3 items-center">
              <Input
                placeholder="搜索邮箱、姓名..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full sm:w-64"
                leftIcon={<Search className="w-4 h-4" />}
              />
              <Select value={roleFilter} onValueChange={setRoleFilter}>
                <SelectTrigger className="w-full sm:w-40">
                  <SelectValue placeholder="角色" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">全部角色</SelectItem>
                  {ROLES.map((role) => (
                    <SelectItem key={role.value} value={role.value}>
                      {role.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Select value={enabledFilter} onValueChange={setEnabledFilter}>
                <SelectTrigger className="w-full sm:w-32">
                  <SelectValue placeholder="状态" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">全部状态</SelectItem>
                  <SelectItem value="true">已启用</SelectItem>
                  <SelectItem value="false">已禁用</SelectItem>
                </SelectContent>
              </Select>
              <Select value={hasPhoneFilter} onValueChange={setHasPhoneFilter}>
                <SelectTrigger className="w-full sm:w-36">
                  <SelectValue placeholder="手机号" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">全部手机号</SelectItem>
                  <SelectItem value="true">有手机号</SelectItem>
                  <SelectItem value="false">无手机号</SelectItem>
                </SelectContent>
              </Select>
              <Button
                variant="outline"
                size="sm"
                onClick={fetchUsers}
                leftIcon={<RefreshCw className="w-4 h-4" />}
              >
                <span>刷新</span>
              </Button>
              <Button
                variant="primary"
                size="sm"
                onClick={handleCreate}
                leftIcon={<Plus className="w-4 h-4" />}
              >
                <span>新建用户</span>
              </Button>
            </div>
          </div>
        </Card>

        {/* Users Table */}
        <Card>
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <RefreshCw className="w-6 h-6 animate-spin text-slate-400" />
            </div>
          ) : users.length === 0 ? (
            <EmptyState
              icon={UsersIcon}
              title="暂无用户"
              description="创建第一个用户开始使用"
            />
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-slate-700">
                    <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-300">ID</th>
                    <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-300">邮箱</th>
                    <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-300">姓名</th>
                    <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-300">角色</th>
                    <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-300">状态</th>
                    <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-300">手机号</th>
                    <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-300">最近登录</th>
                    <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-300">登录次数</th>
                    <th className="text-left py-3 px-4 font-medium text-slate-600 dark:text-slate-300">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {users.map((user) => (
                    <motion.tr
                      key={user.id}
                      className="border-b border-slate-100 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors"
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                    >
                      <td className="py-3 px-4">{user.id}</td>
                      <td className="py-3 px-4">{user.email}</td>
                      <td className="py-3 px-4">
                        {user.displayName || `${user.firstName || ''} ${user.lastName || ''}`.trim() || '-'}
                      </td>
                      <td className="py-3 px-4">
                        <Badge variant={user.role === 'admin' || user.role === 'superadmin' ? 'primary' : 'secondary'}>
                          {user.role || 'user'}
                        </Badge>
                      </td>
                      <td className="py-3 px-4">
                        <div className="flex items-center gap-2 flex-wrap">
                          {user.enabled ? (
                            <Badge variant="success" icon={<UserCheck className="w-3 h-3" />}>
                              已启用
                            </Badge>
                          ) : (
                            <Badge variant="error" icon={<UserX className="w-3 h-3" />}>
                              已禁用
                            </Badge>
                          )}
                        </div>
                      </td>
                      <td className="py-3 px-4">{user.phone || '-'}</td>
                      <td className="py-3 px-4">{user.lastLogin ? new Date(user.lastLogin).toLocaleString('zh-CN') : '-'}</td>
                      <td className="py-3 px-4">{user.loginCount || 0}</td>
                      <td className="py-3 px-4">
                        <div className="flex items-center gap-2">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleViewDetail(user)}
                          >
                            <span>详情</span>
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleEdit(user)}
                            leftIcon={<Edit className="w-4 h-4" />}
                          >
                            <span>编辑</span>
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDelete(user)}
                            leftIcon={<Trash2 className="w-4 h-4" />}
                            className="text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
                          >
                            <span>删除</span>
                          </Button>
                        </div>
                      </td>
                    </motion.tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4 pt-4 border-t border-slate-200 dark:border-slate-700">
              <div className="text-sm text-slate-600 dark:text-slate-400">
                共 {total} 条，第 {currentPage} / {totalPages} 页
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                  disabled={currentPage === 1}
                >
                  上一页
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                  disabled={currentPage === totalPages}
                >
                  下一页
                </Button>
              </div>
            </div>
          )}
        </Card>

        {/* Create Modal */}
        <Modal
          isOpen={showCreateModal}
          onClose={() => setShowCreateModal(false)}
          title="新建用户"
          size="lg"
        >
          <div className="space-y-4">
            <Input
              label="邮箱 *"
              value={formData.email}
              onChange={(e) => setFormData({ ...formData, email: e.target.value })}
              type="email"
              required
            />
            <Input
              label="显示名称"
              value={formData.displayName}
              onChange={(e) => setFormData({ ...formData, displayName: e.target.value })}
            />
            <div className="grid grid-cols-2 gap-4">
              <Input
                label="名"
                value={formData.firstName}
                onChange={(e) => setFormData({ ...formData, firstName: e.target.value })}
              />
              <Input
                label="姓"
                value={formData.lastName}
                onChange={(e) => setFormData({ ...formData, lastName: e.target.value })}
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-2">角色</label>
                <Select value={formData.role || ''} onValueChange={(value) => setFormData({ ...formData, role: value })}>
                  <SelectTrigger>
                    <SelectValue placeholder="选择角色">
                      {ROLES.find(r => r.value === formData.role)?.label || '选择角色'}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {ROLES.map((role) => (
                      <SelectItem key={role.value} value={role.value}>
                        {role.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <Input
                label="手机号"
                value={formData.phone}
                onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
              />
            </div>
              <div className="flex items-center gap-4">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.enabled}
                    onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                    className="rounded"
                  />
                  <span className="text-sm">启用</span>
                </label>
              </div>
              
              <div className="flex justify-end gap-3 pt-4">
              <Button variant="outline" onClick={() => setShowCreateModal(false)}>
                <span>取消</span>
              </Button>
              <Button variant="primary" onClick={handleSubmitCreate} leftIcon={<Save className="w-4 h-4" />}>
                <span>创建</span>
              </Button>
            </div>
          </div>
        </Modal>

        {/* Edit Modal */}
        <Modal
          isOpen={showEditModal}
          onClose={() => {
            setShowEditModal(false)
            setEditingUser(null)
          }}
          title="编辑用户"
          size="lg"
        >
          {editingUser && (
            <div className="space-y-4">
              <Input
                label="邮箱 *"
                value={formData.email}
                onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                type="email"
                required
              />
              <Input
                label="显示名称"
                value={formData.displayName}
                onChange={(e) => setFormData({ ...formData, displayName: e.target.value })}
              />
              <div className="grid grid-cols-2 gap-4">
                <Input
                  label="名"
                  value={formData.firstName}
                  onChange={(e) => setFormData({ ...formData, firstName: e.target.value })}
                />
                <Input
                  label="姓"
                  value={formData.lastName}
                  onChange={(e) => setFormData({ ...formData, lastName: e.target.value })}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-2">角色</label>
                  <Select value={formData.role || 'user'} onValueChange={(value) => setFormData({ ...formData, role: value })}>
                    <SelectTrigger>
                      <SelectValue placeholder="选择角色">
                        {ROLES.find(r => r.value === formData.role)?.label || '选择角色'}
                      </SelectValue>
                    </SelectTrigger>
                    <SelectContent>
                      {ROLES.map((role) => (
                        <SelectItem key={role.value} value={role.value}>
                          {role.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <Input
                  label="手机号"
                  value={formData.phone}
                  onChange={(e) => setFormData({ ...formData, phone: e.target.value })}
                />
              </div>
              <div className="flex items-center gap-4">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.enabled}
                    onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                    className="rounded"
                  />
                  <span className="text-sm">启用</span>
                </label>
              </div>
              
              <div className="flex justify-end gap-3 pt-4">
                <Button variant="outline" onClick={() => {
                  setShowEditModal(false)
                  setEditingUser(null)
                }}>
                  <span>取消</span>
                </Button>
                <Button variant="primary" onClick={handleSubmitUpdate} leftIcon={<Save className="w-4 h-4" />}>
                  <span>保存</span>
                </Button>
              </div>
            </div>
          )}
        </Modal>

        {/* Delete Confirm Dialog */}
        <ConfirmDialog
          isOpen={showDeleteConfirm}
          onClose={() => {
            setShowDeleteConfirm(false)
            setSelectedUser(null)
          }}
          onConfirm={handleConfirmDelete}
          title="确认删除"
          message={`确定要删除用户 "${selectedUser?.email}" 吗？此操作不可恢复。`}
          confirmText="删除"
          cancelText="取消"
          variant="danger"
        />

        <Modal
          isOpen={showDetailModal}
          onClose={() => {
            setShowDetailModal(false)
            setSelectedUser(null)
          }}
          title="用户详情"
          size="lg"
        >
          {selectedUser && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
              <div><span className="text-slate-500">ID：</span>{selectedUser.id}</div>
              <div><span className="text-slate-500">邮箱：</span>{selectedUser.email}</div>
              <div><span className="text-slate-500">显示名：</span>{selectedUser.displayName || '-'}</div>
              <div><span className="text-slate-500">角色：</span>{selectedUser.role || 'user'}</div>
              <div><span className="text-slate-500">手机号：</span>{selectedUser.phone || '-'}</div>
              <div><span className="text-slate-500">时区：</span>{selectedUser.timezone || '-'}</div>
              <div><span className="text-slate-500">城市：</span>{(selectedUser as any).city || '-'}</div>
              <div><span className="text-slate-500">地区：</span>{(selectedUser as any).region || '-'}</div>
              <div><span className="text-slate-500">性别：</span>{(selectedUser as any).gender || '-'}</div>
              <div><span className="text-slate-500">最近登录：</span>{selectedUser.lastLogin ? new Date(selectedUser.lastLogin).toLocaleString('zh-CN') : '-'}</div>
              <div><span className="text-slate-500">登录次数：</span>{selectedUser.loginCount || 0}</div>
              <div><span className="text-slate-500">状态：</span>{selectedUser.enabled ? '启用' : '禁用'}</div>
              <div><span className="text-slate-500">邮件通知：</span>{(selectedUser as any).emailNotifications ? '开' : '关'}</div>
              <div><span className="text-slate-500">推送通知：</span>{(selectedUser as any).pushNotifications ? '开' : '关'}</div>
              <div><span className="text-slate-500">系统通知：</span>{(selectedUser as any).systemNotifications ? '开' : '关'}</div>
            </div>
          )}
        </Modal>
      </div>
    </AdminLayout>
  )
}

export default Users
