import React from 'react'
import Input from '@/components/UI/Input'
import Button from '@/components/UI/Button'

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
}) => {
  return (
    <div
      ref={textInputRef}
      className="border-t dark:border-neutral-700 p-6 bg-gradient-to-r from-purple-50 to-indigo-50 dark:from-purple-900/20 dark:to-indigo-900/20"
    >
      <div className="max-w-2xl mx-auto">
        <div className="flex items-center gap-3">
          {/* 文本模式选择框 */}
          {onTextModeChange && (
            <select
              value={textMode}
              onChange={(e) => onTextModeChange(e.target.value as TextMode)}
              disabled={isWaitingForResponse}
              className="w-32 px-3 py-2 text-sm rounded-lg border border-purple-200 dark:border-purple-800 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 shadow-lg focus:outline-none focus:ring-2 focus:ring-purple-300 dark:focus:ring-purple-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <option value="voice">语音输出</option>
              <option value="text">文本对话</option>
            </select>
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
      </div>
    </div>
  )
}

export default TextInputBox

