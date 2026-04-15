import GenericEntityPage from '@/components/Management/GenericEntityPage'
import { deleteAdminWorkflow, listAdminWorkflows } from '@/services/adminApi'

const Workflows = () => (
  <GenericEntityPage
    title="工作流"
    description="管理工作流定义"
    searchPlaceholder="搜索名称 / slug / 描述 / 状态"
    exportName="workflows"
    fetchList={listAdminWorkflows}
    deleteItem={deleteAdminWorkflow}
    getId={(item) => item.id}
  />
)

export default Workflows
