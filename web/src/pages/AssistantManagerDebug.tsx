import { Navigate, useParams } from 'react-router-dom'

/** Legacy route — debug is now an inline panel on the edit page. */
export default function AssistantManagerDebug() {
  const { id = '' } = useParams<{ id: string }>()
  return <Navigate to={`/assistant-manager/${id}/edit?debug=1`} replace />
}
