import { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { X, ChevronLeft, ChevronRight, Check } from 'lucide-react'
import Button from '../UI/Button'

interface OnboardingStep {
  id: string
  title: string
  description: string
  target?: string
  position?: 'top' | 'bottom' | 'left' | 'right'
  image?: string
  video?: string
  action?: {
    text: string
    onClick: () => void
  }
}

interface OnboardingGuideProps {
  steps: OnboardingStep[]
  isOpen: boolean
  onClose: () => void
  onComplete?: () => void
  className?: string
}

export const OnboardingGuide = ({
  steps,
  isOpen,
  onClose,
  onComplete,
  className = ''
}: OnboardingGuideProps) => {
  const [currentStep, setCurrentStep] = useState(0)
  const [targetElement, setTargetElement] = useState<HTMLElement | null>(null)
  const overlayRef = useRef<HTMLDivElement>(null)

  const currentStepData = steps[currentStep]
  const isLastStep = currentStep === steps.length - 1

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
      }
    }
  }, [isOpen, currentStep, currentStepData?.target])

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

  const getTooltipPosition = () => {
    if (!targetElement) return { top: '50%', left: '50%', transform: 'translate(-50%, -50%)' }

    const rect = targetElement.getBoundingClientRect()
    const position = currentStepData?.position || 'bottom'
    
    switch (position) {
      case 'top':
        return {
          top: `${rect.top - 20}px`,
          left: `${rect.left + rect.width / 2}px`,
          transform: 'translateX(-50%)'
        }
      case 'bottom':
        return {
          top: `${rect.bottom + 20}px`,
          left: `${rect.left + rect.width / 2}px`,
          transform: 'translateX(-50%)'
        }
      case 'left':
        return {
          top: `${rect.top + rect.height / 2}px`,
          left: `${rect.left - 20}px`,
          transform: 'translateY(-50%)'
        }
      case 'right':
        return {
          top: `${rect.top + rect.height / 2}px`,
          left: `${rect.right + 20}px`,
          transform: 'translateY(-50%)'
        }
      default:
        return { top: '50%', left: '50%', transform: 'translate(-50%, -50%)' }
    }
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
        {/* 遮罩层 */}
        <motion.div
          className="absolute inset-0 bg-black/30"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
        />

        {/* 高亮目标元素 */}
        {targetElement && (
          <motion.div
            className="absolute border-2 border-blue-500 rounded-lg pointer-events-none"
            style={{
              top: targetElement.offsetTop - 4,
              left: targetElement.offsetLeft - 4,
              width: targetElement.offsetWidth + 8,
              height: targetElement.offsetHeight + 8,
              boxShadow: '0 0 0 9999px rgba(0, 0, 0, 0.5)'
            }}
            initial={{ scale: 0.8, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            transition={{ duration: 0.3 }}
          />
        )}

        {/* 引导卡片 */}
        <motion.div
          className="absolute bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-md p-6"
          style={getTooltipPosition()}
          initial={{ scale: 0.8, opacity: 0, y: 20 }}
          animate={{ scale: 1, opacity: 1, y: 0 }}
          exit={{ scale: 0.8, opacity: 0, y: 20 }}
          transition={{ duration: 0.3 }}
        >
          {/* 关闭按钮 */}
          <button
            onClick={onClose}
            className="absolute top-4 right-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>

          {/* 步骤指示器 */}
          <div className="flex items-center justify-center mb-4">
            {steps.map((_, index) => (
              <div
                key={index}
                className={`w-2 h-2 rounded-full mx-1 transition-colors ${
                  index === currentStep ? 'bg-blue-500' : 'bg-gray-300 dark:bg-gray-600'
                }`}
              />
            ))}
          </div>

          {/* 内容 */}
          <div className="text-center">
            <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-2">
              {currentStepData?.title}
            </h3>
            <p className="text-gray-600 dark:text-gray-300 mb-6">
              {currentStepData?.description}
            </p>

            {/* 图片或视频 */}
            {currentStepData?.image && (
              <div className="mb-6">
                <img
                  src={currentStepData.image}
                  alt={currentStepData.title}
                  className="w-full h-48 object-cover rounded-lg"
                />
              </div>
            )}

            {currentStepData?.video && (
              <div className="mb-6">
                <video
                  src={currentStepData.video}
                  autoPlay
                  loop
                  muted
                  className="w-full h-48 object-cover rounded-lg"
                />
              </div>
            )}

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

              <div className="flex items-center space-x-2">
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
                  className="flex items-center space-x-2"
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
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}

export default OnboardingGuide
