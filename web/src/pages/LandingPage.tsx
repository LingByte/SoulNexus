import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { BookOpen, Bot, Code, Cpu, GraduationCap, HeartPulse, Key, Mic, Settings, Sparkles, Zap } from 'lucide-react'
import { Button } from '@/components/ui'
import { PLATFORM_HOME_PATH, TENANT_HOME_PATH } from '@/constants/appPaths'
import { useAuthStore } from '@/stores/authStore'
import { useTranslation } from '@/i18n'
import ContentCarousel from '@/components/Home/ContentCarousel'
import SiteFooter from '@/components/Layout/SiteFooter'
import LandingHeader from '@/components/Home/LandingHeader'
import FeatureGridCard from '@/components/Home/FeatureGridCard'
import LandingSectionLink from '@/components/Home/LandingSectionLink'
import { landingSectionIdFromHash, scrollToSection } from '@/utils/scrollToSection'
import CursorGrid from '@/components/Effects/CursorGrid'

const SECTION_SCROLL = 'scroll-mt-[5.5rem]'

export default function LandingPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { user, isAuthenticated, token, logout } = useAuthStore()
  const [mobileOpen, setMobileOpen] = useState(false)
  const loggedIn = Boolean(isAuthenticated && token)

  const featureCards = useMemo(
    () => [
      { icon: Mic, title: t('landing.featureVoiceTitle'), desc: t('landing.featureVoiceDesc') },
      { icon: BookOpen, title: t('landing.featureKbTitle'), desc: t('landing.featureKbDesc') },
      { icon: Settings, title: t('landing.featureWorkflowTitle'), desc: t('landing.featureWorkflowDesc') },
      { icon: Key, title: t('landing.featureTenantTitle'), desc: t('landing.featureTenantDesc') },
      { icon: Zap, title: t('landing.featureVoiceprintTitle'), desc: t('landing.featureVoiceprintDesc') },
    ],
    [t],
  )

  const showcaseSlides = useMemo(
    () => [
      { image: '/images/workflow.png', alt: t('landing.slideWorkflow') },
      { image: '/images/voiceclone.png', alt: t('landing.slideVoiceClone') },
      { image: '/images/debug-assistant.png', alt: t('landing.slideDebug') },
    ],
    [t],
  )

  const knowledgeSlides = useMemo(
    () => [
      { image: '/images/knowledge.png', alt: t('landing.knowledgeTitle') },
      { image: '/images/workflow.png', alt: t('landing.slideWorkflow') },
      { image: '/images/debug-assistant.png', alt: t('landing.slideDebug') },
    ],
    [t],
  )

  const whoWeServe = useMemo(
    () => [
      { icon: Bot, title: t('landing.whoCs'), desc: t('landing.whoCsDesc') },
      { icon: Code, title: t('landing.whoDev'), desc: t('landing.whoDevDesc') },
      { icon: GraduationCap, title: t('landing.whoEdu'), desc: t('landing.whoEduDesc') },
      { icon: HeartPulse, title: t('landing.whoHealth'), desc: t('landing.whoHealthDesc') },
      { icon: Cpu, title: t('landing.whoHardware'), desc: t('landing.whoHardwareDesc') },
      { icon: Sparkles, title: t('landing.whoCreator'), desc: t('landing.whoCreatorDesc') },
    ],
    [t],
  )

  useEffect(() => {
    document.title = t('landing.metaTitle')
    const meta = document.querySelector('meta[name="description"]')
    meta?.setAttribute('content', t('landing.metaDescription'))
  }, [t])

  useEffect(() => {
    const id = landingSectionIdFromHash(window.location.hash)
    if (!id) return
    const tId = window.setTimeout(() => scrollToSection(id), 120)
    return () => window.clearTimeout(tId)
  }, [])

  const goConsole = () => {
    const isPlatform = Boolean(user?.isPlatformAdmin || user?.principal === 'platform')
    navigate(isPlatform ? PLATFORM_HOME_PATH : TENANT_HOME_PATH)
  }

  const handleLogout = () => {
    logout()
    setMobileOpen(false)
  }

  return (
    <div className="relative min-h-screen overflow-x-hidden bg-[hsl(var(--background))] text-[hsl(var(--foreground))]">
      <CursorGrid
        cellSize={70}
        color="#D946EF"
        radius={140}
        falloff="smooth"
        holdTime={400}
        fadeDuration={800}
        lineWidth={1.2}
        maxOpacity={1}
        fillOpacity={0}
        gridOpacity={0}
        cellRadius={0}
        clickPulse
        pulseSpeed={600}
        className="pointer-events-none fixed inset-0 -z-10 opacity-30"
      />

      <LandingHeader
        loggedIn={loggedIn}
        user={user}
        mobileOpen={mobileOpen}
        onToggleMobile={() => setMobileOpen((v) => !v)}
        onCloseMobile={() => setMobileOpen(false)}
        onConsole={goConsole}
        onLogout={handleLogout}
        onLogin={() => navigate('/login')}
      />

      <main>
        <section className={`relative flex min-h-[min(88vh,920px)] flex-col justify-center overflow-hidden py-24 text-center sm:py-32 ${SECTION_SCROLL}`} aria-label="Hero">
          <div className="absolute -left-24 top-10 h-64 w-64 rounded-full bg-violet-400/30 blur-3xl" aria-hidden />
          <div className="absolute -right-16 bottom-0 h-72 w-72 rounded-full bg-fuchsia-400/20 blur-3xl" aria-hidden />
          <motion.div initial={{ opacity: 0, y: 28 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.7 }} className="relative z-10 mx-auto max-w-4xl px-4">
            <p className="mb-4 text-sm font-medium tracking-[0.2em] text-violet-600 uppercase">{t('landing.heroBadge')}</p>
            <h1 className="font-display text-5xl font-bold tracking-tight text-violet-700 sm:text-7xl">{t('landing.heroTitle')}</h1>
            <p className="mx-auto mt-6 max-w-2xl text-base leading-relaxed text-[hsl(var(--muted-foreground))] sm:text-lg">{t('landing.heroSubtitle')}</p>
            <div className="mt-12 flex flex-wrap items-center justify-center gap-3">
              <Button variant="primary" size="lg" onClick={() => (loggedIn ? goConsole() : navigate('/login'))}>{loggedIn ? t('landing.ctaConsole') : t('landing.ctaLogin')}</Button>
              <LandingSectionLink sectionId="features" className="inline-flex h-12 items-center rounded-lg border border-[hsl(var(--border))] px-6 text-sm font-medium hover:bg-[hsl(var(--muted))]">{t('landing.ctaFeatures')}</LandingSectionLink>
            </div>
          </motion.div>
        </section>

        <section id="features" className={`mx-auto max-w-7xl px-4 py-24 sm:px-6 sm:py-32 ${SECTION_SCROLL}`}>
          <div className="mb-14 text-center">
            <h2 className="text-3xl font-bold tracking-tight sm:text-4xl">{t('landing.featuresTitle')}</h2>
            <p className="mx-auto mt-3 max-w-2xl text-[hsl(var(--muted-foreground))]">{t('landing.featuresSubtitle')}</p>
          </div>
          <div className="grid gap-7 sm:grid-cols-2 lg:grid-cols-3">
            {featureCards.map((f, i) => <FeatureGridCard key={f.title} icon={f.icon} title={f.title} description={f.desc} index={i} tall={false} />)}
          </div>
        </section>

        <section id="platform-showcase" className={`relative overflow-hidden py-24 sm:py-32 ${SECTION_SCROLL}`}>
          <div className="relative z-10 mx-auto max-w-7xl px-4 sm:px-6">
            <ContentCarousel subtitle={t('landing.showcaseSubtitle')} title={t('landing.showcaseTitle')} description={t('landing.showcaseDescription')} features={[t('landing.showcaseF1'), t('landing.showcaseF2'), t('landing.showcaseF3')]} carouselItems={showcaseSlides} ctaText={t('landing.showcaseCta')} ctaLink="https://docs.lingecho.com/" />
          </div>
        </section>

        <section id="who-we-serve" className={`mx-auto max-w-7xl px-4 py-24 sm:px-6 sm:py-32 ${SECTION_SCROLL}`}>
          <div className="mb-14 text-center">
            <h2 className="text-3xl font-bold tracking-tight sm:text-4xl">{t('landing.whoTitle')}</h2>
            <p className="mx-auto mt-3 max-w-2xl text-[hsl(var(--muted-foreground))]">{t('landing.whoSubtitle')}</p>
          </div>
          <div className="grid gap-7 sm:grid-cols-2 lg:grid-cols-3">
            {whoWeServe.map((f, i) => <FeatureGridCard key={f.title} icon={f.icon} title={f.title} description={f.desc} index={i} tall />)}
          </div>
        </section>

        <section id="knowledge" className={`relative overflow-hidden border-y border-[hsl(var(--border))] py-24 sm:py-32 ${SECTION_SCROLL}`}>
          <div className="mx-auto max-w-7xl px-4 sm:px-6">
            <ContentCarousel reverse subtitle={t('landing.knowledgeSubtitle')} title={t('landing.knowledgeTitle')} description={t('landing.knowledgeDescription')} features={[t('landing.knowledgeF1'), t('landing.knowledgeF2'), t('landing.knowledgeF3')]} carouselItems={knowledgeSlides} />
          </div>
        </section>
      </main>

      <SiteFooter />
    </div>
  )
}
