import { useParams } from 'react-router-dom'
import AssistantManagerFormPage from '@/pages/assistants/AssistantManagerFormPage'

export default function AssistantManagerEdit() {
  const { id } = useParams<{ id: string }>()
  return (
    <AssistantManagerFormPage
      mode="edit"
      assistantId={id}
      listPath="/assistant-manager"
    />
  )
}
