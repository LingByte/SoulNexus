import LoadingAnimation from '@/components/LoadingAnimation'

interface ContentLoadingSpinnerProps {
  message?: string
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

export const ContentLoadingSpinner = ({
  message,
  size = 'md',
  className = ''
}: ContentLoadingSpinnerProps) => {
  return (
    <div className={`flex flex-col items-center justify-center py-12 ${className}`}>
      <LoadingAnimation type="spinner" size={size} />
      {message && <p className="mt-4 text-sm text-gray-500 dark:text-gray-400">{message}</p>}
    </div>
  )
}

export default ContentLoadingSpinner
