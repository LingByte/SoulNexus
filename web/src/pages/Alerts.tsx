import React, { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { useNavigate } from 'react-router-dom';
import { getAlerts, resolveAlert, muteAlert, Alert, AlertStatus, AlertType } from '@/api/alert';
import { showAlert } from '@/utils/notification';
import { useI18nStore } from '@/stores/i18nStore';
import { Bell, AlertTriangle, AlertCircle, Info, X, CheckCircle, VolumeX, Plus, Filter } from 'lucide-react';
import Button from '@/components/UI/Button';

const Alerts: React.FC = () => {
  const { t } = useI18nStore();
  const navigate = useNavigate();
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [statusFilter, setStatusFilter] = useState<AlertStatus | ''>('');
  const [typeFilter, setTypeFilter] = useState<AlertType | ''>('');
  const [loading, setLoading] = useState(false);

  const fetchAlerts = async () => {
    setLoading(true);
    try {
      const params: any = { page, pageSize };
      if (statusFilter) params.status = statusFilter;
      if (typeFilter) params.alertType = typeFilter;
      
      const res = await getAlerts(params);
      setAlerts(res.data.list);
      setTotal(res.data.total);
    } catch (err: any) {
      showAlert(err?.msg || err?.message || t('alerts.messages.fetchFailed'), 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAlerts();
  }, [page, statusFilter, typeFilter]);

  const handleResolve = async (id: number) => {
    try {
      await resolveAlert(id);
      showAlert(t('alerts.messages.resolveSuccess'), 'success');
      fetchAlerts();
    } catch (err: any) {
      showAlert(err?.msg || err?.message || '操作失败', 'error');
    }
  };

  const handleMute = async (id: number) => {
    try {
      await muteAlert(id);
      showAlert(t('alerts.messages.muteSuccess'), 'success');
      fetchAlerts();
    } catch (err: any) {
      showAlert(err?.msg || err?.message || '操作失败', 'error');
    }
  };

  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'critical':
        return <AlertTriangle className="w-5 h-5 text-red-500" />;
      case 'high':
        return <AlertCircle className="w-5 h-5 text-orange-500" />;
      case 'medium':
        return <Info className="w-5 h-5 text-yellow-500" />;
      case 'low':
        return <Info className="w-5 h-5 text-blue-500" />;
      default:
        return <Bell className="w-5 h-5 text-gray-500" />;
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-200';
      case 'high':
        return 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-200';
      case 'medium':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-200';
      case 'low':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-200';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-200';
    }
  };

  const getStatusColor = (status: AlertStatus) => {
    switch (status) {
      case 'active':
        return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-200';
      case 'resolved':
        return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-200';
      case 'muted':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-200';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-200';
    }
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '';
    return new Date(dateStr).toLocaleString();
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-neutral-900 flex flex-col">
      <div className="max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 pt-8 pb-8 flex flex-col">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
          <div className="relative pl-4">
            <motion.div
              layoutId="pageTitleIndicator"
              className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-8 bg-primary rounded-r-full"
              transition={{ type: 'spring', bounce: 0.2, duration: 0.3 }}
            />
            <h1 className="text-2xl sm:text-3xl font-bold text-gray-900 dark:text-gray-100 mb-1">{t('alerts.title')}</h1>
            <p className="text-sm text-gray-500 dark:text-gray-400">{t('alerts.subtitle')}</p>
          </div>
          <Button
            onClick={() => navigate('/alerts/rules')}
            variant="primary"
            size="md"
            leftIcon={<Plus className="w-4 h-4" />}
            className="w-full sm:w-auto"
          >
            {t('alerts.createRule')}
          </Button>
        </div>

        {/* 过滤器 */}
        <div className="bg-white dark:bg-neutral-800 border border-gray-200 dark:border-neutral-700 rounded-lg p-4 mb-6">
          <div className="flex flex-col sm:flex-row sm:items-center gap-3 sm:gap-4">
            <div className="flex items-center gap-2">
              <Filter className="w-4 h-4 text-gray-500 dark:text-gray-400" />
              <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{t('alerts.filter')}</span>
            </div>
            <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-2 flex-1">
              <div className="flex items-center gap-2">
                <label className="text-sm text-gray-600 dark:text-gray-400 whitespace-nowrap">{t('alerts.status')}</label>
                <select
                  value={statusFilter}
                  onChange={(e) => {
                    setStatusFilter(e.target.value as AlertStatus | '');
                    setPage(1);
                  }}
                  className="flex-1 sm:flex-none px-3 py-1.5 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-sm focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                >
                  <option value="">{t('alerts.all')}</option>
                  <option value="active">{t('alerts.status.active')}</option>
                  <option value="resolved">{t('alerts.status.resolved')}</option>
                  <option value="muted">{t('alerts.status.muted')}</option>
                </select>
              </div>
              <div className="flex items-center gap-2">
                <label className="text-sm text-gray-600 dark:text-gray-400 whitespace-nowrap">{t('alerts.type')}</label>
                <select
                  value={typeFilter}
                  onChange={(e) => {
                    setTypeFilter(e.target.value as AlertType | '');
                    setPage(1);
                  }}
                  className="flex-1 sm:flex-none px-3 py-1.5 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-sm focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                >
                  <option value="">{t('alerts.all')}</option>
                  <option value="system_error">{t('alerts.type.systemError')}</option>
                  <option value="quota_exceeded">{t('alerts.type.quotaExceeded')}</option>
                  <option value="service_error">{t('alerts.type.serviceError')}</option>
                  <option value="custom">{t('alerts.type.custom')}</option>
                </select>
              </div>
            </div>
          </div>
        </div>

        {/* 告警列表 */}
        {loading ? (
          <div className="bg-white dark:bg-neutral-800 border border-gray-200 dark:border-neutral-700 rounded-lg p-16 text-center">
            <div className="text-gray-400 dark:text-gray-500">{t('alerts.loading')}</div>
          </div>
        ) : alerts.length === 0 ? (
          <div className="bg-white dark:bg-neutral-800 border border-gray-200 dark:border-neutral-700 rounded-lg p-16 text-center">
            <Bell className="w-12 h-12 mx-auto mb-4 text-gray-400 dark:text-gray-500" />
            <p className="text-gray-500 dark:text-gray-400">{t('alerts.empty')}</p>
          </div>
        ) : (
          <div className="space-y-3">
            {alerts.map(alert => (
              <div
                key={alert.id}
                className="border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 rounded-lg p-5 hover:border-purple-400 dark:hover:border-purple-500 hover:shadow-md transition-all"
              >
                <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex flex-wrap items-center gap-2 sm:gap-3 mb-2">
                      {getSeverityIcon(alert.severity)}
                      <h3 className="font-semibold text-gray-900 dark:text-gray-100 break-words">{alert.title}</h3>
                      <span className={`px-2 py-0.5 rounded text-xs font-medium whitespace-nowrap ${getSeverityColor(alert.severity)}`}>
                        {alert.severity}
                      </span>
                      <span className={`px-2 py-0.5 rounded text-xs font-medium whitespace-nowrap ${getStatusColor(alert.status)}`}>
                        {alert.status === 'active' ? t('alerts.status.active') : alert.status === 'resolved' ? t('alerts.status.resolved') : t('alerts.status.muted')}
                      </span>
                    </div>
                    <p className="text-gray-600 dark:text-gray-400 mb-3 break-words">{alert.message}</p>
                    <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4 text-xs text-gray-500 dark:text-gray-400">
                      <span className="flex items-center gap-1">
                        <span className="font-medium">{t('alerts.type')}：</span>
                        <span>{alert.alertType}</span>
                      </span>
                      <span className="flex items-center gap-1">
                        <span className="font-medium">{t('alerts.time')}</span>
                        <span>{formatDate(alert.createdAt)}</span>
                      </span>
                      {alert.notified && (
                        <span className="flex items-center gap-1 text-green-600 dark:text-green-400">
                          <CheckCircle className="w-3 h-3" />
                          <span>{t('alerts.notified')}</span>
                        </span>
                      )}
                    </div>
                  </div>
                  <div className="flex flex-wrap items-center gap-2 sm:ml-4">
                    {alert.status === 'active' && (
                      <>
                        <Button
                          onClick={() => handleResolve(alert.id)}
                          variant="ghost"
                          size="sm"
                          leftIcon={<CheckCircle className="w-4 h-4" />}
                          className="flex-1 sm:flex-none"
                        >
                          {t('alerts.resolve')}
                        </Button>
                        <Button
                          onClick={() => handleMute(alert.id)}
                          variant="ghost"
                          size="sm"
                          leftIcon={<VolumeX className="w-4 h-4" />}
                          className="flex-1 sm:flex-none"
                        >
                          {t('alerts.mute')}
                        </Button>
                      </>
                    )}
                    <Button
                      onClick={() => navigate(`/alerts/${alert.id}`)}
                      variant="ghost"
                      size="sm"
                      className="flex-1 sm:flex-none"
                    >
                      {t('alerts.detail')}
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* 分页 */}
        {total > pageSize && (
          <div className="flex items-center justify-center gap-2 mt-6">
            <Button
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1}
              variant="ghost"
              size="sm"
            >
              {t('alerts.prevPage')}
            </Button>
            <span className="text-sm text-gray-600 dark:text-gray-400">
              {t('alerts.pageInfo').replace('{page}', String(page)).replace('{total}', String(Math.ceil(total / pageSize)))}
            </span>
            <Button
              onClick={() => setPage(p => Math.min(Math.ceil(total / pageSize), p + 1))}
              disabled={page >= Math.ceil(total / pageSize)}
              variant="ghost"
              size="sm"
            >
              {t('alerts.nextPage')}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
};

export default Alerts;

