import AssistantManagerFormPage from '@/pages/assistants/AssistantManagerFormPage'

export default function AssistantManagerCreate() {
  const params = new URLSearchParams(window.location.search)
  return (
    <AssistantManagerFormPage
      mode="create"
      scopeTenantId={params.get('tenantId') || undefined}
      templateId={params.get('template') || undefined}
      listPath="/assistant-manager"
    />
  )
}
