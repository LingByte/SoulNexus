import { Navigate, useParams } from 'react-router-dom'
import { isValidJSTemplateId } from '@/api/jsTemplates'
import JSTemplateEditorPage from '@/pages/JSTemplateEditorPage'

export default function JSTemplateEdit() {
  const { id } = useParams<{ id: string }>()
  const templateId = String(id || '').trim()

  if (!isValidJSTemplateId(templateId)) {
    return <Navigate to="/js-templates/new" replace />
  }

  return <JSTemplateEditorPage mode="edit" templateId={templateId} />
}
