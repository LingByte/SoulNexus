import GenericEntityPage from '@/components/Management/GenericEntityPage'
import { deleteAdminVoiceTrainingTask, listAdminVoiceTrainingTasks } from '@/services/adminApi'

const VoiceTraining = () => (
  <GenericEntityPage
    title="音色训练"
    description="管理音色训练任务与结果"
    searchPlaceholder="搜索任务名 / task_id / asset_id"
    exportName="voice_training"
    fetchList={listAdminVoiceTrainingTasks}
    deleteItem={deleteAdminVoiceTrainingTask}
    getId={(item) => item.id}
  />
)

export default VoiceTraining
