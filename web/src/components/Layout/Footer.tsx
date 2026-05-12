import React from 'react'
import { Link } from 'react-router-dom'
import { useI18nStore } from '@/stores/i18nStore'
import { Github, Mail, FileText, Shield } from 'lucide-react'

const Footer: React.FC = () => {
  const { t } = useI18nStore()
  
  // 从环境变量读取配置
  const icpNumber = import.meta.env.VITE_ICP_NUMBER || ''
  const contactEmail = import.meta.env.VITE_CONTACT_EMAIL || ''
  const githubUrl = import.meta.env.VITE_GITHUB_URL || ''

  const currentYear = new Date().getFullYear()

  // 产品链接
  const productLinks = [
    { name: t('nav.sidebar.smartAssistant'), href: '/assistants' },
    { name: t('nav.sidebar.voiceTraining'), href: '/voice-training' },
    { name: t('nav.sidebar.workflow'), href: '/workflows' },
  ]

  // 资源链接
  const resourceLinks = [
    { name: t('nav.docs'), href: 'https://docs.lingecho.com/', icon: FileText, external: true },
  ]

  // 法律链接
  const legalLinks = [
    { name: t('footer.privacy'), href: '/privacy', icon: Shield },
    { name: t('footer.terms'), href: '/terms', icon: FileText },
  ]

  // 社交链接
  const socialLinks = [
    ...(githubUrl ? [{ name: 'GitHub', href: githubUrl, icon: Github }] : []),
    ...(contactEmail ? [{ name: t('footer.contact'), href: `mailto:${contactEmail}`, icon: Mail }] : []),
  ]

  return (
    <footer className="relative bg-gradient-to-b from-gray-50 to-white dark:from-gray-900 dark:to-gray-950 border-t border-gray-200 dark:border-gray-800">
      {/* 装饰性渐变 */}
      <div className="absolute inset-0 bg-gradient-to-r from-blue-50/50 via-purple-50/50 to-pink-50/50 dark:from-blue-950/20 dark:via-purple-950/20 dark:to-pink-950/20 pointer-events-none" />
      
      <div className="relative max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* 主要内容区 */}
        <div className="py-12 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
          {/* 品牌信息 */}
          <div className="space-y-4">
            <Link to="/" className="flex items-center gap-2 group">
              <img
                src="https://cetide-1325039295.cos.ap-chengdu.myqcloud.com/folder/icon-192x192.ico"
                alt={t('brand.name')}
                className="w-8 h-10 rounded transition-transform group-hover:scale-110"
              />
              <span className="text-xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                {t('brand.name')}
              </span>
            </Link>
            <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed">
              {t('footer.description')}
            </p>
            {/* 社交链接 */}
            {socialLinks.length > 0 && (
              <div className="flex items-center gap-3">
                {socialLinks.map((link) => {
                  const Icon = link.icon
                  return (
                    <a
                      key={link.name}
                      href={link.href}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="w-9 h-9 rounded-lg bg-gray-100 dark:bg-gray-800 flex items-center justify-center text-gray-600 dark:text-gray-400 hover:bg-blue-100 dark:hover:bg-blue-900/30 hover:text-blue-600 dark:hover:text-blue-400 transition-all duration-200"
                      title={link.name}
                    >
                      <Icon className="w-4 h-4" />
                    </a>
                  )
                })}
              </div>
            )}
          </div>

          {/* 产品 */}
          <div>
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4 uppercase tracking-wider">
              {t('footer.products')}
            </h3>
            <ul className="space-y-3">
              {productLinks.map((link) => (
                <li key={link.href}>
                  <Link
                    to={link.href}
                    className="text-sm text-gray-600 dark:text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors duration-200 flex items-center gap-2 group"
                  >
                    <span className="w-1 h-1 rounded-full bg-gray-400 dark:bg-gray-600 group-hover:bg-blue-600 dark:group-hover:bg-blue-400 transition-colors" />
                    {link.name}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          {/* 资源 */}
          <div>
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4 uppercase tracking-wider">
              {t('footer.resources')}
            </h3>
            <ul className="space-y-3">
              {resourceLinks.map((link) => {
                const Icon = link.icon
                return (
                  <li key={link.href}>
                    {link.external ? (
                      <a
                        href={link.href}
                        target="_blank"
                        rel="noreferrer"
                        className="text-sm text-gray-600 dark:text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors duration-200 flex items-center gap-2 group"
                      >
                        <Icon className="w-4 h-4 opacity-50 group-hover:opacity-100 transition-opacity" />
                        {link.name}
                      </a>
                    ) : (
                      <Link
                        to={link.href}
                        className="text-sm text-gray-600 dark:text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors duration-200 flex items-center gap-2 group"
                      >
                        <Icon className="w-4 h-4 opacity-50 group-hover:opacity-100 transition-opacity" />
                        {link.name}
                      </Link>
                    )}
                  </li>
                )
              })}
            </ul>
          </div>

          {/* 法律与支持 */}
          <div>
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4 uppercase tracking-wider">
              {t('footer.legal')}
            </h3>
            <ul className="space-y-3">
              {legalLinks.map((link) => {
                const Icon = link.icon
                return (
                  <li key={link.href}>
                    <Link
                      to={link.href}
                      className="text-sm text-gray-600 dark:text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors duration-200 flex items-center gap-2 group"
                    >
                      <Icon className="w-4 h-4 opacity-50 group-hover:opacity-100 transition-opacity" />
                      {link.name}
                    </Link>
                  </li>
                )
              })}
            </ul>
          </div>
        </div>

        {/* 分隔线 */}
        <div className="border-t border-gray-200 dark:border-gray-800" />

        {/* 底部信息 */}
        <div className="py-6 flex flex-col md:flex-row items-center justify-between gap-4">
          <div className="flex flex-col md:flex-row items-center gap-2 md:gap-4 text-xs text-gray-500 dark:text-gray-400">
            <span>© {currentYear} 成都解忧造物科技有限责任公司</span>
            {icpNumber && (
              <>
                <span className="hidden md:inline">•</span>
                <a
                  href="https://beian.miit.gov.cn/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="hover:text-blue-600 dark:hover:text-blue-400 transition-colors duration-200"
                >
                  {icpNumber}
                </a>
              </>
            )}
          </div>
          
          <div className="text-xs text-gray-500 dark:text-gray-400">
            {t('footer.madeWith')} <span className="text-red-500">♥</span> {t('footer.by')} {t('footer.team')}
          </div>
        </div>
      </div>
    </footer>
  )
}

export default Footer

