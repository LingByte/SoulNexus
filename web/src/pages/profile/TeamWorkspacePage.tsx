import React, { useEffect, useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Users,
  Plus,
  Search,
  UserPlus,
  Trash2,
  Crown,
  Shield,
  ChevronDown,
  CheckSquare,
  Square,
  Pencil,
  X,
} from 'lucide-react'
import {
  getGroupList,
  getGroup,
  createGroup,
  inviteUser,
  removeMember,
  updateMemberRole,
  searchUsers,
  getInvitations,
  acceptInvitation,
  rejectInvitation,
  type Group,
  type GroupMember,
  type GroupInvitation,
  type UserSearchResult,
} from '@/api/group'
import { showAlert } from '@/utils/notification'
import { useAuthStore } from '@/stores/authStore'
import { useI18nStore } from '@/stores/i18nStore'
import Button from '@/components/UI/Button'
import LoadingAnimation from '@/components/Animations/LoadingAnimation'
import { prefetch } from '@/utils/prefetch'
import { cn } from '@/utils/cn'

const TeamWorkspacePage: React.FC = () => {
  const { t } = useI18nStore()
  const navigate = useNavigate()
  const { user, currentOrganizationId, setCurrentOrganization } = useAuthStore()

  const [groups, setGroups] = useState<Group[]>([])
  const [loadingList, setLoadingList] = useState(true)
  const [invitations, setInvitations] = useState<GroupInvitation[]>([])
  const [groupDetail, setGroupDetail] = useState<Group | null>(null)
  const [members, setMembers] = useState<GroupMember[]>([])
  const [loadingDetail, setLoadingDetail] = useState(false)

  const [memberQuery, setMemberQuery] = useState('')
  const [showInviteModal, setShowInviteModal] = useState(false)
  const [searchKeyword, setSearchKeyword] = useState('')
  const [searchResults, setSearchResults] = useState<UserSearchResult[]>([])
  const [searching, setSearching] = useState(false)
  const [openRoleMenu, setOpenRoleMenu] = useState<number | null>(null)
  const [selectedMembers, setSelectedMembers] = useState<Set<number>>(new Set())
  const [showBatchMenu, setShowBatchMenu] = useState(false)
  const [teamMenuOpen, setTeamMenuOpen] = useState(false)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [newGroupName, setNewGroupName] = useState('')
  const [newGroupType, setNewGroupType] = useState('')
  const [newGroupDescription, setNewGroupDescription] = useState('')

  const fetchGroups = async () => {
    try {
      setLoadingList(true)
      const res = await getGroupList()
      setGroups(res.data || [])
    } catch (err: unknown) {
      const msg = err && typeof err === 'object' && 'msg' in err ? String((err as { msg?: string }).msg) : ''
      showAlert(msg || t('groups.messages.fetchGroupsFailed'), 'error')
    } finally {
      setLoadingList(false)
    }
  }

  const fetchInvitations = async () => {
    try {
      const res = await getInvitations()
      setInvitations(res.data || [])
    } catch (err: unknown) {
      const msg = err && typeof err === 'object' && 'msg' in err ? String((err as { msg?: string }).msg) : ''
      showAlert(msg || t('groups.messages.fetchInvitationsFailed'), 'error')
    }
  }

  useEffect(() => {
    void fetchGroups()
    void fetchInvitations()
  }, [])

  useEffect(() => {
    if (loadingList) return
    if (groups.length === 0) {
      if (currentOrganizationId != null) setCurrentOrganization(null)
      return
    }
    const ids = new Set(groups.map((g) => g.id))
    const cur = currentOrganizationId
    if (cur == null || !ids.has(cur)) {
      const nextId = groups[0].id
      setCurrentOrganization(nextId)
      prefetch.prefetchOverview(nextId)
    }
  }, [groups, currentOrganizationId, setCurrentOrganization, loadingList])

  const selectedId = currentOrganizationId ?? groups[0]?.id ?? null
  const selectedGroup = useMemo(
    () => groups.find((g) => g.id === selectedId) || null,
    [groups, selectedId]
  )

  const fetchGroupDetail = async (id: number) => {
    try {
      setLoadingDetail(true)
      const res = await getGroup(id)
      setGroupDetail(res.data)
      setMembers(res.data.members || [])
    } catch (err: unknown) {
      const msg = err && typeof err === 'object' && 'msg' in err ? String((err as { msg?: string }).msg) : ''
      showAlert(msg || t('groupMembers.messages.fetchGroupFailed'), 'error')
      setGroupDetail(null)
      setMembers([])
    } finally {
      setLoadingDetail(false)
    }
  }

  useEffect(() => {
    if (!selectedId) return
    void fetchGroupDetail(selectedId)
  }, [selectedId])

  const filteredMembers = useMemo(() => {
    const q = memberQuery.trim().toLowerCase()
    if (!q) return members
    return members.filter((m) => {
      const name = (m.user.displayName || '').toLowerCase()
      const email = (m.user.email || '').toLowerCase()
      return name.includes(q) || email.includes(q)
    })
  }, [members, memberQuery])

  const isAdmin = () => {
    if (!groupDetail || !user) return false
    const userId = user.id ? Number(user.id) : null
    return groupDetail.myRole === 'admin' || groupDetail.creatorId === userId
  }

  const handleInviteUser = async (userId: number) => {
    if (!selectedId) return
    try {
      await inviteUser(selectedId, { userId })
      setShowInviteModal(false)
      setSearchKeyword('')
      setSearchResults([])
      await fetchGroupDetail(selectedId)
      showAlert(t('groups.messages.inviteSuccess'), 'success')
    } catch (err: unknown) {
      const msg = err && typeof err === 'object' && 'msg' in err ? String((err as { msg?: string }).msg) : ''
      showAlert(msg || t('groups.messages.inviteFailed'), 'error')
    }
  }

  const handleSearchUsers = async (keyword: string) => {
    if (!keyword.trim()) {
      setSearchResults([])
      return
    }
    try {
      setSearching(true)
      const res = await searchUsers(keyword, 10)
      setSearchResults(res.data || [])
    } catch (err: unknown) {
      const msg = err && typeof err === 'object' && 'msg' in err ? String((err as { msg?: string }).msg) : ''
      showAlert(msg || t('groups.messages.searchUsersFailed'), 'error')
    } finally {
      setSearching(false)
    }
  }

  useEffect(() => {
    const timer = setTimeout(() => {
      if (showInviteModal) void handleSearchUsers(searchKeyword)
    }, 300)
    return () => clearTimeout(timer)
  }, [searchKeyword, showInviteModal])

  const handleRemoveMember = async (memberRecordId: number) => {
    if (!selectedId) return
    if (!confirm(t('groupMembers.messages.removeConfirm'))) return
    try {
      await removeMember(selectedId, memberRecordId)
      await fetchGroupDetail(selectedId)
      setSelectedMembers((prev) => {
        const next = new Set(prev)
        next.delete(memberRecordId)
        return next
      })
      showAlert(t('groupMembers.messages.removeSuccess'), 'success')
    } catch (err: unknown) {
      const msg = err && typeof err === 'object' && 'msg' in err ? String((err as { msg?: string }).msg) : ''
      showAlert(msg || t('groupMembers.messages.removeFailed'), 'error')
    }
  }

  const handleUpdateRole = async (memberRecordId: number, role: string) => {
    if (!selectedId) return
    try {
      await updateMemberRole(selectedId, memberRecordId, role)
      await fetchGroupDetail(selectedId)
      showAlert(t('groupMembers.messages.roleUpdateSuccess'), 'success')
    } catch (err: unknown) {
      const msg = err && typeof err === 'object' && 'msg' in err ? String((err as { msg?: string }).msg) : ''
      showAlert(msg || t('groupMembers.messages.roleUpdateFailed'), 'error')
    }
  }

  const handleAcceptInvitation = async (invitationId: number) => {
    try {
      await acceptInvitation(invitationId)
      await fetchInvitations()
      await fetchGroups()
      showAlert(t('groups.messages.acceptSuccess'), 'success')
    } catch (err: unknown) {
      const msg = err && typeof err === 'object' && 'msg' in err ? String((err as { msg?: string }).msg) : ''
      showAlert(msg || t('groups.messages.acceptFailed'), 'error')
    }
  }

  const handleRejectInvitation = async (invitationId: number) => {
    try {
      await rejectInvitation(invitationId)
      await fetchInvitations()
      showAlert(t('groups.messages.rejectSuccess'), 'success')
    } catch (err: unknown) {
      const msg = err && typeof err === 'object' && 'msg' in err ? String((err as { msg?: string }).msg) : ''
      showAlert(msg || t('groups.messages.rejectFailed'), 'error')
    }
  }

  const selectableMembers = members.filter(
    (m) =>
      groupDetail?.creatorId !== m.userId && (user?.id ? Number(user.id) !== m.userId : true)
  )

  const handleSelectMember = (memberRecordId: number) => {
    setSelectedMembers((prev) => {
      const next = new Set(prev)
      if (next.has(memberRecordId)) next.delete(memberRecordId)
      else next.add(memberRecordId)
      return next
    })
  }

  const handleSelectAll = () => {
    if (selectedMembers.size === selectableMembers.length && selectableMembers.length > 0) {
      setSelectedMembers(new Set())
    } else {
      setSelectedMembers(new Set(selectableMembers.map((m) => m.id)))
    }
  }

  const handleBatchRemove = async () => {
    if (!selectedId || selectedMembers.size === 0) return
    if (!confirm(t('groupMembers.confirmBatchRemove').replace('{count}', String(selectedMembers.size)))) return
    try {
      await Promise.all(Array.from(selectedMembers).map((mid) => removeMember(selectedId, mid)))
      await fetchGroupDetail(selectedId)
      setSelectedMembers(new Set())
      showAlert(t('groupMembers.batchRemoveSuccess'), 'success')
    } catch (err: unknown) {
      const msg = err && typeof err === 'object' && 'msg' in err ? String((err as { msg?: string }).msg) : ''
      showAlert(msg || t('groupMembers.batchRemoveFailed'), 'error')
    }
  }

  const handleBatchSetRole = async (role: string) => {
    if (!selectedId || selectedMembers.size === 0) return
    try {
      await Promise.all(Array.from(selectedMembers).map((mid) => updateMemberRole(selectedId, mid, role)))
      await fetchGroupDetail(selectedId)
      setSelectedMembers(new Set())
      setShowBatchMenu(false)
      showAlert(t('groupMembers.batchSetRoleSuccess'), 'success')
    } catch (err: unknown) {
      const msg = err && typeof err === 'object' && 'msg' in err ? String((err as { msg?: string }).msg) : ''
      showAlert(msg || t('groupMembers.batchSetRoleFailed'), 'error')
    }
  }

  const handleCreateGroup = async () => {
    if (!newGroupName.trim()) {
      showAlert(t('groups.messages.enterGroupName'), 'error')
      return
    }
    try {
      await createGroup({
        name: newGroupName.trim(),
        type: newGroupType.trim() || undefined,
        extra: newGroupDescription.trim() || undefined,
      })
      setShowCreateModal(false)
      setNewGroupName('')
      setNewGroupType('')
      setNewGroupDescription('')
      await fetchGroups()
      showAlert(t('groups.messages.createSuccess'), 'success')
    } catch (err: unknown) {
      const msg = err && typeof err === 'object' && 'msg' in err ? String((err as { msg?: string }).msg) : ''
      showAlert(msg || t('groups.messages.createFailed'), 'error')
    }
  }

  const notifyDisplay = (m: GroupMember) => {
    const u = m.user as { phone?: string }
    return u.phone && String(u.phone).trim() ? u.phone : '—'
  }

  if (loadingList) {
    return (
      <div className="flex flex-col items-center justify-center py-24">
        <LoadingAnimation type="progress" size="lg" className="mb-4" />
        <p className="text-sm text-gray-500 dark:text-gray-400">{t('groups.loading')}</p>
      </div>
    )
  }

  if (groups.length === 0) {
    return (
      <>
        <div className="rounded-2xl border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 p-12 text-center">
          <Users className="w-16 h-16 text-gray-400 mx-auto mb-4" />
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">{t('groups.empty')}</h2>
          <p className="text-gray-500 dark:text-gray-400 text-sm mb-6">{t('groups.emptyDesc')}</p>
          <Button variant="primary" onClick={() => setShowCreateModal(true)}>
            {t('groups.create')}
          </Button>
        </div>
        {showCreateModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
            <div className="w-full max-w-md rounded-xl bg-white dark:bg-neutral-900 border border-gray-200 dark:border-neutral-700 p-6 shadow-xl">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">{t('groups.createModal.title')}</h2>
              <div className="space-y-4 mb-6">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    {t('groupSettings.name')} <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={newGroupName}
                    onChange={(e) => setNewGroupName(e.target.value)}
                    placeholder={t('groups.createModal.namePlaceholder')}
                    className="w-full rounded-lg border border-gray-300 dark:border-neutral-700 bg-white dark:bg-neutral-950 px-3 py-2 text-sm"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('groupSettings.type')}</label>
                  <input
                    type="text"
                    value={newGroupType}
                    onChange={(e) => setNewGroupType(e.target.value)}
                    placeholder={t('groupSettings.typePlaceholder')}
                    className="w-full rounded-lg border border-gray-300 dark:border-neutral-700 bg-white dark:bg-neutral-950 px-3 py-2 text-sm"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('groupSettings.description')}</label>
                  <textarea
                    value={newGroupDescription}
                    onChange={(e) => setNewGroupDescription(e.target.value)}
                    placeholder={t('groupSettings.descriptionPlaceholder')}
                    rows={3}
                    className="w-full rounded-lg border border-gray-300 dark:border-neutral-700 bg-white dark:bg-neutral-950 px-3 py-2 text-sm resize-none"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-2">
                <Button variant="outline" onClick={() => setShowCreateModal(false)}>
                  {t('groups.createModal.cancel')}
                </Button>
                <Button variant="primary" onClick={() => void handleCreateGroup()}>
                  {t('groups.create')}
                </Button>
              </div>
            </div>
          </div>
        )}
      </>
    )
  }

  return (
    <div className="space-y-6">
      {invitations.length > 0 && (
        <div className="rounded-xl border border-blue-200 dark:border-blue-800 bg-blue-50/80 dark:bg-blue-950/30 p-4">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-3">{t('groups.pendingInvitations')}</h3>
          <div className="space-y-2">
            {invitations.map((inv) => (
              <div
                key={inv.id}
                className="flex flex-wrap items-center justify-between gap-2 rounded-lg bg-white dark:bg-neutral-900 border border-gray-200 dark:border-neutral-700 px-3 py-2"
              >
                <div className="text-sm text-gray-700 dark:text-gray-300">
                  <span className="font-medium">{inv.inviter?.displayName || inv.inviter?.email}</span>
                  <span className="text-gray-500 mx-1">{t('groups.inviteToJoin')}</span>
                  <span>{inv.group.name}</span>
                </div>
                <div className="flex gap-2">
                  <Button size="sm" variant="success" onClick={() => void handleAcceptInvitation(inv.id)}>
                    {t('groups.accept')}
                  </Button>
                  <Button size="sm" variant="secondary" onClick={() => void handleRejectInvitation(inv.id)}>
                    {t('groups.reject')}
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="flex flex-nowrap items-center gap-2 sm:gap-3 w-full min-w-0 overflow-hidden">
        <Users className="w-5 h-5 text-sky-600 shrink-0" />
        <h1 className="text-lg sm:text-xl font-bold tracking-tight text-gray-900 dark:text-gray-100 shrink-0 whitespace-nowrap">
          团队管理
        </h1>
        <div className="relative flex-1 min-w-[120px] max-w-md">
          <button
            type="button"
            onClick={() => setTeamMenuOpen((o) => !o)}
            className="flex w-full items-center gap-2 rounded-lg border border-gray-200 dark:border-neutral-600 bg-white dark:bg-neutral-900 px-3 py-2 text-left text-sm min-h-[38px]"
          >
            <span className="truncate font-medium flex-1">{selectedGroup?.name ?? '—'}</span>
            <ChevronDown className="w-4 h-4 shrink-0 opacity-60" />
          </button>
          {teamMenuOpen && (
            <>
              <button type="button" className="fixed inset-0 z-10 cursor-default" aria-label="close" onClick={() => setTeamMenuOpen(false)} />
              <div className="absolute left-0 right-0 top-full z-20 mt-1 max-h-56 overflow-auto rounded-lg border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 shadow-lg py-1">
                {groups.map((g) => (
                  <button
                    key={g.id}
                    type="button"
                    className={cn(
                      'w-full px-3 py-2 text-left text-sm hover:bg-gray-50 dark:hover:bg-neutral-800 truncate',
                      g.id === selectedId && 'bg-sky-50 dark:bg-sky-950/40 text-sky-800 dark:text-sky-200'
                    )}
                    onClick={() => {
                      setCurrentOrganization(g.id)
                      prefetch.prefetchOverview(g.id)
                      setTeamMenuOpen(false)
                    }}
                  >
                    {g.name}
                  </button>
                ))}
              </div>
            </>
          )}
        </div>
        <button
          type="button"
          title={t('groups.settings')}
          className="inline-flex h-[38px] w-[38px] shrink-0 items-center justify-center rounded-lg border border-gray-200 dark:border-neutral-600 bg-white dark:bg-neutral-900 text-gray-600 hover:bg-gray-50 dark:hover:bg-neutral-800"
          onClick={() => selectedId && navigate(`/groups/${selectedId}/settings`)}
          disabled={!selectedId}
        >
          <Pencil className="w-4 h-4" />
        </button>
        <span className="text-xs sm:text-sm text-gray-500 dark:text-gray-400 shrink-0 whitespace-nowrap ml-auto pl-1">
          共 <span className="font-semibold text-gray-900 dark:text-gray-100">{members.length}</span> 名成员
        </span>
      </div>

      <div className="space-y-4 mt-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex flex-wrap items-center gap-2">
              <select className="rounded-lg border border-gray-200 dark:border-neutral-600 bg-white dark:bg-neutral-900 px-3 py-2 text-sm">
                <option value="joined">已加入</option>
              </select>
              <div className="relative flex-1 min-w-[200px] max-w-md">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                <input
                  type="search"
                  placeholder="搜索成员"
                  value={memberQuery}
                  onChange={(e) => setMemberQuery(e.target.value)}
                  className="w-full rounded-lg border border-gray-200 dark:border-neutral-600 bg-white dark:bg-neutral-900 py-2 pl-9 pr-3 text-sm"
                />
              </div>
            </div>
            {isAdmin() && (
              <Button variant="primary" leftIcon={<UserPlus className="w-4 h-4" />} onClick={() => setShowInviteModal(true)}>
                + 邀请成员
              </Button>
            )}
          </div>

          {isAdmin() && selectableMembers.length > 0 && (
            <div className="rounded-lg border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 px-4 py-3 flex flex-wrap items-center gap-3">
              <button type="button" onClick={handleSelectAll} className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
                {selectedMembers.size === selectableMembers.length ? (
                  <CheckSquare className="w-5 h-5 text-sky-600" />
                ) : (
                  <Square className="w-5 h-5 text-gray-400" />
                )}
                {selectedMembers.size === selectableMembers.length ? t('groupMembers.deselectAll') : t('groupMembers.selectAll')}
              </button>
              {selectedMembers.size > 0 && (
                <>
                  <span className="text-sm text-gray-500">{t('groupMembers.selectedCount').replace('{count}', String(selectedMembers.size))}</span>
                  <div className="relative ml-auto">
                    <Button variant="secondary" size="sm" rightIcon={<ChevronDown className="w-4 h-4" />} onClick={() => setShowBatchMenu((v) => !v)}>
                      {t('groupMembers.batchOperations')}
                    </Button>
                    {showBatchMenu && (
                      <>
                        <button type="button" className="fixed inset-0 z-10" onClick={() => setShowBatchMenu(false)} />
                        <div className="absolute right-0 mt-1 w-44 rounded-lg border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 shadow-lg z-20 py-1">
                          <button
                            type="button"
                            className="w-full px-3 py-2 text-left text-sm hover:bg-gray-50 dark:hover:bg-neutral-800"
                            onClick={() => void handleBatchSetRole('admin')}
                          >
                            {t('groupMembers.setAsAdmin')}
                          </button>
                          <button
                            type="button"
                            className="w-full px-3 py-2 text-left text-sm hover:bg-gray-50 dark:hover:bg-neutral-800"
                            onClick={() => void handleBatchSetRole('member')}
                          >
                            {t('groupMembers.setAsMember')}
                          </button>
                          <button
                            type="button"
                            className="w-full px-3 py-2 text-left text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-950/30"
                            onClick={() => void handleBatchRemove()}
                          >
                            {t('groupMembers.batchRemove')}
                          </button>
                        </div>
                      </>
                    )}
                  </div>
                </>
              )}
            </div>
          )}

          <div className="rounded-xl border border-gray-200 dark:border-neutral-700 overflow-hidden bg-white dark:bg-neutral-900">
            {loadingDetail ? (
              <div className="flex justify-center py-16">
                <LoadingAnimation type="progress" size="md" />
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 dark:bg-neutral-950 border-b border-gray-200 dark:border-neutral-700">
                    <tr>
                      {isAdmin() && <th className="w-10 px-4 py-3" />}
                      <th className="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-400">用户名</th>
                      <th className="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-400">通知接收</th>
                      <th className="px-4 py-3 text-left font-medium text-gray-600 dark:text-gray-400">加入/更新时间</th>
                      {isAdmin() && (
                        <th className="px-4 py-3 text-right font-medium text-gray-600 dark:text-gray-400">操作</th>
                      )}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200 dark:divide-neutral-700">
                    {filteredMembers.length === 0 ? (
                      <tr>
                        <td colSpan={isAdmin() ? 5 : 3} className="px-4 py-12 text-center text-gray-500">
                          暂无成员
                        </td>
                      </tr>
                    ) : (
                      filteredMembers.map((member) => {
                        const isCurrentUser = user?.id ? Number(user.id) === member.userId : false
                        const isMemberCreator = groupDetail?.creatorId === member.userId
                        const canSelect = !isCurrentUser && !isMemberCreator && isAdmin()

                        return (
                          <tr key={member.id} className="hover:bg-gray-50/80 dark:hover:bg-neutral-800/50">
                            {isAdmin() && (
                              <td className="px-4 py-3 align-middle">
                                {canSelect && (
                                  <button type="button" onClick={() => handleSelectMember(member.id)}>
                                    {selectedMembers.has(member.id) ? (
                                      <CheckSquare className="w-5 h-5 text-sky-600" />
                                    ) : (
                                      <Square className="w-5 h-5 text-gray-400" />
                                    )}
                                  </button>
                                )}
                              </td>
                            )}
                            <td className="px-4 py-3">
                              <div className="flex items-center gap-3 min-w-0">
                                <img
                                  src={
                                    member.user.avatar ||
                                    `https://ui-avatars.com/api/?name=${encodeURIComponent(member.user.displayName || member.user.email)}&background=0ea5e9&color=fff`
                                  }
                                  alt=""
                                  className="w-9 h-9 rounded-full shrink-0"
                                />
                                <div className="min-w-0">
                                  <div className="flex flex-wrap items-center gap-1.5 font-medium text-gray-900 dark:text-gray-100">
                                    <span className="truncate">{member.user.displayName || member.user.email}</span>
                                    {isMemberCreator && (
                                      <span className="inline-flex items-center gap-0.5 rounded-full bg-amber-100 dark:bg-amber-900/40 px-2 py-0.5 text-xs text-amber-800 dark:text-amber-200">
                                        <Crown className="w-3 h-3" />
                                        {t('groups.creator')}
                                      </span>
                                    )}
                                    {member.role === 'admin' && !isMemberCreator && (
                                      <span className="inline-flex items-center gap-0.5 rounded-full bg-blue-100 dark:bg-blue-900/40 px-2 py-0.5 text-xs text-blue-800 dark:text-blue-200">
                                        <Shield className="w-3 h-3" />
                                        {t('groups.admin')}
                                      </span>
                                    )}
                                    {isCurrentUser && (
                                      <span className="text-xs text-green-700 dark:text-green-400">({t('groupMembers.me')})</span>
                                    )}
                                  </div>
                                  {member.user.displayName ? (
                                    <div className="text-xs text-gray-500 truncate">{member.user.email}</div>
                                  ) : null}
                                </div>
                              </div>
                            </td>
                            <td className="px-4 py-3 text-gray-600 dark:text-gray-400 whitespace-nowrap">{notifyDisplay(member)}</td>
                            <td className="px-4 py-3 text-gray-600 dark:text-gray-400 whitespace-nowrap">
                              {new Date(member.createdAt).toLocaleString('zh-CN', {
                                year: 'numeric',
                                month: '2-digit',
                                day: '2-digit',
                                hour: '2-digit',
                                minute: '2-digit',
                                second: '2-digit',
                                hour12: false,
                              })}
                            </td>
                            {isAdmin() && (
                              <td className="px-4 py-3 text-right">
                                {!isCurrentUser && !isMemberCreator && (
                                  <div className="flex justify-end gap-2">
                                    <div className="relative">
                                      <Button
                                        variant="secondary"
                                        size="sm"
                                        rightIcon={<ChevronDown className="w-4 h-4" />}
                                        onClick={() => setOpenRoleMenu(openRoleMenu === member.id ? null : member.id)}
                                      >
                                        {member.role === 'admin' ? t('groups.admin') : t('groupMembers.member')}
                                      </Button>
                                      {openRoleMenu === member.id && (
                                        <>
                                          <button type="button" className="fixed inset-0 z-10" onClick={() => setOpenRoleMenu(null)} />
                                          <div className="absolute right-0 mt-1 w-36 rounded-lg border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-900 shadow-lg z-20 py-1">
                                            <button
                                              type="button"
                                              disabled={member.role === 'admin'}
                                              className="w-full px-3 py-2 text-left text-sm hover:bg-gray-50 disabled:opacity-40"
                                              onClick={() => {
                                                void handleUpdateRole(member.id, 'admin')
                                                setOpenRoleMenu(null)
                                              }}
                                            >
                                              {t('groupMembers.setAsAdmin')}
                                            </button>
                                            <button
                                              type="button"
                                              disabled={member.role === 'member'}
                                              className="w-full px-3 py-2 text-left text-sm hover:bg-gray-50 disabled:opacity-40"
                                              onClick={() => {
                                                void handleUpdateRole(member.id, 'member')
                                                setOpenRoleMenu(null)
                                              }}
                                            >
                                              {t('groupMembers.setAsMember')}
                                            </button>
                                          </div>
                                        </>
                                      )}
                                    </div>
                                    <Button
                                      variant="destructive"
                                      size="sm"
                                      leftIcon={<Trash2 className="w-4 h-4" />}
                                      onClick={() => void handleRemoveMember(member.id)}
                                    >
                                      {t('groupMembers.remove')}
                                    </Button>
                                  </div>
                                )}
                              </td>
                            )}
                          </tr>
                        )
                      })
                    )}
                  </tbody>
                </table>
              </div>
            )}
            {!loadingDetail && filteredMembers.length > 0 && (
              <div className="border-t border-gray-100 dark:border-neutral-800 py-3 text-center text-xs text-gray-400">已加载全部</div>
            )}
          </div>
      </div>

      {showInviteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="relative w-full max-w-2xl max-h-[85vh] overflow-y-auto rounded-xl bg-white dark:bg-neutral-900 border border-gray-200 dark:border-neutral-700 shadow-xl p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('groups.inviteModal.title')}</h2>
              <button
                type="button"
                className="rounded-lg p-2 text-gray-500 hover:bg-gray-100 dark:hover:bg-neutral-800"
                onClick={() => {
                  setShowInviteModal(false)
                  setSearchKeyword('')
                  setSearchResults([])
                }}
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="relative mb-4">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
              <input
                type="text"
                value={searchKeyword}
                onChange={(e) => setSearchKeyword(e.target.value)}
                placeholder={t('groups.inviteModal.searchPlaceholder')}
                className="w-full rounded-lg border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-950 py-3 pl-10 pr-4 text-sm"
              />
            </div>
            {searching && <div className="text-center text-gray-400 py-4">{t('groups.inviteModal.searching')}</div>}
            {!searching && searchKeyword && searchResults.length === 0 && (
              <div className="text-center text-gray-400 py-4">{t('groups.inviteModal.noResults')}</div>
            )}
            {!searching && searchResults.length > 0 && (
              <div className="space-y-2">
                {searchResults.map((result) => {
                  const isAlreadyMember = members.some((m) => m.userId === result.id)
                  return (
                    <div
                      key={result.id}
                      className="flex items-center justify-between gap-3 rounded-lg border border-gray-200 dark:border-neutral-700 p-3"
                    >
                      <div className="flex items-center gap-3 min-w-0">
                        <img
                          src={result.avatar || `https://ui-avatars.com/api/?name=${encodeURIComponent(result.displayName || result.email)}&background=0ea5e9&color=fff`}
                          alt=""
                          className="w-10 h-10 rounded-full shrink-0"
                        />
                        <div className="min-w-0">
                          <div className="font-medium text-gray-900 dark:text-gray-100 truncate">{result.displayName || result.email}</div>
                          {result.displayName ? (
                            <div className="text-xs text-gray-500 truncate">{result.email}</div>
                          ) : null}
                        </div>
                      </div>
                      {isAlreadyMember ? (
                        <span className="text-xs text-gray-500 shrink-0">{t('groupMembers.alreadyMember')}</span>
                      ) : (
                        <Button size="sm" variant="primary" leftIcon={<Plus className="w-4 h-4" />} onClick={() => void handleInviteUser(result.id)}>
                          {t('groups.inviteModal.invite')}
                        </Button>
                      )}
                    </div>
                  )
                })}
              </div>
            )}
            {!searchKeyword && (
              <div className="text-center text-gray-400 py-8 text-sm">{t('groups.inviteModal.searchHint')}</div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

export default TeamWorkspacePage
