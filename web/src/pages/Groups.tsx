import React, { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { useNavigate, useLocation } from 'react-router-dom';
import { 
  getGroupList, 
  createGroup, 
  deleteGroup, 
  leaveGroup,
  inviteUser,
  getInvitations,
  acceptInvitation,
  rejectInvitation,
  searchUsers,
  type Group,
  type GroupInvitation,
  type UserSearchResult
} from '@/api/group';
import { showAlert } from '@/utils/notification';
import { useAuthStore } from '@/stores/authStore';
import { Users, Plus, X, UserPlus, LogOut, Trash2, Check, XCircle, Crown, Shield, Search, Settings } from 'lucide-react';
import Button from '@/components/UI/Button';
import { useI18nStore } from '@/stores/i18nStore';

const Groups: React.FC = () => {
  const { t } = useI18nStore();
  const { user } = useAuthStore();
  const navigate = useNavigate();
  const location = useLocation();
  const [groups, setGroups] = useState<Group[]>([]);
  const [invitations, setInvitations] = useState<GroupInvitation[]>([]);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showInviteModal, setShowInviteModal] = useState<number | null>(null);
  const [newGroupName, setNewGroupName] = useState('');
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchResults, setSearchResults] = useState<UserSearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [loading, setLoading] = useState(false);

  const fetchGroups = async () => {
    try {
      setLoading(true);
      const res = await getGroupList();
      setGroups(res.data || []);
    } catch (err: any) {
      showAlert(err?.msg || t('groups.messages.fetchGroupsFailed'), 'error');
    } finally {
      setLoading(false);
    }
  };

  const fetchInvitations = async () => {
    try {
      const res = await getInvitations();
      setInvitations(res.data || []);
    } catch (err: any) {
      showAlert(err?.msg || t('groups.messages.fetchInvitationsFailed'), 'error');
    }
  };

  const [lastFetchTime, setLastFetchTime] = useState(0);

  useEffect(() => {
    fetchGroups();
    fetchInvitations();
    setLastFetchTime(Date.now());
  }, []);

  // 监听路由变化，当从设置页面返回时刷新列表
  useEffect(() => {
    // 如果当前在 /groups 页面，且距离上次刷新超过1秒，则刷新列表
    // 这样可以避免频繁刷新，同时确保从设置页面返回时能获取最新数据
    if (location.pathname === '/groups' && Date.now() - lastFetchTime > 1000) {
      fetchGroups();
      setLastFetchTime(Date.now());
    }
  }, [location.pathname]);

  // 搜索用户
  const handleSearchUsers = async (keyword: string) => {
    if (!keyword.trim()) {
      setSearchResults([]);
      return;
    }
    try {
      setSearching(true);
      const res = await searchUsers(keyword, 10);
      setSearchResults(res.data || []);
    } catch (err: any) {
      showAlert(err?.msg || t('groups.messages.searchUsersFailed'), 'error');
    } finally {
      setSearching(false);
    }
  };

  // 防抖搜索
  useEffect(() => {
    const timer = setTimeout(() => {
      if (showInviteModal) {
        handleSearchUsers(searchKeyword);
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [searchKeyword, showInviteModal]);

  const handleCreateGroup = async () => {
    if (!newGroupName.trim()) {
      showAlert(t('groups.messages.enterGroupName'), 'error');
      return;
    }
    try {
      await createGroup({ name: newGroupName });
      await fetchGroups();
      setShowCreateModal(false);
      setNewGroupName('');
      showAlert(t('groups.messages.createSuccess'), 'success');
    } catch (err: any) {
      showAlert(err?.msg || t('groups.messages.createFailed'), 'error');
    }
  };

  const handleDeleteGroup = async (groupId: number) => {
    if (!confirm(t('groups.messages.deleteConfirm'))) {
      return;
    }
    try {
      await deleteGroup(groupId);
      await fetchGroups();
      showAlert(t('groups.messages.deleteSuccess'), 'success');
    } catch (err: any) {
      showAlert(err?.msg || t('groups.messages.deleteFailed'), 'error');
    }
  };

  const handleLeaveGroup = async (groupId: number) => {
    if (!confirm(t('groups.messages.leaveConfirm'))) {
      return;
    }
    try {
      await leaveGroup(groupId);
      await fetchGroups();
      showAlert(t('groups.messages.leaveSuccess'), 'success');
    } catch (err: any) {
      showAlert(err?.msg || t('groups.messages.leaveFailed'), 'error');
    }
  };

  const handleInviteUser = async (groupId: number, userId: number) => {
    try {
      await inviteUser(groupId, { userId });
      setShowInviteModal(null);
      setSearchKeyword('');
      setSearchResults([]);
      showAlert(t('groups.messages.inviteSuccess'), 'success');
    } catch (err: any) {
      showAlert(err?.msg || t('groups.messages.inviteFailed'), 'error');
    }
  };

  const handleAcceptInvitation = async (invitationId: number) => {
    try {
      await acceptInvitation(invitationId);
      await fetchInvitations();
      await fetchGroups();
      showAlert(t('groups.messages.acceptSuccess'), 'success');
    } catch (err: any) {
      showAlert(err?.msg || t('groups.messages.acceptFailed'), 'error');
    }
  };

  const handleRejectInvitation = async (invitationId: number) => {
    try {
      await rejectInvitation(invitationId);
      await fetchInvitations();
      showAlert(t('groups.messages.rejectSuccess'), 'success');
    } catch (err: any) {
      showAlert(err?.msg || t('groups.messages.rejectFailed'), 'error');
    }
  };

  const isAdmin = (group: Group) => {
    const userId = user?.id ? Number(user.id) : null;
    return group.myRole === 'admin' || group.creatorId === userId;
  };

  const isCreator = (group: Group) => {
    const userId = user?.id ? Number(user.id) : null;
    return group.creatorId === userId;
  };

  return (
    <div className="min-h-screen dark:bg-neutral-900 flex flex-col">
      <div className="max-w-7xl w-full mx-auto pt-10 pb-4 px-4">
        <div className="flex items-center justify-between mb-8">
          <div className="relative pl-4">
            <motion.div
              layoutId="pageTitleIndicator"
              className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-8 bg-primary rounded-r-full"
              transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
            />
            <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">{t('groups.title')}</h1>
            <p className="text-gray-500 dark:text-gray-400 mt-2">{t('groups.subtitle')}</p>
          </div>
          <Button
            onClick={() => setShowCreateModal(true)}
            variant="primary"
            size="lg"
            leftIcon={<Plus className="w-5 h-5" />}
          >
            {t('groups.create')}
          </Button>
        </div>

        {/* 待处理的邀请 */}
        {invitations.length > 0 && (
          <div className="mb-8 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-xl p-6">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">{t('groups.pendingInvitations')}</h2>
            <div className="space-y-3">
              {invitations.map(invitation => (
                <div
                  key={invitation.id}
                  className="flex items-center justify-between p-4 bg-white dark:bg-neutral-800 rounded-lg border border-gray-200 dark:border-neutral-700"
                >
                  <div className="flex-1">
                    <div className="font-medium text-gray-900 dark:text-gray-100">
                      {invitation.inviter?.displayName || invitation.inviter?.email} {t('groups.inviteToJoin')}
                    </div>
                    <div className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                      {invitation.group.name}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      onClick={() => handleAcceptInvitation(invitation.id)}
                      variant="success"
                      size="sm"
                      leftIcon={<Check className="w-4 h-4" />}
                    >
                      {t('groups.accept')}
                    </Button>
                    <Button
                      onClick={() => handleRejectInvitation(invitation.id)}
                      variant="secondary"
                      size="sm"
                      leftIcon={<XCircle className="w-4 h-4" />}
                    >
                      {t('groups.reject')}
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* 组织列表 - 大卡片展示 */}
        {loading ? (
          <div className="text-center text-gray-400 py-16">{t('groups.loading')}</div>
        ) : groups.length === 0 ? (
          <div className="text-center py-16">
            <Users className="w-16 h-16 text-gray-400 mx-auto mb-4" />
            <div className="text-gray-400 text-lg mb-2">{t('groups.empty')}</div>
            <div className="text-gray-500 text-sm mb-6">{t('groups.emptyDesc')}</div>
            <Button
              onClick={() => setShowCreateModal(true)}
              variant="primary"
              size="lg"
            >
              {t('groups.create')}
            </Button>
          </div>
        ) : (
          <div className="space-y-6">
            {groups.map(group => (
              <div
                key={group.id}
                className="bg-white dark:bg-neutral-800 rounded-2xl border border-gray-200 dark:border-neutral-700 p-8 hover:shadow-xl transition-all"
              >
                <div className="flex items-start justify-between mb-6">
                  <div className="flex items-start gap-6 flex-1">
                    <img
                      src={group.avatar 
                        ? `${group.avatar}${group.avatar.includes('?') ? '&' : '?'}t=${new Date(group.updatedAt).getTime() || Date.now()}` 
                        : `https://ui-avatars.com/api/?name=${encodeURIComponent(group.name)}&background=6366f1&color=fff&size=80&bold=true`}
                      alt={group.name}
                      className="w-20 h-20 rounded-xl object-cover border-2 border-gray-200 dark:border-neutral-700 flex-shrink-0"
                      onError={(e) => {
                        // 如果头像加载失败，使用默认头像
                        const target = e.target as HTMLImageElement;
                        target.src = `https://ui-avatars.com/api/?name=${encodeURIComponent(group.name)}&background=6366f1&color=fff&size=80&bold=true`;
                      }}
                    />
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-2">
                        <h2 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                          {group.name}
                        </h2>
                        {isCreator(group) && (
                          <span className="px-3 py-1 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 rounded-full text-xs font-medium flex items-center gap-1">
                            <Crown className="w-3 h-3" />
                            {t('groups.creator')}
                          </span>
                        )}
                        {isAdmin(group) && !isCreator(group) && (
                          <span className="px-3 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200 rounded-full text-xs font-medium flex items-center gap-1">
                            <Shield className="w-3 h-3" />
                            {t('groups.admin')}
                          </span>
                        )}
                      </div>
                      <div className="space-y-2">
                        <div className="flex items-center gap-6 text-sm text-gray-500 dark:text-gray-400">
                          {group.type && (
                            <span className="px-2 py-1 bg-gray-100 dark:bg-neutral-700 rounded text-xs">
                              {group.type}
                            </span>
                          )}
                          <div className="flex items-center gap-2">
                            <Users className="w-4 h-4" />
                            <span>{group.memberCount || 0} {t('groups.members')}</span>
                          </div>
                          {group.creator && (
                            <div>
                              {t('groups.creatorLabel')} {group.creator.displayName || group.creator.email}
                            </div>
                          )}
                          <div>
                            {t('groups.createdAt')} {new Date(group.createdAt).toLocaleDateString()}
                          </div>
                        </div>
                        {group.extra && (
                          <div className="text-sm text-gray-600 dark:text-gray-300 line-clamp-1">
                            {group.extra.length > 12 ? `${group.extra.slice(0, 12)}...` : group.extra}
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 flex-wrap">
                    {isAdmin(group) && (
                      <Button
                        onClick={() => setShowInviteModal(group.id)}
                        variant="primary"
                        size="sm"
                        leftIcon={<UserPlus className="w-4 h-4" />}
                      >
                        {t('groups.inviteMember')}
                      </Button>
                    )}
                    <Button
                      onClick={() => navigate(`/groups/${group.id}/members`)}
                      variant="default"
                      size="sm"
                      leftIcon={<Users className="w-4 h-4" />}
                    >
                      {t('groups.memberManagement')}
                    </Button>
                    {isAdmin(group) && (
                      <Button
                        onClick={() => navigate(`/groups/${group.id}/settings`)}
                        variant="secondary"
                        size="sm"
                        leftIcon={<Settings className="w-4 h-4" />}
                      >
                        {t('groups.settings')}
                      </Button>
                    )}
                    {!isCreator(group) && (
                      <Button
                        onClick={() => handleLeaveGroup(group.id)}
                        variant="destructive"
                        size="sm"
                        leftIcon={<LogOut className="w-4 h-4" />}
                      >
                        {t('groups.leave')}
                      </Button>
                    )}
                    {isCreator(group) && (
                      <Button
                        onClick={() => handleDeleteGroup(group.id)}
                        variant="destructive"
                        size="sm"
                        leftIcon={<Trash2 className="w-4 h-4" />}
                      >
                        {t('groups.delete')}
                      </Button>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 创建组织模态框 */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-neutral-800 rounded-xl p-6 w-full max-w-md">
            <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-4">{t('groups.createModal.title')}</h2>
            <input
              type="text"
              value={newGroupName}
              onChange={(e) => setNewGroupName(e.target.value)}
              placeholder={t('groups.createModal.namePlaceholder')}
              className="w-full px-4 py-3 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-900 text-gray-900 dark:text-gray-100 mb-4"
              onKeyPress={(e) => e.key === 'Enter' && handleCreateGroup()}
            />
            <div className="flex items-center gap-2">
              <Button
                onClick={handleCreateGroup}
                variant="primary"
                size="md"
                fullWidth
              >
                {t('groups.create')}
              </Button>
              <Button
                onClick={() => {
                  setShowCreateModal(false);
                  setNewGroupName('');
                }}
                variant="secondary"
                size="md"
                fullWidth
              >
                {t('groups.createModal.cancel')}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* 邀请用户模态框 - 搜索选择模式 */}
      {showInviteModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-neutral-800 rounded-xl p-6 w-full max-w-2xl max-h-[80vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">{t('groups.inviteModal.title')}</h2>
              <button
                onClick={() => {
                  setShowInviteModal(null);
                  setSearchKeyword('');
                  setSearchResults([]);
                }}
                className="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
              >
                <X className="w-6 h-6" />
              </button>
            </div>
            <div className="relative mb-4">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
              <input
                type="text"
                value={searchKeyword}
                onChange={(e) => setSearchKeyword(e.target.value)}
                placeholder={t('groups.inviteModal.searchPlaceholder')}
                className="w-full pl-10 pr-4 py-3 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-900 text-gray-900 dark:text-gray-100"
              />
            </div>
            {searching && (
              <div className="text-center text-gray-400 py-4">{t('groups.inviteModal.searching')}</div>
            )}
            {!searching && searchKeyword && searchResults.length === 0 && (
              <div className="text-center text-gray-400 py-4">{t('groups.inviteModal.noResults')}</div>
            )}
            {!searching && searchResults.length > 0 && (
              <div className="space-y-2">
                {searchResults.map((result) => (
                  <div
                    key={result.id}
                    className="flex items-center justify-between p-4 border border-gray-200 dark:border-neutral-700 rounded-lg hover:bg-gray-50 dark:hover:bg-neutral-700 transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      <img
                        src={result.avatar || `https://ui-avatars.com/api/?name=${result.displayName || result.email}&background=0ea5e9&color=fff`}
                        alt={result.displayName || result.email}
                        className="w-10 h-10 rounded-full"
                      />
                      <div>
                        <div className="font-medium text-gray-900 dark:text-gray-100">
                          {result.displayName || result.email}
                        </div>
                        {result.displayName && (
                          <div className="text-sm text-gray-500 dark:text-gray-400">
                            {result.email}
                          </div>
                        )}
                      </div>
                    </div>
                    <button
                      onClick={() => handleInviteUser(showInviteModal, result.id)}
                      className="px-4 py-2 rounded-lg bg-purple-600 text-white text-sm hover:bg-purple-700 transition-colors flex items-center gap-2"
                    >
                      <UserPlus className="w-4 h-4" />
                      {t('groups.inviteModal.invite')}
                    </button>
                  </div>
                ))}
              </div>
            )}
            {!searchKeyword && (
              <div className="text-center text-gray-400 py-8">
                <Search className="w-12 h-12 mx-auto mb-2 opacity-50" />
                <p>{t('groups.inviteModal.searchHint')}</p>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default Groups;
