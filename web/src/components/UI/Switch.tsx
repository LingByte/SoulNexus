import React from 'react'
import { Switch as ArcoSwitch, SwitchProps as ArcoSwitchProps } from '@arco-design/web-react'
import { cn } from '@/utils/cn.ts'

interface SwitchProps extends Omit<ArcoSwitchProps, 'onChange'> {
  checked?: boolean
  onCheckedChange?: (checked: boolean) => void
  size?: 'small' | 'default'
}

const Switch = React.forwardRef<any, SwitchProps>(
  ({
    checked,
    onCheckedChange,
    disabled = false,
    className = '',
    size = 'default',
    ...props
  }, ref) => {
    const PURPLE_PRIMARY = '#6d28d9'

    return (
      <ArcoSwitch
        ref={ref}
        checked={checked}
        onChange={onCheckedChange}
        disabled={disabled}
        size={size}
        className={cn(className)}
        style={{
          '--color-primary': PURPLE_PRIMARY,
        } as React.CSSProperties}
        {...props}
      />
    )
  }
)

Switch.displayName = 'Switch'

export { Switch }
export default Switch
