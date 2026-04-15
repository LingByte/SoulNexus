import GenericEntityPage from '@/components/Management/GenericEntityPage'
import { deleteAdminMCPServer, listAdminMCPServers } from '@/services/adminApi'

const MCPServers = () => (
  <GenericEntityPage
    title="MCP 管理"
    description="管理 MCP 服务器配置"
    searchPlaceholder="搜索名称 / 描述 / 类型 / 状态"
    exportName="mcp_servers"
    fetchList={listAdminMCPServers}
    deleteItem={deleteAdminMCPServer}
    getId={(item) => item.id}
  />
)

export default MCPServers
