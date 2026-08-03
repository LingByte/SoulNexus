import { Navigate, useSearchParams } from 'react-router-dom'
import { isPixelTab } from '@/pages/pixel/PixelToolPage'

/** 旧 /pixel/tools?tab= 链接重定向到独立路由 */
export default function PixelLegacyRedirect() {
  const [searchParams] = useSearchParams()
  const tab = searchParams.get('tab')
  if (tab === 'matte') {
    return <Navigate to="/pixel/process" replace />
  }
  const target = isPixelTab(tab ?? undefined) ? `/pixel/${tab}` : '/pixel/sheet'
  return <Navigate to={target} replace />
}
