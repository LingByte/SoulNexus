import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import {
  BookOpen,
  Bot,
  Code,
  GraduationCap,
  HeartPulse,
  Key,
  Mic,
  Settings,
  Sparkles,
  Zap,
  Cpu,
} from 'lucide-react'
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
      { image: '/images/js-template.png', alt: t('landing.slideJsTemplate') },
      { image: '/images/device-log.png', alt: t('landing.slideDevice') },
    ],
    [t],
  )

  const knowledgeSlides = useMemo(
    () => [
      { image: '/images/knowledge.png', alt: t('landing.knowledgeTitle') },
      { image: '/images/debug-assistant.png', alt: t('landing.slideDebug') },
      { image: '/images/workflow.png', alt: t('landing.slideWorkflow') },
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

  const coreMatrix = useMemo(
    () => [
      {
        icon: Zap,
        title: t('landing.coreVoiceTitle'),
        desc: t('landing.coreVoiceDesc'),
        tags: [t('landing.coreVoiceTag1'), t('landing.coreVoiceTag2'), t('landing.coreVoiceTag3'), t('landing.coreVoiceTag4')],
      },
      {
        icon: Mic,
        title: t('landing.coreCloneTitle'),
        desc: t('landing.coreCloneDesc'),
        tags: [t('landing.coreCloneTag1'), t('landing.coreCloneTag2'), t('landing.coreCloneTag3'), t('landing.coreCloneTag4')],
      },
      {
        icon: Settings,
        title: t('landing.coreWidgetTitle'),
        desc: t('landing.coreWidgetDesc'),
        tags: [t('landing.coreWidgetTag1'), t('landing.coreWidgetTag2'), t('landing.coreWidgetTag3'), t('landing.coreWidgetTag4')],
      },
      {
        icon: Sparkles,
        title: t('landing.coreFlowTitle'),
        desc: t('landing.coreFlowDesc'),
        tags: [t('landing.coreFlowTag1'), t('landing.coreFlowTag2'), t('landing.coreFlowTag3'), t('landing.coreFlowTag4')],
      },
      {
        icon: Key,
        title: t('landing.coreKeyTitle'),
        desc: t('landing.coreKeyDesc'),
        tags: [t('landing.coreKeyTag1'), t('landing.coreKeyTag2'), t('landing.coreKeyTag3'), t('landing.coreKeyTag4')],
      },
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

  const primaryCta = () => (loggedIn ? goConsole() : navigate('/login'))

  return (
    <div className="relative min-h-screen overflow-x-hidden bg-[hsl(var(--background))] text-[hsl(var(--foreground))]">
      <div
        className="pointer-events-none fixed inset-0 -z-10 bg-gradient-to-br from-violet-100 via-indigo-50 to-purple-100 dark:from-violet-950/40 dark:via-gray-950 dark:to-purple-950/30"
        aria-hidden
      />
      <div
        className="pointer-events-none fixed inset-0 -z-10 opacity-40 [background-image:linear-gradient(to_right,rgba(139,92,246,0.08)_1px,transparent_1px),linear-gradient(to_bottom,rgba(139,92,246,0.08)_1px,transparent_1px)] [background-size:48px_48px]"
        aria-hidden
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
        <section
          className={`relative flex min-h-[min(88vh,920px)] flex-col justify-center overflow-hidden py-24 text-center sm:py-32 ${SECTION_SCROLL}`}
          aria-label="Hero"
        >
          <div className="absolute -left-24 top-10 h-64 w-64 rounded-full bg-violet-400/30 blur-3xl" aria-hidden />
          <div className="absolute -right-16 bottom-0 h-72 w-72 rounded-full bg-fuchsia-400/20 blur-3xl" aria-hidden />

          <motion.div
            initial={{ opacity: 0, y: 28 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.7 }}
            className="relative z-10 mx-auto max-w-4xl px-4"
          >
            <p className="mb-4 text-sm font-medium tracking-[0.2em] text-violet-600 uppercase dark:text-violet-300">
              {t('landing.heroBadge')}
            </p>
            <h1 className="font-display text-5xl font-bold tracking-tight text-violet-700 sm:text-7xl dark:text-violet-200">
              {t('landing.heroTitle')}
            </h1>
            <p className="mx-auto mt-6 max-w-2xl text-base leading-relaxed text-[hsl(var(--muted-foreground))] sm:text-lg">
              {t('landing.heroSubtitle')}
            </p>
            <div className="mt-12 flex flex-wrap items-center justify-center gap-3">
              <Button variant="primary" size="lg" onClick={primaryCta}>
                {loggedIn ? t('landing.ctaConsole') : t('landing.ctaLogin')}
              </Button>
              <LandingSectionLink
                sectionId="features"
                className="inline-flex h-12 items-center rounded-lg border border-[hsl(var(--border))] px-6 text-sm font-medium hover:bg-[hsl(var(--muted))]"
              >
                {t('landing.ctaFeatures')}
              </LandingSectionLink>
            </div>

            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.35, duration: 0.6 }}
              className="mx-auto mt-14 max-w-3xl rounded-2xl border border-violet-200/50 bg-[hsl(var(--card)/0.75)] p-6 text-left backdrop-blur dark:border-violet-800/40"
            >
              <p className="text-sm leading-relaxed text-[hsl(var(--foreground)/0.9)] sm:text-base">{t('landing.heroMission')}</p>
            </motion.div>
          </motion.div>
        </section>

        <section id="platform-showcase" className={`relative overflow-hidden py-24 sm:py-32 ${SECTION_SCROLL}`}>
          <div className="absolute inset-0 bg-gradient-to-br from-violet-50/80 via-transparent to-purple-50/80 dark:from-violet-950/20 dark:to-purple-950/20" aria-hidden />
          <div className="relative z-10 mx-auto max-w-7xl px-4 sm:px-6">
            <ContentCarousel
              subtitle={t('landing.showcaseSubtitle')}
              title={t('landing.showcaseTitle')}
              description={t('landing.showcaseDescription')}
              features={[
                t('landing.showcaseF1'),
                t('landing.showcaseF2'),
                t('landing.showcaseF3'),
                t('landing.showcaseF4'),
              ]}
              carouselItems={showcaseSlides}
              ctaText={t('landing.showcaseCta')}
              ctaLink="https://docs.lingecho.com/"
            />
          </div>
        </section>

        <section id="features" className={`mx-auto max-w-7xl px-4 py-24 sm:px-6 sm:py-32 ${SECTION_SCROLL}`}>
          <div className="mb-14 text-center">
            <h2 className="text-3xl font-bold tracking-tight sm:text-4xl">{t('landing.featuresTitle')}</h2>
            <p className="mx-auto mt-3 max-w-2xl text-[hsl(var(--muted-foreground))]">{t('landing.featuresSubtitle')}</p>
          </div>
          <div className="grid gap-7 sm:grid-cols-2 lg:grid-cols-3">
            {featureCards.map((f, i) => (
              <FeatureGridCard key={f.title} icon={f.icon} title={f.title} description={f.desc} index={i} tall={false} />
            ))}
          </div>
        </section>

        <section id="knowledge" className={`relative overflow-hidden border-y border-[hsl(var(--border))] py-24 sm:py-32 ${SECTION_SCROLL}`}>
          <div className="mx-auto max-w-7xl px-4 sm:px-6">
            <ContentCarousel
              reverse
              subtitle={t('landing.knowledgeSubtitle')}
              title={t('landing.knowledgeTitle')}
              description={t('landing.knowledgeDescription')}
              features={[
                t('landing.knowledgeF1'),
                t('landing.knowledgeF2'),
                t('landing.knowledgeF3'),
                t('landing.knowledgeF4'),
              ]}
              carouselItems={knowledgeSlides}
            />
          </div>
        </section>

        <section id="who-we-serve" className={`relative py-24 sm:py-32 ${SECTION_SCROLL}`}>
          <div className="mx-auto max-w-6xl px-4 sm:px-6">
            <motion.div
              initial={{ opacity: 0, y: 16 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              className="mb-12 text-center"
            >
              <h2 className="text-3xl font-bold sm:text-4xl">{t('landing.whoTitle')}</h2>
              <p className="mx-auto mt-3 max-w-3xl text-[hsl(var(--muted-foreground))]">{t('landing.whoSubtitle')}</p>
            </motion.div>
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {whoWeServe.map((item, i) => (
                <motion.div
                  key={item.title}
                  initial={{ opacity: 0, y: 20 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  viewport={{ once: true }}
                  transition={{ delay: i * 0.06 }}
                  className="rounded-2xl border border-[hsl(var(--border))] bg-[hsl(var(--card)/0.85)] p-5 backdrop-blur transition hover:border-violet-400/50 hover:shadow-lg hover:shadow-violet-500/10"
                >
                  <div className="flex gap-4">
                    <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-violet-500 to-purple-600 text-white">
                      <item.icon className="h-6 w-6" />
                    </div>
                    <div>
                      <h3 className="font-semibold">{item.title}</h3>
                      <p className="mt-2 text-sm leading-relaxed text-[hsl(var(--muted-foreground))]">{item.desc}</p>
                    </div>
                  </div>
                </motion.div>
              ))}
            </div>
            <p className="mx-auto mt-10 max-w-2xl text-center text-sm text-[hsl(var(--muted-foreground))]">
              {t('landing.whoCta')}
            </p>
          </div>
        </section>

        <section id="highlights" className={`relative overflow-hidden py-24 sm:py-32 ${SECTION_SCROLL}`}>
          <div className="absolute inset-0 bg-gradient-to-br from-purple-50/90 via-transparent to-indigo-50/90 dark:from-purple-950/30 dark:to-indigo-950/20" aria-hidden />
          <div className="relative mx-auto max-w-6xl px-4 sm:px-6">
            <div className="mb-12 text-center">
              <h2 className="text-3xl font-bold sm:text-4xl">{t('landing.highlightsTitle')}</h2>
              <p className="mx-auto mt-3 max-w-3xl text-[hsl(var(--muted-foreground))]">{t('landing.highlightsSubtitle')}</p>
            </div>
            <div className="grid gap-6 md:grid-cols-3">
              {[
                { title: t('landing.perfTitle'), desc: t('landing.perfDesc'), icon: Zap },
                { title: t('landing.securityTitle'), desc: t('landing.securityDesc'), icon: Key },
                { title: t('landing.archTitle'), desc: t('landing.archDesc'), icon: Code },
              ].map((item, i) => (
                <motion.div
                  key={item.title}
                  initial={{ opacity: 0, y: 20 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  viewport={{ once: true }}
                  transition={{ delay: i * 0.08 }}
                  className="rounded-2xl border border-violet-200/50 bg-[hsl(var(--card)/0.9)] p-7 shadow-md dark:border-violet-800/40"
                >
                  <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-violet-500 to-purple-600 text-white">
                    <item.icon className="h-6 w-6" />
                  </div>
                  <h3 className="text-xl font-bold">{item.title}</h3>
                  <p className="mt-3 text-sm leading-relaxed text-[hsl(var(--muted-foreground))]">{item.desc}</p>
                </motion.div>
              ))}
            </div>
            <div className="mt-12 grid grid-cols-2 gap-6 md:grid-cols-4">
              {[
                ['99.9%', t('landing.statUptime')],
                ['<600ms', t('landing.statLatency')],
                ['1K+', t('landing.statConcurrent')],
                ['100%', t('landing.statOpen')],
              ].map(([value, label]) => (
                <div key={label} className="text-center">
                  <div className="text-2xl font-bold text-violet-600 dark:text-violet-300 sm:text-3xl">{value}</div>
                  <div className="mt-1 text-sm text-[hsl(var(--muted-foreground))]">{label}</div>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section id="more" className={`mx-auto max-w-7xl px-4 py-24 sm:px-6 sm:py-32 ${SECTION_SCROLL}`}>
          <div className="mb-14 text-center">
            <h2 className="text-3xl font-bold sm:text-4xl">{t('landing.coreTitle')}</h2>
            <p className="mx-auto mt-3 max-w-2xl text-[hsl(var(--muted-foreground))]">{t('landing.coreDesc')}</p>
          </div>
          <div className="grid gap-7 sm:grid-cols-2 lg:grid-cols-3">
            {coreMatrix.map((card, idx) => (
              <FeatureGridCard
                key={card.title}
                icon={card.icon}
                title={card.title}
                description={card.desc}
                tags={card.tags}
                index={idx}
                tall
              />
            ))}
          </div>
        </section>

        {loggedIn ? (
          <section className="border-y border-[hsl(var(--border))] bg-gradient-to-r from-violet-600 via-purple-600 to-indigo-600 py-14 text-white">
            <div className="mx-auto flex max-w-6xl flex-col items-center justify-between gap-6 px-4 text-center sm:flex-row sm:text-left sm:px-6">
              <div>
                <h2 className="text-2xl font-bold">{t('landing.ctaTitle')}</h2>
                <p className="mt-2 text-white/85">{t('landing.ctaBody')}</p>
              </div>
              <Button size="lg" className="!bg-white !text-violet-700 hover:!bg-violet-50" onClick={goConsole}>
                {t('landing.ctaConsole')}
              </Button>
            </div>
          </section>
        ) : null}
      </main>

      <SiteFooter />
    </div>
  )
}
