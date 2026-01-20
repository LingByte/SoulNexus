import { useState, useEffect, useMemo } from 'react'
import { Target, TrendingUp, Clock, FileText } from 'lucide-react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/cn'
import Card from '../UI/Card'
import ProgressBar from '../Data/ProgressBar'

interface WordCounterProps {
  content: string
  targetWords?: number
  className?: string
  showStats?: boolean
  showProgress?: boolean
  showSpeed?: boolean
  onTargetReached?: () => void
}

interface WritingStats {
  words: number
  characters: number
  charactersNoSpaces: number
  paragraphs: number
  sentences: number
  readingTime: number
  writingSpeed: number
  progress: number
}

const WordCounter = ({
  content,
  targetWords = 1000,
  className,
  showStats = true,
  showProgress = true,
  showSpeed = false,
  onTargetReached
}: WordCounterProps) => {
  const [startTime, setStartTime] = useState<number | null>(null)
  const [lastWordCount, setLastWordCount] = useState(0)
  const [writingSpeed, setWritingSpeed] = useState(0)

  const stats = useMemo((): WritingStats => {
    const words = content.trim().split(/\s+/).filter(word => word.length > 0).length
    const characters = content.length
    const charactersNoSpaces = content.replace(/\s/g, '').length
    const paragraphs = content.split(/\n\s*\n/).filter(p => p.trim().length > 0).length
    const sentences = content.split(/[.!?]+/).filter(s => s.trim().length > 0).length
    const readingTime = Math.ceil(words / 200) // 假设每分钟阅读200字
    const progress = targetWords > 0 ? Math.min((words / targetWords) * 100, 100) : 0

    return {
      words,
      characters,
      charactersNoSpaces,
      paragraphs,
      sentences,
      readingTime,
      writingSpeed,
      progress
    }
  }, [content, targetWords, writingSpeed])

  // 计算写作速度
  useEffect(() => {
    if (startTime && stats.words > lastWordCount) {
      const timeElapsed = (Date.now() - startTime) / 1000 / 60 // 分钟
      const wordsWritten = stats.words - lastWordCount
      if (timeElapsed > 0) {
        setWritingSpeed(wordsWritten / timeElapsed)
      }
    }
    setLastWordCount(stats.words)
  }, [stats.words, startTime, lastWordCount])

  // 开始计时
  useEffect(() => {
    if (content.length > 0 && !startTime) {
      setStartTime(Date.now())
    }
  }, [content, startTime])

  // 目标达成提醒
  useEffect(() => {
    if (stats.words >= targetWords && onTargetReached) {
      onTargetReached()
    }
  }, [stats.words, targetWords, onTargetReached])

  const getProgressColor = (progress: number) => {
    if (progress < 30) return 'error'
    if (progress < 70) return 'warning'
    return 'success'
  }

  const getMotivationalMessage = (progress: number) => {
    if (progress < 10) return '开始你的创作之旅！'
    if (progress < 30) return '坚持下去，你做得很好！'
    if (progress < 50) return '已经完成一半了，加油！'
    if (progress < 70) return '接近目标了，继续努力！'
    if (progress < 90) return '马上就要完成了！'
    if (progress < 100) return '最后冲刺阶段！'
    return '恭喜你完成了目标！'
  }

  return (
    <div className={cn('space-y-4', className)}>
      {/* 主要统计 */}
      <Card className="p-4">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="text-center">
            <div className="flex items-center justify-center w-12 h-12 bg-primary/10 rounded-lg mb-2 mx-auto">
              <FileText className="w-6 h-6 text-primary" />
            </div>
            <div className="text-2xl font-bold text-neutral-900 dark:text-white">
              {stats.words.toLocaleString()}
            </div>
            <div className="text-sm text-neutral-600 dark:text-neutral-400">
              字数
            </div>
          </div>

          <div className="text-center">
            <div className="flex items-center justify-center w-12 h-12 bg-secondary/10 rounded-lg mb-2 mx-auto">
              <Target className="w-6 h-6 text-secondary" />
            </div>
            <div className="text-2xl font-bold text-neutral-900 dark:text-white">
              {targetWords.toLocaleString()}
            </div>
            <div className="text-sm text-neutral-600 dark:text-neutral-400">
              目标
            </div>
          </div>

          <div className="text-center">
            <div className="flex items-center justify-center w-12 h-12 bg-accent/10 rounded-lg mb-2 mx-auto">
              <Clock className="w-6 h-6 text-accent" />
            </div>
            <div className="text-2xl font-bold text-neutral-900 dark:text-white">
              {stats.readingTime}
            </div>
            <div className="text-sm text-neutral-600 dark:text-neutral-400">
              分钟阅读
            </div>
          </div>

          {showSpeed && (
            <div className="text-center">
              <div className="flex items-center justify-center w-12 h-12 bg-green-100 dark:bg-green-900/20 rounded-lg mb-2 mx-auto">
                <TrendingUp className="w-6 h-6 text-green-600 dark:text-green-400" />
              </div>
              <div className="text-2xl font-bold text-neutral-900 dark:text-white">
                {Math.round(writingSpeed)}
              </div>
              <div className="text-sm text-neutral-600 dark:text-neutral-400">
                字/分钟
              </div>
            </div>
          )}
        </div>
      </Card>

      {/* 进度条 */}
      {showProgress && (
        <Card className="p-4">
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold text-neutral-900 dark:text-white">
                写作进度
              </h3>
              <span className="text-sm text-neutral-600 dark:text-neutral-400">
                {stats.words} / {targetWords} 字
              </span>
            </div>

            <ProgressBar
              value={stats.progress}
              variant={getProgressColor(stats.progress) as any}
              showValue
            />

            <motion.p
              className="text-center text-sm text-neutral-600 dark:text-neutral-400"
              key={stats.progress}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
            >
              {getMotivationalMessage(stats.progress)}
            </motion.p>
          </div>
        </Card>
      )}

      {/* 详细统计 */}
      {showStats && (
        <Card className="p-4">
          <h3 className="text-lg font-semibold text-neutral-900 dark:text-white mb-4">
            详细统计
          </h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <div className="text-neutral-600 dark:text-neutral-400">字符数</div>
              <div className="font-medium">{stats.characters.toLocaleString()}</div>
            </div>
            <div>
              <div className="text-neutral-600 dark:text-neutral-400">不含空格</div>
              <div className="font-medium">{stats.charactersNoSpaces.toLocaleString()}</div>
            </div>
            <div>
              <div className="text-neutral-600 dark:text-neutral-400">段落数</div>
              <div className="font-medium">{stats.paragraphs}</div>
            </div>
            <div>
              <div className="text-neutral-600 dark:text-neutral-400">句子数</div>
              <div className="font-medium">{stats.sentences}</div>
            </div>
          </div>
        </Card>
      )}
    </div>
  )
}

export default WordCounter
