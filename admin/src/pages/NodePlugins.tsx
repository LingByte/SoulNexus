import GenericEntityPage from '@/components/Management/GenericEntityPage'
import { deleteAdminNodePlugin, listAdminNodePlugins } from '@/services/adminApi'

const NodePlugins = () => (
  <GenericEntityPage
    title="插件市场(节点)"
    description="管理节点插件与发布状态"
    searchPlaceholder="搜索名称 / slug / 分类 / 状态"
    exportName="node_plugins"
    fetchList={listAdminNodePlugins}
    deleteItem={deleteAdminNodePlugin}
    getId={(item) => item.id}
  />
)

export default NodePlugins
