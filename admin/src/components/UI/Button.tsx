import {ButtonHTMLAttributes, forwardRef} from 'react'
import {motion} from 'framer-motion'
import {cn} from '@/utils/cn.ts'
// @ts-ignore
import {getCurrentTheme, getThemeClasses} from '@/utils/themeAdapter.ts'
// @ts-ignore
import {playClickSound, playHoverSound} from '@/utils/audioEffects.ts'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: 'default' | 'primary' | 'secondary' | 'outline' | 'ghost' | 'destructive' | 'success' | 'warning'
    size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'icon'
    loading?: boolean
    leftIcon?: React.ReactNode
    rightIcon?: React.ReactNode
    fullWidth?: boolean
    animation?: 'none' | 'scale' | 'bounce' | 'pulse' | 'slide'
    enableAudio?: boolean
}

const Button = forwardRef<HTMLButtonElement, ButtonProps>(
    ({
         className,
         variant = 'default',
         size = 'md',
         loading = false,
         leftIcon,
         rightIcon,
         fullWidth = false,
         animation = 'scale',
         enableAudio = true,
         children,
         disabled,
         onClick,
         onMouseEnter,
         ...props
     }, ref) => {
        const baseClasses = 'relative inline-flex items-center justify-center font-medium transition-all duration-300 focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none select-none overflow-hidden'

        // 获取当前主题
        const currentTheme = getCurrentTheme()
        const themeClasses = getThemeClasses(currentTheme, 'primary')

        const variantClasses = {
            default: `${themeClasses.bg} ${themeClasses.text} ${themeClasses.hover} focus:ring-${themeClasses.ring} shadow-sm hover:shadow-lg active:shadow-md`,
            primary: `${themeClasses.bg} ${themeClasses.text} ${themeClasses.hover} focus:ring-${themeClasses.ring} shadow-sm hover:shadow-lg active:shadow-md`,
            secondary: 'bg-gray-100 text-gray-900 hover:bg-gray-200 focus:ring-gray-500 shadow-sm hover:shadow-lg active:shadow-md',
            outline: 'border border-gray-300 text-gray-700 hover:bg-gray-50 focus:ring-blue-500 shadow-sm hover:shadow-lg active:shadow-md',
            ghost: 'text-gray-700 hover:bg-gray-100 focus:ring-gray-500 hover:shadow-sm',
            destructive: 'bg-red-600 hover:bg-red-700 focus:ring-red-500 shadow-sm hover:shadow-lg active:shadow-md',
            success: 'bg-green-600 hover:bg-green-700 focus:ring-green-500 shadow-sm hover:shadow-lg active:shadow-md',
            warning: 'bg-yellow-500 hover:bg-yellow-600 focus:ring-yellow-500 shadow-sm hover:shadow-lg active:shadow-md',
        }

        const sizeClasses = {
            xs: 'h-7 px-2 text-xs rounded-md',
            sm: 'h-8 px-3 text-sm rounded-md',
            md: 'h-9 px-4 text-sm rounded-lg',
            lg: 'h-10 px-6 text-base rounded-lg',
            xl: 'h-12 px-8 text-lg rounded-xl',
            icon: 'h-9 w-9 rounded-lg',
        }

        const iconSizeClasses = {
            xs: 'w-3 h-3',
            sm: 'w-3.5 h-3.5',
            md: 'w-4 h-4',
            lg: 'w-5 h-5',
            xl: 'w-6 h-6',
            icon: 'w-4 h-4',
        }

        const animationVariants = {
            none: {},
            scale: {
                hover: {scale: 1.05},
                tap: {scale: 0.95}
            },
            bounce: {
                hover: {
                    scale: 1.05,
                    transition: {type: "spring", stiffness: 400, damping: 10}
                },
                tap: {scale: 0.95}
            },
            pulse: {
                hover: {
                    scale: 1.05,
                    boxShadow: "0 0 0 8px rgba(78, 205, 196, 0.18)"
                },
                tap: {scale: 0.95}
            },
            slide: {
                hover: {
                    x: 2,
                    scale: 1.02
                },
                tap: {x: 0, scale: 0.98}
            }
        }

        const iconSize = iconSizeClasses[size]
        
        // 检测是否使用垂直布局
        const isVertical = className?.includes('flex-col')

        const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
            if (enableAudio && !disabled && !loading) {
                playClickSound()
            }
            onClick?.(e)
        }

        const handleMouseEnter = (e: React.MouseEvent<HTMLButtonElement>) => {
            if (enableAudio && !disabled && !loading) {
                playHoverSound()
            }
            onMouseEnter?.(e)
        }

        return (
            <motion.button
                ref={ref}
                className={cn(
                    isVertical ? 'relative flex' : baseClasses,
                    isVertical ? 'flex-col items-center' : '',
                    variantClasses[variant],
                    sizeClasses[size],
                    fullWidth && 'w-full',
                    'font-medium transition-all duration-300 focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none select-none overflow-hidden',
                    className
                )}
                disabled={disabled || loading}
                variants={animationVariants[animation]}
                whileHover={disabled || loading ? {} : (animationVariants[animation] as any).hover}
                whileTap={disabled || loading ? {} : (animationVariants[animation] as any).tap}
                transition={{duration: 0.2, ease: "easeOut"}}
                onClick={handleClick}
                onMouseEnter={handleMouseEnter}
                {...(props as any)}
            >
                <motion.div
                    className="absolute inset-0 bg-white/20 rounded-inherit"
                    initial={{scale: 0, opacity: 0}}
                    whileTap={{scale: 1, opacity: 1}}
                    transition={{duration: 0.3}}
                />

                {/* 如果有leftIcon或rightIcon props，使用原来的布局；否则直接渲染children（支持flex-col） */}
                {(leftIcon || rightIcon) && !isVertical ? (
                    <div className={cn('relative flex items-center justify-center gap-2 whitespace-nowrap')}>
                        {loading && (
                            <motion.div
                                animate={{rotate: 360}}
                                transition={{duration: 1, repeat: Infinity, ease: 'linear'}}
                                className={cn('border-2 border-current border-t-transparent rounded-full flex-shrink-0', iconSize)}
                            />
                        )}
                        {!loading && leftIcon && (
                            <span className={cn('flex-shrink-0 inline-flex items-center', iconSize)}>
                                {leftIcon}
                            </span>
                        )}
                        {children && (
                            <span className="truncate inline-block">
                                {children}
                            </span>
                        )}
                        {!loading && rightIcon && (
                            <span className={cn('flex-shrink-0 inline-flex items-center', iconSize)}>
                                {rightIcon}
                            </span>
                        )}
                    </div>
                ) : (
                    <div className={cn('relative', isVertical ? 'flex flex-col items-center gap-1' : 'flex items-center gap-2')}>
                        {loading && (
                            <motion.div
                                animate={{rotate: 360}}
                                transition={{duration: 1, repeat: Infinity, ease: 'linear'}}
                                className={cn('border-2 border-current border-t-transparent rounded-full flex-shrink-0', iconSize)}
                            />
                        )}
                        {!loading && leftIcon && (
                            <span className={cn('flex-shrink-0 inline-flex items-center', iconSize)}>
                                {leftIcon}
                            </span>
                        )}
                        {children}
                        {!loading && rightIcon && (
                            <span className={cn('flex-shrink-0 inline-flex items-center', iconSize)}>
                                {rightIcon}
                            </span>
                        )}
                    </div>
                )}
            </motion.button>
        )
    }
)

Button.displayName = 'Button'

export default Button