import BaseLayout from '@/components/Layout/BaseLayout'
import AssistantManagerTab from '@/pages/assistants/AssistantManagerTab'
import { useTranslation } from '@/i18n'

const AssistantManager = () => {
  const { t } = useTranslation()
  return (
    <BaseLayout title={t('assistant.pageTitle')} description={t('assistant.pageDesc')}>
      <AssistantManagerTab />
    </BaseLayout>
  )
}

export default AssistantManager
