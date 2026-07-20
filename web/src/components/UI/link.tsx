import React, { forwardRef, AnchorHTMLAttributes } from 'react'
import { Link as ArcoLink } from '@arco-design/web-react'
import type { LinkProps as ArcoLinkProps } from '@arco-design/web-react'
import { Link as RouterLink } from 'react-router-dom'
import type { LinkProps as RouterLinkProps } from 'react-router-dom'
import { cn } from '@/utils/utils.ts'
import { ChevronRight, ExternalLink as ExternalLinkIcon } from 'lucide-react'

export type LinkVariant = 'default' | 'primary' | 'muted' | 'nav' | 'underline'
export type LinkSize = 'xs' | 'sm' | 'md' | 'lg'

interface BaseLinkProps {
  variant?: LinkVariant
  size?: LinkSize
  leftIcon?: React.ReactNode
  rightIcon?: React.ReactNode
  showExternalIcon?: boolean
  className?: string
  children?: React.ReactNode
}

export interface LinkProps extends BaseLinkProps, Omit<RouterLinkProps, 'className'> {}

export interface ExternalLinkProps
  extends BaseLinkProps,
    Omit<AnchorHTMLAttributes<HTMLAnchorElement>, 'className'> {}

const variantClassMap: Record<LinkVariant, string> = {
  default:
    'text-[hsl(var(--foreground))] hover:text-[hsl(var(--primary))] hover:underline underline-offset-4',
  primary:
    'text-[hsl(var(--primary))] hover:text-[hsl(var(--primary)/0.8)] hover:underline underline-offset-4 font-medium',
  muted:
    'text-[hsl(var(--muted-foreground))] hover:text-[hsl(var(--foreground))] transition-colors',
  nav:
    'text-[hsl(var(--muted-foreground))] hover:text-[hsl(var(--foreground))] transition-colors font-medium',
  underline:
    'text-[hsl(var(--foreground))] underline decoration-[hsl(var(--border))] underline-offset-4 hover:text-[hsl(var(--primary))] hover:decoration-[hsl(var(--primary))] transition-colors',
}

const sizeClassMap: Record<LinkSize, string> = {
  xs: 'text-xs',
  sm: 'text-sm',
  md: 'text-sm',
  lg: 'text-base',
}

const iconSizeBySize: Record<LinkSize, number> = {
  xs: 12,
  sm: 14,
  md: 16,
  lg: 18,
}

function linkContent(
  size: LinkSize,
  leftIcon: React.ReactNode | undefined,
  rightIcon: React.ReactNode | undefined,
  showExternalIcon: boolean,
  children: React.ReactNode,
) {
  const iconSize = iconSizeBySize[size]
  return (
    <>
      {leftIcon && (
        <span className="shrink-0 inline-flex" style={{ width: iconSize, height: iconSize }}>
          {leftIcon}
        </span>
      )}
      {children}
      {rightIcon && (
        <span className="shrink-0 inline-flex" style={{ width: iconSize, height: iconSize }}>
          {rightIcon}
        </span>
      )}
      {showExternalIcon && (
        <span className="shrink-0 inline-flex opacity-70" style={{ width: iconSize, height: iconSize }}>
          <ExternalLinkIcon size={iconSize} />
        </span>
      )}
    </>
  )
}

export const Link = forwardRef<HTMLAnchorElement, LinkProps>(function Link(
  {
    variant = 'default',
    size = 'md',
    leftIcon,
    rightIcon,
    showExternalIcon = false,
    className,
    children,
    ...rest
  },
  ref,
) {
  const classes = cn(
    'inline-flex items-center gap-1.5 transition-colors',
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--primary)/0.4)] focus-visible:ring-offset-2 focus-visible:ring-offset-[hsl(var(--background))] rounded-sm',
    variantClassMap[variant],
    sizeClassMap[size],
    className,
  )

  return (
    <RouterLink ref={ref} className={classes} {...rest}>
      {linkContent(size, leftIcon, rightIcon, showExternalIcon, children)}
    </RouterLink>
  )
})

Link.displayName = 'Link'

/** External URL — Arco Link styling with security defaults. */
export const ExternalLink = forwardRef<HTMLAnchorElement, ExternalLinkProps>(function ExternalLink(
  {
    variant = 'default',
    size = 'md',
    leftIcon,
    rightIcon,
    showExternalIcon = true,
    className,
    children,
    target = '_blank',
    rel = 'noopener noreferrer',
    href,
    ...rest
  },
  ref,
) {
  const classes = cn(
    'inline-flex items-center gap-1.5 transition-colors',
    variantClassMap[variant],
    sizeClassMap[size],
    className,
  )

  const arcoStatus: ArcoLinkProps['status'] =
    variant === 'primary' ? undefined : variant === 'muted' ? undefined : undefined

  return (
    <ArcoLink
      ref={ref}
      href={href}
      target={target}
      rel={rel}
      status={arcoStatus}
      className={classes}
      icon={showExternalIcon && !rightIcon ? <ExternalLinkIcon size={iconSizeBySize[size]} /> : undefined}
      {...rest}
    >
      {leftIcon && (
        <span className="shrink-0 inline-flex mr-1" style={{ width: iconSizeBySize[size], height: iconSizeBySize[size] }}>
          {leftIcon}
        </span>
      )}
      {children}
      {rightIcon && (
        <span className="shrink-0 inline-flex ml-1" style={{ width: iconSizeBySize[size], height: iconSizeBySize[size] }}>
          {rightIcon}
        </span>
      )}
    </ArcoLink>
  )
})

ExternalLink.displayName = 'ExternalLink'

export const ArrowLink = forwardRef<HTMLAnchorElement, LinkProps>(function ArrowLink(
  { rightIcon, children, variant = 'primary', size = 'md', ...rest },
  ref,
) {
  return (
    <Link
      ref={ref}
      variant={variant}
      size={size}
      rightIcon={rightIcon ?? <ChevronRight size={iconSizeBySize[size]} />}
      {...rest}
    >
      {children}
    </Link>
  )
})

ArrowLink.displayName = 'ArrowLink'

export default Link
