import React from 'react'
import { LineMode } from '@/pages/VoiceAssistant/types'

interface LineSelectorProps {
  lineMode: LineMode
  onLineModeChange: (mode: LineMode) => void
}

const LineSelector: React.FC<LineSelectorProps> = ({ lineMode, onLineModeChange }) => {
  return (
    <div className="flex items-center justify-center gap-1.5">
      <div className="relative flex-1 bg-gray-100 dark:bg-neutral-700/80 rounded-lg p-0.5 flex shadow-inner max-w-[11rem]">
        <div
          className={`absolute top-1 bottom-1 w-1/2 rounded-lg bg-gradient-to-r from-blue-500 to-cyan-500 transition-transform duration-300 ease-out shadow-[0_0_20px_rgba(59,130,246,0.6)] ${lineMode === 'webrtc' ? 'translate-x-0' : 'translate-x-full'}`}
        />
        <button
          className={`relative z-10 flex-1 px-2 py-1 text-[11px] font-medium rounded-md transition-all duration-200 ${lineMode === 'webrtc' ? 'text-blue-700 dark:text-blue-300' : 'text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100'} `}
          onClick={() => onLineModeChange('webrtc')}
        >
          WebRTC
        </button>
        <button
          className={`relative z-10 flex-1 px-2 py-1 text-[11px] font-medium rounded-md transition-all duration-200 ${lineMode === 'websocket' ? 'text-blue-700 dark:text-blue-300' : 'text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100'} `}
          onClick={() => onLineModeChange('websocket')}
        >
          WebSocket
        </button>
      </div>
      
      {/* 提示图标 */}
      <div className="relative group shrink-0">
        <div className="w-4 h-4 bg-neutral-300 dark:bg-neutral-600 rounded-full flex items-center justify-center cursor-help">
          <span className="text-white dark:text-neutral-200 text-[10px] font-bold leading-none">?</span>
        </div>
        
        {/* 悬停提示框 - 显示在屏幕中央，避免被遮挡 */}
        <div className="fixed inset-0 flex items-center justify-center pointer-events-none z-[9999] opacity-0 group-hover:opacity-100 transition-opacity duration-200">
          <div className="px-4 py-3 bg-gray-900 dark:bg-gray-100 text-white dark:text-gray-900 text-sm rounded-lg shadow-xl max-w-sm mx-4">
            <div className="font-medium mb-2">线路选择说明：</div>
            <div className="space-y-2">
              <div className="flex items-start gap-2">
                <span>
                  <span className="text-blue-300 dark:text-blue-600 font-medium">WebRTC：</span>
                  页内对接 cmd/voice（<code className="text-xs">VITE_CMD_VOICE_BASE</code>），媒体在语音进程，对话走业务 WS。
                </span>
              </div>
              <div className="flex items-start gap-2">
                <span>
                  <span className="text-cyan-300 dark:text-cyan-600 font-medium">WebSocket：</span>
                  同样连 cmd/voice 的 xiaozhi 协议（<code className="text-xs">/xiaozhi/v1/?payload=</code> 透传鉴权），PCM 上行 / TTS 下行。
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

