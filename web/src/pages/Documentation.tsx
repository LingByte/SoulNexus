import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { 
  BookOpen, 
  Code, 
  Zap, 
  Users, 
  Settings, 
  Download,
  Github,
  ExternalLink,
  Star,
  Heart,
  GitBranch,
  ChevronRight
} from 'lucide-react'
import Button from '@/components/UI/Button'
import Badge from '@/components/UI/Badge'
import DocumentRenderer from '@/components/Documentation/DocumentRenderer'
import { useI18nStore } from '@/stores/i18nStore'

interface DocumentationData {
  project: {
    name: string
    version: string
    description: string
    github: string
    license: string
  }
  sections: Array<{
    id: string
    title: string
    icon: string
    description: string
    content: any[]
  }>
}

const Documentation = () => {
  const { t } = useI18nStore()
  const [activeSection, setActiveSection] = useState('getting-started')
  const [data, setData] = useState<DocumentationData | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // 加载文档数据
    import('@/data/documentation.json')
      .then((docData) => {
        setData(docData.default)
        setLoading(false)
      })
      .catch((error) => {
        console.error('Failed to load documentation data:', error)
        setLoading(false)
      })
  }, [])

  const getIcon = (iconName: string) => {
    const icons: { [key: string]: any } = {
      BookOpen,
      Code,
      Zap,
      Users,
      Settings,
      Download,
      Github,
      Star,
      Heart,
      GitBranch
    }
    return icons[iconName] || BookOpen
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <div className="w-8 h-8 border-4 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-muted-foreground">{t('docs.loading')}</p>
        </div>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <BookOpen className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
          <h2 className="text-xl font-semibold mb-2">{t('docs.loadFailed')}</h2>
          <p className="text-muted-foreground">{t('docs.loadFailedDesc')}</p>
        </div>
      </div>
    )
  }

  const currentSection = data.sections.find(s => s.id === activeSection)

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <div className="border-b bg-background/95 backdrop-blur">
        <div className="max-w-7xl mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 bg-primary rounded-lg flex items-center justify-center">
                <BookOpen className="w-6 h-6 text-primary-foreground" />
              </div>
              <div>
                <h1 className="text-2xl font-bold text-foreground">{data.project.name} {t('docs.documentation')}</h1>
                <p className="text-muted-foreground">{data.project.description}</p>
              </div>
            </div>
            
            <div className="flex items-center gap-4">
              <Badge className="bg-green-100 text-green-800">
                <Star className="w-3 h-3 mr-1" />
                v{data.project.version}
              </Badge>
              <Badge variant="outline">
                <Heart className="w-3 h-3 mr-1" />
                {data.project.license} {t('docs.openSource')}
              </Badge>
              <a 
                href={data.project.github} 
                target="_blank" 
                rel="noopener noreferrer"
                className="flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors"
              >
                <Github className="w-4 h-4" />
                <span className="text-sm">GitHub</span>
                <ExternalLink className="w-3 h-3" />
              </a>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="flex gap-8">
          {/* Sidebar Navigation */}
          <aside className="w-64 flex-shrink-0">
            <nav className="space-y-2 sticky top-8">
              {data.sections.map((section) => {
                const Icon = getIcon(section.icon)
                return (
                  <button
                    key={section.id}
                    onClick={() => setActiveSection(section.id)}
                    className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg text-left transition-colors ${
                      activeSection === section.id
                        ? 'bg-accent text-accent-foreground'
                        : 'text-muted-foreground hover:text-foreground hover:bg-accent'
                    }`}
                  >
                    <Icon className="w-5 h-5" />
                    <div className="flex-1">
                      <div className="font-medium">{section.title}</div>
                      <div className="text-xs opacity-75">{section.description}</div>
                    </div>
                    {activeSection === section.id && (
                      <ChevronRight className="w-4 h-4" />
                    )}
                  </button>
                )
              })}
            </nav>
          </aside>

          {/* Main Content */}
          <main className="flex-1 min-w-0">
            <motion.div
              key={activeSection}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
            >
              <div className="mb-8">
                <div className="flex items-center gap-3 mb-4">
                  <div className="w-10 h-10 bg-primary/10 rounded-lg flex items-center justify-center">
                    {React.createElement(getIcon(currentSection?.icon || 'BookOpen'), { className: "w-5 h-5 text-primary" })}
                  </div>
                  <div>
                    <h2 className="text-3xl font-bold text-foreground">{currentSection?.title}</h2>
                    <p className="text-lg text-muted-foreground">{currentSection?.description}</p>
                  </div>
                </div>
                <div className="h-px bg-border"></div>
              </div>
              
              <div className="prose prose-gray max-w-none">
                {currentSection?.content && (
                  <DocumentRenderer content={currentSection.content} />
                )}
              </div>
            </motion.div>
          </main>
        </div>

        {/* Footer */}
        <div className="mt-16 pt-8 border-t border-border">
          <div className="flex items-center justify-between">
            <div className="text-sm text-muted-foreground">
              <p>{t('docs.copyright').replace('{name}', data.project.name).replace('{license}', data.project.license)}</p>
            </div>
            <div className="flex items-center gap-4">
              <Button variant="outline" size="sm">
                <Download className="w-4 h-4 mr-2" />
                {t('docs.downloadSource')}
              </Button>
              <Button size="sm">
                <Github className="w-4 h-4 mr-2" />
                {t('docs.contribute')}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Documentation