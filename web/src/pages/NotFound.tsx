import { motion } from 'framer-motion'
import { Home, ArrowLeft, Search } from 'lucide-react'
import { Link } from 'react-router-dom'
import Button from '../components/UI/Button'
import FadeIn from '../components/Animations/FadeIn'
import { useI18nStore } from '../stores/i18nStore'

const NotFound = () => {
  const { t } = useI18nStore()
  return (
      <div className="min-h-screen flex items-center justify-center px-4 py-12">
        <div className="max-w-2xl mx-auto text-center">
          <FadeIn direction="up">
            <motion.div
                initial={{ scale: 0.8, opacity: 0 }}
                animate={{ scale: 1, opacity: 1 }}
                transition={{ delay: 0.2, duration: 0.6 }}
                className="text-9xl font-bold text-primary mb-8"
            >
              404
            </motion.div>
          </FadeIn>

          <FadeIn direction="up" delay={0.3}>
            <motion.div
                initial={{ scale: 0.8, opacity: 0 }}
                animate={{ scale: 1, opacity: 1 }}
                transition={{ delay: 0.2, duration: 0.6 }}
                className="mb-8"
            >
              <h1 className="text-4xl md:text-5xl font-display font-bold">
                {t('notFound.title')}
              </h1>
            </motion.div>
          </FadeIn>

          <FadeIn direction="up" delay={0.4}>
            <p className="text-xl text-neutral-600 mb-8 leading-relaxed">
              {t('notFound.description')}
            </p>
          </FadeIn>

          <FadeIn direction="up" delay={0.5}>
            <div className="flex flex-col sm:flex-row gap-4 justify-center">
              <Link to="/">
                <Button variant="primary" size="lg" leftIcon={<Home className="w-5 h-5" />}>
                  {t('notFound.backHome')}
                </Button>
              </Link>
              <Button
                  variant="outline"
                  size="lg"
                  leftIcon={<ArrowLeft className="w-5 h-5" />}
                  onClick={() => window.history.back()}
              >
                {t('notFound.back')}
              </Button>
            </div>
          </FadeIn>

          <FadeIn direction="up" delay={0.6}>
            <div className="mt-12 p-6 rounded-xl">
              <div className="flex items-center justify-center mb-4">
                <Search className="w-6 h-6 mr-2" />
                <h3 className="text-lg font-semibold">{t('notFound.needHelp')}</h3>
              </div>
              <p className="mb-4">
                {t('notFound.helpDesc')}
              </p>
              <Button variant="ghost" size="sm">
                {t('notFound.contactSupport')}
              </Button>
            </div>
          </FadeIn>
        </div>
      </div>
  )
}

export default NotFound