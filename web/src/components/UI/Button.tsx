import React, { forwardRef } from 'react'
import { Button as ArcoButton } from '@arco-design/web-react'
import type { ButtonProps as ArcoButtonProps } from '@arco-design/web-react'
import { cn } from '@/utils/utils.ts'

export type ButtonVariant =
  | 'primary'
  | 'secondary'
  | 'outline'
  | 'ghost'
  | 'destructive'
  | 'success'
  | 'warning'
  | 'link'

/** Primary size tokens — new code should prefer these. */
export type ButtonSize = 'xs' | 'sm' | 'md' | 'lg' | 'icon'
/** Arco-style size tokens — accepted for backward compatibility. */
export type ArcoButtonSize = 'mini' | 'small' | 'default' | 'large'
export type AnyButtonSize = ButtonSize | ArcoButtonSize

/** Arco-style button type alias (backward compatible). */
export type ArcoButtonType = 'primary' | 'secondary' | 'dashed' | 'outline' | 'text' | 'default'

export interface ButtonProps extends Omit<ArcoButtonProps, 'size' | 'type' | 'htmlType' | 'icon' | 'status'> {
  variant?: ButtonVariant
  /** Arco alias for variant — e.g. type="primary" | "outline" | "text" */
  type?: ArcoButtonType
  size?: AnyButtonSize
  block?: boolean
  /** Arco alias for block */
  long?: boolean
  rounded?: 'none' | 'sm' | 'md' | 'lg' | 'full'
  leftIcon?: React.ReactNode
  rightIcon?: React.ReactNode
  /** Arco alias for leftIcon */
  icon?: React.ReactNode
  /** Arco status — maps to semantic variants */
  status?: 'danger' | 'warning' | 'success' | 'default'
  children?: React.ReactNode
  className?: string
  htmlType?: 'button' | 'submit' | 'reset'
}

const variantClassMap: Record<ButtonVariant, string> = {
  primary:
    'ui-btn-primary bg-[hsl(var(--primary))] text-white hover:bg-[hsl(var(--primary)/0.9)] active:bg-[hsl(var(--primary)/0.8)] shadow-[0_1px_2px_rgba(0,0,0,0.05)] hover:shadow-[0_4px_12px_hsl(var(--primary)/0.25)]',
  secondary:
    'ui-btn-secondary bg-[hsl(var(--secondary))] text-[hsl(var(--secondary-foreground))] hover:bg-[hsl(var(--secondary)/0.8)] border border-[hsl(var(--border))]',
  outline:
    'ui-btn-outline bg-transparent border border-[hsl(var(--border))] text-[hsl(var(--foreground))] hover:bg-[hsl(var(--accent)/0.4)] hover:border-[hsl(var(--primary)/0.4)]',
  ghost:
    'ui-btn-ghost bg-transparent text-[hsl(var(--foreground))] hover:bg-[hsl(var(--accent)/0.5)]',
  destructive:
    'ui-btn-destructive bg-[#ef4444] text-white hover:bg-[#dc2626] shadow-[0_1px_2px_rgba(239,68,68,0.15)] hover:shadow-[0_4px_12px_rgba(239,68,68,0.3)]',
  success:
    'ui-btn-success bg-[#10b981] text-white hover:bg-[#059669] shadow-[0_1px_2px_rgba(16,185,129,0.15)] hover:shadow-[0_4px_12px_rgba(16,185,129,0.3)]',
  warning:
    'ui-btn-warning bg-[#f59e0b] text-white hover:bg-[#d97706] shadow-[0_1px_2px_rgba(245,158,11,0.15)] hover:shadow-[0_4px_12px_rgba(245,158,11,0.3)]',
  link: 'ui-btn-link bg-transparent text-[hsl(var(--primary))] hover:underline px-0 shadow-none',
}

const sizeClassMap: Record<ButtonSize, string> = {
  xs: 'h-7 px-2.5 text-xs gap-1.5',
  sm: 'h-8 px-3 text-xs gap-1.5',
  md: 'h-10 px-4 text-sm gap-2',
  lg: 'h-12 px-6 text-base gap-2.5',
  icon: 'h-10 w-10 p-0',
}

const roundedClassMap: Record<Exclude<ButtonProps['rounded'], undefined>, string> = {
  none: 'rounded-none',
  sm: 'rounded-sm',
  md: 'rounded-md',
  lg: 'rounded-lg',
  full: 'rounded-full',
}

const variantToArcoType: Record<ButtonVariant, ArcoButtonProps['type']> = {
  primary: 'text',
  secondary: 'text',
  outline: 'text',
  ghost: 'text',
  destructive: 'text',
  success: 'text',
  warning: 'text',
  link: 'text',
}

const sizeToArcoSize: Record<ButtonSize, ArcoButtonProps['size']> = {
  xs: 'mini',
  sm: 'small',
  md: 'default',
  lg: 'large',
  icon: 'default',
}

/** Normalize Arco-style size tokens (e.g. "small" | "mini" | "default" | "large") to internal ButtonSize. */
function normalizeSize(size: AnyButtonSize): ButtonSize {
  switch (size) {
    case 'mini':
      return 'xs'
    case 'small':
      return 'sm'
    case 'default':
      return 'md'
    case 'large':
      return 'lg'
    default:
      return size as ButtonSize
  }
}

function resolveVariant(props: ButtonProps): ButtonVariant {
  const { variant, type, status } = props
  if (variant) return variant
  if (status === 'danger') return 'destructive'
  if (status === 'warning') return 'warning'
  if (status === 'success') return 'success'
  switch (type) {
    case 'primary':
      return 'primary'
    case 'secondary':
    case 'default':
      return 'secondary'
    case 'outline':
    case 'dashed':
      return 'outline'
    case 'text':
      return 'ghost'
    default:
      return 'primary'
  }
}

export const Button = forwardRef<HTMLElement, ButtonProps>(function Button(
  {
    variant: variantProp,
    type,
    size = 'md',
    block = false,
    long,
    rounded = 'md',
    leftIcon,
    rightIcon,
    icon,
    status,
    children,
    className,
    disabled,
    loading,
    htmlType = 'button',
    ...rest
  },
  ref,
) {
  const variant = resolveVariant({ variant: variantProp, type, status })
  const normalizedSize = normalizeSize(size ?? 'md')
  const isBlock = block || long
  const resolvedLeftIcon = leftIcon ?? icon
  const isLink = variant === 'link'
  const iconSize = normalizedSize === 'lg' ? 18 : normalizedSize === 'xs' || normalizedSize === 'sm' ? 14 : 16

  const baseClasses = cn(
    'ui-btn inline-flex flex-row items-center justify-center font-medium tracking-tight select-none',
    'transition-all duration-200 ease-out',
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--primary)/0.4)] focus-visible:ring-offset-2 focus-visible:ring-offset-[hsl(var(--background))]',
    'disabled:opacity-50 disabled:cursor-not-allowed disabled:shadow-none',
    'active:scale-[0.98]',
    variantClassMap[variant],
    sizeClassMap[normalizedSize],
    rounded !== undefined && roundedClassMap[rounded],
    isBlock && 'w-full',
    isLink && 'h-auto py-1',
    className,
  )

  const renderIcon = (node: React.ReactNode) => {
    if (!node) return null
    return (
      <span className="ui-btn-icon inline-flex items-center justify-center" style={{ fontSize: iconSize }}>
        {node}
      </span>
    )
  }

  return (
    <ArcoButton
      ref={ref}
      type={variantToArcoType[variant]}
      size={sizeToArcoSize[normalizedSize]}
      htmlType={htmlType}
      disabled={disabled}
      loading={loading}
      long={isBlock}
      className={baseClasses}
      {...rest}
    >
      {loading ? null : renderIcon(resolvedLeftIcon)}
      {children && <span className="ui-btn-label">{children}</span>}
      {loading ? null : renderIcon(rightIcon)}
    </ArcoButton>
  )
})

Button.displayName = 'Button'

export default Button
