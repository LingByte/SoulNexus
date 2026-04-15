import React from 'react'
import Input from '../UI/Input'
import { ProviderConfig, ProviderField } from '../../config/providerConfig'
import { Info } from 'lucide-react'

interface ProviderConfigFormProps {
  provider: string
  config: ProviderConfig | null
  values: Record<string, any>
  onChange: (key: string, value: any) => void
  prefix?: string // 用于区分 ASR 和 TTS 字段
}

const ProviderConfigForm: React.FC<ProviderConfigFormProps> = ({
  provider,
  config,
  values,
  onChange,
  prefix = ''
}) => {
  if (!provider) {
    return (
      <div className="text-sm text-gray-500 dark:text-gray-400 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
        请先选择服务提供商
      </div>
    )
  }

  if (!config) {
    return (
      <div className="text-sm text-yellow-600 dark:text-yellow-400 p-4 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg">
        未知的服务提供商: {provider}
      </div>
    )
  }

  const renderField = (field: ProviderField) => {
    const fieldKey = prefix ? `${prefix}_${field.key}` : field.key
    const value = values[fieldKey] || ''

    switch (field.type) {
      case 'select':
        return (
          <div key={field.key} className="space-y-1">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {field.label}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <select
              value={value}
              onChange={(e) => onChange(fieldKey, e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary focus:border-transparent"
              required={field.required}
            >
              <option value="">请选择</option>
              {field.options?.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
            {field.description && (
              <p className="text-xs text-gray-500 dark:text-gray-400 flex items-center gap-1 mt-1">
                <Info className="w-3 h-3" />
                {field.description}
              </p>
            )}
          </div>
        )

      case 'password':
        return (
          <div key={field.key} className="space-y-1">
            <Input
              label={field.label}
              type="password"
              value={value}
              onChange={(e) => onChange(fieldKey, e.target.value)}
              placeholder={field.placeholder}
              required={field.required}
              helperText={field.description}
            />
          </div>
        )

      case 'number':
        return (
          <div key={field.key} className="space-y-1">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {field.label}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <input
              type="number"
              value={value}
              onChange={(e) => onChange(fieldKey, e.target.value)}
              placeholder={field.placeholder}
              required={field.required}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary focus:border-transparent"
            />
            {field.description && (
              <p className="text-xs text-gray-500 dark:text-gray-400 flex items-center gap-1 mt-1">
                <Info className="w-3 h-3" />
                {field.description}
              </p>
            )}
          </div>
        )

      default:
        return (
          <div key={field.key} className="space-y-1">
            <Input
              label={field.label}
              type="text"
              value={value}
              onChange={(e) => onChange(fieldKey, e.target.value)}
              placeholder={field.placeholder}
              required={field.required}
              helperText={field.description}
            />
          </div>
        )
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 mb-4">
        <span className="text-sm font-semibold text-gray-700 dark:text-gray-300">
          {config.name} 配置
        </span>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {config.fields.map((field) => renderField(field))}
      </div>
    </div>
  )
}

export default ProviderConfigForm

