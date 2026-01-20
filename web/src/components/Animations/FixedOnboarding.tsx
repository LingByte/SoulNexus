import { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { X, ChevronLeft, ChevronRight, Check, Play, Pause } from 'lucide-react'
import Button from '../UI/Button'

interface OnboardingStep {
  id: string
  title: string
  description: string
  target?: string
  position?: 'top' | 'bottom' | 'left' | 'right' | 'center'
  image?: string
  video?: string
  action?: {
    text: string
    onClick: () => void
  }
}

interface FixedOnboardingProps {
  steps: OnboardingStep[]
  isOpen: boolean
  onClose: () => void
  onComplete?: () => void
  className?: string
  autoPlay?: boolean
  showProgress?: boolean
  allowSkip?: boolean
}

export const FixedOnboarding = ({
  steps,
  isOpen,
  onClose,
  onComplete,
  className = '',
  autoPlay = false,
  showProgress = true,
  allowSkip = true
}: FixedOnboardingProps) => {
  const [currentStep, setCurrentStep] = useState(0)
  const [targetElement, setTargetElement] = useState<HTMLElement | null>(null)
  const [isPlaying, setIsPlaying] = useState(autoPlay)
  const [highlightStyle, setHighlightStyle] = useState<React.CSSProperties>({})
  const overlayRef = useRef<HTMLDivElement>(null)
  const intervalRef = useRef<NodeJS.Timeout>()

  const currentStepData = steps[currentStep]
  const isLastStep = currentStep === steps.length - 1
  const progress = ((currentStep + 1) / steps.length) * 100

  useEffect(() => {
    if (isOpen && currentStepData?.target) {
      const element = document.querySelector(currentStepData.target) as HTMLElement
      setTargetElement(element)
      
      if (element) {
        // 滚动到目标元素
        element.scrollIntoView({
          behavior: 'smooth',
          block: 'center',
          inline: 'center'
        })
        
        // 等待滚动完成后更新高亮
        setTimeout(() => {
          const updateHighlight = () => {
            const rect = element.getBoundingClientRect()
            setHighlightStyle({
              position: 'fixed',
              top: rect.top - 8,
              left: rect.left - 8,
              width: rect.width + 16,
              height: rect.height + 16,
              zIndex: 40,
              pointerEvents: 'none',
              boxShadow: '0 0 0 9999px rgba(0, 0, 0, 0.7), 0 0 30px rgba(59, 130, 246, 0.8)',
              border: '4px solid #3b82f6',
              borderRadius: '16px',
              transition: 'all 0.3s ease'
            })
          }
          
          updateHighlight()
          
          // 监听滚动和窗口大小变化
          const handleUpdate = () => {
            if (element && document.contains(element)) {
              updateHighlight()
            }
          }
          
          window.addEventListener('scroll', handleUpdate, true)
          window.addEventListener('resize', handleUpdate)
          
          return () => {
            window.removeEventListener('scroll', handleUpdate, true)
            window.removeEventListener('resize', handleUpdate)
          }
        }, 500) // 等待滚动动画完成
      }
    }
  }, [isOpen, currentStep, currentStepData?.target])

  useEffect(() => {
    if (isPlaying && !isLastStep) {
      intervalRef.current = setInterval(() => {
        setCurrentStep(prev => prev + 1)
      }, 5000)
    } else {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
      }
    }

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
      }
    }
  }, [isPlaying, isLastStep])

  const nextStep = () => {
    if (isLastStep) {
      onComplete?.()
      onClose()
    } else {
      setCurrentStep(prev => prev + 1)
    }
  }

  const prevStep = () => {
    if (currentStep > 0) {
      setCurrentStep(prev => prev - 1)
    }
  }

  const skipTutorial = () => {
    onComplete?.()
    onClose()
  }

  const togglePlayPause = () => {
    setIsPlaying(!isPlaying)
  }

  if (!isOpen) return null

  return (
    <AnimatePresence>
      <motion.div
        ref={overlayRef}
        className={`fixed inset-0 z-50 ${className}`}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
      >
        {/* 背景遮罩 */}
        <motion.div
          className="absolute inset-0 bg-black/40"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
        />

        {/* 高亮目标元素 - 使用固定定位 */}
        {targetElement && (
          <div
            style={highlightStyle}
            className="animate-pulse"
          />
        )}

        {/* 引导卡片 - 固定在屏幕中央 */}
        <motion.div
          className="fixed top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-2xl shadow-2xl max-w-lg p-8 z-50"
          initial={{ scale: 0.8, opacity: 0, y: 20 }}
          animate={{ scale: 1, opacity: 1, y: 0 }}
          exit={{ scale: 0.8, opacity: 0, y: 20 }}
          transition={{ duration: 0.3 }}
        >
          {/* 顶部控制栏 */}
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center space-x-4">
              {/* 播放控制 */}
              <button
                onClick={togglePlayPause}
                className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors rounded-full hover:bg-gray-100 dark:hover:bg-gray-700"
              >
                {isPlaying ? <Pause className="w-5 h-5" /> : <Play className="w-5 h-5" />}
              </button>

              {/* 步骤指示器 */}
              <div className="flex items-center space-x-2">
                {steps.map((_, index) => (
                  <div
                    key={index}
                    className={`w-2 h-2 rounded-full transition-all duration-300 ${
                      index === currentStep 
                        ? 'bg-blue-500 w-8' 
                        : index < currentStep 
                          ? 'bg-green-500' 
                          : 'bg-gray-300 dark:bg-gray-600'
                    }`}
                  />
                ))}
              </div>
            </div>

            <div className="flex items-center space-x-2">
              {allowSkip && (
                <button
                  onClick={skipTutorial}
                  className="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
                >
                  跳过
                </button>
              )}
              <button
                onClick={onClose}
                className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors rounded-full hover:bg-gray-100 dark:hover:bg-gray-700"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
          </div>

          {/* 进度条 */}
          {showProgress && (
            <div className="mb-6">
              <div className="flex justify-between text-sm text-gray-500 mb-2">
                <span>步骤 {currentStep + 1} / {steps.length}</span>
                <span>{Math.round(progress)}%</span>
              </div>
              <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                <motion.div
                  className="bg-gradient-to-r from-blue-500 to-purple-500 h-2 rounded-full"
                  initial={{ width: 0 }}
                  animate={{ width: `${progress}%` }}
                  transition={{ duration: 0.5 }}
                />
              </div>
            </div>
          )}

          {/* 内容区域 */}
          <div className="text-center mb-8">
            <motion.h3 
              className="text-2xl font-bold text-gray-900 dark:text-white mb-4"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.2 }}
            >
              {currentStepData?.title}
            </motion.h3>
            
            <motion.p 
              className="text-gray-600 dark:text-gray-300 mb-6 leading-relaxed"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.3 }}
            >
              {currentStepData?.description}
            </motion.p>

            {/* 图片或视频 */}
            {currentStepData?.image && (
              <motion.div 
                className="mb-6"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: 0.4 }}
              >
                <img
                  src={currentStepData.image}
                  alt={currentStepData.title}
                  className="w-full h-48 object-cover rounded-xl shadow-lg"
                />
              </motion.div>
            )}

            {currentStepData?.video && (
              <motion.div 
                className="mb-6"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: 0.4 }}
              >
                <video
                  src={currentStepData.video}
                  autoPlay
                  loop
                  muted
                  className="w-full h-48 object-cover rounded-xl shadow-lg"
                />
              </motion.div>
            )}
          </div>

          {/* 操作按钮 */}
          <div className="flex items-center justify-between">
            <Button
              variant="outline"
              onClick={prevStep}
              disabled={currentStep === 0}
              className="flex items-center space-x-2"
            >
              <ChevronLeft className="w-4 h-4" />
              <span>上一步</span>
            </Button>

            <div className="flex items-center space-x-3">
              {currentStepData?.action && (
                <Button
                  variant="outline"
                  onClick={currentStepData.action.onClick}
                  className="flex items-center space-x-2"
                >
                  <span>{currentStepData.action.text}</span>
                </Button>
              )}

              <Button
                onClick={nextStep}
                className="flex items-center space-x-2 bg-gradient-to-r from-blue-500 to-purple-500 hover:from-blue-600 hover:to-purple-600"
              >
                <span>{isLastStep ? '完成' : '下一步'}</span>
                {isLastStep ? (
                  <Check className="w-4 h-4" />
                ) : (
                  <ChevronRight className="w-4 h-4" />
                )}
              </Button>
            </div>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}

export default FixedOnboarding
