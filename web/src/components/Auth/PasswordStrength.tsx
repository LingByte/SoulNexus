import React from 'react'

interface PasswordStrengthProps {
  password: string
  minLength?: number
  requireUppercase?: boolean
  requireLowercase?: boolean
  requireNumber?: boolean
  requireSpecialChar?: boolean
}

interface StrengthRule {
  label: string
  test: (pwd: string) => boolean
  met: boolean
}

const PasswordStrength: React.FC<PasswordStrengthProps> = ({
  password,
  minLength = 8,
  requireUppercase = false,
  requireLowercase = true,
  requireNumber = false,
  requireSpecialChar = false,
}) => {
  const rules: StrengthRule[] = [
    {
      label: `至少 ${minLength} 个字符`,
      test: (pwd) => pwd.length >= minLength,
      met: password.length >= minLength,
    },
    {
      label: '包含小写字母',
      test: (pwd) => /[a-z]/.test(pwd),
      met: /[a-z]/.test(password),
      required: requireLowercase,
    },
    {
      label: '包含大写字母',
      test: (pwd) => /[A-Z]/.test(pwd),
      met: /[A-Z]/.test(password),
      required: requireUppercase,
    },
    {
      label: '包含数字',
      test: (pwd) => /[0-9]/.test(pwd),
      met: /[0-9]/.test(password),
      required: requireNumber,
    },
    {
      label: '包含特殊字符',
      test: (pwd) => /[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(pwd),
      met: /[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(password),
      required: requireSpecialChar,
    },
  ].filter((rule) => rule.required !== false)

  const metCount = rules.filter((r) => r.met).length
  const totalCount = rules.length
  const strength = metCount === totalCount ? 'strong' : metCount >= totalCount * 0.6 ? 'medium' : 'weak'

  if (!password) return null

  return (
    <div className="mt-2 space-y-2">
      <div className="flex items-center gap-2">
        <div className="flex-1 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
          <div
            className={`h-full transition-all duration-300 ${
              strength === 'strong'
                ? 'bg-green-500 w-full'
                : strength === 'medium'
                  ? 'bg-yellow-500 w-2/3'
                  : 'bg-red-500 w-1/3'
            }`}
          />
        </div>
        <span
          className={`text-xs font-medium ${
            strength === 'strong'
              ? 'text-green-600 dark:text-green-400'
              : strength === 'medium'
                ? 'text-yellow-600 dark:text-yellow-400'
                : 'text-red-600 dark:text-red-400'
          }`}
        >
          {strength === 'strong' ? '强' : strength === 'medium' ? '中' : '弱'}
        </span>
      </div>
      <div className="space-y-1">
        {rules.map((rule, index) => (
          <div key={index} className="flex items-center gap-2 text-xs">
            <div
              className={`w-4 h-4 rounded-full flex items-center justify-center ${
                rule.met
                  ? 'bg-green-500 text-white'
                  : 'bg-gray-300 dark:bg-gray-600 text-gray-500 dark:text-gray-400'
              }`}
            >
              {rule.met ? '✓' : '○'}
            </div>
            <span
              className={
                rule.met
                  ? 'text-green-600 dark:text-green-400'
                  : 'text-gray-500 dark:text-gray-400'
              }
            >
              {rule.label}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

export default PasswordStrength

