import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { 
  getGroup,
  removeMember,
  inviteUser,
  searchUsers,
  updateMemberRole,
  type Group,
  type GroupMember,
  type UserSearchResult
} from '@/api/group';
import { showAlert } from '@/utils/notification';
import { useAuthStore } from '@/stores/authStore';
import { useI18nStore } from '@/stores/i18nStore';
import { Users, X, UserPlus, Trash2, Crown, Shield, Search, ArrowLeft, Settings, ChevronDown } from 'lucide-react';
import Button from '@/components/UI/Button';

const GroupMembers: React.FC = () => {
  const { t } = useI18nStore();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const [group, setGroup] = useState<Group | null>(null);
  const [members, setMembers] = useState<GroupMember[]>([]);
  const [showInviteModal, setShowInviteModal] = useState(false);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchResults, setSearchResults] = useState<UserSearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [loading, setLoading] = useState(false);
  const [openRoleMenu, setOpenRoleMenu] = useState<number | null>(null);

  const fetchGroup = async () => {
    if (!id) return;
    try {
      setLoading(true);
      const res = await getGroup(Number(id));
      setGroup(res.data);
      setMembers(res.data.members || []);
    } catch (err: any) {
      showAlert(err?.msg || t('groupMembers.messages.fetchGroupFailed'), 'error');
      navigate('/groups');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchGroup();
  }, [id]);

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

  const handleInviteUser = async (userId: number) => {
    if (!id) return;
    try {
      await inviteUser(Number(id), { userId });
      setShowInviteModal(false);
      setSearchKeyword('');
      setSearchResults([]);
      await fetchGroup();
      showAlert(t('groups.messages.inviteSuccess'), 'success');
    } catch (err: any) {
      showAlert(err?.msg || t('groups.messages.inviteFailed'), 'error');
    }
  };

  const handleRemoveMember = async (memberId: number) => {
    if (!id) return;
    if (!confirm(t('groupMembers.messages.removeConfirm'))) {
      return;
    }
    try {
      await removeMember(Number(id), memberId);
      await fetchGroup();
      showAlert(t('groupMembers.messages.removeSuccess'), 'success');
    } catch (err: any) {
      showAlert(err?.msg || t('groupMembers.messages.removeFailed'), 'error');
    }
  };

  const handleUpdateRole = async (memberId: number, role: string) => {
    if (!id) return;
    try {
      await updateMemberRole(Number(id), memberId, role);
      await fetchGroup();
      showAlert(t('groupMembers.messages.roleUpdateSuccess'), 'success');
    } catch (err: any) {
      showAlert(err?.msg || t('groupMembers.messages.roleUpdateFailed'), 'error');
    }
  };

  const isAdmin = () => {
    if (!group || !user) return false;
    const userId = user.id ? Number(user.id) : null;
    return group.myRole === 'admin' || group.creatorId === userId;
  };

  const isCreator = () => {
    if (!group || !user) return false;
    const userId = user.id ? Number(user.id) : null;
    return group.creatorId === userId;
  };

  if (loading) {
    return (
      <div className="min-h-screen dark:bg-neutral-900 flex items-center justify-center">
        <div className="text-gray-400">{t('groups.loading')}</div>
      </div>
    );
  }

  if (!group) {
    return null;
  }

  return (
    <div className="min-h-screen dark:bg-neutral-900 flex flex-col">
      <div className="max-w-6xl w-full mx-auto pt-10 pb-4 px-4">
        {/* 头部 */}
        <div className="mb-8">
          <Button
            onClick={() => navigate('/groups')}
            variant="ghost"
            size="sm"
            leftIcon={<ArrowLeft className="w-4 h-4" />}
            className="mb-4"
          >
            {t('groupMembers.backToList')}
          </Button>
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-2">
                {group.name} - {t('groupMembers.title')}
              </h1>
              <p className="text-gray-500 dark:text-gray-400">
                {t('groupMembers.memberCount').replace('{count}', String(members.length))}
              </p>
            </div>
            {isAdmin() && (
              <Button
                onClick={() => setShowInviteModal(true)}
                variant="primary"
                size="lg"
                leftIcon={<UserPlus className="w-5 h-5" />}
              >
                {t('groups.inviteMember')}
              </Button>
            )}
          </div>
        </div>

        {/* 成员列表 */}
        <div className="bg-white dark:bg-neutral-800 rounded-2xl border border-gray-200 dark:border-neutral-700 p-6">
          <div className="space-y-4">
            {members.map((member) => {
              const isCurrentUser = user?.id ? Number(user.id) === member.userId : false;
              const isMemberCreator = group.creatorId === member.userId;
              
              return (
                <div
                  key={member.id}
                  className="flex items-center justify-between p-4 border border-gray-200 dark:border-neutral-700 rounded-lg hover:bg-gray-50 dark:hover:bg-neutral-700 transition-colors"
                >
                  <div className="flex items-center gap-4">
                    <img
                      src={member.user.avatar || `https://ui-avatars.com/api/?name=${member.user.displayName || member.user.email}&background=0ea5e9&color=fff`}
                      alt={member.user.displayName || member.user.email}
                      className="w-12 h-12 rounded-full"
                    />
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-gray-900 dark:text-gray-100">
                          {member.user.displayName || member.user.email}
                        </span>
                        {isMemberCreator && (
                          <span className="px-2 py-1 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 rounded-full text-xs font-medium flex items-center gap-1">
                            <Crown className="w-3 h-3" />
                            {t('groups.creator')}
                          </span>
                        )}
                        {member.role === 'admin' && !isMemberCreator && (
                          <span className="px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200 rounded-full text-xs font-medium flex items-center gap-1">
                            <Shield className="w-3 h-3" />
                            {t('groups.admin')}
                          </span>
                        )}
                        {member.role === 'member' && (
                          <span className="px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded-full text-xs font-medium">
                            {t('groupMembers.member')}
                          </span>
                        )}
                        {isCurrentUser && (
                          <span className="px-2 py-1 bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200 rounded-full text-xs font-medium">
                            {t('groupMembers.me')}
                          </span>
                        )}
                      </div>
                      {member.user.displayName && (
                        <div className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                          {member.user.email}
                        </div>
                      )}
                      <div className="text-xs text-gray-400 dark:text-gray-500 mt-1">
                        {t('groupMembers.joinedAt')} {new Date(member.createdAt).toLocaleDateString()}
                      </div>
                    </div>
                  </div>
                  {isAdmin() && !isCurrentUser && !isMemberCreator && (
                    <div className="flex items-center gap-2">
                      <div className="relative">
                        <Button
                          variant="secondary"
                          size="sm"
                          rightIcon={<ChevronDown className="w-4 h-4" />}
                          onClick={(e) => {
                            e.stopPropagation();
                            setOpenRoleMenu(openRoleMenu === member.id ? null : member.id);
                          }}
                        >
                          {member.role === 'admin' ? t('groups.admin') : t('groupMembers.member')}
                        </Button>
                        {openRoleMenu === member.id && (
                          <>
                            <div 
                              className="fixed inset-0 z-10" 
                              onClick={() => setOpenRoleMenu(null)}
                            />
                            <div className="absolute right-0 mt-1 w-36 bg-white dark:bg-neutral-800 border border-gray-200 dark:border-neutral-700 rounded-lg shadow-lg z-20">
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleUpdateRole(member.id, 'admin');
                                  setOpenRoleMenu(null);
                                }}
                                disabled={member.role === 'admin'}
                                className="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-neutral-700 rounded-t-lg disabled:opacity-50 disabled:cursor-not-allowed"
                              >
                                {t('groupMembers.setAsAdmin')}
                              </button>
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleUpdateRole(member.id, 'member');
                                  setOpenRoleMenu(null);
                                }}
                                disabled={member.role === 'member'}
                                className="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-neutral-700 rounded-b-lg disabled:opacity-50 disabled:cursor-not-allowed"
                              >
                                {t('groupMembers.setAsMember')}
                              </button>
                            </div>
                          </>
                        )}
                      </div>
                      <Button
                        onClick={() => handleRemoveMember(member.id)}
                        variant="destructive"
                        size="sm"
                        leftIcon={<Trash2 className="w-4 h-4" />}
                      >
                        {t('groupMembers.remove')}
                      </Button>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* 邀请用户模态框 */}
      {showInviteModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-neutral-800 rounded-xl p-6 w-full max-w-2xl max-h-[80vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">{t('groups.inviteModal.title')}</h2>
              <Button
                onClick={() => {
                  setShowInviteModal(false);
                  setSearchKeyword('');
                  setSearchResults([]);
                }}
                variant="ghost"
                size="icon"
              >
                <X className="w-6 h-6" />
              </Button>
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
                {searchResults.map((result) => {
                  const isAlreadyMember = members.some(m => m.userId === result.id);
                  return (
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
                      {isAlreadyMember ? (
                        <span className="px-4 py-2 rounded-lg bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-400 text-sm">
                          {t('groupMembers.alreadyMember')}
                        </span>
                      ) : (
                        <Button
                          onClick={() => handleInviteUser(result.id)}
                          variant="primary"
                          size="sm"
                          leftIcon={<UserPlus className="w-4 h-4" />}
                        >
                          {t('groups.inviteModal.invite')}
                        </Button>
                      )}
                    </div>
                  );
                })}
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

export default GroupMembers;

