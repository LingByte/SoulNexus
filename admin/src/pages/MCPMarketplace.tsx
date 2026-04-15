import GenericEntityPage from '@/components/Management/GenericEntityPage'
import { deleteAdminMCPMarketplace, listAdminMCPMarketplace } from '@/services/adminApi'

const MCPMarketplace = () => (
  <GenericEntityPage
    title="MCP 广场"
    description="管理 MCP 广场项目"
    searchPlaceholder="搜索名称 / 作者 / 分类 / 状态"
    exportName="mcp_marketplace"
    fetchList={listAdminMCPMarketplace}
    deleteItem={deleteAdminMCPMarketplace}
    getId={(item) => item.id}
  />
)

export default MCPMarketplace
