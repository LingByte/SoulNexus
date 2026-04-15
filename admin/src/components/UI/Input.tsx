import {
    forwardRef,
    useEffect,
    useId,
    useImperativeHandle,
    useMemo,
    useRef,
    useState,
} from 'react'
import { motion, type HTMLMotionProps } from 'framer-motion'
import { Eye, EyeOff, X, Loader2 } from 'lucide-react'
import { clsx } from 'clsx'


type Size = 'sm' | 'md' | 'lg'

// 以 HTMLMotionProps<'input'> 作为基础，移除我们要自定义的 size
type BaseMotionInputProps = Omit<HTMLMotionProps<'input'>, 'size'>

// 我们的最终 Props：在 Base 上扩展并覆盖 onChange 的签名
interface InputProps extends BaseMotionInputProps {
    label?: string
    error?: string
    helperText?: string
    leftIcon?: React.ReactNode
    rightIcon?: React.ReactNode
    clearable?: boolean
    onClear?: () => void
    size?: Size
    loading?: boolean
    showCount?: boolean
    countMax?: number
    onValueChange?: (value: string) => void
    changeDebounceMs?: number
    wrapperClassName?: string
    inputClassName?: string
    // 关键：覆盖 onChange 的类型，供内部解构与调用
    onChange?: React.ChangeEventHandler<HTMLInputElement>
}

const sizeMap: Record<Size, { px: string; py: string; text: string; icon: string }> = {
    sm: { px: 'px-3',   py: 'py-1.5', text: 'text-sm',  icon: 'w-4 h-4' },
    md: { px: 'px-3.5', py: 'py-2.5', text: 'text-base',icon: 'w-4 h-4' },
    lg: { px: 'px-4',   py: 'py-3',   text: 'text-lg',  icon: 'w-5 h-5' },
}

// 简易节流
function useDebouncedCallback<T extends (...args: any[]) => void>(cb: T | undefined, delay = 0) {
    const timer = useRef<number | null>(null)
    return (...args: Parameters<T>) => {
        if (!cb) return
        if (delay <= 0) return cb(...args)
        if (timer.current) window.clearTimeout(timer.current)
        timer.current = window.setTimeout(() => cb(...args), delay)
    }
}

const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
    {
        className,
        inputClassName,
        wrapperClassName,
        type = 'text',
        label,
        error,
        helperText,
        leftIcon,
        rightIcon,
        clearable = false,
        onClear,
        value,
        defaultValue,
        onChange,
        onValueChange,
        size = 'md',
        loading = false,
        showCount = false,
        countMax,
        maxLength,
        disabled,
        readOnly,
        changeDebounceMs = 0,
        ...restProps // 其余作为 HTMLMotionProps<'input'> 传给 motion.input
    },
    ref
) {
    const generatedId = useId()
    const inputId = restProps.id ?? `input-${generatedId}`
    const errorId = `${inputId}-error`
    const helpId = `${inputId}-help`

    const isControlled = value !== undefined
    const [inner, setInner] = useState<string>(String(defaultValue ?? ''))
    const currentValue = String(isControlled ? value ?? '' : inner)

    const [showPassword, setShowPassword] = useState(false)

    const inputRef = useRef<HTMLInputElement>(null)
    useImperativeHandle(ref, () => inputRef.current as HTMLInputElement)

    // 触发外部回调（可加节流）
    const emitChange = useDebouncedCallback((val: string, e?: React.ChangeEvent<HTMLInputElement>) => {
        onValueChange?.(val)
        onChange?.(e as any)
    }, changeDebounceMs)

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const val = e.target.value
        if (!isControlled) setInner(val)
        emitChange(val, e)
    }

    const handleClear = () => {
        if (onClear) {
            onClear()
            return
        }
        if (!isControlled) setInner('')
        const event = new Event('input', { bubbles: true })
        if (inputRef.current) {
            inputRef.current.value = ''
            inputRef.current.dispatchEvent(event)
        }
        onValueChange?.('')
    }

    const inputType = type === 'password' && showPassword ? 'text' : type
    const hasValue = currentValue.length > 0
    const hasRightAction = loading || clearable || type === 'password' || rightIcon

    const sizeTokens = sizeMap[size]

    const countCurrent = useMemo(() => currentValue.length, [currentValue])
    const countLimit = countMax ?? maxLength

    const describedBy =
        [
            error ? errorId : null,
            helperText ? helpId : null,
            restProps['aria-describedby'] ?? null,
        ]
            .filter(Boolean)
            .join(' ') || undefined

    useEffect(() => {
        return () => {}
    }, [])

    return (
        <div className={clsx('w-full', wrapperClassName)}>
            {label && (
                <motion.label
                    htmlFor={inputId}
                    initial={{ opacity: 0, y: -6 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="mb-1.5 block text-sm font-medium text-foreground"
                >
                    {label}
                    {restProps.required && <span className="ml-0.5 text-destructive">*</span>}
                </motion.label>
            )}

            <div className="relative">
                {leftIcon && (
                    <div className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground">
                        {leftIcon}
                    </div>
                )}

                <motion.input
                    ref={inputRef}
                    id={inputId}
                    type={inputType}
                    value={currentValue}
                    onChange={handleChange}
                    disabled={disabled || loading}
                    readOnly={readOnly}
                    aria-invalid={!!error || undefined}
                    aria-describedby={describedBy}
                    className={clsx(
                        'input', // 若项目里已有 .input 基类，保留；否则可以仅用以下兜底样式
                        'w-full rounded-md border bg-background text-foreground placeholder:text-muted-foreground',
                        'border-input focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                        sizeTokens.px,
                        sizeTokens.py,
                        sizeTokens.text,
                        leftIcon && 'pl-10',
                        hasRightAction && 'pr-10',
                        disabled && 'opacity-60 cursor-not-allowed',
                        readOnly && 'opacity-90',
                        error && 'border-destructive focus-visible:ring-destructive',
                        inputClassName,
                        className
                    )}
                    // 把剩余属性（已是 HTMLMotionProps<'input'>）传下去，避免 MotionProps 冲突
                    {...(restProps as HTMLMotionProps<'input'>)}
                />

                <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-1">
                    {loading && <Loader2 className={clsx('animate-spin text-neutral-400', sizeTokens.icon)} />}

                    {clearable && hasValue && !readOnly && !disabled && (
                        <motion.button
                            type="button"
                            aria-label="清空输入"
                            initial={{ opacity: 0, scale: 0.8 }}
                            animate={{ opacity: 1, scale: 1 }}
                            whileHover={{ scale: 1.08 }}
                            whileTap={{ scale: 0.95 }}
                            onClick={handleClear}
                            className="p-1 text-neutral-400 hover:text-neutral-600 transition-colors"
                        >
                            <X className="w-4 h-4" />
                        </motion.button>
                    )}

                    {type === 'password' && !readOnly && (
                        <motion.button
                            type="button"
                            aria-label={showPassword ? '隐藏密码' : '显示密码'}
                            aria-pressed={showPassword}
                            title={showPassword ? '隐藏密码' : '显示密码'}
                            whileHover={{ scale: 1.08 }}
                            whileTap={{ scale: 0.95 }}
                            onClick={() => setShowPassword(v => !v)}
                            className="p-1 text-neutral-400 hover:text-neutral-600 transition-colors"
                        >
                            {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                        </motion.button>
                    )}

                    {!loading && type !== 'password' && !clearable && rightIcon && (
                        <span className="text-neutral-400">{rightIcon}</span>
                    )}
                </div>
            </div>

            <div className="mt-1.5 flex items-start justify-between gap-3">
                <div className="min-w-0">
                    {error ? (
                        <motion.p
                            id={errorId}
                            initial={{ opacity: 0, y: -4 }}
                            animate={{ opacity: 1, y: 0 }}
                            className="text-sm text-destructive"
                        >
                            {error}
                        </motion.p>
                    ) : helperText ? (
                        <motion.p
                            id={helpId}
                            initial={{ opacity: 0, y: -4 }}
                            animate={{ opacity: 1, y: 0 }}
                            className="text-sm text-muted-foreground"
                        >
                            {helperText}
                        </motion.p>
                    ) : null}
                </div>

                {showCount && (type === 'text' || type === 'search' || type === 'password') && (
                    <div
                        className={clsx(
                            'text-xs tabular-nums',
                            countLimit && countCurrent > (countLimit || 0)
                                ? 'text-destructive'
                                : 'text-muted-foreground'
                        )}
                        title="字符统计"
                    >
                        {countCurrent}
                        {countLimit ? ` / ${countLimit}` : null}
                    </div>
                )}
            </div>
        </div>
    )
})

Input.displayName = 'Input'

export default Input
