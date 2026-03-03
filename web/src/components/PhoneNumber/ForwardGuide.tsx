import { motion } from 'framer-motion'
import { Copy, Check } from 'lucide-react'
import { useState } from 'react'
import { FORWARD_GUIDES } from '@/types/phoneNumber'
import Button from '@/components/UI/Button'
import { showAlert } from '@/utils/notification'

const ForwardGuide = () => {
  const [copiedCode, setCopiedCode] = useState<string | null>(null)
  const platformNumber = '10086' // 这里应该从配置中获取平台接入号

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopiedCode(label)
      showAlert('已复制到剪贴板', 'success')
      setTimeout(() => setCopiedCode(null), 2000)
    })
  }

  return (
    <div className="space-y-6">
      {/* Platform Number */}
      <div className="p-4 bg-primary/10 border border-primary/20 rounded-lg">
        <div className="text-sm text-muted-foreground mb-1">平台接入号</div>
        <div className="flex items-center justify-between">
          <div className="text-2xl font-bold text-primary">{platformNumber}</div>
          <Button
            size="sm"
            variant="outline"
            onClick={() => copyToClipboard(platformNumber, 'platform')}
            leftIcon={copiedCode === 'platform' ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
          >
            {copiedCode === 'platform' ? '已复制' : '复制'}
          </Button>
        </div>
      </div>

      {/* Instructions */}
      <div className="space-y-4">
        <h3 className="font-semibold text-foreground">设置步骤</h3>
        <ol className="space-y-3 text-sm text-muted-foreground">
          <li className="flex gap-3">
            <span className="flex-shrink-0 w-6 h-6 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-xs font-medium">
              1
            </span>
            <span>复制上方的平台接入号</span>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 w-6 h-6 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-xs font-medium">
              2
            </span>
            <span>根据您的运营商，复制对应的呼叫转移代码</span>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 w-6 h-6 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-xs font-medium">
              3
            </span>
            <span>在手机拨号界面输入代码并拨打</span>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 w-6 h-6 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-xs font-medium">
              4
            </span>
            <span>听到提示音后，呼叫转移设置成功</span>
          </li>
        </ol>
      </div>

      {/* Carrier Guides */}
      <div className="space-y-4">
        <h3 className="font-semibold text-foreground">运营商代码</h3>
        {FORWARD_GUIDES.map((guide, index) => (
          <motion.div
            key={guide.carrier}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.1 }}
            className="p-4 border border-border rounded-lg space-y-3"
          >
            <div className="flex items-center justify-between">
              <h4 className="font-medium text-foreground">{guide.carrier}</h4>
              <span className="text-xs text-muted-foreground">{guide.description}</span>
            </div>

            <div className="space-y-2">
              {/* Enable Code */}
              <div className="flex items-center justify-between p-2 bg-muted/50 rounded">
                <div className="flex-1">
                  <div className="text-xs text-muted-foreground mb-0.5">开启转移</div>
                  <code className="text-sm font-mono text-foreground">
                    {guide.enableCode.replace('{number}', platformNumber)}
                  </code>
                </div>
                <Button
                  size="xs"
                  variant="ghost"
                  onClick={() => copyToClipboard(
                    guide.enableCode.replace('{number}', platformNumber),
                    `${guide.carrier}-enable`
                  )}
                >
                  {copiedCode === `${guide.carrier}-enable` ? <Check className="w-3 h-3" /> : <Copy className="w-3 h-3" />}
                </Button>
              </div>

              {/* Disable Code */}
              <div className="flex items-center justify-between p-2 bg-muted/50 rounded">
                <div className="flex-1">
                  <div className="text-xs text-muted-foreground mb-0.5">关闭转移</div>
                  <code className="text-sm font-mono text-foreground">{guide.disableCode}</code>
                </div>
                <Button
                  size="xs"
                  variant="ghost"
                  onClick={() => copyToClipboard(guide.disableCode, `${guide.carrier}-disable`)}
                >
                  {copiedCode === `${guide.carrier}-disable` ? <Check className="w-3 h-3" /> : <Copy className="w-3 h-3" />}
                </Button>
              </div>

              {/* Check Code */}
              <div className="flex items-center justify-between p-2 bg-muted/50 rounded">
                <div className="flex-1">
                  <div className="text-xs text-muted-foreground mb-0.5">查询状态</div>
                  <code className="text-sm font-mono text-foreground">{guide.checkCode}</code>
                </div>
                <Button
                  size="xs"
                  variant="ghost"
                  onClick={() => copyToClipboard(guide.checkCode, `${guide.carrier}-check`)}
                >
                  {copiedCode === `${guide.carrier}-check` ? <Check className="w-3 h-3" /> : <Copy className="w-3 h-3" />}
                </Button>
              </div>
            </div>
          </motion.div>
        ))}
      </div>

      {/* Warning */}
      <div className="p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
        <div className="text-sm text-yellow-800 dark:text-yellow-200">
          <strong>注意：</strong>
          <ul className="mt-2 space-y-1 list-disc list-inside">
            <li>呼叫转移可能会产生额外费用，请咨询运营商</li>
            <li>部分套餐可能不支持呼叫转移功能</li>
            <li>设置后请拨打测试号码验证是否生效</li>
          </ul>
        </div>
      </div>
    </div>
  )
}

export default ForwardGuide
