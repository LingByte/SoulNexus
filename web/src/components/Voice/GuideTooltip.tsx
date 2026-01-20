import React, { useState, useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'

interface GuideTooltipProps {
  text: string
  position?: 'top' | 'bottom' | 'left' | 'right'
  onNext?: () => void
  onClose?: () => void
  isLast?: boolean
}

const GuideTooltip: React.FC<GuideTooltipProps> = ({
  text,
  position = 'bottom',
  onNext,
  onClose,
  isLast = false
}) => {
  const tooltipRef = useRef<HTMLDivElement>(null);
  const adjustedPosition = position;
  const tooltipStyle: React.CSSProperties = {};
  const [isVisible, setIsVisible] = useState(false);

  // 获取指示器样式
  const getIndicatorStyle = (pos: string) => {
    switch (pos) {
      case 'top':
        return {
          top: '100%',
          left: '50%',
          transform: 'translateX(-50%)',
          borderTop: 'none',
          borderLeft: 'none'
        };
      case 'bottom':
        return {
          bottom: '100%',
          left: '50%',
          transform: 'translateX(-50%)',
          borderBottom: 'none',
          borderRight: 'none'
        };
      case 'left':
        return {
          left: '100%',
          top: '50%',
          transform: 'translateY(-50%)',
          borderLeft: 'none',
          borderTop: 'none'
        };
      case 'right':
        return {
          right: '100%',
          top: '50%',
          transform: 'translateY(-50%)',
          borderRight: 'none',
          borderBottom: 'none'
        };
      default:
        return {
          top: '100%',
          left: '50%',
          transform: 'translateX(-50%)'
        };
    }
  };

  useEffect(() => {
    setIsVisible(true);
  }, []);

  return createPortal(
    <div
      ref={tooltipRef}
      className={`fixed inset-0 z-[9998] pointer-events-none transition-opacity duration-300 ${
        isVisible ? 'opacity-100' : 'opacity-0'
      }`}
    >
      <div
        className="absolute bg-white dark:bg-neutral-800 border-2 border-blue-500 rounded-lg shadow-xl p-4 min-w-[200px] max-w-[300px] pointer-events-auto"
        style={{
          ...tooltipStyle,
          ...getIndicatorStyle(adjustedPosition)
        }}
      >
        <div className="text-sm text-gray-800 dark:text-gray-200 mb-3">
          {text}
        </div>
        <div className="flex justify-end gap-2">
          {!isLast && onNext && (
            <button
              onClick={onNext}
              className="px-3 py-1.5 text-xs bg-blue-500 text-white rounded hover:bg-blue-600 transition-colors"
            >
              下一步
            </button>
          )}
          {onClose && (
            <button
              onClick={onClose}
              className="px-3 py-1.5 text-xs bg-gray-500 text-white rounded hover:bg-gray-600 transition-colors"
            >
              {isLast ? '完成' : '跳过'}
            </button>
          )}
        </div>
        {/* 指示器三角形 */}
        <div
          className={`absolute w-0 h-0 border-8 border-transparent ${
            adjustedPosition === 'top'
              ? 'border-b-blue-500 bottom-[-16px] left-1/2 transform -translate-x-1/2'
              : adjustedPosition === 'bottom'
              ? 'border-t-blue-500 top-[-16px] left-1/2 transform -translate-x-1/2'
              : adjustedPosition === 'left'
              ? 'border-r-blue-500 right-[-16px] top-1/2 transform -translate-y-1/2'
              : 'border-l-blue-500 left-[-16px] top-1/2 transform -translate-y-1/2'
          }`}
        />
      </div>
    </div>,
    document.body
  );
};

export default GuideTooltip;

