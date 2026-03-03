import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { Save, X, Plus, Trash2 } from 'lucide-react'
import Button from '@/components/UI/Button'
import Input from '@/components/UI/Input'
import { showAlert } from '@/utils/notification'
import { createScheme, updateScheme } from '@/api/scheme'
import { getAssistantList } from '@/api/assistant'
import type { Scheme, CreateSchemeRequest, KeywordReply } from '@/types/scheme'
import type { AssistantListItem } from '@/api/assistant'

interface SchemeFormProps {
  scheme?: Scheme | null
  onSuccess: () => void
  onCancel: () => void
}

const SchemeForm = ({ scheme, onSuccess, onCancel }: SchemeFormProps) => {
  const [loading, setLoading] = useState(false)
  const [assistants, setAssistants] = useState<AssistantListItem[]>([])
  const [countryCode, setCountryCode] = useState('+86') // 默认中国
  const [phoneNumber, setPhoneNumber] = useState('')
  const [formData, setFormData] = useState<CreateSchemeRequest>({
    schemeName: '',
    description: '',
    assistantId: undefined,
    autoAnswer: true,
    autoAnswerDelay: 0,
    openingMessage: '',
    keywordReplies: [],
    fallbackMessage: '',
    recordingEnabled: true,
    recordingMode: 'full',
    messageEnabled: true,
    messageDuration: 20,
    messagePrompt: '',
    boundPhoneNumber: ''
  })

  // 常用国家代码
  const countryCodes = [
    { code: '+86', name: '中国', flag: '🇨🇳' },
    { code: '+1', name: '美国/加拿大', flag: '🇺🇸' },
    { code: '+44', name: '英国', flag: '🇬🇧' },
    { code: '+81', name: '日本', flag: '🇯🇵' },
    { code: '+82', name: '韩国', flag: '🇰🇷' },
    { code: '+65', name: '新加坡', flag: '🇸🇬' },
    { code: '+852', name: '香港', flag: '🇭🇰' },
    { code: '+853', name: '澳门', flag: '🇲🇴' },
    { code: '+886', name: '台湾', flag: '🇹🇼' },
    { code: '+61', name: '澳大利亚', flag: '🇦🇺' },
  ]

  // 加载助手列表
  useEffect(() => {
    loadAssistants()
  }, [])

  const loadAssistants = async () => {
    try {
      const res = await getAssistantList()
      if (res.code === 200 && res.data) {
        setAssistants(res.data)
      }
    } catch (error) {
      console.error('加载助手列表失败:', error)
    }
  }

  useEffect(() => {
    if (scheme) {
      // 解析已有的号码，分离国家代码和号码
      let parsedCountryCode = '+86'
      let parsedPhoneNumber = ''
      
      if (scheme.boundPhoneNumber) {
        // 使用已知国家代码进行精确匹配（避免贪婪匹配问题）
        // 常见国家代码：+1(美国), +86(中国), +852(香港), +853(澳门), +886(台湾), +44(英国), +81(日本), +82(韩国) 等
        const knownCountryCodes = ['+1', '+7', '+20', '+27', '+30', '+31', '+32', '+33', '+34', '+36', '+39', '+40', '+41', '+43', '+44', '+45', '+46', '+47', '+48', '+49', '+51', '+52', '+53', '+54', '+55', '+56', '+57', '+58', '+60', '+61', '+62', '+63', '+64', '+65', '+66', '+81', '+82', '+84', '+86', '+90', '+91', '+92', '+93', '+94', '+95', '+98', '+212', '+213', '+216', '+218', '+220', '+221', '+222', '+223', '+224', '+225', '+226', '+227', '+228', '+229', '+230', '+231', '+232', '+233', '+234', '+235', '+236', '+237', '+238', '+239', '+240', '+241', '+242', '+243', '+244', '+245', '+246', '+248', '+249', '+250', '+251', '+252', '+253', '+254', '+255', '+256', '+257', '+258', '+260', '+261', '+262', '+263', '+264', '+265', '+266', '+267', '+268', '+269', '+290', '+291', '+297', '+298', '+299', '+350', '+351', '+352', '+353', '+354', '+355', '+356', '+357', '+358', '+359', '+370', '+371', '+372', '+373', '+374', '+375', '+376', '+377', '+378', '+380', '+381', '+382', '+383', '+385', '+386', '+387', '+389', '+420', '+421', '+423', '+500', '+501', '+502', '+503', '+504', '+505', '+506', '+507', '+508', '+509', '+590', '+591', '+592', '+593', '+594', '+595', '+596', '+597', '+598', '+599', '+670', '+672', '+673', '+674', '+675', '+676', '+677', '+678', '+679', '+680', '+681', '+682', '+683', '+685', '+686', '+687', '+688', '+689', '+690', '+691', '+692', '+850', '+852', '+853', '+855', '+856', '+880', '+886', '+960', '+961', '+962', '+963', '+964', '+965', '+966', '+967', '+968', '+970', '+971', '+972', '+973', '+974', '+975', '+976', '+977', '+992', '+993', '+994', '+995', '+996', '+998']
        
        let matched = false
        // 按长度从长到短排序，优先匹配更长的国家代码（如 +852 优先于 +86）
        const sortedCodes = knownCountryCodes.sort((a, b) => b.length - a.length)
        
        for (const code of sortedCodes) {
          if (scheme.boundPhoneNumber.startsWith(code)) {
            parsedCountryCode = code
            parsedPhoneNumber = scheme.boundPhoneNumber.substring(code.length)
            matched = true
            break
          }
        }
        
        if (!matched) {
          // 如果没有匹配到已知国家代码，尝试通用匹配
          const match = scheme.boundPhoneNumber.match(/^(\+\d{1,4})(\d+)$/)
          if (match) {
            parsedCountryCode = match[1]
            parsedPhoneNumber = match[2]
          } else {
            parsedPhoneNumber = scheme.boundPhoneNumber
          }
        }
      }
      
      setCountryCode(parsedCountryCode)
      setPhoneNumber(parsedPhoneNumber)
      
      setFormData({
        schemeName: scheme.schemeName,
        description: scheme.description || '',
        assistantId: scheme.assistantId,
        autoAnswer: scheme.autoAnswer,
        autoAnswerDelay: scheme.autoAnswerDelay || 0,
        openingMessage: scheme.openingMessage || '',
        keywordReplies: scheme.keywordReplies || [],
        fallbackMessage: scheme.fallbackMessage || '',
        recordingEnabled: scheme.recordingEnabled,
        recordingMode: scheme.recordingMode,
        messageEnabled: scheme.messageEnabled,
        messageDuration: scheme.messageDuration || 20,
        messagePrompt: scheme.messagePrompt || '',
        boundPhoneNumber: scheme.boundPhoneNumber || ''
      })
    }
  }, [scheme])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!formData.schemeName.trim()) {
      showAlert('请输入方案名称', 'warning')
      return
    }

    if (!formData.assistantId) {
      showAlert('请选择AI助手', 'warning')
      return
    }

    // 组合完整的电话号码（国家代码 + 号码）
    const fullPhoneNumber = phoneNumber ? `${countryCode}${phoneNumber}` : ''

    try {
      setLoading(true)
      const submitData = {
        ...formData,
        boundPhoneNumber: fullPhoneNumber
      }
      
      const res = scheme
        ? await updateScheme(scheme.id, submitData)
        : await createScheme(submitData)

      if (res.code === 200) {
        showAlert(scheme ? '更新成功' : '创建成功', 'success')
        onSuccess()
      } else {
        throw new Error(res.msg)
      }
    } catch (error: any) {
      showAlert(error.msg || '操作失败', 'error')
    } finally {
      setLoading(false)
    }
  }

  const addKeywordReply = () => {
    setFormData({
      ...formData,
      keywordReplies: [...(formData.keywordReplies || []), { keyword: '', reply: '' }]
    })
  }

  const removeKeywordReply = (index: number) => {
    const newReplies = [...(formData.keywordReplies || [])]
    newReplies.splice(index, 1)
    setFormData({ ...formData, keywordReplies: newReplies })
  }

  const updateKeywordReply = (index: number, field: 'keyword' | 'reply', value: string) => {
    const newReplies = [...(formData.keywordReplies || [])]
    newReplies[index] = { ...newReplies[index], [field]: value }
    setFormData({ ...formData, keywordReplies: newReplies })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Basic Info */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-foreground">基本信息</h3>
        
        <Input
          label="方案名称"
          placeholder="例如：工作模式、会议中、防骚扰"
          value={formData.schemeName}
          onChange={(e) => setFormData({ ...formData, schemeName: e.target.value })}
          required
        />

        <div>
          <label className="block text-sm font-medium text-foreground mb-1.5">
            方案描述
          </label>
          <textarea
            className="w-full px-3.5 py-2.5 border border-input rounded-md bg-background text-foreground placeholder:text-muted-foreground focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
            rows={2}
            placeholder="简单描述这个方案的用途"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
          />
        </div>
      </div>

      {/* AI Assistant Selection */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-foreground">AI助手</h3>
        
        <div>
          <label className="block text-sm font-medium text-foreground mb-1.5">
            选择AI助手 <span className="text-destructive">*</span>
          </label>
          <select
            className="w-full px-3.5 py-2.5 border border-input rounded-md bg-background text-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
            value={formData.assistantId || ''}
            onChange={(e) => setFormData({ ...formData, assistantId: e.target.value ? Number(e.target.value) : undefined })}
            required
          >
            <option value="">请选择助手</option>
            {assistants.map((assistant) => (
              <option key={assistant.id} value={assistant.id}>
                {assistant.name} {assistant.description && `- ${assistant.description}`}
              </option>
            ))}
          </select>
          <p className="mt-1.5 text-xs text-muted-foreground">
            助手的音色、大模型等配置在 Smart Assistant 中设置
          </p>
        </div>
      </div>

      {/* Auto Answer Configuration */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-foreground">自动接听</h3>
        
        <div className="flex items-center gap-3">
          <input
            type="checkbox"
            id="autoAnswer"
            className="w-4 h-4 rounded border-input text-primary focus:ring-2 focus:ring-ring"
            checked={formData.autoAnswer}
            onChange={(e) => setFormData({ ...formData, autoAnswer: e.target.checked })}
          />
          <label htmlFor="autoAnswer" className="text-sm font-medium text-foreground cursor-pointer">
            启用自动接听
          </label>
        </div>

        {formData.autoAnswer && (
          <Input
            label="接听延迟（秒）"
            type="number"
            min="0"
            placeholder="0"
            value={formData.autoAnswerDelay}
            onChange={(e) => setFormData({ ...formData, autoAnswerDelay: Number(e.target.value) })}
          />
        )}
      </div>

      {/* AI Response Configuration */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-foreground">AI回复配置</h3>
        
        <div>
          <label className="block text-sm font-medium text-foreground mb-1.5">
            开场白
          </label>
          <textarea
            className="w-full px-3.5 py-2.5 border border-input rounded-md bg-background text-foreground placeholder:text-muted-foreground focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
            rows={2}
            placeholder="例如：您好，我是XX的助理，请问有什么可以帮您？"
            value={formData.openingMessage}
            onChange={(e) => setFormData({ ...formData, openingMessage: e.target.value })}
          />
        </div>

        {/* Keyword Replies */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="text-sm font-medium text-foreground">
              关键词回复（可选）
            </label>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={addKeywordReply}
              leftIcon={<Plus className="w-3.5 h-3.5" />}
            >
              添加
            </Button>
          </div>
          
          {formData.keywordReplies && formData.keywordReplies.length > 0 && (
            <div className="space-y-3">
              {formData.keywordReplies.map((item, index) => (
                <motion.div
                  key={index}
                  initial={{ opacity: 0, y: -10 }}
                  animate={{ opacity: 1, y: 0 }}
                  className="flex gap-2 items-start p-3 bg-muted/50 rounded-md"
                >
                  <div className="flex-1 space-y-2">
                    <input
                      type="text"
                      className="w-full px-3 py-1.5 text-sm border border-input rounded-md bg-background text-foreground placeholder:text-muted-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
                      placeholder="关键词"
                      value={item.keyword}
                      onChange={(e) => updateKeywordReply(index, 'keyword', e.target.value)}
                    />
                    <input
                      type="text"
                      className="w-full px-3 py-1.5 text-sm border border-input rounded-md bg-background text-foreground placeholder:text-muted-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
                      placeholder="回复内容"
                      value={item.reply}
                      onChange={(e) => updateKeywordReply(index, 'reply', e.target.value)}
                    />
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    onClick={() => removeKeywordReply(index)}
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </Button>
                </motion.div>
              ))}
            </div>
          )}
        </div>

        <div>
          <label className="block text-sm font-medium text-foreground mb-1.5">
            兜底回复
          </label>
          <textarea
            className="w-full px-3.5 py-2.5 border border-input rounded-md bg-background text-foreground placeholder:text-muted-foreground focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
            rows={2}
            placeholder="当没有匹配到关键词时的默认回复"
            value={formData.fallbackMessage}
            onChange={(e) => setFormData({ ...formData, fallbackMessage: e.target.value })}
          />
        </div>
      </div>

      {/* Recording Configuration */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-foreground">录音设置</h3>
        
        <div className="flex items-center gap-3">
          <input
            type="checkbox"
            id="recordingEnabled"
            className="w-4 h-4 rounded border-input text-primary focus:ring-2 focus:ring-ring"
            checked={formData.recordingEnabled}
            onChange={(e) => setFormData({ ...formData, recordingEnabled: e.target.checked })}
          />
          <label htmlFor="recordingEnabled" className="text-sm font-medium text-foreground cursor-pointer">
            启用录音
          </label>
        </div>

        {formData.recordingEnabled && (
          <div>
            <label className="block text-sm font-medium text-foreground mb-2">
              录音模式
            </label>
            <div className="space-y-2">
              <label className="flex items-center gap-3 p-3 border border-input rounded-md cursor-pointer hover:bg-muted/50 transition-colors">
                <input
                  type="radio"
                  name="recordingMode"
                  value="full"
                  checked={formData.recordingMode === 'full'}
                  onChange={(e) => setFormData({ ...formData, recordingMode: e.target.value as 'full' | 'message' })}
                  className="w-4 h-4 text-primary focus:ring-2 focus:ring-ring"
                />
                <div>
                  <div className="text-sm font-medium text-foreground">全程录音</div>
                  <div className="text-xs text-muted-foreground">记录完整的通话内容</div>
                </div>
              </label>
              <label className="flex items-center gap-3 p-3 border border-input rounded-md cursor-pointer hover:bg-muted/50 transition-colors">
                <input
                  type="radio"
                  name="recordingMode"
                  value="message"
                  checked={formData.recordingMode === 'message'}
                  onChange={(e) => setFormData({ ...formData, recordingMode: e.target.value as 'full' | 'message' })}
                  className="w-4 h-4 text-primary focus:ring-2 focus:ring-ring"
                />
                <div>
                  <div className="text-sm font-medium text-foreground">仅留言录音</div>
                  <div className="text-xs text-muted-foreground">只记录留言阶段的内容</div>
                </div>
              </label>
            </div>
          </div>
        )}
      </div>

      {/* Voicemail Configuration */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-foreground">留言设置</h3>
        
        <div className="flex items-center gap-3">
          <input
            type="checkbox"
            id="messageEnabled"
            className="w-4 h-4 rounded border-input text-primary focus:ring-2 focus:ring-ring"
            checked={formData.messageEnabled}
            onChange={(e) => setFormData({ ...formData, messageEnabled: e.target.checked })}
          />
          <label htmlFor="messageEnabled" className="text-sm font-medium text-foreground cursor-pointer">
            启用留言功能
          </label>
        </div>

        {formData.messageEnabled && (
          <>
            <Input
              label="留言时长（秒）"
              type="number"
              min="5"
              max="120"
              placeholder="20"
              value={formData.messageDuration}
              onChange={(e) => setFormData({ ...formData, messageDuration: Number(e.target.value) })}
            />
            
            <div>
              <label className="block text-sm font-medium text-foreground mb-1.5">
                留言提示语
              </label>
              <textarea
                className="w-full px-3.5 py-2.5 border border-input rounded-md bg-background text-foreground placeholder:text-muted-foreground focus:ring-2 focus:ring-ring focus:border-transparent resize-none"
                rows={2}
                placeholder="例如：请在嘀声后留言"
                value={formData.messagePrompt}
                onChange={(e) => setFormData({ ...formData, messagePrompt: e.target.value })}
              />
            </div>
          </>
        )}
      </div>

      {/* Bound Phone Number */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-foreground">绑定号码</h3>
        
        <div>
          <label className="block text-sm font-medium text-foreground mb-1.5">
            手机号码（可选）
          </label>
          <div className="flex gap-2">
            <select
              className="px-3.5 py-2.5 border border-input rounded-md bg-background text-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
              value={countryCode}
              onChange={(e) => setCountryCode(e.target.value)}
            >
              {countryCodes.map((country) => (
                <option key={country.code} value={country.code}>
                  {country.flag} {country.code} {country.name}
                </option>
              ))}
            </select>
            <input
              type="tel"
              className="flex-1 px-3.5 py-2.5 border border-input rounded-md bg-background text-foreground placeholder:text-muted-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
              placeholder="13344444444"
              value={phoneNumber}
              onChange={(e) => setPhoneNumber(e.target.value.replace(/\D/g, ''))}
            />
          </div>
          {phoneNumber && (
            <p className="mt-1.5 text-xs text-muted-foreground">
              完整号码：<span className="font-mono font-medium">{countryCode}{phoneNumber}</span>
            </p>
          )}
          <p className="mt-1.5 text-xs text-muted-foreground">
            绑定手机号后，来电到该号码时将使用此方案自动接听
          </p>
        </div>
      </div>

      {/* Actions */}
      <div className="flex justify-end gap-3 pt-4 border-t border-border">
        <Button
          type="button"
          variant="ghost"
          onClick={onCancel}
          disabled={loading}
        >
          取消
        </Button>
        <Button
          type="submit"
          loading={loading}
          leftIcon={<Save className="w-4 h-4" />}
        >
          {scheme ? '保存' : '创建'}
        </Button>
      </div>
    </form>
  )
}

export default SchemeForm
