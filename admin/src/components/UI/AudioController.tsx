import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { Volume2, VolumeX, Music, Zap } from 'lucide-react'
import { audioManager, setAudioEnabled, setAudioVolume } from '@/utils/audioEffects.ts'

interface AudioControllerProps {
  className?: string
  position?: 'top-right' | 'top-left' | 'bottom-right' | 'bottom-left'
}

const AudioController = ({ 
  className = '', 
  position = 'bottom-right' 
}: AudioControllerProps) => {
  const [isEnabled, setIsEnabled] = useState(true)
  const [volume, setVolume] = useState(0.3)
  const [isExpanded, setIsExpanded] = useState(false)

  useEffect(() => {
    setAudioEnabled(isEnabled)
  }, [isEnabled])

  useEffect(() => {
    setAudioVolume(volume)
  }, [volume])

  const getPositionClasses = () => {
    switch (position) {
      case 'top-right':
        return 'top-4 right-4'
      case 'top-left':
        return 'top-4 left-4'
      case 'bottom-right':
        return 'bottom-4 right-4'
      case 'bottom-left':
        return 'bottom-4 left-4'
      default:
        return 'bottom-4 right-4'
    }
  }

  const handleVolumeChange = (newVolume: number) => {
    setVolume(newVolume)
  }

  const toggleEnabled = () => {
    setIsEnabled(!isEnabled)
  }

  const testSound = () => {
    audioManager.playMagicSound()
  }

  return (
    <div className={`fixed z-50 ${getPositionClasses()} ${className}`}>
      <motion.div
        initial={{ opacity: 0, scale: 0.8 }}
        animate={{ opacity: 1, scale: 1 }}
        className="bg-white dark:bg-gray-800 rounded-xl shadow-lg border border-gray-200 dark:border-gray-700 overflow-hidden"
      >
        {/* 主控制按钮 */}
        <motion.button
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
          onClick={() => setIsExpanded(!isExpanded)}
          className="flex items-center gap-2 p-3 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
        >
          {isEnabled ? (
            <Volume2 className="w-5 h-5 text-blue-600" />
          ) : (
            <VolumeX className="w-5 h-5 text-gray-400" />
          )}
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
            音频
          </span>
        </motion.button>

        {/* 展开的控制面板 */}
        <motion.div
          initial={{ height: 0, opacity: 0 }}
          animate={{ 
            height: isExpanded ? 'auto' : 0, 
            opacity: isExpanded ? 1 : 0 
          }}
          transition={{ duration: 0.2 }}
          className="overflow-hidden"
        >
          <div className="p-4 space-y-4 border-t border-gray-200 dark:border-gray-700">
            {/* 开关控制 */}
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-600 dark:text-gray-400">启用音效</span>
              <button
                onClick={toggleEnabled}
                className={`relative w-12 h-6 rounded-full transition-colors ${
                  isEnabled 
                    ? 'bg-blue-600' 
                    : 'bg-gray-300 dark:bg-gray-600'
                }`}
              >
                <motion.div
                  className="absolute top-1 w-4 h-4 bg-white rounded-full shadow-md"
                  animate={{ x: isEnabled ? 28 : 2 }}
                  transition={{ duration: 0.2 }}
                />
              </button>
            </div>

            {/* 音量控制 */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600 dark:text-gray-400">音量</span>
                <span className="text-xs text-gray-500 dark:text-gray-400">
                  {Math.round(volume * 100)}%
                </span>
              </div>
              <input
                type="range"
                min="0"
                max="1"
                step="0.1"
                value={volume}
                onChange={(e) => handleVolumeChange(Number(e.target.value))}
                className="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer slider"
                style={{
                  background: `linear-gradient(to right, #3b82f6 0%, #3b82f6 ${volume * 100}%, #e5e7eb ${volume * 100}%, #e5e7eb 100%)`
                }}
              />
            </div>

            {/* 测试按钮 */}
            <motion.button
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              onClick={testSound}
              className="w-full flex items-center justify-center gap-2 py-2 px-3 bg-gradient-to-r from-purple-500 to-pink-500 text-white rounded-lg text-sm font-medium hover:from-purple-600 hover:to-pink-600 transition-all"
            >
              <Zap className="w-4 h-4" />
              测试音效
            </motion.button>

            {/* 快捷音效按钮 */}
            <div className="grid grid-cols-2 gap-2">
              <motion.button
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.98 }}
                onClick={() => audioManager.playSuccessSound()}
                className="flex items-center justify-center gap-1 py-2 px-2 bg-green-100 dark:bg-green-900/20 text-green-700 dark:text-green-400 rounded-lg text-xs hover:bg-green-200 dark:hover:bg-green-900/40 transition-colors"
              >
                <Music className="w-3 h-3" />
                成功
              </motion.button>
              <motion.button
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.98 }}
                onClick={() => audioManager.playErrorSound()}
                className="flex items-center justify-center gap-1 py-2 px-2 bg-red-100 dark:bg-red-900/20 text-red-700 dark:text-red-400 rounded-lg text-xs hover:bg-red-200 dark:hover:bg-red-900/40 transition-colors"
              >
                <Music className="w-3 h-3" />
                错误
              </motion.button>
            </div>
          </div>
        </motion.div>
      </motion.div>

      <style>{`
        .slider::-webkit-slider-thumb {
          appearance: none;
          width: 16px;
          height: 16px;
          border-radius: 50%;
          background: #3b82f6;
          cursor: pointer;
          border: 2px solid white;
          box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
        }
        
        .slider::-moz-range-thumb {
          width: 16px;
          height: 16px;
          border-radius: 50%;
          background: #3b82f6;
          cursor: pointer;
          border: 2px solid white;
          box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
        }
      `}</style>
    </div>
  )
}

export default AudioController
