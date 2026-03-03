'use client'

import { useRef, useState, useEffect } from 'react'

export interface ParsedMessage {
  role: 'user' | 'agent' | 'system'
  content: string
  timeInCallSecs: number
}

interface CallAudioPlayerProps {
  callId: string
  audioUrl: string
  hasAudio: boolean
  durationSeconds: number | null
  messages?: ParsedMessage[]
}

export default function CallAudioPlayer({ 
  callId: _, // Unused but kept for API consistency
  audioUrl,
  hasAudio, 
  durationSeconds, 
  messages = [] 
}: CallAudioPlayerProps) {
  const audioRef = useRef<HTMLAudioElement>(null)
  const waveformRef = useRef<HTMLDivElement>(null)
  const [isPlaying, setIsPlaying] = useState(false)
  const [currentTime, setCurrentTime] = useState(0)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [waveformData, setWaveformData] = useState<number[]>([])

  // Use duration from props (from database)
  const duration = durationSeconds || 0

  // Generate waveform data from audio
  useEffect(() => {
    const audio = audioRef.current
    if (!audio || !audio.src) return

    const generateWaveform = async () => {
      try {
        const response = await fetch(audio.src)
        const arrayBuffer = await response.arrayBuffer()
        const audioContext = new (window.AudioContext || (window as any).webkitAudioContext)()
        const audioBuffer = await audioContext.decodeAudioData(arrayBuffer)

        // Get audio data from the first channel
        const rawData = audioBuffer.getChannelData(0)
        const samples = 100 // Number of bars to display
        const blockSize = Math.floor(rawData.length / samples)
        const filteredData: number[] = []

        for (let i = 0; i < samples; i++) {
          const blockStart = blockSize * i
          let sum = 0
          for (let j = 0; j < blockSize; j++) {
            sum += Math.abs(rawData[blockStart + j])
          }
          filteredData.push(sum / blockSize)
        }

        // Normalize the data
        const max = Math.max(...filteredData)
        const normalized = filteredData.map(n => (n / max) * 100)
        setWaveformData(normalized)
      } catch (err) {
        console.error('Failed to generate waveform:', err)
        // Fallback to random data if generation fails
        setWaveformData(Array.from({ length: 100 }, () => Math.random() * 80 + 20))
      }
    }

    generateWaveform()
  }, [audioUrl])

  useEffect(() => {
    const audio = audioRef.current
    if (!audio) return

    const handleTimeUpdate = () => {
      setCurrentTime(audio.currentTime)
    }

    const handleEnded = () => {
      setIsPlaying(false)
      setCurrentTime(0)
    }

    const handleLoadStart = () => setIsLoading(true)
    const handleCanPlay = () => setIsLoading(false)

    const handleError = () => {
      console.error('音频加载失败:', {
        audioUrl,
        error: audioRef.current?.error
      });
      setError('音频加载失败');
      setIsLoading(false);
    };

    const handlePlay = () => setIsPlaying(true)
    const handlePause = () => setIsPlaying(false)

    audio.addEventListener('timeupdate', handleTimeUpdate)
    audio.addEventListener('ended', handleEnded)
    audio.addEventListener('loadstart', handleLoadStart)
    audio.addEventListener('canplay', handleCanPlay)
    audio.addEventListener('error', handleError)
    audio.addEventListener('play', handlePlay)
    audio.addEventListener('pause', handlePause)

    return () => {
      audio.removeEventListener('timeupdate', handleTimeUpdate)
      audio.removeEventListener('ended', handleEnded)
      audio.removeEventListener('loadstart', handleLoadStart)
      audio.removeEventListener('canplay', handleCanPlay)
      audio.removeEventListener('error', handleError)
      audio.removeEventListener('play', handlePlay)
      audio.removeEventListener('pause', handlePause)
    }
  }, [])

  const togglePlayPause = async () => {
    const audio = audioRef.current
    if (!audio) return

    try {
      if (isPlaying) {
        audio.pause()
      } else {
        await audio.play()
      }
    } catch (err) {
      console.error('Playback error:', err)
      setError('播放失败')
    }
  }

  const handleWaveformClick = async (e: React.MouseEvent<HTMLDivElement>) => {
    const audio = audioRef.current
    const waveform = waveformRef.current
    if (!audio || !waveform || !duration || duration === 0) return

    const rect = waveform.getBoundingClientRect()
    const x = e.clientX - rect.left
    const percentage = Math.max(0, Math.min(1, x / rect.width))
    const newTime = percentage * duration

    audio.currentTime = newTime

    // Auto play if not playing
    if (!isPlaying) {
      try {
        await audio.play()
      } catch (err) {
        console.error('Playback error:', err)
      }
    }
  }

  const formatTime = (seconds: number) => {
    if (isNaN(seconds) || seconds === null) return '0:00'
    const mins = Math.floor(seconds / 60)
    const secs = Math.floor(seconds % 60)
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }

  if (!hasAudio) {
    return null
  }

  const progress = duration > 0 ? (currentTime / duration) * 100 : 0

  return (
    <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white">通话录音</h3>
        <div className="text-sm text-gray-500 dark:text-gray-400 font-mono">
          {formatTime(currentTime)} / {formatTime(duration)}
        </div>
      </div>

      <audio
        ref={audioRef}
        src={audioUrl}
        preload="metadata"
      />

      {error && (
        <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
          <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
        </div>
      )}

      <div className="flex items-center gap-3">
        {/* Play/Pause Button */}
        <button
          onClick={togglePlayPause}
          disabled={isLoading}
          className="flex-shrink-0 w-10 h-10 rounded-full bg-black dark:bg-gray-700 hover:bg-gray-800 dark:hover:bg-gray-600 disabled:bg-gray-400 text-white flex items-center justify-center transition-colors"
        >
          {isLoading ? (
            <svg className="animate-spin h-5 w-5" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          ) : isPlaying ? (
            <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 24 24">
              <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z" />
            </svg>
          ) : (
            <svg className="h-5 w-5 ml-0.5" fill="currentColor" viewBox="0 0 24 24">
              <path d="M8 5v14l11-7z" />
            </svg>
          )}
        </button>

        {/* Waveform-style Progress Bar */}
        <div
          ref={waveformRef}
          onClick={handleWaveformClick}
          className="flex-1 h-16 relative bg-gray-100 dark:bg-gray-700 rounded cursor-pointer overflow-hidden group"
        >
          {/* Speaker segments background */}
          {duration > 0 && messages.length > 0 && (
            <div className="absolute inset-0">
              {messages.map((msg, idx) => {
                const nextMsg = messages[idx + 1]
                const startPercent = (msg.timeInCallSecs / duration) * 100
                const endTime = nextMsg ? nextMsg.timeInCallSecs : duration
                const widthPercent = ((endTime - msg.timeInCallSecs) / duration) * 100

                // Color based on speaker role
                const bgColor = msg.role === 'user'
                  ? 'bg-blue-100/50 dark:bg-blue-900/30'
                  : msg.role === 'agent'
                  ? 'bg-green-100/50 dark:bg-green-900/30'
                  : 'bg-gray-100/50 dark:bg-gray-700/50'

                return (
                  <div
                    key={idx}
                    className={`absolute top-0 bottom-0 ${bgColor}`}
                    style={{
                      left: `${startPercent}%`,
                      width: `${widthPercent}%`,
                    }}
                  />
                )
              })}
            </div>
          )}

          {/* Waveform bars */}
          <div className="absolute inset-0 flex items-center justify-around px-1 z-10">
            {(waveformData.length > 0 ? waveformData : Array.from({ length: 100 }, () => 50)).map((height, i) => {
              const isPlayed = (i / 100) * 100 < progress
              return (
                <div
                  key={i}
                  className={`w-0.5 rounded-full transition-colors ${
                    isPlayed ? 'bg-black dark:bg-white' : 'bg-gray-300 dark:bg-gray-600 group-hover:bg-gray-400 dark:group-hover:bg-gray-500'
                  }`}
                  style={{ height: `${Math.max(20, height)}%` }}
                />
              )
            })}
          </div>

          {/* Progress indicator */}
          <div
            className="absolute top-0 bottom-0 w-0.5 bg-red-500 z-20"
            style={{ left: `${progress}%` }}
          />
        </div>
      </div>
    </div>
  )
}

