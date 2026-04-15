import GenericEntityPage from '@/components/Management/GenericEntityPage'
import { deleteAdminAlert, listAdminAlerts } from '@/services/adminApi'

const AlertCenter = () => (
  <GenericEntityPage
    title="告警管理"
    description="管理系统告警记录"
    searchPlaceholder="搜索标题 / 消息 / 类型 / 状态"
    exportName="alerts"
    fetchList={listAdminAlerts}
    deleteItem={deleteAdminAlert}
    getId={(item) => item.id}
  />
)

export default AlertCenter
