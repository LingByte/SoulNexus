import {forwardRef} from 'react'
import {motion} from 'framer-motion'
import {Button as ArcoButton, ButtonProps as ArcoButtonProps} from '@arco-design/web-react'
import {cn} from '@/utils/cn.ts'
// @ts-ignore
import {playClickSound, playHoverSound} from '@/utils/audioEffects.ts'

interface ButtonProps extends Omit<ArcoButtonProps, 'type' | 'size' | 'loading'> {
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
        // 深紫色主题 RGB: (109, 40, 217)
        const PURPLE_PRIMARY = '#6d28d9'
        const isDark = typeof document !== 'undefined' && document.documentElement.classList.contains('dark')

        // 映射variant到Arco Design的type和自定义样式
        const variantConfig: Record<string, { type: any, customStyle?: React.CSSProperties }> = {
            default: { 
                type: 'primary', 
                customStyle: { 
                    backgroundColor: PURPLE_PRIMARY,
                    borderColor: PURPLE_PRIMARY,
                    color: '#ffffff'
                } 
            },
            primary: { 
                type: 'primary', 
                customStyle: { 
                    backgroundColor: PURPLE_PRIMARY,
                    borderColor: PURPLE_PRIMARY,
                    color: '#ffffff'
                } 
            },
            secondary: { type: 'secondary' },
            outline: { type: 'outline' },
            ghost: { type: 'text' },
            destructive: { 
                type: 'primary', 
                customStyle: { 
                    backgroundColor: '#dc2626',
                    borderColor: '#dc2626',
                    color: '#ffffff'
                } 
            },
            success: { 
                type: 'primary', 
                customStyle: { 
                    backgroundColor: '#16a34a',
                    borderColor: '#16a34a',
                    color: '#ffffff'
                } 
            },
            warning: { 
                type: 'primary', 
                customStyle: { 
                    backgroundColor: '#ea580c',
                    borderColor: '#ea580c',
                    color: '#ffffff'
                } 
            },
        }

        // 映射size到Arco Design的size
        const sizeToArcoSize = {
            xs: 'small',
            sm: 'small',
            md: 'middle',
            lg: 'large',
            xl: 'large',
            icon: 'large',
        }

        const sizeClasses = {
            xs: 'h-7 px-2 text-xs rounded-md min-w-fit',
            sm: 'h-8 px-3 text-sm rounded-md min-w-fit',
            md: 'h-9 px-4 text-sm rounded-lg min-w-fit',
            lg: 'h-10 px-6 text-base rounded-lg min-w-fit',
            xl: 'h-12 px-8 text-lg rounded-xl min-w-fit',
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
                    boxShadow: "0 0 0 8px rgba(59, 130, 246, 0.1)"
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
        const config = variantConfig[variant]
        const arcoSize = sizeToArcoSize[size]

        const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
            if (enableAudio && !disabled && !loading) {
                playClickSound()
            }
            onClick?.(e as any)
        }

        const handleMouseEnter = (e: React.MouseEvent<HTMLButtonElement>) => {
            if (enableAudio && !disabled && !loading) {
                playHoverSound()
            }
            onMouseEnter?.(e as any)
        }

        return (
            <motion.div
                variants={animationVariants[animation]}
                whileHover={disabled || loading ? {} : (animationVariants[animation] as any).hover}
                whileTap={disabled || loading ? {} : (animationVariants[animation] as any).tap}
                transition={{duration: 0.2, ease: "easeOut"}}
                className={cn(fullWidth && 'w-full')}
            >
                <ArcoButton
                    ref={ref}
                    type={config.type}
                    size={arcoSize as any}
                    loading={loading}
                    disabled={disabled || loading}
                    style={config.customStyle || {}}
                    className={cn(
                        sizeClasses[size],
                        fullWidth && 'w-full',
                        className
                    )}
                    onClick={handleClick}
                    onMouseEnter={handleMouseEnter}
                    {...(props as any)}
                >
                    <motion.div
                        className="absolute inset-0 rounded-inherit"
                        initial={{scale: 0, opacity: 0}}
                        whileTap={{scale: 1, opacity: 1}}
                        transition={{duration: 0.3}}
                        style={{backgroundColor: 'rgba(255, 255, 255, 0.1)'}}
                    />

                    <div
                        className={cn(
                            'relative z-[1] flex min-w-0 flex-row flex-nowrap items-center justify-center gap-2',
                            fullWidth && 'w-full min-w-0'
                        )}
                    >
                        {!loading && leftIcon && (
                            <motion.span
                                className={cn('inline-flex flex-shrink-0 items-center justify-center', iconSize)}
                                whileHover={{scale: 1.1}}
                                transition={{duration: 0.2}}
                            >
                                {leftIcon}
                            </motion.span>
                        )}
                        {children != null && children !== false && (
                            <motion.span
                                className="inline-flex min-w-0 max-w-full flex-row flex-nowrap items-center justify-center gap-1.5 whitespace-nowrap [&_svg]:shrink-0"
                                initial={{opacity: 0, y: 10}}
                                animate={{opacity: 1, y: 0}}
                                transition={{duration: 0.3, delay: 0.1}}
                            >
                                {children}
                            </motion.span>
                        )}
                        {!loading && rightIcon && (
                            <motion.span
                                className={cn('inline-flex flex-shrink-0 items-center justify-center', iconSize)}
                                whileHover={{scale: 1.1, x: 2}}
                                transition={{duration: 0.2}}
                            >
                                {rightIcon}
                            </motion.span>
                        )}
                    </div>
                </ArcoButton>
            </motion.div>
        )
    }
)

Button.displayName = 'Button'

export default Button