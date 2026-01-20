import React, { useEffect, useState, useRef } from 'react';
import { motion } from 'framer-motion';
import { useParams, useNavigate } from 'react-router-dom';
import { 
  getGroup,
  updateGroup,
  deleteGroup,
  getGroupSharedResources,
  uploadGroupAvatar,
  type Group,
  type UpdateGroupRequest,
  type GroupSharedResources
} from '@/api/group';
import {
  getGroupQuotas,
  deleteGroupQuota,
  type GroupQuota,
  getQuotaTypeLabel,
  formatQuotaValue
} from '@/api/quota';
import { showAlert } from '@/utils/notification';
import { useAuthStore } from '@/stores/authStore';
import { useI18nStore } from '@/stores/i18nStore';
import { ArrowLeft, Save, Trash2, AlertTriangle, Bot, BookOpen, Upload, X, Plus, Edit, Database, LayoutDashboard } from 'lucide-react';
import Button from '@/components/UI/Button';
import QuotaModal from '@/components/Quota/QuotaModal';
import { getOverviewConfig } from '@/api/overview';
import { OverviewConfig, defaultOverviewConfig } from '@/types/overview';

const GroupSettings: React.FC = () => {
  const { t } = useI18nStore();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const [group, setGroup] = useState<Group | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [resources, setResources] = useState<GroupSharedResources | null>(null);
  const [loadingResources, setLoadingResources] = useState(false);
  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const [avatarPreview, setAvatarPreview] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [quotas, setQuotas] = useState<GroupQuota[]>([]);
  const [loadingQuotas, setLoadingQuotas] = useState(false);
  const [showQuotaModal, setShowQuotaModal] = useState(false);
  const [editingQuota, setEditingQuota] = useState<GroupQuota | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    type: '',
    extra: '',
  });
  const [overviewConfig, setOverviewConfig] = useState<OverviewConfig | null>(null);
  const [loadingOverview, setLoadingOverview] = useState(false);

  const fetchGroup = async () => {
    if (!id) return;
    try {
      setLoading(true);
      const res = await getGroup(Number(id));
      setGroup(res.data);
      setFormData({
        name: res.data.name || '',
        type: res.data.type || '',
        extra: res.data.extra || '',
      });
      setAvatarPreview(res.data.avatar || null);
    } catch (err: any) {
      showAlert(err?.msg || t('groupSettings.messages.fetchGroupFailed'), 'error');
      navigate('/groups');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchGroup();
    if (id) {
      fetchResources();
      fetchQuotas();
      fetchOverviewConfig();
    }
  }, [id]);

  const fetchQuotas = async () => {
    if (!id) return;
    try {
      setLoadingQuotas(true);
      const res = await getGroupQuotas(Number(id));
      setQuotas(res.data || []);
    } catch (err: any) {
      showAlert(err?.msg || t('groupSettings.messages.fetchQuotasFailed'), 'error');
    } finally {
      setLoadingQuotas(false);
    }
  };

  const fetchResources = async () => {
    if (!id) return;
    try {
      setLoadingResources(true);
      const res = await getGroupSharedResources(Number(id));
      setResources(res.data);
    } catch (err: any) {
      console.error('获取组织资源失败', err);
    } finally {
      setLoadingResources(false);
    }
  };

  const fetchOverviewConfig = async () => {
    if (!id) return;
    try {
      setLoadingOverview(true);
      const configRes = await getOverviewConfig(Number(id));
      if (configRes.code === 200 && configRes.data) {
        const backendConfig = configRes.data as any;
        const currentGroup = group || (await getGroup(Number(id))).data;
        const loadedConfig: OverviewConfig = {
          id: backendConfig.id || `config-${Date.now()}`,
          organizationId: backendConfig.organizationId || Number(id),
          name: backendConfig.name || `${currentGroup?.name || ''} - 概览`,
          description: backendConfig.description,
          layout: {
            ...defaultOverviewConfig.layout,
            ...(backendConfig.layout || {})
          },
          widgets: backendConfig.widgets || [],
          theme: {
            ...defaultOverviewConfig.theme,
            ...(backendConfig.theme || {})
          },
          header: backendConfig.header || defaultOverviewConfig.header,
          footer: backendConfig.footer || defaultOverviewConfig.footer,
          createdAt: backendConfig.createdAt,
          updatedAt: backendConfig.updatedAt
        };
        setOverviewConfig(loadedConfig);
        
      } else {
        // 如果没有配置，创建默认配置
        const currentGroup = group || (await getGroup(Number(id))).data;
        const defaultConfig: OverviewConfig = {
          ...defaultOverviewConfig,
          id: `config-${Date.now()}`,
          organizationId: Number(id),
          name: `${currentGroup?.name || ''} - 概览`,
        };
        setOverviewConfig(defaultConfig);
      }
    } catch (err: any) {
      console.warn('获取概览配置失败:', err);
      // 使用默认配置
      const currentGroup = group || (await getGroup(Number(id))).data;
      const defaultConfig: OverviewConfig = {
        ...defaultOverviewConfig,
        id: `config-${Date.now()}`,
        organizationId: Number(id),
        name: `${currentGroup?.name || ''} - 概览`,
      };
      setOverviewConfig(defaultConfig);
    } finally {
      setLoadingOverview(false);
    }
  };


  const handleSave = async () => {
    if (!id || !group) return;
    if (!formData.name.trim()) {
      showAlert(t('groupSettings.messages.nameRequired'), 'error');
      return;
    }

    try {
      setSaving(true);
      const updateData: UpdateGroupRequest = {
        name: formData.name,
        type: formData.type || undefined,
        extra: formData.extra || undefined,
      };
      await updateGroup(Number(id), updateData);
      await fetchGroup();
      showAlert(t('groupSettings.messages.saveSuccess'), 'success');
    } catch (err: any) {
      showAlert(err?.msg || t('groupSettings.messages.saveFailed'), 'error');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!id) return;
    if (!confirm(t('groupSettings.deleteConfirm'))) {
      return;
    }
    if (!confirm(t('groupSettings.deleteConfirmAgain'))) {
      return;
    }
    try {
      await deleteGroup(Number(id));
      showAlert(t('groupSettings.messages.deleteSuccess'), 'success');
      navigate('/groups');
    } catch (err: any) {
      showAlert(err?.msg || t('groupSettings.messages.deleteFailed'), 'error');
    }
  };

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // 验证文件类型
    if (!file.type.startsWith('image/')) {
      showAlert(t('groupSettings.messages.selectImage'), 'error');
      return;
    }

    // 验证文件大小 (5MB)
    if (file.size > 5 * 1024 * 1024) {
      showAlert(t('groupSettings.messages.imageTooLarge'), 'error');
      return;
    }

    // 预览
    const reader = new FileReader();
    reader.onloadend = () => {
      setAvatarPreview(reader.result as string);
    };
    reader.readAsDataURL(file);
  };

  const handleAvatarUpload = async () => {
    if (!id || !avatarPreview) return;

    const file = fileInputRef.current?.files?.[0];
    if (!file) {
      showAlert(t('groupSettings.messages.selectImageFirst'), 'error');
      return;
    }

    try {
      setUploadingAvatar(true);
      const res = await uploadGroupAvatar(Number(id), file);
      setGroup(prev => prev ? { ...prev, avatar: res.data.avatar } : null);
      setAvatarPreview(res.data.avatar);
      showAlert(t('groupSettings.messages.avatarUploadSuccess'), 'success');
    } catch (err: any) {
      showAlert(err?.msg || t('groupSettings.messages.avatarUploadFailed'), 'error');
      // 恢复原来的头像
      if (group) {
        setAvatarPreview(group.avatar || null);
      }
    } finally {
      setUploadingAvatar(false);
    }
  };

  const handleRemoveAvatar = () => {
    setAvatarPreview(group?.avatar || null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const getDefaultAvatar = (groupName: string) => {
    return `https://ui-avatars.com/api/?name=${encodeURIComponent(groupName)}&background=6366f1&color=fff&size=96&bold=true`;
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

  if (!isCreator()) {
    return (
      <div className="min-h-screen dark:bg-neutral-900 flex items-center justify-center">
        <div className="text-center">
          <AlertTriangle className="w-16 h-16 text-yellow-500 mx-auto mb-4" />
          <div className="text-gray-400 text-lg mb-2">{t('groupSettings.insufficientPermissions')}</div>
          <div className="text-gray-500 text-sm mb-6">{t('groupSettings.insufficientPermissionsDesc')}</div>
          <button
            onClick={() => navigate('/groups')}
            className="px-6 py-3 rounded-lg bg-purple-600 text-white hover:bg-purple-700 transition-colors"
          >
            {t('groupSettings.backToList')}
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen dark:bg-neutral-900 flex flex-col">
      <div className="max-w-4xl w-full mx-auto pt-10 pb-4 px-4">
        {/* 头部 */}
        <div className="mb-8">
          <Button
            onClick={() => navigate('/groups')}
            variant="ghost"
            size="sm"
            leftIcon={<ArrowLeft className="w-4 h-4" />}
            className="mb-4"
          >
            {t('groupSettings.backToList')}
          </Button>
          <div className="relative pl-4">
            <motion.div
              layoutId="pageTitleIndicator"
              className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-8 bg-primary rounded-r-full"
              transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
            />
            <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-2">
              {group.name} - {t('groupSettings.title')}
            </h1>
            <p className="text-gray-500 dark:text-gray-400">
              {t('groupSettings.subtitle')}
            </p>
          </div>
        </div>

        {/* 组织头像 */}
        <div className="bg-white dark:bg-neutral-800 rounded-2xl border border-gray-200 dark:border-neutral-700 p-6 mb-6">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-6">{t('groupSettings.avatar')}</h2>
          <div className="flex items-center gap-6">
            <div className="relative">
              <img
                src={avatarPreview 
                  ? `${avatarPreview}${avatarPreview.includes('?') ? '&' : '?'}t=${Date.now()}` 
                  : (group ? getDefaultAvatar(group.name) : '')}
                alt={t('groupSettings.avatar')}
                className="w-24 h-24 rounded-xl object-cover border-2 border-gray-200 dark:border-neutral-700"
                onError={(e) => {
                  // 如果头像加载失败，使用默认头像
                  const target = e.target as HTMLImageElement;
                  if (group) {
                    target.src = getDefaultAvatar(group.name);
                  }
                }}
              />
            </div>
            <div className="flex-1 space-y-3">
              <div className="flex items-center gap-3">
                <Button
                  onClick={() => fileInputRef.current?.click()}
                  variant="outline"
                  size="md"
                  leftIcon={<Upload className="w-4 h-4" />}
                >
                  {t('groupSettings.uploadAvatar')}
                </Button>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/jpeg,image/jpg,image/png,image/gif,image/webp"
                  onChange={handleAvatarChange}
                  className="hidden"
                />
                {avatarPreview && avatarPreview !== group?.avatar && (
                  <>
                    <Button
                      onClick={handleAvatarUpload}
                      variant="primary"
                      size="md"
                      loading={uploadingAvatar}
                      disabled={uploadingAvatar}
                    >
                      {t('groupSettings.changeAvatar')}
                    </Button>
                    <Button
                      onClick={handleRemoveAvatar}
                      variant="ghost"
                      size="md"
                      leftIcon={<X className="w-4 h-4" />}
                    >
                      {t('groups.createModal.cancel')}
                    </Button>
                  </>
                )}
              </div>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                {t('groupSettings.avatarFormatHint')}
              </p>
            </div>
          </div>
        </div>

        {/* 基本信息 */}
        <div className="bg-white dark:bg-neutral-800 rounded-2xl border border-gray-200 dark:border-neutral-700 p-6 mb-6">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-6">{t('groupSettings.basicInfo')}</h2>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {t('groupSettings.name')} <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className="w-full px-4 py-3 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-900 text-gray-900 dark:text-gray-100"
                placeholder={t('groupSettings.namePlaceholder')}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {t('groupSettings.type')}
              </label>
              <input
                type="text"
                value={formData.type}
                onChange={(e) => setFormData({ ...formData, type: e.target.value })}
                className="w-full px-4 py-3 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-900 text-gray-900 dark:text-gray-100"
                placeholder={t('groupSettings.typePlaceholder')}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {t('groupSettings.description')}
              </label>
              <textarea
                value={formData.extra}
                onChange={(e) => setFormData({ ...formData, extra: e.target.value })}
                rows={4}
                className="w-full px-4 py-3 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-900 text-gray-900 dark:text-gray-100"
                placeholder={t('groupSettings.descriptionPlaceholder')}
              />
            </div>
          </div>
          <div className="flex items-center gap-3 mt-6">
            <Button
              onClick={handleSave}
              disabled={saving}
              loading={saving}
              variant="primary"
              size="lg"
              leftIcon={<Save className="w-4 h-4" />}
            >
              {t('groupSettings.save')}
            </Button>
          </div>
        </div>

        {/* 组织共享的资源 */}
        <div className="bg-white dark:bg-neutral-800 rounded-2xl border border-gray-200 dark:border-neutral-700 p-6 mb-6">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-6">{t('groupSettings.sharedResources')}</h2>
          
          {loadingResources ? (
            <div className="text-center text-gray-400 py-8">{t('groups.loading')}</div>
          ) : (
            <div className="space-y-6">
              {/* 助手列表 */}
              <div>
                <h3 className="text-lg font-medium text-gray-800 dark:text-gray-200 mb-4 flex items-center gap-2">
                  <Bot className="w-5 h-5" />
                  {t('groupSettings.assistants')} ({resources?.assistants?.length || 0})
                </h3>
                {resources?.assistants && resources.assistants.length > 0 ? (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {resources.assistants.map(assistant => (
                      <div
                        key={assistant.id}
                        className="p-4 border border-gray-200 dark:border-neutral-700 rounded-lg hover:border-purple-400 transition-colors cursor-pointer"
                        onClick={() => navigate(`/voice-assistant/${assistant.id}`)}
                      >
                        <div className="font-medium text-gray-900 dark:text-gray-100 mb-1">{assistant.name}</div>
                        <div className="text-sm text-gray-500 dark:text-gray-400 line-clamp-2">{assistant.description}</div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-gray-400 text-sm py-4">{t('groupSettings.noSharedAssistants')}</div>
                )}
              </div>

              {/* 知识库列表 */}
              <div>
                <h3 className="text-lg font-medium text-gray-800 dark:text-gray-200 mb-4 flex items-center gap-2">
                  <BookOpen className="w-5 h-5" />
                  {t('groupSettings.knowledgeBases')} ({resources?.knowledgeBases?.length || 0})
                </h3>
                {resources?.knowledgeBases && resources.knowledgeBases.length > 0 ? (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {resources.knowledgeBases.map(kb => (
                      <div
                        key={kb.id}
                        className="p-4 border border-gray-200 dark:border-neutral-700 rounded-lg"
                      >
                        <div className="font-medium text-gray-900 dark:text-gray-100 mb-1">{kb.knowledge_name}</div>
                        <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">Key: {kb.knowledge_key}</div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-gray-400 text-sm py-4">{t('groupSettings.noSharedKnowledgeBases')}</div>
                )}
              </div>
            </div>
          )}
        </div>

        {/* 概览页面配置 */}
        <div className="bg-white dark:bg-neutral-800 rounded-2xl border border-gray-200 dark:border-neutral-700 p-6 mb-6">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 flex items-center gap-2">
              <LayoutDashboard className="w-5 h-5" />
              概览页面
            </h2>
            <Button
              onClick={() => navigate(`/groups/${id}/overview/edit`)}
              variant="primary"
              size="sm"
              leftIcon={<Edit className="w-4 h-4" />}
            >
              编辑概览页面
            </Button>
          </div>

          {loadingOverview ? (
            <div className="text-center text-gray-400 py-8">{t('groups.loading')}</div>
          ) : (
            <div className="space-y-4">
              <p className="text-gray-600 dark:text-gray-400 text-sm">
                {overviewConfig 
                  ? `当前概览页面已配置，包含 ${overviewConfig.widgets?.length || 0} 个组件。点击"编辑概览页面"按钮进行编辑。`
                  : '概览页面尚未配置，点击"编辑概览页面"按钮开始创建。'}
              </p>
              {overviewConfig && (
                <div className="bg-gray-50 dark:bg-neutral-900 rounded-lg p-4">
                  <div className="text-sm space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="text-gray-600 dark:text-gray-400">配置名称:</span>
                      <span className="font-medium text-gray-900 dark:text-gray-100">{overviewConfig.name}</span>
                    </div>
                    {overviewConfig.description && (
                      <div className="flex items-center justify-between">
                        <span className="text-gray-600 dark:text-gray-400">描述:</span>
                        <span className="font-medium text-gray-900 dark:text-gray-100">{overviewConfig.description}</span>
                      </div>
                    )}
                    <div className="flex items-center justify-between">
                      <span className="text-gray-600 dark:text-gray-400">组件数量:</span>
                      <span className="font-medium text-gray-900 dark:text-gray-100">{overviewConfig.widgets?.length || 0}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-gray-600 dark:text-gray-400">主题样式:</span>
                      <span className="font-medium text-gray-900 dark:text-gray-100">{overviewConfig.theme?.style || 'modern'}</span>
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* 配额管理 */}
        <div className="bg-white dark:bg-neutral-800 rounded-2xl border border-gray-200 dark:border-neutral-700 p-6 mb-6">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 flex items-center gap-2">
              <Database className="w-5 h-5" />
              {t('groupSettings.quotaManagement')}
            </h2>
            <Button
              onClick={() => {
                setEditingQuota(null);
                setShowQuotaModal(true);
              }}
              variant="primary"
              size="sm"
              leftIcon={<Plus className="w-4 h-4" />}
            >
              {t('groupSettings.addQuota')}
            </Button>
          </div>

          {loadingQuotas ? (
            <div className="text-center text-gray-400 py-8">{t('groups.loading')}</div>
          ) : quotas.length === 0 ? (
            <div className="text-center text-gray-400 py-8">
              <p className="mb-4">{t('groupSettings.noQuotas')}</p>
              <p className="text-sm text-gray-500 dark:text-gray-500">
                {t('groupSettings.quotaDesc')}
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              {quotas.map(quota => {
                const percentage = quota.totalQuota > 0 
                  ? ((quota.usedQuota / quota.totalQuota) * 100).toFixed(2)
                  : '0';
                return (
                  <div
                    key={quota.id}
                    className="border border-gray-200 dark:border-neutral-700 rounded-lg p-4 hover:border-purple-400 transition-colors"
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-3 mb-2">
                          <h3 className="font-semibold text-gray-900 dark:text-gray-100">
                            {getQuotaTypeLabel(quota.quotaType)}
                          </h3>
                          <span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-neutral-700 text-gray-600 dark:text-gray-400">
                            {quota.period === 'lifetime' ? t('groupSettings.period.lifetime') : quota.period === 'monthly' ? t('groupSettings.period.monthly') : t('groupSettings.period.yearly')}
                          </span>
                        </div>
                        <div className="space-y-2">
                          <div className="flex items-center justify-between text-sm">
                            <span className="text-gray-600 dark:text-gray-400">{t('groupSettings.used')}</span>
                            <span className="font-medium">{formatQuotaValue(quota.quotaType, quota.usedQuota)}</span>
                          </div>
                          <div className="flex items-center justify-between text-sm">
                            <span className="text-gray-600 dark:text-gray-400">{t('groupSettings.totalQuota')}</span>
                            <span className="font-medium">{quota.totalQuota === 0 ? t('groupSettings.unlimited') : formatQuotaValue(quota.quotaType, quota.totalQuota)}</span>
                          </div>
                          {quota.totalQuota > 0 && (
                            <div className="mt-3">
                              <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400 mb-1">
                                <span>{t('groupSettings.usageRate')}</span>
                                <span>{percentage}%</span>
                              </div>
                              <div className="w-full bg-gray-200 dark:bg-neutral-700 rounded-full h-2">
                                <div
                                  className={`h-2 rounded-full transition-all ${
                                    parseFloat(percentage) >= 90
                                      ? 'bg-red-500'
                                      : parseFloat(percentage) >= 75
                                      ? 'bg-orange-500'
                                      : parseFloat(percentage) >= 50
                                      ? 'bg-yellow-500'
                                      : 'bg-green-500'
                                  }`}
                                  style={{ width: `${Math.min(parseFloat(percentage), 100)}%` }}
                                />
                              </div>
                            </div>
                          )}
                          {quota.description && (
                            <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">{quota.description}</p>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-2 ml-4">
                        <Button
                          onClick={() => {
                            setEditingQuota(quota);
                            setShowQuotaModal(true);
                          }}
                          variant="ghost"
                          size="sm"
                          leftIcon={<Edit className="w-4 h-4" />}
                        >
                          {t('alertRules.edit')}
                        </Button>
                        <Button
                          onClick={async () => {
                            if (!confirm(t('groupSettings.messages.deleteConfirm'))) return;
                            try {
                              await deleteGroupQuota(Number(id), quota.quotaType);
                              showAlert(t('groupSettings.messages.deleteSuccess'), 'success');
                              fetchQuotas();
                            } catch (err: any) {
                              showAlert(err?.msg || t('groupSettings.messages.deleteFailed'), 'error');
                            }
                          }}
                          variant="ghost"
                          size="sm"
                          leftIcon={<Trash2 className="w-4 h-4" />}
                        >
                          {t('alertRules.delete')}
                        </Button>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* 危险操作 */}
        <div className="bg-white dark:bg-neutral-800 rounded-2xl border border-red-200 dark:border-red-900/30 p-6">
          <h2 className="text-xl font-semibold text-red-600 dark:text-red-400 mb-4">{t('groupSettings.dangerousOperations')}</h2>
          <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 mb-4">
            <div className="flex items-start gap-3">
              <AlertTriangle className="w-5 h-5 text-red-600 dark:text-red-400 mt-0.5" />
              <div>
                <div className="font-medium text-red-900 dark:text-red-200 mb-1">{t('groupSettings.deleteGroup')}</div>
                <div className="text-sm text-red-700 dark:text-red-300">
                  {t('groupSettings.deleteGroupDesc')}
                  <ul className="list-disc list-inside mt-2 space-y-1">
                    <li>{t('groupSettings.deleteGroupItem1')}</li>
                    <li>{t('groupSettings.deleteGroupItem2')}</li>
                    <li>{t('groupSettings.deleteGroupItem3')}</li>
                  </ul>
                  {t('groupSettings.deleteGroupWarning')}
                </div>
              </div>
            </div>
          </div>
          <Button
            onClick={handleDelete}
            variant="destructive"
            size="lg"
            leftIcon={<Trash2 className="w-4 h-4" />}
          >
            {t('groupSettings.delete')}
          </Button>
        </div>
      </div>

      {/* 配额管理弹窗 */}
      {showQuotaModal && (
        <QuotaModal
          isOpen={showQuotaModal}
          onClose={() => {
            setShowQuotaModal(false);
            setEditingQuota(null);
          }}
          groupId={Number(id)}
          quota={editingQuota}
          onSuccess={() => {
            fetchQuotas();
          }}
        />
      )}
    </div>
  );
};

export default GroupSettings;

