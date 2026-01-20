import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { getAlert, resolveAlert, muteAlert, Alert, AlertNotification } from '@/api/alert';
import { showAlert } from '@/utils/notification';
import { useI18nStore } from '@/stores/i18nStore';
import { ArrowLeft, CheckCircle, VolumeX, Bell, Mail, Webhook, MessageSquare } from 'lucide-react';
import Button from '@/components/UI/Button';

const AlertDetail: React.FC = () => {
  const { t } = useI18nStore();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [alert, setAlert] = useState<Alert | null>(null);
  const [notifications, setNotifications] = useState<AlertNotification[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (id) {
      fetchAlert();
    }
  }, [id]);

  const fetchAlert = async () => {
    if (!id) return;
    setLoading(true);
    try {
      const res = await getAlert(parseInt(id));
      setAlert(res.data.alert);
      setNotifications(res.data.notifications);
    } catch (err: any) {
      showAlert(err?.msg || err?.message || '获取告警详情失败', 'error');
      navigate('/alerts');
    } finally {
      setLoading(false);
    }
  };

  const handleResolve = async () => {
    if (!id) return;
    try {
      await resolveAlert(parseInt(id));
      showAlert('告警已解决', 'success');
      fetchAlert();
    } catch (err: any) {
      showAlert(err?.msg || err?.message || '操作失败', 'error');
    }
  };

  const handleMute = async () => {
    if (!id) return;
    try {
      await muteAlert(parseInt(id));
      showAlert('告警已静音', 'success');
      fetchAlert();
    } catch (err: any) {
      showAlert(err?.msg || err?.message || '操作失败', 'error');
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

  const getStatusColor = (status: string) => {
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

  const getChannelIcon = (channel: string) => {
    switch (channel) {
      case 'email':
        return <Mail className="w-4 h-4" />;
      case 'internal':
        return <MessageSquare className="w-4 h-4" />;
      case 'webhook':
        return <Webhook className="w-4 h-4" />;
      default:
        return <Bell className="w-4 h-4" />;
    }
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '';
    return new Date(dateStr).toLocaleString('zh-CN');
  };

  if (loading) {
    return (
      <div className="min-h-screen dark:bg-neutral-900 flex items-center justify-center">
        <div className="text-gray-400">加载中...</div>
      </div>
    );
  }

  if (!alert) {
    return null;
  }

  let alertData: any = {};
  if (alert.data) {
    try {
      alertData = JSON.parse(alert.data);
    } catch (e) {
      // 忽略解析错误
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-neutral-900 flex flex-col">
      <div className="max-w-4xl w-full mx-auto px-4 sm:px-6 lg:px-8 pt-8 pb-8 flex flex-col">
        <div className="flex items-center gap-4 mb-6">
          <Button
            onClick={() => navigate('/alerts')}
            variant="ghost"
            size="sm"
            leftIcon={<ArrowLeft className="w-4 h-4" />}
          >
            返回
          </Button>
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">告警详情</h1>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">查看告警的详细信息和通知记录</p>
          </div>
        </div>

        <div className="space-y-6">
          {/* 告警信息 */}
          <div className="border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 rounded-lg p-6 shadow-sm">
            <div className="flex items-start justify-between mb-4">
              <div>
                <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">{alert.title}</h2>
                <div className="flex items-center gap-2 mb-2">
                  <span className={`px-2 py-0.5 rounded text-xs font-medium ${getSeverityColor(alert.severity)}`}>
                    {alert.severity}
                  </span>
                  <span className={`px-2 py-0.5 rounded text-xs font-medium ${getStatusColor(alert.status)}`}>
                    {alert.status === 'active' ? '活跃' : alert.status === 'resolved' ? '已解决' : '已静音'}
                  </span>
                </div>
              </div>
              {alert.status === 'active' && (
                <div className="flex items-center gap-2">
                  <Button
                    onClick={handleResolve}
                    variant="primary"
                    size="sm"
                    leftIcon={<CheckCircle className="w-4 h-4" />}
                  >
                    解决
                  </Button>
                  <Button
                    onClick={handleMute}
                    variant="ghost"
                    size="sm"
                    leftIcon={<VolumeX className="w-4 h-4" />}
                  >
                    静音
                  </Button>
                </div>
              )}
            </div>
            <p className="text-gray-600 dark:text-gray-400 mb-4">{alert.message}</p>
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-gray-500 dark:text-gray-500">类型：</span>
                <span className="ml-2">{alert.alertType}</span>
              </div>
              <div>
                <span className="text-gray-500 dark:text-gray-500">创建时间：</span>
                <span className="ml-2">{formatDate(alert.createdAt)}</span>
              </div>
              {alert.notified && (
                <div>
                  <span className="text-gray-500 dark:text-gray-500">通知时间：</span>
                  <span className="ml-2">{formatDate(alert.notifiedAt)}</span>
                </div>
              )}
              {alert.resolvedAt && (
                <div>
                  <span className="text-gray-500 dark:text-gray-500">解决时间：</span>
                  <span className="ml-2">{formatDate(alert.resolvedAt)}</span>
                </div>
              )}
            </div>
          </div>

          {/* 告警数据 */}
          {Object.keys(alertData).length > 0 && (
            <div className="border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 rounded-lg p-6 shadow-sm">
              <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">告警数据</h3>
              <pre className="bg-gray-50 dark:bg-neutral-900 p-4 rounded-lg overflow-auto text-sm">
                {JSON.stringify(alertData, null, 2)}
              </pre>
            </div>
          )}

          {/* 通知记录 */}
          {notifications.length > 0 && (
            <div className="border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 rounded-lg p-6 shadow-sm">
              <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">通知记录</h3>
              <div className="space-y-3">
                {notifications.map(notification => (
                  <div
                    key={notification.id}
                    className="flex items-center justify-between p-3 bg-gray-50 dark:bg-neutral-900 rounded-lg"
                  >
                    <div className="flex items-center gap-3">
                      {getChannelIcon(notification.channel)}
                      <div>
                        <div className="font-medium">{notification.channel}</div>
                        <div className="text-xs text-gray-500 dark:text-gray-500">
                          {formatDate(notification.sentAt || notification.createdAt)}
                        </div>
                      </div>
                    </div>
                    <span className={`px-2 py-0.5 rounded text-xs ${
                      notification.status === 'success'
                        ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-200'
                        : 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-200'
                    }`}>
                      {notification.status === 'success' ? '成功' : '失败'}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* 规则信息 */}
          {alert.rule && (
            <div className="border border-gray-200 dark:border-neutral-700 bg-white dark:bg-neutral-800 rounded-lg p-6 shadow-sm">
              <h3 className="text-lg font-semibold mb-4 text-gray-900 dark:text-gray-100">关联规则</h3>
              <div className="space-y-2">
                <div>
                  <span className="text-gray-500 dark:text-gray-500">规则名称：</span>
                  <span className="ml-2">{alert.rule.name}</span>
                </div>
                {alert.rule.description && (
                  <div>
                    <span className="text-gray-500 dark:text-gray-500">描述：</span>
                    <span className="ml-2">{alert.rule.description}</span>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default AlertDetail;

