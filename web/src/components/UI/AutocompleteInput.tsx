import { useState, useRef, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Check, ChevronDown } from 'lucide-react'
import Input from './Input'

interface AutocompleteOption {
  value: string
  label: string
  description?: string
}

interface AutocompleteInputProps {
  label?: string
  value: string
  onChange: (value: string) => void
  placeholder?: string
  options: AutocompleteOption[]
  leftIcon?: React.ReactNode
  helperText?: string
  error?: string
}

const AutocompleteInput = ({
  label,
  value,
  onChange,
  placeholder = '输入或选择...',
  options,
  leftIcon,
  helperText,
  error,
}: AutocompleteInputProps) => {
  const [isOpen, setIsOpen] = useState(false)
  const [inputValue, setInputValue] = useState(value)
  const wrapperRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // 同步外部value变化
  useEffect(() => {
    setInputValue(value)
  }, [value])

  // 点击外部关闭下拉菜单
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (wrapperRef.current && !wrapperRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [])

  // 过滤选项
  const filteredOptions = options.filter(option =>
    option.label.toLowerCase().includes(inputValue.toLowerCase()) ||
    option.value.toLowerCase().includes(inputValue.toLowerCase()) ||
    option.description?.toLowerCase().includes(inputValue.toLowerCase())
  )

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value
    setInputValue(newValue)
    onChange(newValue)
    setIsOpen(true)
  }

  const handleSelectOption = (option: AutocompleteOption) => {
    setInputValue(option.value)
    onChange(option.value)
    setIsOpen(false)
    if (inputRef.current) {
      inputRef.current.blur()
    }
  }

  const handleFocus = () => {
    if (inputValue && filteredOptions.length > 0) {
      setIsOpen(true)
    }
  }

  return (
    <div ref={wrapperRef} className="relative w-full">
      <Input
        ref={inputRef}
        label={label}
        value={inputValue}
        onChange={handleInputChange}
        onFocus={handleFocus}
        placeholder={placeholder}
        leftIcon={leftIcon}
        rightIcon={
          <ChevronDown 
            className={`w-4 h-4 transition-transform text-gray-400 ${isOpen ? 'rotate-180' : ''}`}
          />
        }
        helperText={helperText}
        error={error}
      />

      {/* 下拉建议列表 */}
      <AnimatePresence>
        {isOpen && inputValue && filteredOptions.length > 0 && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            transition={{ duration: 0.2 }}
            className="absolute z-50 w-full top-full mt-1 bg-white dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded-lg shadow-lg max-h-60 overflow-auto"
          >
            {filteredOptions.map((option) => {
              const isSelected = option.value === inputValue
              return (
                <motion.div
                  key={option.value}
                  onClick={() => handleSelectOption(option)}
                  className={`
                    px-4 py-3 cursor-pointer transition-colors
                    ${isSelected 
                      ? 'bg-primary/10 dark:bg-primary/20' 
                      : 'hover:bg-gray-100 dark:hover:bg-gray-700'
                    }
                  `}
                  whileHover={{ backgroundColor: isSelected ? undefined : 'rgba(0, 0, 0, 0.05)' }}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center space-x-2">
                        <span className="font-medium text-gray-900 dark:text-white">
                          {option.label}
                        </span>
                        {isSelected && (
                          <Check className="w-4 h-4 text-primary" />
                        )}
                      </div>
                      {option.description && (
                        <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                          {option.description}
                        </p>
                      )}
                    </div>
                  </div>
                </motion.div>
              )
            })}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

export default AutocompleteInput

