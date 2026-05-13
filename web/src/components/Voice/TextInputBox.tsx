import React from 'react'
import Input from '@/components/UI/Input'
import Button from '@/components/UI/Button'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '@/components/UI/Select'
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
  inputRef,
  textInputRef,
  attachmentName = '',
  isParsingAttachment = false,
  onAttachmentSelect,
  onAttachmentClear,
}) => {
  return (
    <div
      ref={textInputRef}
      className="border-t dark:border-neutral-700 p-6 bg-gradient-to-r from-purple-50 to-indigo-50 dark:from-purple-900/20 dark:to-indigo-900/20"
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
              <Button
                variant="outline"
                size="md"
                disabled={isWaitingForResponse || isParsingAttachment}
                onClick={() => {
                  const el = document.getElementById('voice-attachment-input') as HTMLInputElement | null
                  el?.click()
                }}
                className="px-3"
              >
                <Paperclip className="w-4 h-4" />
              </Button>
            </div>
          )}
          {/* 文本模式选择框 */}
          {onTextModeChange && (
            <Select
              value={textMode}
              onValueChange={(v) => onTextModeChange(v as TextMode)}
              disabled={isWaitingForResponse}
              className="w-36 shrink-0"
            >
              <SelectTrigger className="h-10 w-full border-purple-200 bg-white text-sm shadow-lg dark:border-purple-800 dark:bg-gray-800">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="voice">语音输出</SelectItem>
                <SelectItem value="text">文本对话</SelectItem>
              </SelectContent>
            </Select>
          )}
          <Input
            ref={inputRef}
            value={inputValue}
            onChange={(e) => onInputChange(e.target.value)}
            placeholder={isWaitingForResponse ? "正在处理中..." : textMode === 'text' ? "输入文本进行文本对话..." : "输入文本直接发送"}
            size="md"
            disabled={isWaitingForResponse}
            className="shadow-lg border-purple-200 dark:border-purple-800 focus:ring-purple-300 dark:focus:ring-purple-700 flex-1"
            onKeyDown={onEnter}
          />
          <Button
            variant="primary"
            size="md"
            disabled={isWaitingForResponse}
            onClick={onSend}
            className="shadow-lg hover:shadow-xl hover:scale-105 active:scale-95 transition-all duration-200 px-6 bg-gradient-to-r from-purple-600 to-indigo-600 hover:from-purple-700 hover:to-indigo-700"
            animation="scale"
          >
            {isWaitingForResponse ? "处理中..." : "发送"}
          </Button>
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

