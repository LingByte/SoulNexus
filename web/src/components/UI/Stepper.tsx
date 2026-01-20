import { ReactNode } from 'react'
import { Check } from 'lucide-react'
import { cn } from '@/utils/cn.ts'

interface Step {
  title: string
  description?: string
  content?: ReactNode
  completed?: boolean
  disabled?: boolean
}

interface StepperProps {
  steps: Step[]
  currentStep: number
  onStepClick?: (stepIndex: number) => void
  orientation?: 'horizontal' | 'vertical'
  className?: string
  showContent?: boolean
}

const Stepper = ({
  steps,
  currentStep,
  onStepClick,
  orientation = 'horizontal',
  className,
  showContent = false
}: StepperProps) => {
  const isHorizontal = orientation === 'horizontal'

  return (
    <div className={cn(
      'w-full',
      isHorizontal ? 'flex flex-col' : 'flex',
      className
    )}>
      {/* 步骤指示器 */}
      <div className={cn(
        'flex',
        isHorizontal ? 'flex-row justify-between' : 'flex-col'
      )}>
        {steps.map((step, index) => {
          const isActive = index === currentStep
          const isCompleted = step.completed || index < currentStep
          const isClickable = onStepClick && !step.disabled

          return (
            <div
              key={index}
              className={cn(
                'flex items-center',
                isHorizontal ? 'flex-col' : 'flex-row',
                isHorizontal && index < steps.length - 1 && 'flex-1'
              )}
            >
              {/* 步骤圆圈 */}
              <div
                className={cn(
                  'flex items-center justify-center w-8 h-8 rounded-full border-2 transition-all',
                  isCompleted && 'bg-primary border-primary text-primary-foreground',
                  isActive && !isCompleted && 'border-primary text-primary',
                  !isActive && !isCompleted && 'border-neutral-300 dark:border-neutral-600 text-neutral-400',
                  step.disabled && 'opacity-50 cursor-not-allowed',
                  isClickable && !step.disabled && 'cursor-pointer hover:scale-110'
                )}
                onClick={() => isClickable && onStepClick?.(index)}
              >
                {isCompleted ? (
                  <Check className="w-4 h-4" />
                ) : (
                  <span className="text-sm font-medium">{index + 1}</span>
                )}
              </div>

              {/* 步骤信息 */}
              <div className={cn(
                'flex flex-col',
                isHorizontal ? 'items-center mt-2' : 'ml-3',
                isHorizontal && 'text-center'
              )}>
                <div className={cn(
                  'text-sm font-medium',
                  isActive && 'text-primary',
                  isCompleted && 'text-primary',
                  !isActive && !isCompleted && 'text-neutral-500 dark:text-neutral-400'
                )}>
                  {step.title}
                </div>
                {step.description && (
                  <div className="text-xs text-neutral-500 dark:text-neutral-400 mt-1">
                    {step.description}
                  </div>
                )}
              </div>

              {/* 连接线 */}
              {index < steps.length - 1 && (
                <div className={cn(
                  'flex-1',
                  isHorizontal ? 'h-0.5 bg-neutral-200 dark:bg-neutral-700 mx-4 mt-4' : 'w-0.5 h-8 bg-neutral-200 dark:bg-neutral-700 ml-4',
                  isCompleted && 'bg-primary'
                )} />
              )}
            </div>
          )
        })}
      </div>

      {/* 步骤内容 */}
      {showContent && steps[currentStep]?.content && (
        <div className="mt-6 p-4 bg-neutral-50 dark:bg-neutral-800 rounded-lg">
          {steps[currentStep].content}
        </div>
      )}
    </div>
  )
}

export default Stepper
