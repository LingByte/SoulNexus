import React from 'react'
import { motion } from 'framer-motion'
import { ChevronDown } from 'lucide-react'

interface CollapsibleSectionHeaderProps {
  title: string
  icon?: React.ReactNode
  expanded: boolean
  onToggle: () => void
  rightContent?: React.ReactNode
  children?: React.ReactNode
  showChevron?: boolean
  clickable?: boolean
  compact?: boolean
  titleSize?: 'sm' | 'md' | 'lg'
  withDivider?: boolean
  className?: string
}

const CollapsibleSectionHeader: React.FC<CollapsibleSectionHeaderProps> = ({
  title,
  icon,
  expanded,
  onToggle,
  rightContent,
  children,
  showChevron = true,
  clickable = true,
  compact = false,
  titleSize,
  withDivider = false,
  className = '',
}) => {
  const rightSlot = rightContent ?? children
  const computedTitleSize =
    titleSize ?? (compact ? 'sm' : 'lg')
  const titleClass =
    computedTitleSize === 'lg'
      ? 'text-lg'
      : computedTitleSize === 'md'
        ? 'text-base'
        : 'text-sm'

  return (
    <div
      className={`w-full flex justify-between items-end group ${clickable ? 'cursor-pointer' : ''} ${withDivider ? 'pb-1 border-b border-gray-300 dark:border-neutral-600' : ''} ${className}`}
      onClick={clickable ? onToggle : undefined}
    >
      <div className="flex items-center">
        <h3 className={`${titleClass} leading-5 font-semibold flex items-center`}>
          {icon}
          <span className={`${compact ? 'ml-1.5' : 'ml-2'}`}>{title}</span>
        </h3>
        {showChevron && (
          <motion.div
            animate={{ rotate: expanded ? 0 : -90 }}
            transition={{ duration: 0.2 }}
            className="ml-2"
          >
            <ChevronDown className="w-4 h-4 text-gray-500 group-hover:text-purple-600 transition-colors" />
          </motion.div>
        )}
      </div>
      {rightSlot}
    </div>
  )
}

export default CollapsibleSectionHeader
