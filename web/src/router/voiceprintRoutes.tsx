import { lazy } from 'react'
import { RouteObject } from 'react-router-dom'

// 懒加载声纹识别相关页面
const VoiceprintManagement = lazy(() => import('@/pages/VoiceprintManagement'))

// 声纹识别相关路由配置
export const voiceprintRoutes: RouteObject[] = [
  {
    path: '/voiceprint-management',
    element: <VoiceprintManagement />,
  },
]

export default voiceprintRoutes