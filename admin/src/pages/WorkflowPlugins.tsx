import GenericEntityPage from '@/components/Management/GenericEntityPage'
import { deleteAdminWorkflowPlugin, listAdminWorkflowPlugins } from '@/services/adminApi'

const WorkflowPlugins = () => (
  <GenericEntityPage
    title="插件市场"
    description="管理工作流插件市场"
    searchPlaceholder="搜索名称 / slug / 分类 / 状态"
    exportName="workflow_plugins"
    fetchList={listAdminWorkflowPlugins}
    deleteItem={deleteAdminWorkflowPlugin}
    getId={(item) => item.id}
  />
)

export default WorkflowPlugins
