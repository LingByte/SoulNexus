import React from 'react'
import { LineMode } from '@/pages/VoiceAssistant/types'

interface LineSelectorProps {
  lineMode: LineMode
  onLineModeChange: (mode: LineMode) => void
}

const LineSelector: React.FC<LineSelectorProps> = ({ lineMode, onLineModeChange }) => {
  return (
    <div className="mb-3 flex items-center justify-center">
      <div className="relative bg-gray-100 dark:bg-neutral-700 rounded-xl p-1 flex gap-1 shadow-inner">
        <div
          className={`absolute top-1 bottom-1 w-1/2 rounded-lg bg-gradient-to-r from-blue-500 to-cyan-500 transition-transform duration-300 ease-out shadow-[0_0_20px_rgba(59,130,246,0.6)] ${lineMode === 'webrtc' ? 'translate-x-0' : 'translate-x-full'}`}
        />
        <button
          className={`relative z-10 px-4 py-1 text-sm rounded-lg transition-all duration-200 ${lineMode === 'webrtc' ? 'text-blue-700 dark:text-blue-300 scale-[1.02]' : 'text-gray-700 dark:text-gray-200 hover:text-gray-900'} `}
          onClick={() => onLineModeChange('webrtc')}
        >
          WebRTC
        </button>
        <button
          className={`relative z-10 px-4 py-1 text-sm rounded-lg transition-all duration-200 ${lineMode === 'websocket' ? 'text-blue-700 dark:text-blue-300 scale-[1.02]' : 'text-gray-700 dark:text-gray-200 hover:text-gray-900'} `}
          onClick={() => onLineModeChange('websocket')}
        >
          WebSocket
        </button>
      </div>
      
      {/* 提示图标 */}
      <div className="ml-2 relative group">
        <div className="w-5 h-5 bg-gray-400 dark:bg-gray-500 rounded-full flex items-center justify-center cursor-help">
          <span className="text-white text-xs font-bold">?</span>
        </div>
        
        {/* 悬停提示框 - 显示在屏幕中央，避免被遮挡 */}
        <div className="fixed inset-0 flex items-center justify-center pointer-events-none z-[9999] opacity-0 group-hover:opacity-100 transition-opacity duration-200">
          <div className="px-4 py-3 bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900 text-sm rounded-lg shadow-xl max-w-sm mx-4">
            <div className="font-medium mb-2">线路选择说明：</div>
            <div className="space-y-2">
              <div className="flex items-start gap-2">
                <span>
                  <span className="text-blue-300 dark:text-blue-600 font-medium">WebRTC：</span>
                  此线路是WebRTC实时通信，超低延迟，适合连续对话
                </span>
              </div>
              <div className="flex items-start gap-2">
                <span>
                  <span className="text-cyan-300 dark:text-cyan-600 font-medium">WebSocket：</span>
                  WebSocket语音服务，高精度识别，略有延迟
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default LineSelector

