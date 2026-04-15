import { useState, useRef, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { cn } from '@/utils/cn.ts'

interface FormField {
  name: string
  label: string
  type: 'text' | 'email' | 'password' | 'textarea' | 'select' | 'checkbox' | 'radio' | 'date' | 'file'
  required?: boolean
  placeholder?: string
  options?: { value: string; label: string }[]
  validation?: (value: any) => string | null
  className?: string
}

interface AdvancedFormProps {
  fields: FormField[]
  onSubmit: (data: Record<string, any>) => void
  submitText?: string
  className?: string
  animated?: boolean
  showProgress?: boolean
  autoSave?: boolean
  autoSaveDelay?: number
}

const AdvancedForm = ({
  fields,
  onSubmit,
  submitText = '提交',
  className = '',
  animated = true,
  showProgress = true,
  autoSave = false,
  autoSaveDelay = 2000
}: AdvancedFormProps) => {
  const [formData, setFormData] = useState<Record<string, any>>({})
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [touched, setTouched] = useState<Record<string, boolean>>({})
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [currentStep, setCurrentStep] = useState(0)
  // @ts-ignore
  const autoSaveTimeoutRef = useRef<NodeJS.Timeout>()

  const steps = Math.ceil(fields.length / 3) // 每步3个字段
  const currentFields = fields.slice(currentStep * 3, (currentStep + 1) * 3)

  const validateField = (name: string, value: any): string | null => {
    const field = fields.find(f => f.name === name)
    if (!field) return null

    if (field.required && (!value || value === '')) {
      return `${field.label}是必填项`
    }

    if (field.validation) {
      return field.validation(value)
    }

    return null
  }

  const handleChange = (name: string, value: any) => {
    setFormData(prev => ({ ...prev, [name]: value }))
    
    // 清除错误
    if (errors[name]) {
      setErrors(prev => ({ ...prev, [name]: '' }))
    }

    // 自动保存
    if (autoSave) {
      if (autoSaveTimeoutRef.current) {
        clearTimeout(autoSaveTimeoutRef.current)
      }
      autoSaveTimeoutRef.current = setTimeout(() => {
        localStorage.setItem('form_autosave', JSON.stringify(formData))
      }, autoSaveDelay)
    }
  }

  const handleBlur = (name: string) => {
    setTouched(prev => ({ ...prev, [name]: true }))
    const error = validateField(name, formData[name])
    if (error) {
      setErrors(prev => ({ ...prev, [name]: error }))
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    // 验证所有字段
    const newErrors: Record<string, string> = {}
    fields.forEach(field => {
      const error = validateField(field.name, formData[field.name])
      if (error) {
        newErrors[field.name] = error
      }
    })

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors)
      return
    }

    setIsSubmitting(true)
    try {
      await onSubmit(formData)
      // 清除自动保存
      if (autoSave) {
        localStorage.removeItem('form_autosave')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  const nextStep = () => {
    if (currentStep < steps - 1) {
      setCurrentStep(prev => prev + 1)
    }
  }

  const prevStep = () => {
    if (currentStep > 0) {
      setCurrentStep(prev => prev - 1)
    }
  }

  // 加载自动保存的数据
  useEffect(() => {
    if (autoSave) {
      const saved = localStorage.getItem('form_autosave')
      if (saved) {
        try {
          setFormData(JSON.parse(saved))
        } catch (e) {
          console.warn('Failed to load autosaved form data')
        }
      }
    }
  }, [autoSave])

  const renderField = (field: FormField) => {
    const hasError = touched[field.name] && errors[field.name]
    const fieldId = `field-${field.name}`

    return (
      <motion.div
        key={field.name}
        className={cn('space-y-2', field.className)}
        initial={animated ? { opacity: 0, y: 20 } : {}}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
      >
        <label
          htmlFor={fieldId}
          className="block text-sm font-medium text-gray-700"
        >
          {field.label}
          {field.required && <span className="text-red-500 ml-1">*</span>}
        </label>

        {field.type === 'textarea' ? (
          <textarea
            id={fieldId}
            value={formData[field.name] || ''}
            onChange={(e) => handleChange(field.name, e.target.value)}
            onBlur={() => handleBlur(field.name)}
            placeholder={field.placeholder}
            className={cn(
              'w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all duration-200',
              hasError ? 'border-red-500' : 'border-gray-300'
            )}
            rows={4}
          />
        ) : field.type === 'select' ? (
          <select
            id={fieldId}
            value={formData[field.name] || ''}
            onChange={(e) => handleChange(field.name, e.target.value)}
            onBlur={() => handleBlur(field.name)}
            className={cn(
              'w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all duration-200',
              hasError ? 'border-red-500' : 'border-gray-300'
            )}
          >
            <option value="">请选择...</option>
            {field.options?.map(option => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        ) : field.type === 'checkbox' ? (
          <div className="flex items-center space-x-2">
            <input
              id={fieldId}
              type="checkbox"
              checked={formData[field.name] || false}
              onChange={(e) => handleChange(field.name, e.target.checked)}
              onBlur={() => handleBlur(field.name)}
              className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
            <label htmlFor={fieldId} className="text-sm text-gray-700">
              {field.placeholder}
            </label>
          </div>
        ) : field.type === 'radio' ? (
          <div className="space-y-2">
            {field.options?.map(option => (
              <div key={option.value} className="flex items-center space-x-2">
                <input
                  id={`${fieldId}-${option.value}`}
                  type="radio"
                  name={field.name}
                  value={option.value}
                  checked={formData[field.name] === option.value}
                  onChange={(e) => handleChange(field.name, e.target.value)}
                  onBlur={() => handleBlur(field.name)}
                  className="w-4 h-4 text-blue-600 border-gray-300 focus:ring-blue-500"
                />
                <label htmlFor={`${fieldId}-${option.value}`} className="text-sm text-gray-700">
                  {option.label}
                </label>
              </div>
            ))}
          </div>
        ) : field.type === 'file' ? (
          <div className="relative">
            <input
              id={fieldId}
              type="file"
              onChange={(e) => handleChange(field.name, e.target.files?.[0])}
              onBlur={() => handleBlur(field.name)}
              className="w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all duration-200 file:mr-4 file:py-2 file:px-4 file:rounded-full file:border-0 file:text-sm file:font-semibold file:bg-blue-50 file:text-blue-700 hover:file:bg-blue-100"
            />
          </div>
        ) : (
          <input
            id={fieldId}
            type={field.type}
            value={formData[field.name] || ''}
            onChange={(e) => handleChange(field.name, e.target.value)}
            onBlur={() => handleBlur(field.name)}
            placeholder={field.placeholder}
            className={cn(
              'w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all duration-200',
              hasError ? 'border-red-500' : 'border-gray-300'
            )}
          />
        )}

        {hasError && (
          <motion.p
            className="text-sm text-red-600"
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
          >
            {errors[field.name]}
          </motion.p>
        )}
      </motion.div>
    )
  }

  return (
    <motion.form
      onSubmit={handleSubmit}
      className={cn('space-y-6', className)}
      initial={animated ? { opacity: 0, y: 20 } : {}}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
    >
      {/* 进度条 */}
      {showProgress && steps > 1 && (
        <div className="mb-6">
          <div className="flex justify-between text-sm text-gray-600 mb-2">
            <span>步骤 {currentStep + 1} / {steps}</span>
            <span>{Math.round(((currentStep + 1) / steps) * 100)}%</span>
          </div>
          <div className="w-full bg-gray-200 rounded-full h-2">
            <motion.div
              className="bg-blue-600 h-2 rounded-full"
              initial={{ width: 0 }}
              animate={{ width: `${((currentStep + 1) / steps) * 100}%` }}
              transition={{ duration: 0.3 }}
            />
          </div>
        </div>
      )}

      {/* 字段 */}
      <AnimatePresence mode="wait">
        <motion.div
          key={currentStep}
          className="space-y-4"
          initial={{ opacity: 0, x: 20 }}
          animate={{ opacity: 1, x: 0 }}
          exit={{ opacity: 0, x: -20 }}
          transition={{ duration: 0.3 }}
        >
          {currentFields.map(renderField)}
        </motion.div>
      </AnimatePresence>

      {/* 按钮 */}
      <div className="flex justify-between">
        <div>
          {currentStep > 0 && (
            <button
              type="button"
              onClick={prevStep}
              className="px-4 py-2 text-gray-600 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              上一步
            </button>
          )}
        </div>

        <div className="flex space-x-3">
          {currentStep < steps - 1 ? (
            <button
              type="button"
              onClick={nextStep}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              下一步
            </button>
          ) : (
            <motion.button
              type="submit"
              disabled={isSubmitting}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
            >
              {isSubmitting ? (
                <div className="flex items-center space-x-2">
                  <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                  <span>提交中...</span>
                </div>
              ) : (
                submitText
              )}
            </motion.button>
          )}
        </div>
      </div>
    </motion.form>
  )
}

export default AdvancedForm
