import React, { useState, useRef, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Play, Pause, Volume2, VolumeX, Download, RotateCcw } from 'lucide-react';

interface VoicePlayerProps {
  audioUrl?: string;
  audioData?: string; // base64 encoded audio data
  title?: string;
  autoPlay?: boolean;
  onPlay?: () => void;
  onPause?: () => void;
  onEnd?: () => void;
  className?: string;
}

const VoicePlayer: React.FC<VoicePlayerProps> = ({
  audioUrl,
  audioData,
  title = '语音播放',
  autoPlay = false,
  onPlay,
  onPause,
  onEnd,
  className = ''
}) => {
  const [isPlaying, setIsPlaying] = useState(false);
  const [isMuted, setIsMuted] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(1);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const progressRef = useRef<HTMLDivElement | null>(null);

  // 初始化音频
  useEffect(() => {
    if (audioRef.current) {
      const audio = audioRef.current;
      
      const handleLoadedMetadata = () => {
        setDuration(audio.duration);
        setIsLoading(false);
      };
      
      const handleTimeUpdate = () => {
        setCurrentTime(audio.currentTime);
      };
      
      const handlePlay = () => {
        setIsPlaying(true);
        onPlay?.();
      };
      
      const handlePause = () => {
        setIsPlaying(false);
        onPause?.();
      };
      
      const handleEnded = () => {
        setIsPlaying(false);
        setCurrentTime(0);
        onEnd?.();
      };
      
      const handleError = (e: Event) => {
        const audio = e.target as HTMLAudioElement;
        console.error('音频加载失败:', {
          src: audio.src,
          error: audio.error,
          networkState: audio.networkState,
          readyState: audio.readyState
        });
        setError(`音频加载失败: ${audio.error?.message || '未知错误'}`);
        setIsLoading(false);
      };
      
      audio.addEventListener('loadedmetadata', handleLoadedMetadata);
      audio.addEventListener('timeupdate', handleTimeUpdate);
      audio.addEventListener('play', handlePlay);
      audio.addEventListener('pause', handlePause);
      audio.addEventListener('ended', handleEnded);
      audio.addEventListener('error', handleError);
      
      return () => {
        audio.removeEventListener('loadedmetadata', handleLoadedMetadata);
        audio.removeEventListener('timeupdate', handleTimeUpdate);
        audio.removeEventListener('play', handlePlay);
        audio.removeEventListener('pause', handlePause);
        audio.removeEventListener('ended', handleEnded);
        audio.removeEventListener('error', handleError);
      };
    }
  }, [onPlay, onPause, onEnd]);

  // 设置音频源
  useEffect(() => {
    if (audioRef.current) {
      setIsLoading(true);
      setError(null);
      
      if (audioUrl) {
        console.log('设置音频URL:', audioUrl);
        audioRef.current.src = audioUrl;
        
        // 添加crossOrigin属性以支持跨域音频
        audioRef.current.crossOrigin = 'anonymous';
        
        // 尝试预加载
        audioRef.current.load();
      } else if (audioData) {
        // 根据数据格式设置正确的MIME类型
        const mimeType = audioData.startsWith('/9j/') ? 'audio/wav' : 'audio/mp3';
        const dataUrl = `data:${mimeType};base64,${audioData}`;
        console.log('设置音频数据URL:', dataUrl.substring(0, 100) + '...');
        audioRef.current.src = dataUrl;
      }
      
      if (autoPlay) {
        audioRef.current.play().catch((error) => {
          console.error('自动播放失败:', error);
          setError('自动播放失败');
        });
      }
    }
  }, [audioUrl, audioData, autoPlay]);

  // 播放/暂停
  const togglePlay = async () => {
    if (audioRef.current) {
      try {
        if (isPlaying) {
          audioRef.current.pause();
        } else {
          await audioRef.current.play();
        }
      } catch (error) {
        console.error('播放失败:', error);
        setError('播放失败');
      }
    }
  };

  // 静音/取消静音
  const toggleMute = () => {
    if (audioRef.current) {
      audioRef.current.muted = !isMuted;
      setIsMuted(!isMuted);
    }
  };

  // 设置音量
  const handleVolumeChange = (newVolume: number) => {
    if (audioRef.current) {
      audioRef.current.volume = newVolume;
      setVolume(newVolume);
    }
  };

  // 跳转到指定时间
  const handleSeek = (event: React.MouseEvent<HTMLDivElement>) => {
    if (audioRef.current && progressRef.current) {
      const rect = progressRef.current.getBoundingClientRect();
      const clickX = event.clientX - rect.left;
      const percentage = clickX / rect.width;
      const newTime = percentage * duration;
      
      audioRef.current.currentTime = newTime;
      setCurrentTime(newTime);
    }
  };

  // 重置播放
  const resetPlayback = () => {
    if (audioRef.current) {
      audioRef.current.currentTime = 0;
      setCurrentTime(0);
    }
  };

  // 下载音频
  const downloadAudio = () => {
    if (audioRef.current && audioRef.current.src) {
      const link = document.createElement('a');
      // 如果是API URL，添加download参数
      if (audioRef.current.src.includes('/api/recordings/')) {
        link.href = audioRef.current.src + '?download=1';
      } else {
        link.href = audioRef.current.src;
      }
      
      // 根据URL推断文件扩展名
      let extension = '.wav';
      if (audioRef.current.src.includes('.mp3')) extension = '.mp3';
      if (audioRef.current.src.includes('.opus')) extension = '.opus';
      
      link.download = `${title}${extension}`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    }
  };

  // 格式化时间
  const formatTime = (time: number) => {
    if (isNaN(time)) return '0:00';
    const minutes = Math.floor(time / 60);
    const seconds = Math.floor(time % 60);
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  };

  // 计算进度百分比
  const progressPercentage = duration > 0 ? (currentTime / duration) * 100 : 0;

  return (
    <div className={`voice-player ${className}`}>
      {/* 音频元素 */}
      <audio
        ref={audioRef}
        preload="metadata"
        className="hidden"
      />
      
      {/* 播放器界面 */}
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg p-4">
        {/* 标题 */}
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-medium text-gray-900 dark:text-white">
            {title}
          </h3>
          <div className="flex items-center space-x-2">
            <motion.button
              onClick={downloadAudio}
              className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              <Download className="w-4 h-4" />
            </motion.button>
            <motion.button
              onClick={resetPlayback}
              className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              <RotateCcw className="w-4 h-4" />
            </motion.button>
          </div>
        </div>

        {/* 错误信息 */}
        {error && (
          <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
            <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
            {audioUrl && (
              <div className="mt-2">
                <p className="text-xs text-gray-500">音频URL: {audioUrl}</p>
                <button 
                  onClick={() => window.open(audioUrl, '_blank')}
                  className="text-xs text-blue-500 hover:text-blue-700 underline"
                >
                  在新窗口中测试URL
                </button>
              </div>
            )}
          </div>
        )}

        {/* 加载状态 */}
        {isLoading && (
          <div className="mb-4 flex items-center justify-center">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-500"></div>
            <span className="ml-2 text-sm text-gray-600 dark:text-gray-400">加载中...</span>
          </div>
        )}

        {/* 进度条 */}
        <div className="mb-4">
          <div
            ref={progressRef}
            className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2 cursor-pointer"
            onClick={handleSeek}
          >
            <div
              className="bg-blue-500 h-2 rounded-full transition-all duration-300"
              style={{ width: `${progressPercentage}%` }}
            />
          </div>
          <div className="flex justify-between text-xs text-gray-500 dark:text-gray-400 mt-1">
            <span>{formatTime(currentTime)}</span>
            <span>{formatTime(duration)}</span>
          </div>
        </div>

        {/* 控制按钮 */}
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            {/* 播放/暂停按钮 */}
            <motion.button
              onClick={togglePlay}
              disabled={isLoading || !!error}
              className="flex items-center justify-center w-12 h-12 bg-blue-500 hover:bg-blue-600 disabled:bg-gray-400 disabled:cursor-not-allowed text-white rounded-full transition-colors"
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              {isPlaying ? <Pause className="w-6 h-6" /> : <Play className="w-6 h-6" />}
            </motion.button>

            {/* 静音按钮 */}
            <motion.button
              onClick={toggleMute}
              className="flex items-center justify-center w-10 h-10 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              {isMuted ? <VolumeX className="w-5 h-5" /> : <Volume2 className="w-5 h-5" />}
            </motion.button>

            {/* 音量控制 */}
            <div className="flex items-center space-x-2">
              <Volume2 className="w-4 h-4 text-gray-500 dark:text-gray-400" />
              <input
                type="range"
                min="0"
                max="1"
                step="0.1"
                value={isMuted ? 0 : volume}
                onChange={(e) => handleVolumeChange(parseFloat(e.target.value))}
                className="w-20 h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer"
              />
            </div>
          </div>

          {/* 状态指示器 */}
          <div className="flex items-center space-x-2">
            {isPlaying && (
              <div className="flex items-center space-x-1">
                <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
                <span className="text-xs text-green-600 dark:text-green-400">播放中</span>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default VoicePlayer;
