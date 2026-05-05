import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { getGroup, getGroupActivityLogs, type Group, type GroupActivityLog } from '@/api/group';
import { showAlert } from '@/utils/notification';
import { useI18nStore } from '@/stores/i18nStore';
import { ArrowLeft, Clock, User, RefreshCw } from 'lucide-react';
import Button from '@/components/UI/Button';
import LoadingAnimation from '@/components/Animations/LoadingAnimation';
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/UI/Select';
import { cn } from '@/utils/cn';

const GroupActivityLogs: React.FC = () => {
  const { t } = useI18nStore();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [group, setGroup] = useState<Group | null>(null);
  const [logs, setLogs] = useState<GroupActivityLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(20);
  const [actionFilter, setActionFilter] = useState('');
  const [resourceFilter, setResourceFilter] = useState('');

  const fetchGroup = async () => {
    if (!id) return;
    try {
      const res = await getGroup(Number(id));
      setGroup(res.data);
    } catch (err: any) {
      showAlert(err?.msg || 'Failed to fetch group', 'error');
      navigate('/profile/teams');
    }
  };

  const fetchLogs = async () => {
    if (!id) return;
    try {
      setLoading(true);
      const res = await getGroupActivityLogs(Number(id), {
        page: currentPage,
        pageSize,
        action: actionFilter || undefined,
        resourceType: resourceFilter || undefined,
      });
      setLogs(res.data.logs || []);
      setTotal(res.data.total || 0);
    } catch (err: any) {
      showAlert(err?.msg || 'Failed to fetch activity logs', 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchGroup();
  }, [id]);

  useEffect(() => {
    if (currentPage === 1) {
      fetchLogs();
    } else {
      setCurrentPage(1);
    }
  }, [actionFilter, resourceFilter]);

  useEffect(() => {
    fetchLogs();
  }, [currentPage]);

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const getActionLabel = (action: string) => {
    return t(`activityLog.action.${action}`) || action;
  };

  const getResourceLabel = (resourceType: string) => {
    return t(`activityLog.resource.${resourceType}`) || resourceType;
  };

  const totalPages = Math.ceil(total / pageSize);

  if (!group) {
    return (
      <div className="min-h-screen dark:bg-neutral-900 flex items-center justify-center">
        <LoadingAnimation type="progress" size="lg" />
      </div>
    );
  }

  return (
    <div className="min-h-screen dark:bg-neutral-900 flex flex-col">
      <div className="max-w-7xl w-full mx-auto pt-10 pb-4 px-4">
        <div className="mb-8">
          <Button
            onClick={() => navigate(`/groups/${id}/settings`)}
            variant="ghost"
            size="sm"
            leftIcon={<ArrowLeft className="w-4 h-4" />}
            className="mb-4"
          >
            {t('activityLog.backToSettings')}
          </Button>
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-2">
              {group.name} - {t('activityLog.title')}
            </h1>
            <p className="text-gray-500 dark:text-gray-400">
              {t('activityLog.totalRecords').replace('{count}', String(total))}
            </p>
          </div>
        </div>

        <div className="bg-white dark:bg-neutral-800 rounded-xl border border-gray-200 dark:border-neutral-700 p-4 mb-6">
          <div className="flex flex-col sm:flex-row gap-4">
            <Select value={actionFilter} onValueChange={setActionFilter}>
              <SelectTrigger className="w-full sm:w-[200px]">
                <SelectValue placeholder={t('activityLog.filterByAction')}>
                  {actionFilter === '' ? t('activityLog.allActions') : getActionLabel(actionFilter)}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">{t('activityLog.allActions')}</SelectItem>
                <SelectItem value="member_invited">{t('activityLog.action.member_invited')}</SelectItem>
                <SelectItem value="member_joined">{t('activityLog.action.member_joined')}</SelectItem>
                <SelectItem value="member_removed">{t('activityLog.action.member_removed')}</SelectItem>
                <SelectItem value="member_role_changed">{t('activityLog.action.member_role_changed')}</SelectItem>
                <SelectItem value="resource_added">{t('activityLog.action.resource_added')}</SelectItem>
                <SelectItem value="resource_removed">{t('activityLog.action.resource_removed')}</SelectItem>
                <SelectItem value="group_created">{t('activityLog.action.group_created')}</SelectItem>
                <SelectItem value="group_updated">{t('activityLog.action.group_updated')}</SelectItem>
                <SelectItem value="group_archived">{t('activityLog.action.group_archived')}</SelectItem>
                <SelectItem value="group_restored">{t('activityLog.action.group_restored')}</SelectItem>
                <SelectItem value="group_exported">{t('activityLog.action.group_exported')}</SelectItem>
                <SelectItem value="group_cloned">{t('activityLog.action.group_cloned')}</SelectItem>
                <SelectItem value="quota_added">{t('activityLog.action.quota_added')}</SelectItem>
                <SelectItem value="quota_updated">{t('activityLog.action.quota_updated')}</SelectItem>
                <SelectItem value="quota_deleted">{t('activityLog.action.quota_deleted')}</SelectItem>
              </SelectContent>
            </Select>

            <Select value={resourceFilter} onValueChange={setResourceFilter}>
              <SelectTrigger className="w-full sm:w-[200px]">
                <SelectValue placeholder={t('activityLog.filterByResource')}>
                  {resourceFilter === '' ? t('activityLog.allResources') : getResourceLabel(resourceFilter)}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">{t('activityLog.allResources')}</SelectItem>
                <SelectItem value="assistant">{t('activityLog.resource.assistant')}</SelectItem>
                <SelectItem value="knowledge">{t('activityLog.resource.knowledge')}</SelectItem>
                <SelectItem value="member">{t('activityLog.resource.member')}</SelectItem>
                <SelectItem value="quota">{t('activityLog.resource.quota')}</SelectItem>
                <SelectItem value="group">{t('activityLog.resource.group')}</SelectItem>
              </SelectContent>
            </Select>

            <Button
              variant="outline"
              onClick={fetchLogs}
              disabled={loading}
              leftIcon={<RefreshCw className={cn('w-4 h-4', loading && 'animate-spin')} />}
            >
              {t('common.refresh')}
            </Button>
          </div>
        </div>

        <div className="bg-white dark:bg-neutral-800 rounded-2xl border border-gray-200 dark:border-neutral-700 overflow-hidden">
          {loading && logs.length === 0 ? (
            <div className="flex items-center justify-center py-12">
              <RefreshCw className="w-6 h-6 animate-spin text-gray-400" />
            </div>
          ) : logs.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12">
              <Clock className="w-12 h-12 text-gray-400 mb-4" />
              <p className="text-gray-500 dark:text-gray-400">{t('activityLog.empty')}</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-50 dark:bg-neutral-900 border-b border-gray-200 dark:border-neutral-700">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider w-[15%]">
                      {t('activityLog.action')}
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider w-[12%]">
                      {t('activityLog.resource')}
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider w-[20%]">
                      {t('activityLog.resourceName')}
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider w-[15%]">
                      {t('activityLog.user')}
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider w-[18%]">
                      {t('activityLog.time')}
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider w-[20%]">
                      {t('activityLog.details')}
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200 dark:divide-neutral-700">
                  {logs.map((log) => (
                    <tr key={log.id} className="hover:bg-gray-50 dark:hover:bg-neutral-700/50">
                      <td className="px-6 py-4">
                        <span className="px-3 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200 rounded-full text-xs font-medium">
                          {getActionLabel(log.action)}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        {log.resourceType ? (
                          <span className="px-3 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded-full text-xs">
                            {getResourceLabel(log.resourceType)}
                          </span>
                        ) : (
                          <span className="text-gray-400">-</span>
                        )}
                      </td>
                      <td className="px-6 py-4">
                        <span className="font-medium text-gray-900 dark:text-gray-100 text-sm">
                          {log.resourceName || '-'}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-2">
                          <User className="w-4 h-4 text-gray-400" />
                          <span className="text-sm text-gray-700 dark:text-gray-300">
                            {log.user.displayName || log.user.email}
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
                          <Clock className="w-4 h-4" />
                          <span>{formatDate(log.createdAt)}</span>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        {log.details ? (
                          <details className="text-xs">
                            <summary className="cursor-pointer text-blue-600 dark:text-blue-400 hover:underline">
                              {t('activityLog.viewDetails')}
                            </summary>
                            <pre className="mt-2 p-2 bg-gray-50 dark:bg-neutral-900 rounded text-gray-600 dark:text-gray-400 whitespace-pre-wrap overflow-x-auto max-w-xs">
                              {log.details}
                            </pre>
                          </details>
                        ) : (
                          <span className="text-gray-400">-</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {totalPages > 1 && (
            <div className="flex items-center justify-between px-6 py-4 border-t border-gray-200 dark:border-neutral-700">
              <div className="text-sm text-gray-500 dark:text-gray-400">
                {t('common.pagination').replace('{current}', String(currentPage)).replace('{total}', String(totalPages)).replace('{count}', String(total))}
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                  disabled={currentPage === 1}
                >
                  {t('common.previousPage')}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                  disabled={currentPage === totalPages}
                >
                  {t('common.nextPage')}
                </Button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default GroupActivityLogs;
