import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { createAlertRule, updateAlertRule, getAlertRule, AlertType, AlertSeverity, NotificationChannel, AlertCondition } from '@/api/alert';
import { showAlert } from '@/utils/notification';
import { ArrowLeft, Save } from 'lucide-react';
import Button from '@/components/UI/Button';
import { useI18nStore } from '@/stores/i18nStore';

const AlertRuleForm: React.FC = () => {
  const navigate = useNavigate();
  const { t } = useI18nStore();
  const { id } = useParams<{ id: string }>();
  const isEdit = !!id;

  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    alertType: 'quota_exceeded' as AlertType,
    severity: 'medium' as AlertSeverity,
    conditions: {
      quotaType: '',
      quotaThreshold: 80,
      errorCount: 10,
      errorWindow: 300,
      serviceName: '',
      failureRate: 20,
      responseTime: 3000,
    } as AlertCondition,
    channels: ['internal'] as NotificationChannel[],
    webhookUrl: '',
    webhookMethod: 'POST',
    cooldown: 300,
    enabled: true,
  });

  useEffect(() => {
    if (isEdit && id) {
      fetchRule();
    }
  }, [isEdit, id]);

  const fetchRule = async () => {
    if (!id) return;
    try {
      const res = await getAlertRule(parseInt(id));
      const rule = res.data;
      setFormData({
        name: rule.name,
        description: rule.description || '',
        alertType: rule.alertType,
        severity: rule.severity,
        conditions: JSON.parse(rule.conditions || '{}'),
        channels: JSON.parse(rule.channels || '["internal"]'),
        webhookUrl: rule.webhookUrl || '',
        webhookMethod: rule.webhookMethod || 'POST',
        cooldown: rule.cooldown,
        enabled: rule.enabled,
      });
    } catch (err: any) {
      showAlert(err?.msg || err?.message || t('alertRuleForm.fetchError'), 'error');
      navigate('/alerts/rules');
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      if (isEdit && id) {
        await updateAlertRule(parseInt(id), formData);
        showAlert(t('alertRuleForm.updateSuccess'), 'success');
      } else {
        await createAlertRule(formData);
        showAlert(t('alertRuleForm.createSuccess'), 'success');
      }
      navigate('/alerts/rules');
    } catch (err: any) {
      showAlert(err?.msg || err?.message || t('alertRuleForm.operationFailed'), 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleChannelToggle = (channel: NotificationChannel) => {
    setFormData(prev => {
      const channels = prev.channels.includes(channel)
        ? prev.channels.filter(c => c !== channel)
        : [...prev.channels, channel];
      return { ...prev, channels };
    });
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-neutral-900 flex flex-col">
      <div className="max-w-4xl w-full mx-auto px-4 sm:px-6 lg:px-8 pt-8 pb-8 flex flex-col">
        <div className="flex items-center gap-4 mb-6">
          <Button
            onClick={() => navigate('/alerts/rules')}
            variant="ghost"
            size="sm"
            leftIcon={<ArrowLeft className="w-4 h-4" />}
          >
            {t('alertRuleForm.back')}
          </Button>
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
              {isEdit ? t('alertRuleForm.titleEdit') : t('alertRuleForm.titleCreate')}
            </h1>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
              {isEdit ? t('alertRuleForm.subtitleEdit') : t('alertRuleForm.subtitleCreate')}
            </p>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* 基本信息 */}
          <div className="border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 rounded-lg p-6 shadow-sm">
            <h2 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">{t('alertRuleForm.basicInfo')}</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">{t('alertRuleForm.nameRequired')}</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">{t('alertRuleForm.description')}</label>
                <textarea
                  value={formData.description}
                  onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800"
                  rows={3}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-2">{t('alertRuleForm.alertTypeRequired')}</label>
                  <select
                    value={formData.alertType}
                    onChange={(e) => setFormData(prev => ({ ...prev, alertType: e.target.value as AlertType }))}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800"
                    required
                  >
                    <option value="system_error">{t('alertRuleForm.type.systemError')}</option>
                    <option value="quota_exceeded">{t('alertRuleForm.type.quotaExceeded')}</option>
                    <option value="service_error">{t('alertRuleForm.type.serviceError')}</option>
                    <option value="custom">{t('alertRuleForm.type.custom')}</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-2">{t('alertRuleForm.severityRequired')}</label>
                  <select
                    value={formData.severity}
                    onChange={(e) => setFormData(prev => ({ ...prev, severity: e.target.value as AlertSeverity }))}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800"
                    required
                  >
                    <option value="critical">{t('alertRuleForm.severity.critical')}</option>
                    <option value="high">{t('alertRuleForm.severity.high')}</option>
                    <option value="medium">{t('alertRuleForm.severity.medium')}</option>
                    <option value="low">{t('alertRuleForm.severity.low')}</option>
                  </select>
                </div>
              </div>
            </div>
          </div>

          {/* 触发条件 */}
          <div className="border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 rounded-lg p-6 shadow-sm">
            <h2 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">{t('alertRuleForm.triggerConditions')}</h2>
            <div className="space-y-4">
              {formData.alertType === 'quota_exceeded' && (
                <>
                  <div>
                    <label className="block text-sm font-medium mb-2">{t('alertRuleForm.quotaTypeRequired')}</label>
                    <select
                      value={formData.conditions.quotaType || ''}
                      onChange={(e) => setFormData(prev => ({
                        ...prev,
                        conditions: { ...prev.conditions, quotaType: e.target.value }
                      }))}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                      required
                    >
                      <option value="">{t('alertRuleForm.quotaTypePlaceholder')}</option>
                      <option value="storage">{t('alertRuleForm.quotaType.storage')}</option>
                      <option value="llm_tokens">{t('alertRuleForm.quotaType.llmTokens')}</option>
                      <option value="llm_calls">{t('alertRuleForm.quotaType.llmCalls')}</option>
                      <option value="api_calls">{t('alertRuleForm.quotaType.apiCalls')}</option>
                      <option value="call_duration">{t('alertRuleForm.quotaType.callDuration')}</option>
                      <option value="call_count">{t('alertRuleForm.quotaType.callCount')}</option>
                      <option value="asr_duration">{t('alertRuleForm.quotaType.asrDuration')}</option>
                      <option value="asr_count">{t('alertRuleForm.quotaType.asrCount')}</option>
                      <option value="tts_duration">{t('alertRuleForm.quotaType.ttsDuration')}</option>
                      <option value="tts_count">{t('alertRuleForm.quotaType.ttsCount')}</option>
                    </select>
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      {t('alertRuleForm.quotaTypeHint')}
                    </p>
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-2">{t('alertRuleForm.quotaThresholdRequired')}</label>
                    <input
                      type="number"
                      min="0"
                      max="100"
                      step="0.1"
                      value={formData.conditions.quotaThreshold || 80}
                      onChange={(e) => setFormData(prev => ({
                        ...prev,
                        conditions: { ...prev.conditions, quotaThreshold: parseFloat(e.target.value) || 0 }
                      }))}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                      required
                    />
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      {t('alertRuleForm.quotaThresholdHint')}
                    </p>
                  </div>
                </>
              )}
              {formData.alertType === 'system_error' && (
                <>
                  <div>
                    <label className="block text-sm font-medium mb-2">{t('alertRuleForm.errorCount')}</label>
                    <input
                      type="number"
                      min="1"
                      value={formData.conditions.errorCount || 10}
                      onChange={(e) => setFormData(prev => ({
                        ...prev,
                        conditions: { ...prev.conditions, errorCount: parseInt(e.target.value) }
                      }))}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-2">{t('alertRuleForm.errorWindow')}</label>
                    <input
                      type="number"
                      min="1"
                      value={formData.conditions.errorWindow || 300}
                      onChange={(e) => setFormData(prev => ({
                        ...prev,
                        conditions: { ...prev.conditions, errorWindow: parseInt(e.target.value) }
                      }))}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800"
                    />
                  </div>
                </>
              )}
              {formData.alertType === 'service_error' && (
                <>
                  <div>
                    <label className="block text-sm font-medium mb-2">{t('alertRuleForm.serviceName')}</label>
                    <input
                      type="text"
                      value={formData.conditions.serviceName || ''}
                      onChange={(e) => setFormData(prev => ({
                        ...prev,
                        conditions: { ...prev.conditions, serviceName: e.target.value }
                      }))}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800"
                    />
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-medium mb-2">{t('alertRuleForm.failureRate')}</label>
                      <input
                        type="number"
                        min="0"
                        max="100"
                        value={formData.conditions.failureRate || 20}
                        onChange={(e) => setFormData(prev => ({
                          ...prev,
                          conditions: { ...prev.conditions, failureRate: parseFloat(e.target.value) }
                        }))}
                        className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium mb-2">{t('alertRuleForm.responseTime')}</label>
                      <input
                        type="number"
                        min="0"
                        value={formData.conditions.responseTime || 3000}
                        onChange={(e) => setFormData(prev => ({
                          ...prev,
                          conditions: { ...prev.conditions, responseTime: parseInt(e.target.value) }
                        }))}
                        className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800"
                      />
                    </div>
                  </div>
                </>
              )}
            </div>
          </div>

          {/* 通知配置 */}
          <div className="border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 rounded-lg p-6 shadow-sm">
            <h2 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">{t('alertRuleForm.notificationConfig')}</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">{t('alertRuleForm.channelsRequired')}</label>
                <div className="flex flex-wrap gap-2">
                  {(['email', 'internal', 'webhook', 'sms'] as NotificationChannel[]).map(channel => (
                    <label key={channel} className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={formData.channels.includes(channel)}
                        onChange={() => handleChannelToggle(channel)}
                        className="rounded"
                      />
                      <span className="text-sm">
                        {t(`alertRuleForm.channel.${channel}`)}
                      </span>
                    </label>
                  ))}
                </div>
              </div>
              {formData.channels.includes('webhook') && (
                <>
                  <div>
                    <label className="block text-sm font-medium mb-2">{t('alertRuleForm.webhookUrlRequired')}</label>
                    <input
                      type="url"
                      value={formData.webhookUrl}
                      onChange={(e) => setFormData(prev => ({ ...prev, webhookUrl: e.target.value }))}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800"
                      required={formData.channels.includes('webhook')}
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-2">{t('alertRuleForm.webhookMethod')}</label>
                    <select
                      value={formData.webhookMethod}
                      onChange={(e) => setFormData(prev => ({ ...prev, webhookMethod: e.target.value }))}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800"
                    >
                      <option value="POST">POST</option>
                      <option value="PUT">PUT</option>
                      <option value="PATCH">PATCH</option>
                    </select>
                  </div>
                </>
              )}
              <div>
                <label className="block text-sm font-medium mb-2">{t('alertRuleForm.cooldown')}</label>
                <input
                  type="number"
                  min="0"
                  value={formData.cooldown}
                  onChange={(e) => setFormData(prev => ({ ...prev, cooldown: parseInt(e.target.value) || 300 }))}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800"
                />
                <p className="text-xs text-gray-500 mt-1">{t('alertRuleForm.cooldownHint')}</p>
              </div>
              <div>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.enabled}
                    onChange={(e) => setFormData(prev => ({ ...prev, enabled: e.target.checked }))}
                    className="rounded"
                  />
                  <span className="text-sm">{t('alertRuleForm.enabled')}</span>
                </label>
              </div>
            </div>
          </div>

          <div className="flex items-center justify-end gap-3">
            <Button
              type="button"
              onClick={() => navigate('/alerts/rules')}
              variant="ghost"
            >
              {t('alertRuleForm.cancel')}
            </Button>
            <Button
              type="submit"
              variant="primary"
              leftIcon={<Save className="w-4 h-4" />}
              disabled={loading}
            >
              {loading ? t('alertRuleForm.saving') : t('alertRuleForm.save')}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default AlertRuleForm;

