import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { 
  ArrowLeft, 
  Phone, 
  MapPin, 
  Clock, 
  Star,
  Trash2,
  MessageSquare,
  FileText,
  Play,
  Pause,
  Volume2
} from 'lucide-react'
import Button from '@/components/UI/Button'
import { showAlert } from '@/utils/notification'
import { 
  getVoicemailDetail, 
  markVoicemailAsRead,
  markVoicemailAsImportant,
  deleteVoicemail,
  transcribeVoicemail,
  summarizeVoicemail
} from '@/api/voicemail'
import type { Voicemail } from '@/types/voicemail'
import { clsx } from 'clsx'

const VoicemailDetail = () => {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [voicemail, setVoicemail] = useState<Voicemail | null>(null)
  const [loading, setLoading] = useState(false)
  const [playing, setPlaying] = useState(false)
  const [transcribing, setTranscribing] = useState(false)
  const [summarizing, setSummarizing] = useState(false)

  useEffect(() => {
    if (id) {
      loadVoicemail(parseInt(id))
    }
  }, [id])

  const loadVoicemail = async (voicemailId: number) => {
    try {
      setLoading(true)
      const res = await getVoicemailDetail(voicemailId)
      if (res.code === 200 && res.data) {
        setVoicemail(res.data)
        // 自动标记为已读
        if (!res.data.isRead) {
          await markVoicemailAsRead(voicemailId)
        }
      }
    } catch (error: any) {
      showAlert(error.msg || '加载失败', 'error')
      navigate('/voicemail')
    } finally {
      setLoading(false)
    }
  }

  const handleToggleImportant = async () => {
    if (!voicemail) return
    try {
      const res = await markVoicemailAsImportant(voicemail.id, !voicemail.isImportant)
      if (res.code === 200) {
        setVoicemail({ ...voicemail, isImportant: !voicemail.isImportant })
        showAlert(voicemail.isImportant ? '已取消重要标记' : '已标记为重要', 'success')
      }
    } catch (error: any) {
      showAlert(error.msg || '操作失败', 'error')
    }
  }

  const handleDelete = async () => {
    if (!voicemail || !confirm('确定要删除这条留言吗？')) return
    try {
      const res = await deleteVoicemail(voicemail.id)
      if (res.code === 200) {
        showAlert('删除成功', 'success')
        navigate('/voicemail')
      }
    } catch (error: any) {
      showAlert(error.msg || '删除失败', 'error')
    }
  }

  const handleTranscribe = async () => {
    if (!voicemail) return
    try {
      setTranscribing(true)
      const res = await transcribeVoicemail(voicemail.id)
      if (res.code === 200) {
        showAlert('转录任务已提交，请稍后刷新查看结果', 'success')
      }
    } catch (error: any) {
      showAlert(error.msg || '转录失败', 'error')
    } finally {
      setTranscribing(false)
    }
  }

  const handleSummarize = async () => {
    if (!voicemail) return
    try {
      setSummarizing(true)
      const res = await summarizeVoicemail(voicemail.id)
      if (res.code === 200) {
        showAlert('摘要生成任务已提交，请稍后刷新查看结果', 'success')
      }
    } catch (error: any) {
      showAlert(error.msg || '生成摘要失败', 'error')
    } finally {
      setSummarizing(false)
    }
  }

  const formatDuration = (seconds: number) => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }

  if (loading || !voicemail) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="inline-block w-8 h-8 border-4 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  return (
    <div className="p-6 max-w-4xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <Button
          variant="ghost"
          onClick={() => navigate('/voicemail')}
          leftIcon={<ArrowLeft className="w-4 h-4" />}
          className="mb-4"
        >
          返回列表
        </Button>

        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-3xl font-bold text-foreground mb-2">
              {voicemail.callerName || voicemail.callerNumber}
            </h1>
            <div className="flex items-center gap-4 text-sm text-muted-foreground">
              <span className="flex items-center gap-1">
                <Phone className="w-4 h-4" />
                {voicemail.callerNumber}
              </span>
              {voicemail.callerLocation && (
                <span className="flex items-center gap-1">
                  <MapPin className="w-4 h-4" />
                  {voicemail.callerLocation}
                </span>
              )}
              <span className="flex items-center gap-1">
                <Clock className="w-4 h-4" />
                {new Date(voicemail.createdAt).toLocaleString('zh-CN')}
              </span>
            </div>
          </div>

          <div className="flex gap-2">
            <Button
              variant="ghost"
              onClick={handleToggleImportant}
            >
              <Star 
                className={clsx(
                  'w-5 h-5',
                  voicemail.isImportant && 'fill-yellow-500 text-yellow-500'
                )}
              />
            </Button>
            <Button
              variant="ghost"
              onClick={handleDelete}
            >
              <Trash2 className="w-5 h-5" />
            </Button>
          </div>
        </div>
      </div>

      {/* Audio Player */}
      {voicemail.audioUrl && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="mb-6 p-6 bg-card border border-border rounded-lg"
        >
          <div className="flex items-center gap-4">
            <Button
              size="lg"
              variant="primary"
              onClick={() => setPlaying(!playing)}
            >
              {playing ? <Pause className="w-5 h-5" /> : <Play className="w-5 h-5" />}
            </Button>
            <div className="flex-1">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium text-foreground">录音播放</span>
                <span className="text-sm text-muted-foreground">
                  {formatDuration(voicemail.duration)}
                </span>
              </div>
              <div className="h-2 bg-muted rounded-full overflow-hidden">
                <div className="h-full bg-primary w-0" />
              </div>
            </div>
            <Volume2 className="w-5 h-5 text-muted-foreground" />
          </div>
          <audio 
            src={voicemail.audioUrl} 
            className="hidden"
            onPlay={() => setPlaying(true)}
            onPause={() => setPlaying(false)}
          />
        </motion.div>
      )}

      {/* Summary */}
      {voicemail.summary && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="mb-6 p-6 bg-primary/5 border border-primary/20 rounded-lg"
        >
          <div className="flex items-start gap-3">
            <FileText className="w-5 h-5 text-primary flex-shrink-0 mt-0.5" />
            <div className="flex-1">
              <h3 className="font-semibold text-foreground mb-2">留言摘要</h3>
              <p className="text-foreground">{voicemail.summary}</p>
              {voicemail.keywords && (
                <div className="flex flex-wrap gap-2 mt-3">
                  {voicemail.keywords.split(',').map((keyword, i) => (
                    <span 
                      key={i}
                      className="px-3 py-1 bg-primary/10 text-primary text-sm rounded-full"
                    >
                      {keyword.trim()}
                    </span>
                  ))}
                </div>
              )}
            </div>
          </div>
        </motion.div>
      )}

      {/* Transcript */}
      {voicemail.transcribedText ? (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
          className="mb-6 p-6 bg-card border border-border rounded-lg"
        >
          <div className="flex items-start gap-3">
            <MessageSquare className="w-5 h-5 text-muted-foreground flex-shrink-0 mt-0.5" />
            <div className="flex-1">
              <h3 className="font-semibold text-foreground mb-2">转录文本</h3>
              <p className="text-foreground whitespace-pre-wrap">{voicemail.transcribedText}</p>
            </div>
          </div>
        </motion.div>
      ) : (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
          className="mb-6 p-6 bg-card border border-border rounded-lg text-center"
        >
          <MessageSquare className="w-12 h-12 mx-auto text-muted-foreground mb-3" />
          <p className="text-muted-foreground mb-4">
            {voicemail.transcribeStatus === 'pending' 
              ? '转录任务等待处理中...'
              : voicemail.transcribeStatus === 'processing'
              ? '正在转录中...'
              : '暂无转录文本'}
          </p>
          {voicemail.transcribeStatus === 'pending' && (
            <Button
              onClick={handleTranscribe}
              loading={transcribing}
              leftIcon={<MessageSquare className="w-4 h-4" />}
            >
              开始转录
            </Button>
          )}
        </motion.div>
      )}

      {/* Actions */}
      {!voicemail.summary && voicemail.transcribedText && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
          className="text-center"
        >
          <Button
            onClick={handleSummarize}
            loading={summarizing}
            leftIcon={<FileText className="w-4 h-4" />}
          >
            生成摘要
          </Button>
        </motion.div>
      )}
    </div>
  )
}

export default VoicemailDetail
