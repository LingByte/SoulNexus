import { motion } from 'framer-motion'
import { useState } from 'react'
import { 
  Target, 
  Eye, 
  Users, 
  Award, 
  CheckCircle, 
  Heart,
  Building2,
  Building,
  Mail,
  Globe,
  Github,
  Send
} from 'lucide-react'
import Card, { CardContent, CardDescription, CardHeader, CardTitle } from '../components/UI/Card'
import FadeIn from '../components/Animations/FadeIn'
import StaggeredList from '../components/Animations/StaggeredList'
import { useI18nStore } from '../stores/i18nStore'

const About = () => {
  const { t } = useI18nStore()
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    message: ''
  })
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isSubmitted, setIsSubmitted] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsSubmitting(true)
    
    // 模拟提交
    await new Promise(resolve => setTimeout(resolve, 1500))
    
    setIsSubmitting(false)
    setIsSubmitted(true)
    
    // 重置表单
    setTimeout(() => {
      setFormData({ name: '', email: '', message: '' })
      setIsSubmitted(false)
    }, 3000)
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value
    })
  }

  const values = [
    {
      icon: <Target className="w-8 h-8 text-indigo-500" />,
      title: t('about.values.tech.title'),
      description: t('about.values.tech.desc'),
    },
    {
      icon: <Eye className="w-8 h-8 text-purple-500" />,
      title: t('about.values.ux.title'),
      description: t('about.values.ux.desc'),
    },
    {
      icon: <Users className="w-8 h-8 text-pink-500" />,
      title: t('about.values.features.title'),
      description: t('about.values.features.desc'),
    },
    {
      icon: <Award className="w-8 h-8 text-blue-500" />,
      title: t('about.values.openSource.title'),
      description: t('about.values.openSource.desc'),
    },
  ]

  return (
      <div className="space-y-20">
        {/* Hero Section */}
        <section className="py-20 text-center">
          <FadeIn direction="up">
            <h1 className="text-5xl md:text-6xl font-display font-bold mb-6 text-foreground">
              {t('about.title')}
            </h1>
            <p className="text-xl text-muted-foreground max-w-3xl mx-auto leading-relaxed">
              {t('about.subtitle')}
            </p>
          </FadeIn>
        </section>

        {/* Mission Section */}
        <section className="py-8 bg-muted min-h-screen flex items-center">
          <div className="max-w-5xl mx-auto px-4 w-full">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 items-center">
              <FadeIn direction="left">
                <div>
                  <h2 className="text-4xl font-display font-bold mb-6 text-foreground">
                    {t('about.mission.title')}
                  </h2>
                  <p className="text-lg text-muted-foreground mb-6 leading-relaxed">
                    {t('about.mission.desc')}
                  </p>
                  <div className="space-y-4">
                    <div className="flex items-start space-x-3">
                      <CheckCircle className="w-6 h-6 text-primary mt-1 flex-shrink-0" />
                      <p className="text-foreground">
                        {t('about.mission.item1')}
                      </p>
                    </div>
                    <div className="flex items-start space-x-3">
                      <CheckCircle className="w-6 h-6 text-primary mt-1 flex-shrink-0" />
                      <p className="text-foreground">
                        {t('about.mission.item2')}
                      </p>
                    </div>
                    <div className="flex items-start space-x-3">
                      <CheckCircle className="w-6 h-6 text-primary mt-1 flex-shrink-0" />
                      <p className="text-foreground">
                        {t('about.mission.item3')}
                      </p>
                    </div>
                  </div>
                </div>
              </FadeIn>

              <FadeIn direction="right">
                <div className="relative">
                  <div className="absolute inset-0 bg-gradient-to-r from-primary/80 via-secondary/80 to-primary/80 rounded-3xl transform rotate-3"></div>
                  <div className="relative bg-background rounded-3xl p-8 shadow-2xl border">
                    <div className="text-center">
                      <div className="w-20 h-20 bg-accent from-primary via-secondary to-primary rounded-full flex items-center justify-center mx-auto mb-6">
                        <Heart className="w-10 h-10 text-foreground " />
                      </div>
                      <h3 className="text-2xl font-bold mb-4 text-foreground">{t('about.madeWithHeart')}</h3>
                      <p className="text-muted-foreground">
                        {t('about.madeWithHeartDesc')}
                      </p>
                    </div>
                  </div>
                </div>
              </FadeIn>
            </div>
          </div>
        </section>

        {/* Values Section */}
        <section className="py-20">
          <div className="max-w-5xl mx-auto px-4 w-full">
            <FadeIn direction="up" className="text-center mb-6">
              <h2 className="text-4xl font-display font-bold mb-6 text-foreground">
                {t('about.values.title')}
              </h2>
              <p className="text-xl text-muted-foreground max-w-3xl mx-auto">
                {t('about.values.desc')}
              </p>
            </FadeIn>

            <StaggeredList className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
              {values.map((value) => (
                  <motion.div
                      key={value.title}
                      whileHover={{ y: -5 }}
                      transition={{ duration: 0.2 }}
                  >
                    <Card hover className="h-full text-center border shadow-lg hover:shadow-xl transition-all duration-300">
                      <CardHeader>
                        <div className="w-16 h-16 bg-muted rounded-2xl flex items-center justify-center mx-auto mb-4">
                          {value.icon}
                        </div>
                        <CardTitle className="text-xl text-foreground">{value.title}</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <CardDescription className="text-base leading-relaxed text-muted-foreground">
                          {value.description}
                        </CardDescription>
                      </CardContent>
                    </Card>
                  </motion.div>
              ))}
            </StaggeredList>
          </div>
        </section>

        {/* Contact Section */}
        <section className="py-12 bg-muted min-h-[calc(100vh-4rem)]">
          <div className="max-w-5xl mx-auto px-4 h-full flex flex-col justify-center">
            <FadeIn direction="up" className="text-center mb-8">
              <h2 className="text-3xl font-display font-bold mb-3 text-foreground">
                {t('contact.title')}
              </h2>
              <p className="text-base text-muted-foreground">
                {t('contact.subtitle')}
              </p>
            </FadeIn>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Contact Form */}
              <motion.div
                initial={{ opacity: 0, x: -30 }}
                whileInView={{ opacity: 1, x: 0 }}
                viewport={{ once: false, margin: "-100px" }}
                transition={{ duration: 0.6 }}
              >
                <Card className="border shadow-xl">
                  <CardHeader>
                    <CardTitle className="text-2xl text-foreground">{t('contact.form.title')}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <form onSubmit={handleSubmit} className="space-y-6">
                      <div>
                        <label htmlFor="name" className="block text-sm font-medium text-muted-foreground mb-2">
                          {t('contact.form.name')}
                        </label>
                        <input
                          type="text"
                          id="name"
                          name="name"
                          value={formData.name}
                          onChange={handleChange}
                          required
                          className="w-full px-4 py-3 rounded-lg border border-input bg-background text-foreground focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                          placeholder={t('contact.form.namePlaceholder')}
                        />
                      </div>

                      <div>
                        <label htmlFor="email" className="block text-sm font-medium text-muted-foreground mb-2">
                          {t('contact.form.email')}
                        </label>
                        <input
                          type="email"
                          id="email"
                          name="email"
                          value={formData.email}
                          onChange={handleChange}
                          required
                          className="w-full px-4 py-3 rounded-lg border border-input bg-background text-foreground focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                          placeholder={t('contact.form.emailPlaceholder')}
                        />
                      </div>

                      <div>
                        <label htmlFor="message" className="block text-sm font-medium text-muted-foreground mb-2">
                          {t('contact.form.message')}
                        </label>
                        <textarea
                          id="message"
                          name="message"
                          value={formData.message}
                          onChange={handleChange}
                          required
                          rows={6}
                          className="w-full px-4 py-3 rounded-lg border border-input bg-background text-foreground focus:ring-2 focus:ring-primary focus:border-transparent transition-all resize-none"
                          placeholder={t('contact.form.messagePlaceholder')}
                        />
                      </div>

                      <button
                        type="submit"
                        disabled={isSubmitting || isSubmitted}
                        className="w-full px-6 py-3 rounded-lg font-semibold shadow-lg bg-gradient-to-r from-indigo-500 via-purple-500 to-blue-500 hover:from-indigo-600 hover:via-purple-600 hover:to-blue-600 transition-all duration-300 hover:scale-105 hover:shadow-xl disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100 flex items-center justify-center gap-2"
                      >
                        {isSubmitted ? (
                          <>
                            <CheckCircle className="w-5 h-5" />
                            {t('contact.form.sent')}
                          </>
                        ) : isSubmitting ? (
                          <>
                            <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin" />
                            {t('contact.form.sending')}
                          </>
                        ) : (
                          <>
                            <Send className="w-5 h-5" />
                            {t('contact.form.send')}
                          </>
                        )}
                      </button>
                    </form>
                  </CardContent>
                </Card>
              </motion.div>

              {/* Contact Info */}
              <motion.div
                initial={{ opacity: 0, x: 30 }}
                whileInView={{ opacity: 1, x: 0 }}
                viewport={{ once: false, margin: "-100px" }}
                transition={{ duration: 0.6 }}
                className="space-y-6"
              >
                <Card className="border shadow-lg">
                  <CardHeader>
                    <CardTitle className="text-xl text-foreground">{t('contact.info.title')}</CardTitle>
                    <CardDescription>{t('contact.info.description')}</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    {/* Email */}
                    <div className="flex items-start gap-4 p-4 bg-muted rounded-lg hover:bg-accent transition-colors">
                      <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center flex-shrink-0">
                        <Mail className="w-5 h-5" />
                      </div>
                      <div>
                        <h3 className="font-semibold text-foreground mb-1">{t('contact.info.email')}</h3>
                        <a href="mailto:support@lingecho.com" className="text-primary hover:underline">
                          support@lingecho.com
                        </a>
                      </div>
                    </div>

                    {/* Website */}
                    <div className="flex items-start gap-4 p-4 bg-muted rounded-lg hover:bg-accent transition-colors">
                      <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-purple-500 to-pink-600 flex items-center justify-center flex-shrink-0">
                        <Globe className="w-5 h-5" />
                      </div>
                      <div>
                        <h3 className="font-semibold text-foreground mb-1">{t('contact.info.website')}</h3>
                        <a href="https://lingecho.com" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
                          lingecho.com
                        </a>
                      </div>
                    </div>

                    {/* GitHub */}
                    <div className="flex items-start gap-4 p-4 bg-muted rounded-lg hover:bg-accent transition-colors">
                      <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-blue-500 to-cyan-600 flex items-center justify-center flex-shrink-0">
                        <Github className="w-5 h-5" />
                      </div>
                      <div>
                        <h3 className="font-semibold text-foreground mb-1">{t('contact.info.github')}</h3>
                        <a href="https://github.com/LingByte/SoulNexus" target="_blank" rel="noopener noreferrer" className="text-primary hover:underline break-all">
                          github.com/LingByte/SoulNexus
                        </a>
                      </div>
                    </div>
                  </CardContent>
                </Card>

                {/* Company Info */}
                <Card className="border shadow-lg">
                  <CardHeader>
                    <div className="w-12 h-12 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center mb-4">
                      <Building2 className="w-6 h-6" />
                    </div>
                    <CardTitle className="text-xl text-foreground">{t('contact.company.title')}</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div className="flex items-start gap-3">
                      <Building className="w-5 h-5 text-primary mt-1 flex-shrink-0" />
                      <div>
                        <p className="font-medium text-foreground">{t('contact.company.name')}</p>
                        <p className="text-sm text-muted-foreground">{t('contact.company.nameEn')}</p>
                      </div>
                    </div>
                    <div className="flex items-start gap-3">
                      <Target className="w-5 h-5 text-primary mt-1 flex-shrink-0" />
                      <div>
                        <p className="font-medium text-foreground">{t('contact.company.address')}</p>
                        <p className="text-sm text-muted-foreground">{t('contact.company.addressDetail')}</p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </motion.div>
            </div>
          </div>
        </section>
      </div>
  )
}

export default About
