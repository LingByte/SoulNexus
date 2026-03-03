import React from 'react';
import { cn } from '@/utils/cn';

interface ProgressProps {
  value: number;
  max?: number;
  className?: string;
  showValue?: boolean;
  size?: 'sm' | 'md' | 'lg';
  variant?: 'default' | 'success' | 'warning' | 'danger';
}

const Progress: React.FC<ProgressProps> = ({
  value,
  max = 100,
  className,
  showValue = false,
  size = 'md',
  variant = 'default',
}) => {
  const percentage = Math.min(Math.max((value / max) * 100, 0), 100);

  const sizeClasses = {
    sm: 'h-1',
    md: 'h-2',
    lg: 'h-3',
  };

  const variantClasses = {
    default: 'bg-blue-500',
    success: 'bg-green-500',
    warning: 'bg-yellow-500',
    danger: 'bg-red-500',
  };

  return (
    <div className={cn('w-full', className)}>
      <div className={cn(
        'w-full bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden',
        sizeClasses[size]
      )}>
        <div
          className={cn(
            'h-full transition-all duration-300 ease-in-out rounded-full',
            variantClasses[variant]
          )}
          style={{ width: `${percentage}%` }}
        />
      </div>
      {showValue && (
        <div className="flex justify-between text-xs text-gray-600 dark:text-gray-400 mt-1">
          <span>{value}</span>
          <span>{max}</span>
        </div>
      )}
    </div>
  );
};

export default Progress;