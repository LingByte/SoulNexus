import React, { useEffect, useState } from 'react';
import { motion } from 'framer-motion';
import { useNavigate } from 'react-router-dom';
import { getAlertRules, deleteAlertRule, AlertRule, AlertType, AlertSeverity, NotificationChannel } from '@/api/alert';
import { showAlert } from '@/utils/notification';
import { useI18nStore } from '@/stores/i18nStore';
import { Bell, Plus, Edit, Trash2, ToggleLeft, ToggleRight, Settings } from 'lucide-react';
import Button from '@/components/UI/Button';

const AlertRules: React.FC = () => {
  const { t } = useI18nStore();
  const navigate = useNavigate();
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchRules = async () => {
    setLoading(true);
    try {
      const res = await getAlertRules();
      setRules(res.data);
    } catch (err: any) {
      showAlert(err?.msg || err?.message || t('alertRules.messages.fetchFailed'), 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRules();
  }, []);

  const handleDelete = async (id: number) => {
    if (!confirm(t('alertRules.messages.deleteConfirm'))) return;
    
    try {
      await deleteAlertRule(id);
      showAlert(t('alertRules.messages.deleteSuccess'), 'success');
      fetchRules();
    } catch (err: any) {
      showAlert(err?.msg || err?.message || t('alertRules.messages.deleteFailed'), 'error');
    }
  };

  const getTypeLabel = (type: AlertType) => {
    const labels: Record<AlertType, string> = {
      system_error: t('alerts.type.systemError'),
      quota_exceeded: t('alerts.type.quotaExceeded'),
      service_error: t('alerts.type.serviceError'),
      custom: t('alerts.type.custom'),
    };
    return labels[type] || type;
  };

  const getSeverityLabel = (severity: AlertSeverity) => {
    const labels: Record<AlertSeverity, string> = {
      critical: t('alerts.severity.critical'),
      high: t('alerts.severity.high'),
      medium: t('alerts.severity.medium'),
      low: t('alerts.severity.low'),
    };
    return labels[severity] || severity;
  };

  const getSeverityColor = (severity: AlertSeverity) => {
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

  const parseChannels = (channelsStr: string): NotificationChannel[] => {
    try {
      return JSON.parse(channelsStr);
    } catch {
      return [];
    }
  };

  const getChannelLabel = (channel: NotificationChannel) => {
    const labels: Record<NotificationChannel, string> = {
      email: t('alertRules.channel.email'),
      internal: t('alertRules.channel.internal'),
      webhook: t('alertRules.channel.webhook'),
      sms: t('alertRules.channel.sms'),
    };
    return labels[channel] || channel;
  };

  const formatDate = (dateStr: string) => {
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
            <h1 className="text-2xl sm:text-3xl font-bold text-gray-900 dark:text-gray-100 mb-1">{t('alertRules.title')}</h1>
            <p className="text-sm text-gray-500 dark:text-gray-400">{t('alertRules.subtitle')}</p>
          </div>
          <Button
            onClick={() => navigate('/alerts/rules/new')}
            variant="primary"
            size="md"
            leftIcon={<Plus className="w-4 h-4" />}
            className="w-full sm:w-auto"
          >
            {t('alertRules.create')}
          </Button>
        </div>

        {loading ? (
          <div className="bg-white dark:bg-neutral-800 border border-gray-200 dark:border-neutral-700 rounded-lg p-16 text-center">
            <div className="text-gray-400 dark:text-gray-500">{t('alertRules.loading')}</div>
          </div>
        ) : rules.length === 0 ? (
          <div className="bg-white dark:bg-neutral-800 border border-gray-200 dark:border-neutral-700 rounded-lg p-16 text-center">
            <Bell className="w-12 h-12 mx-auto mb-4 text-gray-400 dark:text-gray-500" />
            <p className="text-gray-500 dark:text-gray-400 mb-4">{t('alertRules.empty')}</p>
            <Button
              onClick={() => navigate('/alerts/rules/new')}
              variant="primary"
              size="md"
              leftIcon={<Plus className="w-4 h-4" />}
            >
              {t('alertRules.createFirst')}
            </Button>
          </div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {rules.map(rule => (
              <div
                key={rule.id}
                className="border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 rounded-lg p-6 hover:border-purple-400 dark:hover:border-purple-500 hover:shadow-md transition-all"
              >
                <div className="flex items-start justify-between mb-3">
                  <div className="flex-1">
                    <h3 className="font-semibold text-gray-900 dark:text-gray-100 mb-1">{rule.name}</h3>
                    {rule.description && (
                      <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">{rule.description}</p>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    {rule.enabled ? (
                      <ToggleRight className="w-6 h-6 text-green-500" />
                    ) : (
                      <ToggleLeft className="w-6 h-6 text-gray-400" />
                    )}
                  </div>
                </div>

                <div className="space-y-2 mb-4">
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-gray-500 dark:text-gray-500">{t('alertRules.type')}</span>
                    <span className="text-xs font-medium">{getTypeLabel(rule.alertType)}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-gray-500 dark:text-gray-500">{t('alertRules.severity')}</span>
                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${getSeverityColor(rule.severity)}`}>
                      {getSeverityLabel(rule.severity)}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-gray-500 dark:text-gray-500">{t('alertRules.channels')}</span>
                    <div className="flex items-center gap-1">
                      {parseChannels(rule.channels).map((channel, idx) => (
                        <span key={idx} className="px-2 py-0.5 rounded bg-gray-100 dark:bg-neutral-700 text-xs">
                          {getChannelLabel(channel)}
                        </span>
                      ))}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-gray-500 dark:text-gray-500">{t('alertRules.triggerCount')}</span>
                    <span className="text-xs font-medium">{rule.triggerCount}</span>
                  </div>
                  {rule.lastTriggerAt && (
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-gray-500 dark:text-gray-500">{t('alertRules.lastTrigger')}</span>
                      <span className="text-xs">{formatDate(rule.lastTriggerAt)}</span>
                    </div>
                  )}
                </div>

                <div className="flex items-center justify-between pt-3 border-t border-gray-200 dark:border-neutral-700">
                  <div className="text-xs text-gray-500 dark:text-gray-500">
                    {t('alertRules.createdAt')} {formatDate(rule.createdAt)}
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      onClick={() => navigate(`/alerts/rules/${rule.id}/edit`)}
                      variant="ghost"
                      size="sm"
                      leftIcon={<Edit className="w-4 h-4" />}
                    >
                      {t('alertRules.edit')}
                    </Button>
                    <Button
                      onClick={() => handleDelete(rule.id)}
                      variant="ghost"
                      size="sm"
                      leftIcon={<Trash2 className="w-4 h-4" />}
                    >
                      {t('alertRules.delete')}
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default AlertRules;

