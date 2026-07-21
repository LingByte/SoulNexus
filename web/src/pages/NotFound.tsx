import { useNavigate } from 'react-router-dom'
import { Empty, Button } from '@/components/ui'
import { useTranslation } from '@/i18n'
import { PLATFORM_HOME_PATH, TENANT_HOME_PATH } from '@/constants/appPaths'
import { useAuthStore } from '@/stores/authStore'

export default function NotFound() {
  const navigate = useNavigate()
  const { t } = useTranslation()
  const user = useAuthStore((s) => s.user)
  const isPlatform = Boolean(user?.isPlatformAdmin || user?.principal === 'platform')
  const homePath = user ? (isPlatform ? PLATFORM_HOME_PATH : TENANT_HOME_PATH) : '/'

  return (
    <div className="flex min-h-screen w-full flex-col items-center justify-center px-4 py-8">
      <Empty preset="404" description={t('notFound.description')}>
        <Button type="primary" onClick={() => navigate(homePath)}>
          {t('notFound.backHome')}
        </Button>
      </Empty>
    </div>
  )
}
