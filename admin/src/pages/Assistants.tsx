import GenericEntityPage from '@/components/Management/GenericEntityPage'
import { deleteAdminAssistant, listAdminAssistants } from '@/services/adminApi'

const Assistants = () => (
  <GenericEntityPage
    title="智能体管理"
    description="管理智能体配置与基础信息"
    searchPlaceholder="搜索智能体名称 / 描述 / 模型"
    exportName="assistants"
    fetchList={async ({ page, pageSize, search }) => {
      const res = await listAdminAssistants({ page, pageSize, search })
      return {
        items: res.agents || [],
        total: res.total || 0,
        page: res.page || page,
        pageSize: res.pageSize || pageSize,
      }
    }}
    deleteItem={deleteAdminAssistant}
    getId={(item) => item.id}
  />
)

export default Assistants
