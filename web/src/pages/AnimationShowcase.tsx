import { useState } from 'react'
import { motion } from 'framer-motion'
import { Rocket, Lightbulb, Target, Palette } from 'lucide-react'
import Card from '../components/UI/Card'
import Button from '../components/UI/Button'
import {
  WaterRipple,
  RealWaterRipple,
  SimpleWaterRipple,
  PageTransition,
  OnboardingGuide,
  FixedOnboarding,
  MagneticButton,
  GlitchEffect,
  Typewriter
} from '../components/Animations/AdvancedAnimations'
import EnhancedMagneticButton from '../components/UI/EnhancedMagneticButton'
import { showAlertWithScroll } from '../utils/notification'
import EnhancedParticleEffect from "@/components/Animations/EnhancedParticleEffect.tsx";
import EnhancedPageTransition from "@/components/Animations/EnhancedPageTransition.tsx";
import {ScrollReveal} from "@/components/UX/MicroInteractions.tsx";
import { useI18nStore } from '../stores/i18nStore'

const AnimationShowcase = () => {
  const { t } = useI18nStore()
  const [showOnboarding, setShowOnboarding] = useState(false)
  const [showFixedOnboarding, setShowFixedOnboarding] = useState(false)
  const [showParticles, setShowParticles] = useState(false)

  const onboardingSteps = [
    {
      id: 'welcome',
      title: t('animation.onboarding.welcome.title'),
      description: t('animation.onboarding.welcome.description'),
      target: '.welcome-card'
    },
    {
      id: 'features',
      title: t('animation.onboarding.features.title'),
      description: t('animation.onboarding.features.description'),
      target: '.features-card'
    },
    {
      id: 'get-started',
      title: t('animation.onboarding.getStarted.title'),
      description: t('animation.onboarding.getStarted.description'),
      target: '.get-started-card'
    }
  ]

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100 p-8">
      <div className="max-w-7xl mx-auto space-y-12">
        {/* 页面标题 */}
        <motion.div
          className="text-center"
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
        >
          <h1 className="text-4xl font-bold mb-4">
            <Typewriter
              text={t('animation.title')}
              speed={100}
              className="text-transparent bg-clip-text bg-gradient-to-r from-blue-600 to-purple-600"
            />
          </h1>
          <p className="text-lg text-gray-600 dark:text-gray-400">
            {t('animation.subtitle')}
          </p>
        </motion.div>

        {/* 真实水波涟漪效果 */}
        <ScrollReveal direction="up" delay={0.2}>
          <Card
            title={t('animation.waterRipple.title')}
            subtitle={t('animation.waterRipple.subtitle')}
            className="welcome-card"
          >
            <div className="space-y-6">
              {/* 简单水波纹效果 */}
              <div>
                <h4 className="text-lg font-semibold mb-3">{t('animation.waterRipple.simple')}</h4>
                <SimpleWaterRipple
                  intensity="high"
                  color="rgba(59, 130, 246, 0.4)"
                  className="p-8 bg-gradient-to-br from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 rounded-lg"
                >
                  <div className="text-center">
                    <h3 className="text-2xl font-bold mb-2">{t('animation.waterRipple.clickMe')}</h3>
                    <p className="text-gray-600 dark:text-gray-400">
                      {t('animation.waterRipple.simpleDescription')}
                    </p>
                  </div>
                </SimpleWaterRipple>
              </div>

              {/* 真实水波纹效果 */}
              <div>
                <h4 className="text-lg font-semibold mb-3">{t('animation.waterRipple.real')}</h4>
                <RealWaterRipple
                  intensity="high"
                  color="rgba(59, 130, 246, 0.4)"
                  className="p-8 bg-gradient-to-br from-green-50 to-blue-50 dark:from-green-900/20 dark:to-blue-900/20 rounded-lg"
                >
                  <div className="text-center">
                    <h3 className="text-2xl font-bold mb-2">{t('animation.waterRipple.clickMe')}</h3>
                    <p className="text-gray-600 dark:text-gray-400">
                      {t('animation.waterRipple.realDescription')}
                    </p>
                  </div>
                </RealWaterRipple>
              </div>
            </div>
          </Card>
        </ScrollReveal>

        {/* 磁性按钮 */}
        <ScrollReveal direction="left" delay={0.4}>
          <Card
            title={t('animation.magnetic.title')}
            subtitle={t('animation.magnetic.subtitle')}
            className="features-card"
          >
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
              <MagneticButton
                variant="primary"
                size="lg"
                intensity={0.5}
                className="flex flex-col items-center gap-3 p-8 h-auto min-h-[140px]"
              >
                <Rocket className="w-10 h-10 text-blue-600" />
                <div className="text-center">
                  <div className="font-semibold text-lg">{t('animation.magnetic.button1')}</div>
                  <div className="text-sm opacity-80 mt-1">{t('animation.magnetic.button1Desc')}</div>
                </div>
              </MagneticButton>
              <MagneticButton
                variant="outline"
                size="lg"
                intensity={0.3}
                className="flex flex-col items-center gap-3 p-8 h-auto min-h-[140px]"
              >
                <div className="text-4xl">⚡</div>
                <div className="text-center">
                  <div className="font-semibold text-lg">{t('animation.magnetic.button2')}</div>
                  <div className="text-sm opacity-80 mt-1">{t('animation.magnetic.button2Desc')}</div>
                </div>
              </MagneticButton>
              <MagneticButton
                variant="ghost"
                size="lg"
                intensity={0.4}
                className="flex flex-col items-center gap-3 p-8 h-auto min-h-[140px]"
              >
                <div className="text-4xl">✨</div>
                <div className="text-center">
                  <div className="font-semibold text-lg">{t('animation.magnetic.button3')}</div>
                  <div className="text-sm opacity-80 mt-1">{t('animation.magnetic.button3Desc')}</div>
                </div>
              </MagneticButton>
            </div>
          </Card>
        </ScrollReveal>

        {/* 故障效果 */}
        <ScrollReveal direction="right" delay={0.6}>
          <Card
            title={t('animation.glitch.title')}
            subtitle={t('animation.glitch.subtitle')}
          >
            <div className="flex flex-wrap gap-4 justify-center">
              <GlitchEffect
                trigger="hover"
                intensity="medium"
                className="text-4xl font-bold text-blue-600"
              >
                {t('animation.glitch.hoverMe')}
              </GlitchEffect>
              <GlitchEffect
                trigger="click"
                intensity="high"
                className="text-4xl font-bold text-purple-600 cursor-pointer"
              >
                {t('animation.glitch.clickMe')}
              </GlitchEffect>
            </div>
          </Card>
        </ScrollReveal>

        {/* 粒子效果 */}
        <ScrollReveal direction="up" delay={0.8}>
          <Card
            title={t('animation.particle.title')}
            subtitle={t('animation.particle.subtitle')}
            className="get-started-card"
          >
            <div className="relative h-64 bg-gradient-to-br from-gray-900 to-blue-900 rounded-lg overflow-hidden">
              {showParticles && (
                <EnhancedParticleEffect
                  count={80}
                  colors={['#3b82f6', '#8b5cf6', '#06b6d4', '#10b981', '#f59e0b']}
                  interactive={true}
                  enableAudio={true}
                  intensity="high"
                  shape="star"
                  className="absolute inset-0"
                />
              )}
              <div className="absolute inset-0 flex items-center justify-center">
                <Button
                  onClick={() => setShowParticles(!showParticles)}
                  variant="primary"
                  size="lg"
                  className="z-10"
                >
                  {showParticles ? t('animation.particle.hide') : t('animation.particle.show')}
                </Button>
              </div>
            </div>
          </Card>
        </ScrollReveal>

        {/* 页面过渡效果 */}
        <ScrollReveal direction="down" delay={1.0}>
          <Card
            title={t('animation.pageTransition.title')}
            subtitle={t('animation.pageTransition.subtitle')}
          >
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              {['fade', 'slide', 'scale', 'flip'].map((variant) => (
                <PageTransition
                  key={variant}
                  variant={variant as any}
                  className="p-4 bg-gray-100 dark:bg-gray-800 rounded-lg text-center"
                >
                  <div className="text-sm font-medium capitalize">
                    {t(`animation.pageTransition.${variant}`)}
                  </div>
                </PageTransition>
              ))}
            </div>
          </Card>
        </ScrollReveal>

        {/* 引导动画 */}
        <ScrollReveal direction="up" delay={1.2}>
          <Card
            title={t('animation.onboarding.title')}
            subtitle={t('animation.onboarding.subtitle')}
          >
            <div className="text-center space-y-4">
              <div className="mb-6 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                <p className="text-sm text-blue-700 dark:text-blue-300 flex items-center gap-2">
                  <Lightbulb className="w-4 h-4" />
                  {t('animation.onboarding.hint')}
                </p>
              </div>
              <div className="flex flex-col sm:flex-row gap-4 justify-center">
                <Button
                  onClick={() => setShowOnboarding(true)}
                  variant="primary"
                  size="lg"
                  className="px-8 py-4"
                >
                  {t('animation.onboarding.basicButton')}
                </Button>
                <Button
                  onClick={() => setShowFixedOnboarding(true)}
                  variant="outline"
                  size="lg"
                  className="px-8 py-4"
                >
                  {t('animation.onboarding.fixedButton')}
                </Button>
              </div>
            </div>
          </Card>
        </ScrollReveal>

        {/* 滚动显示效果 */}
        <ScrollReveal direction="up" delay={1.4}>
          <Card
            title="滚动显示效果"
            subtitle="元素进入视口时的动画效果"
          >
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              {[
                { direction: 'up', text: '从下方滑入' },
                { direction: 'left', text: '从左侧滑入' },
                { direction: 'right', text: '从右侧滑入' }
              ].map((item, index) => (
                <ScrollReveal
                  key={item.direction}
                  direction={item.direction as any}
                  delay={index * 0.2}
                  className="p-6 bg-gradient-to-br from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 rounded-lg text-center"
                >
                  <div className="text-lg font-semibold">{item.text}</div>
                </ScrollReveal>
              ))}
            </div>
          </Card>
        </ScrollReveal>

        {/* 增强效果展示 */}
        <ScrollReveal direction="up" delay={1.6}>
          <Card
            title="增强效果展示"
            subtitle="全新的增强动画和交互效果"
          >
            <div className="space-y-8">
              {/* 增强磁吸按钮 */}
              <div className="text-center">
                <h4 className="text-lg font-semibold mb-4">增强磁吸按钮</h4>
                <div className="flex flex-wrap gap-4 justify-center">
                  <EnhancedMagneticButton
                    variant="primary"
                    size="lg"
                    intensity={0.4}
                    enableAudio={true}
                    enableRipple={true}
                    enableGlow={true}
                  >
                    ✨ 增强磁吸
                  </EnhancedMagneticButton>
                  <EnhancedMagneticButton
                    variant="outline"
                    size="lg"
                    intensity={0.3}
                    enableAudio={true}
                    enableRipple={true}
                    enableGlow={true}
                  >
                    <Target className="w-4 h-4 mr-2" />
                    3D 效果
                  </EnhancedMagneticButton>
                </div>
              </div>

              {/* 增强页面过渡 */}
              <div className="text-center">
                <h4 className="text-lg font-semibold mb-4">增强页面过渡</h4>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  {['fade', 'slide', 'scale', 'glitch', 'morph', 'wave'].map((variant) => (
                    <EnhancedPageTransition
                      key={variant}
                      variant={variant as any}
                      enableAudio={true}
                      className="p-4 bg-gradient-to-br from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 rounded-lg text-center"
                    >
                      <div className="text-sm font-medium capitalize">
                        {variant} 过渡
                      </div>
                    </EnhancedPageTransition>
                  ))}
                </div>
              </div>

              {/* 组合效果展示 */}
              <div className="text-center">
                <h4 className="text-lg font-semibold mb-4">组合效果展示</h4>
                <GlitchEffect trigger="hover" intensity="low">
                  <WaterRipple intensity="medium" color="rgba(139, 92, 246, 0.3)">
                    <EnhancedMagneticButton
                      variant="primary"
                      size="lg"
                      intensity={0.3}
                      enableAudio={true}
                      enableRipple={true}
                      enableGlow={true}
                    >
                      <Palette className="w-4 h-4 mr-2" />
                      组合动画效果
                    </EnhancedMagneticButton>
                  </WaterRipple>
                </GlitchEffect>
              </div>
            </div>
          </Card>
        </ScrollReveal>

        {/* 额外内容用于测试滚动 */}
        <ScrollReveal direction="up" delay={1.8}>
          <Card
            title="测试滚动区域 1"
            subtitle="这个区域用于测试引导功能的滚动效果"
          >
            <div className="text-center py-8">
              <p className="text-gray-600 dark:text-gray-400 mb-4">
                当您使用引导功能时，页面会自动滚动到这里
              </p>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                {Array.from({ length: 8 }, (_, i) => (
                  <div
                    key={i}
                    className="p-4 bg-gradient-to-br from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 rounded-lg text-center"
                  >
                    <Target className="w-8 h-8 text-blue-600 mb-2" />
                    <div className="text-sm font-medium">测试项 {i + 1}</div>
                  </div>
                ))}
              </div>
            </div>
          </Card>
        </ScrollReveal>

        <ScrollReveal direction="up" delay={2.0}>
          <Card
            title="测试滚动区域 2"
            subtitle="更多内容用于测试滚动效果"
          >
            <div className="text-center py-8">
              <p className="text-gray-600 dark:text-gray-400 mb-4">
                这是另一个测试区域，确保引导功能能够正确滚动到目标位置
              </p>
              <div className="space-y-4">
                {Array.from({ length: 5 }, (_, i) => (
                  <div
                    key={i}
                    className="p-6 bg-gradient-to-r from-green-50 to-blue-50 dark:from-green-900/20 dark:to-blue-900/20 rounded-lg"
                  >
                    <h4 className="text-lg font-semibold mb-2">测试卡片 {i + 1}</h4>
                    <p className="text-gray-600 dark:text-gray-400">
                      这是用于测试滚动功能的卡片内容，确保引导组件能够正确显示在指定位置。
                    </p>
                  </div>
                ))}
              </div>
            </div>
          </Card>
        </ScrollReveal>
      </div>

      {/* 基础引导动画组件 */}
      <OnboardingGuide
        steps={onboardingSteps}
        isOpen={showOnboarding}
        onClose={() => setShowOnboarding(false)}
        onComplete={() => {
          setShowOnboarding(false)
          showAlertWithScroll(t('animation.onboarding.completeMessage'), 'success', t('animation.onboarding.completeTitle'))
        }}
      />

      {/* 固定引导动画组件 */}
      <FixedOnboarding
        steps={[
          {
            id: 'welcome',
            title: t('animation.fixed.welcome.title'),
            description: t('animation.fixed.welcome.description'),
            target: '.welcome-card',
            position: 'center'
          },
          {
            id: 'features',
            title: t('animation.fixed.features.title'),
            description: t('animation.fixed.features.description'),
            target: '.features-card',
            position: 'center'
          },
          {
            id: 'water-ripple',
            title: t('animation.fixed.waterRipple.title'),
            description: t('animation.fixed.waterRipple.description'),
            target: '.welcome-card',
            position: 'center',
            action: {
              text: t('animation.fixed.waterRipple.action'),
              onClick: () => {
                // 触发水波效果
                const element = document.querySelector('.welcome-card') as HTMLElement
                if (element) {
                  element.click()
                }
              }
            }
          },
          {
            id: 'get-started',
            title: t('animation.fixed.getStarted.title'),
            icon: <Rocket className="w-5 h-5" />,
            description: t('animation.fixed.getStarted.description'),
            target: '.get-started-card',
            position: 'center',
            action: {
              text: t('animation.fixed.getStarted.action'),
              onClick: () => {
                showAlertWithScroll(t('animation.fixed.getStarted.actionMessage'), 'info', t('animation.fixed.getStarted.actionTitle'))
              }
            }
          }
        ]}
        isOpen={showFixedOnboarding}
        onClose={() => setShowFixedOnboarding(false)}
        onComplete={() => {
          setShowFixedOnboarding(false)
          showAlertWithScroll(t('animation.fixed.completeMessage'), 'success', t('animation.fixed.completeTitle'))
        }}
        autoPlay={false}
        showProgress={true}
        allowSkip={true}
      />
    </div>
  )
}

export default AnimationShowcase
