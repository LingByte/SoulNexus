import GenericEntityPage from '@/components/Management/GenericEntityPage'
import { deleteAdminKnowledgeBase, listAdminKnowledgeBases } from '@/services/adminApi'

const KnowledgeBases = () => (
  <GenericEntityPage
    title="知识库"
    description="管理知识库连接与索引配置"
    searchPlaceholder="搜索名称 / 描述 / provider / 索引"
    exportName="knowledge_bases"
    fetchList={listAdminKnowledgeBases}
    deleteItem={deleteAdminKnowledgeBase}
    getId={(item) => item.id}
  />
)

export default KnowledgeBases
