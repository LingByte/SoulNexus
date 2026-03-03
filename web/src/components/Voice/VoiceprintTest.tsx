import { useState } from 'react'
import { Mic, Play } from 'lucide-react'
import Button from '@/components/UI/Button'
import Card, { CardContent, CardHeader, CardTitle } from '@/components/UI/Card'
import AudioFileUpload from '@/components/UI/AudioFileUpload'
import { showAlert } from '@/utils/notification'
import { identifyVoiceprint } from '@/api/voiceprint'

interface VoiceprintTestProps {
  assistantId: string
  assistantName: string
}

const VoiceprintTest = ({ assistantId, assistantName }: VoiceprintTestProps) => {
  const [audioFile, setAudioFile] = useState<File | null>(null)
  const [isIdentifying, setIsIdentifying] = useState(false)
  const [identifyResult, setIdentifyResult] = useState<any>(null)

  const handleIdentify = async () => {
    if (!audioFile) {
      showAlert('请先选择音频文件', 'warning')
      return
    }

    setIsIdentifying(true)
    try {
      const response = await identifyVoiceprint(assistantId, audioFile)
      if (response.code === 200) {
        setIdentifyResult(response.data)
        if (response.data.is_match) {
          showAlert(`识别到说话人: ${response.data.speaker_id}`, 'success', '识别成功')
        } else {
          showAlert('未找到匹配的声纹', 'info', '识别结果')
        }
      } else {
        throw new Error(response.msg || '识别失败')
      }
    } catch (error: any) {
      showAlert(error?.msg || error?.message || '声纹识别失败', 'error', '识别失败')
    } finally {
      setIsIdentifying(false)
    }
  }

  const getConfidenceColor = (confidence: string) => {
    switch (confidence) {
      case 'very_high': return 'text-green-600 dark:text-green-400'
      case 'high': return 'text-blue-600 dark:text-blue-400'
      case 'medium': return 'text-yellow-600 dark:text-yellow-400'
      case 'low': return 'text-orange-600 dark:text-orange-400'
      case 'very_low': return 'text-red-600 dark:text-red-400'
      default: return 'text-gray-600 dark:text-gray-400'
    }
  }

  const getConfidenceText = (confidence: string) => {
    switch (confidence) {
      case 'very_high': return '非常高'
      case 'high': return '高'
      case 'medium': return '中等'
      case 'low': return '低'
      case 'very_low': return '非常低'
      default: return '未知'
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Mic className="w-5 h-5" />
          声纹识别测试 - {assistantName}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
            选择测试音频
          </label>
          <AudioFileUpload
            onFileSelect={setAudioFile}
            placeholder="选择WAV音频文件进行声纹识别测试"
            maxSize={50}
          />
        </div>

        <div className="flex justify-center">
          <Button
            variant="primary"
            onClick={handleIdentify}
            loading={isIdentifying}
            disabled={!audioFile}
            leftIcon={<Play className="w-4 h-4" />}
          >
            {isIdentifying ? '识别中...' : '开始识别'}
          </Button>
        </div>

        {identifyResult && (
          <div className="mt-6 p-4 bg-gray-50 dark:bg-gray-800/50 rounded-lg">
            <h4 className="font-medium text-gray-900 dark:text-white mb-3">识别结果</h4>
            <div className="space-y-2">
              <div className="flex justify-between items-center">
                <span className="text-sm text-gray-600 dark:text-gray-400">说话人ID:</span>
                <span className="font-medium text-gray-900 dark:text-white">
                  {identifyResult.speaker_id || '未识别'}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm text-gray-600 dark:text-gray-400">相似度分数:</span>
                <span className="font-medium text-gray-900 dark:text-white">
                  {(identifyResult.score * 100).toFixed(2)}%
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm text-gray-600 dark:text-gray-400">置信度:</span>
                <span className={`font-medium ${getConfidenceColor(identifyResult.confidence)}`}>
                  {getConfidenceText(identifyResult.confidence)}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm text-gray-600 dark:text-gray-400">匹配状态:</span>
                <span className={`font-medium ${identifyResult.is_match ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}>
                  {identifyResult.is_match ? '匹配成功' : '未匹配'}
                </span>
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default VoiceprintTest