import LoadingAnimation from '@/components/LoadingAnimation'

interface PageLoadingScreenProps {
  message?: string
  fullscreen?: boolean
}

export const PageLoadingScreen = ({
  message = '加载中…',
  fullscreen = true
}: PageLoadingScreenProps) => {
  const containerClass = fullscreen
    ? 'min-h-screen bg-gray-50 dark:bg-gray-900'
    : 'h-full bg-gray-50 dark:bg-gray-900'

  return (
    <div className={`${containerClass} flex flex-col items-center justify-center px-4`}>
      <LoadingAnimation type="spinner" size="lg" className="mb-4" />
      <p className="text-sm text-gray-500 dark:text-gray-400 text-center">{message}</p>
    </div>
  )
}

export default PageLoadingScreen
