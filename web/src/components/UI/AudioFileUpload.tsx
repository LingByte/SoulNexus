import { useState, useRef } from 'react'
import { Upload, Mic, X, Play, Pause, Volume2 } from 'lucide-react'
import Button from './Button'

interface AudioFileUploadProps {
  onFileSelect: (file: File | null) => void
  accept?: string
  maxSize?: number // MB
  className?: string
  placeholder?: string
  required?: boolean
}

const AudioFileUpload = ({
  onFileSelect,
  accept = '.wav',
  maxSize = 10,
  className = '',
  placeholder = '选择音频文件或拖拽到此处',
  required = false
}: AudioFileUploadProps) => {
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [isDragging, setIsDragging] = useState(false)
  const [isPlaying, setIsPlaying] = useState(false)
  const [error, setError] = useState<string>('')
  const fileInputRef = useRef<HTMLInputElement>(null)
  const audioRef = useRef<HTMLAudioElement>(null)

  const validateFile = (file: File): string | null => {
    // 检查文件类型
    if (!file.name.toLowerCase().endsWith('.wav')) {
      return '仅支持 WAV 格式的音频文件'
    }

    // 检查文件大小
    if (file.size > maxSize * 1024 * 1024) {
      return `文件大小不能超过 ${maxSize}MB`
    }

    return null
  }

  const handleFileSelect = (file: File) => {
    const validationError = validateFile(file)
    if (validationError) {
      setError(validationError)
      return
    }

    setError('')
    setSelectedFile(file)
    onFileSelect(file)
  }

  const handleFileInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      handleFileSelect(file)
    }
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(true)
  }

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(false)
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(false)
    
    const files = e.dataTransfer.files
    if (files.length > 0) {
      handleFileSelect(files[0])
    }
  }

  const handleRemoveFile = () => {
    setSelectedFile(null)
    setError('')
    setIsPlaying(false)
    onFileSelect(null)
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  const togglePlayback = () => {
    if (!selectedFile || !audioRef.current) return

    if (isPlaying) {
      audioRef.current.pause()
      setIsPlaying(false)
    } else {
      const url = URL.createObjectURL(selectedFile)
      audioRef.current.src = url
      audioRef.current.play()
      setIsPlaying(true)
    }
  }

  const handleAudioEnded = () => {
    setIsPlaying(false)
  }

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }
  return (
    <div className={`space-y-3 ${className}`}>
      {!selectedFile ? (
        <div
          className={`
            relative border-2 border-dashed rounded-lg p-6 text-center cursor-pointer transition-colors
            ${isDragging 
              ? 'border-purple-400 bg-purple-50 dark:bg-purple-900/20' 
              : 'border-gray-300 dark:border-gray-600 hover:border-purple-400 hover:bg-gray-50 dark:hover:bg-gray-800/50'
            }
            ${error ? 'border-red-400 bg-red-50 dark:bg-red-900/20' : ''}
          `}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          onClick={() => fileInputRef.current?.click()}
        >
          <input
            ref={fileInputRef}
            type="file"
            accept={accept}
            onChange={handleFileInputChange}
            className="hidden"
            required={required}
          />
          
          <div className="flex flex-col items-center space-y-3">
            <div className={`
              p-3 rounded-full 
              ${error 
                ? 'bg-red-100 dark:bg-red-900/30' 
                : 'bg-purple-100 dark:bg-purple-900/30'
              }
            `}>
              {error ? (
                <X className="w-6 h-6 text-red-600 dark:text-red-400" />
              ) : (
                <Upload className="w-6 h-6 text-purple-600 dark:text-purple-400" />
              )}
            </div>
            
            <div>
              <p className={`text-sm font-medium ${error ? 'text-red-700 dark:text-red-300' : 'text-gray-700 dark:text-gray-300'}`}>
                {error || placeholder}
              </p>
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                支持 WAV 格式，最大 {maxSize}MB
              </p>
            </div>
          </div>
        </div>
      ) : (
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 bg-gray-50 dark:bg-gray-800/50">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3 flex-1 min-w-0">
              <div className="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
                <Mic className="w-5 h-5 text-purple-600 dark:text-purple-400" />
              </div>
              
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                  {selectedFile.name}
                </p>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  {formatFileSize(selectedFile.size)}
                </p>
              </div>
            </div>
            
            <div className="flex items-center space-x-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={togglePlayback}
                className="text-purple-600 dark:text-purple-400 hover:bg-purple-50 dark:hover:bg-purple-900/20"
              >
                {isPlaying ? (
                  <Pause className="w-4 h-4" />
                ) : (
                  <Play className="w-4 h-4" />
                )}
              </Button>
              
              <Button
                variant="ghost"
                size="sm"
                onClick={handleRemoveFile}
                className="text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
              >
                <X className="w-4 h-4" />
              </Button>
            </div>
          </div>
          
          {/* 音频播放器 */}
          <audio
            ref={audioRef}
            onEnded={handleAudioEnded}
            className="hidden"
          />
        </div>
      )}
      
      {error && (
        <p className="text-sm text-red-600 dark:text-red-400 flex items-center space-x-1">
          <X className="w-4 h-4" />
          <span>{error}</span>
        </p>
      )}
      
      {selectedFile && !error && (
        <div className="flex items-center space-x-2 text-xs text-gray-500 dark:text-gray-400">
          <Volume2 className="w-3 h-3" />
          <span>点击播放按钮可以预览音频</span>
        </div>
      )}
    </div>
  )
}

export default AudioFileUpload