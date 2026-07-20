import { Tabs } from '@arco-design/web-react'
import { useTranslation } from '@/i18n'
import WebhooksPanel from '@/components/profile/WebhooksPanel'
import IMChannelsPanel from '@/components/profile/IMChannelsPanel'
import DialogChannelsPanel from '@/components/profile/DialogChannelsPanel'

/**
 * Combined tenant outbound notification settings: HTTP webhooks + WeCom/Feishu IM + dialog inbound channels.
 */
export default function NotificationChannelsPanel() {
  const { t } = useTranslation()
  return (
    <div>
      <div className="mb-4">
        <div className="text-base font-medium">{t('profile.navNotifications')}</div>
        <div className="text-sm text-neutral-500">{t('profile.notificationsDesc')}</div>
      </div>
      <Tabs defaultActiveTab="webhook" type="rounded">
        <Tabs.TabPane key="webhook" title={t('profile.tabWebhook')}>
          <WebhooksPanel embedded />
        </Tabs.TabPane>
        <Tabs.TabPane key="im" title={t('profile.tabIM')}>
          <IMChannelsPanel embedded />
        </Tabs.TabPane>
        <Tabs.TabPane key="dialog" title={t('profile.tabDialogChannels')}>
          <DialogChannelsPanel embedded />
        </Tabs.TabPane>
      </Tabs>
    </div>
  )
}
