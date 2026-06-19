import React from 'react'
import { Input as ArcoInput, Select as ArcoSelect, Button as ArcoButton, Spin } from '@arco-design/web-react'
import { Paperclip, X } from 'lucide-react'

export type TextMode = 'voice' | 'text'

interface TextInputBoxProps {
  inputValue: string
  onInputChange: (value: string) => void
  isWaitingForResponse: boolean
  onEnter: (e: React.KeyboardEvent<HTMLInputElement>) => void
  onSend: () => void
  textMode?: TextMode
  onTextModeChange?: (mode: TextMode) => void
  inputRef?: React.RefObject<HTMLInputElement>
  textInputRef?: React.RefObject<HTMLDivElement>
  attachmentName?: string
  isParsingAttachment?: boolean
  onAttachmentSelect?: (file: File) => void
  onAttachmentClear?: () => void
}

const TextInputBox: React.FC<TextInputBoxProps> = ({
  inputValue,
  onInputChange,
  isWaitingForResponse,
  onEnter,
  onSend,
  textMode = 'voice',
  onTextModeChange,
  inputRef: _inputRef,
  textInputRef,
  attachmentName = '',
  isParsingAttachment = false,
  onAttachmentSelect,
  onAttachmentClear,
}) => {
  return (
    <div
      ref={textInputRef}
      className="border-t dark:border-neutral-700 px-4 py-3 bg-gradient-to-r from-purple-50 to-indigo-50 dark:from-purple-900/20 dark:to-indigo-900/20"
    >
      <div className="max-w-2xl mx-auto">
        <div className="flex items-center gap-3">
          {onAttachmentSelect && (
            <div className="relative">
              <input
                id="voice-attachment-input"
                type="file"
                className="hidden"
                onChange={(e) => {
                  const file = e.target.files?.[0]
                  if (file) onAttachmentSelect(file)
                  e.currentTarget.value = ''
                }}
              />
              <ArcoButton
                type="outline"
                disabled={isWaitingForResponse || isParsingAttachment}
                onClick={() => {
                  const el = document.getElementById('voice-attachment-input') as HTMLInputElement | null
                  el?.click()
                }}
              >
                <Paperclip className="w-4 h-4" />
              </ArcoButton>
            </div>
          )}
          {/* 文本模式选择框 */}
          {onTextModeChange && (
            <ArcoSelect
              value={textMode}
              onChange={(v) => onTextModeChange(v as TextMode)}
              disabled={isWaitingForResponse}
              size="small"
              style={{ width: 72, flexShrink: 0 }}
              options={[
                { label: '语音', value: 'voice' },
                { label: '文本', value: 'text' }
              ]}
            />
          )}
          {isWaitingForResponse ? (
            <div className="flex-1 flex items-center gap-2 px-3 h-10 border border-gray-200 dark:border-neutral-600 rounded-lg bg-gray-50 dark:bg-neutral-800">
              <Spin size={14} />
              <span className="text-sm text-gray-400">AI 正在思考中...</span>
            </div>
          ) : (
            <ArcoInput
              size="large"
              className="!h-10 !text-base ![&::placeholder]:text-base"
              value={inputValue}
              onChange={(val) => onInputChange(val)}
              placeholder={textMode === 'text' ? "输入文本进行文本对话..." : "输入文本直接发送"}
              onKeyDown={onEnter}
            />
          )}
          <ArcoButton
            type="primary"
            loading={isWaitingForResponse}
            disabled={isWaitingForResponse}
            onClick={onSend}
            style={{ background: 'linear-gradient(to right, #7c3aed, #4f46e5)', border: 'none' }}
          >
            {isWaitingForResponse ? "处理中..." : "发送"}
          </ArcoButton>
        </div>
        {(attachmentName || isParsingAttachment) && (
          <div className="mt-2 flex items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
            <span>{isParsingAttachment ? '附件解析中...' : `已附加：${attachmentName}`}</span>
            {attachmentName && onAttachmentClear && (
              <button type="button" onClick={onAttachmentClear} className="text-red-500 hover:text-red-600">
                <X className="w-3 h-3" />
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

export default TextInputBox

