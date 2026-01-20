import React from 'react'
import { useI18nStore } from '@/stores/i18nStore'

const Footer: React.FC = () => {
  const { t } = useI18nStore()
  
  // 从环境变量读取ICP备案号
  const icpNumber = import.meta.env.VITE_ICP_NUMBER || ''

  // 如果没有设置ICP备案号，不显示Footer
  if (!icpNumber) {
    return null
  }

  // 工信部备案查询网站（可以带上备案号参数进行查询）
  const icpUrl = `https://beian.miit.gov.cn/`

  return (
    <footer className="w-full py-4 px-6 border-t border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 mt-auto">
      <div className="flex flex-col items-center justify-center gap-2">
        {/* 团队信息 */}
        <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
          <span>{t('footer.team')}：</span>
          <span className="font-medium">{t('footer.teamDesc')}</span>
        </div>
        
        {/* ICP备案号 */}
        <div className="flex items-center justify-center">
          <a
            href={icpUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors flex items-center gap-1"
          >
            <span>{t('footer.icp')}：</span>
            <span className="underline">{icpNumber}</span>
          </a>
        </div>
      </div>
    </footer>
  )
}

export default Footer

