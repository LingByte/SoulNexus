import GenericEntityPage from '@/components/Management/GenericEntityPage'
import { deleteAdminNotification, listAdminNotifications } from '@/services/adminApi'

const NotificationCenter = () => (
  <GenericEntityPage
    title="通知中心"
    description="管理站内通知消息"
    searchPlaceholder="搜索标题 / 内容"
    exportName="notifications"
    fetchList={listAdminNotifications}
    deleteItem={deleteAdminNotification}
    getId={(item) => item.id}
  />
)

export default NotificationCenter
