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

const AnimationShowcase = () => {
  const [showOnboarding, setShowOnboarding] = useState(false)
  const [showFixedOnboarding, setShowFixedOnboarding] = useState(false)
  const [showParticles, setShowParticles] = useState(false)

  const onboardingSteps = [
    {
      id: 'welcome',
      title: '欢迎使用 SoulNexus',
      description: '这是一个强大的AI写作助手，帮助您创作出优秀的内容。',
      target: '.welcome-card'
    },
    {
      id: 'features',
      title: '核心功能',
      description: '探索我们的各种功能，包括智能写作、内容优化和实时协作。',
      target: '.features-card'
    },
    {
      id: 'get-started',
      title: '开始使用',
      description: '点击下方按钮开始您的创作之旅！',
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
              text="高级动画效果展示"
              speed={100}
              className="text-transparent bg-clip-text bg-gradient-to-r from-blue-600 to-purple-600"
            />
          </h1>
          <p className="text-lg text-gray-600 dark:text-gray-400">
            体验各种炫酷的动画效果和交互体验
          </p>
        </motion.div>

        {/* 真实水波涟漪效果 */}
        <ScrollReveal direction="up" delay={0.2}>
          <Card
            title="真实水波涟漪效果"
            subtitle="点击卡片体验真实的水波涟漪动画"
            className="welcome-card"
          >
            <div className="space-y-6">
              {/* 简单水波纹效果 */}
              <div>
                <h4 className="text-lg font-semibold mb-3">简单水波纹效果</h4>
                <SimpleWaterRipple
                  intensity="high"
                  color="rgba(59, 130, 246, 0.4)"
                  className="p-8 bg-gradient-to-br from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 rounded-lg"
                >
                  <div className="text-center">
                    <h3 className="text-2xl font-bold mb-2">点击我！</h3>
                    <p className="text-gray-600 dark:text-gray-400">
                      感受稳定的水波涟漪效果
                    </p>
                  </div>
                </SimpleWaterRipple>
              </div>

              {/* 真实水波纹效果 */}
              <div>
                <h4 className="text-lg font-semibold mb-3">真实水波纹效果</h4>
                <RealWaterRipple
                  intensity="high"
                  color="rgba(59, 130, 246, 0.4)"
                  className="p-8 bg-gradient-to-br from-green-50 to-blue-50 dark:from-green-900/20 dark:to-blue-900/20 rounded-lg"
                >
                  <div className="text-center">
                    <h3 className="text-2xl font-bold mb-2">点击我！</h3>
                    <p className="text-gray-600 dark:text-gray-400">
                      感受更真实的水波效果
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
            title="磁性按钮效果"
            subtitle="鼠标悬停体验磁性吸附效果"
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
                  <div className="font-semibold text-lg">磁性按钮</div>
                  <div className="text-sm opacity-80 mt-1">鼠标跟随效果</div>
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
                  <div className="font-semibold text-lg">悬停我</div>
                  <div className="text-sm opacity-80 mt-1">交互式体验</div>
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
                  <div className="font-semibold text-lg">交互体验</div>
                  <div className="text-sm opacity-80 mt-1">沉浸式感受</div>
                </div>
              </MagneticButton>
            </div>
          </Card>
        </ScrollReveal>

        {/* 故障效果 */}
        <ScrollReveal direction="right" delay={0.6}>
          <Card
            title="故障效果"
            subtitle="悬停或点击触发故障动画"
          >
            <div className="flex flex-wrap gap-4 justify-center">
              <GlitchEffect
                trigger="hover"
                intensity="medium"
                className="text-4xl font-bold text-blue-600"
              >
                HOVER ME
              </GlitchEffect>
              <GlitchEffect
                trigger="click"
                intensity="high"
                className="text-4xl font-bold text-purple-600 cursor-pointer"
              >
                CLICK ME
              </GlitchEffect>
            </div>
          </Card>
        </ScrollReveal>

        {/* 粒子效果 */}
        <ScrollReveal direction="up" delay={0.8}>
          <Card
            title="粒子效果"
            subtitle="动态粒子背景效果"
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
                  {showParticles ? '隐藏粒子' : '显示粒子'}
                </Button>
              </div>
            </div>
          </Card>
        </ScrollReveal>

        {/* 页面过渡效果 */}
        <ScrollReveal direction="down" delay={1.0}>
          <Card
            title="页面过渡效果"
            subtitle="各种页面切换动画"
          >
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              {['fade', 'slide', 'scale', 'flip'].map((variant) => (
                <PageTransition
                  key={variant}
                  variant={variant as any}
                  className="p-4 bg-gray-100 dark:bg-gray-800 rounded-lg text-center"
                >
                  <div className="text-sm font-medium capitalize">
                    {variant} 过渡
                  </div>
                </PageTransition>
              ))}
            </div>
          </Card>
        </ScrollReveal>

        {/* 引导动画 */}
        <ScrollReveal direction="up" delay={1.2}>
          <Card
            title="引导动画"
            subtitle="用户引导和教程动画 - 支持滚动到指定位置"
          >
            <div className="text-center space-y-4">
              <div className="mb-6 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                <p className="text-sm text-blue-700 dark:text-blue-300 flex items-center gap-2">
                  <Lightbulb className="w-4 h-4" />
                  点击下面的按钮体验引导功能，页面会自动滚动到相关元素位置
                </p>
              </div>
              <div className="flex flex-col sm:flex-row gap-4 justify-center">
                <Button
                  onClick={() => setShowOnboarding(true)}
                  variant="primary"
                  size="lg"
                  className="px-8 py-4"
                >
                  基础引导教程
                </Button>
                <Button
                  onClick={() => setShowFixedOnboarding(true)}
                  variant="outline"
                  size="lg"
                  className="px-8 py-4"
                >
                  固定引导教程
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
          showAlertWithScroll('引导完成！欢迎使用 SoulNexus！', 'success', '欢迎！')
        }}
      />

      {/* 固定引导动画组件 */}
      <FixedOnboarding
        steps={[
          {
            id: 'welcome',
            title: '🎉 欢迎使用 SoulNexus',
            description: '这是一个强大的AI语音助手，帮助您创作出优秀的内容。让我们开始这段精彩的旅程！',
            target: '.welcome-card',
            position: 'center'
          },
          {
            id: 'features',
            title: '✨ 核心功能',
            description: '探索我们的各种功能，包括智能语音。每个功能都经过精心设计，为您提供最佳体验。',
            target: '.features-card',
            position: 'center'
          },
          {
            id: 'water-ripple',
            title: '🌊 水波涟漪效果',
            description: '体验我们独特的水波涟漪交互效果，让每一次点击都充满惊喜！',
            target: '.welcome-card',
            position: 'center',
            action: {
              text: '试试水波效果',
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
            title: '开始使用',
            icon: <Rocket className="w-5 h-5" />,
            description: '现在您已经了解了所有功能，点击下方按钮开始您的创作之旅！',
            target: '.get-started-card',
            position: 'center',
            action: {
              text: '开始创作',
              onClick: () => {
                showAlertWithScroll('开始您的创作之旅！', 'info', '创作开始')
              }
            }
          }
        ]}
        isOpen={showFixedOnboarding}
        onClose={() => setShowFixedOnboarding(false)}
        onComplete={() => {
          setShowFixedOnboarding(false)
          showAlertWithScroll('🎉 引导完成！欢迎使用 SoulNexus！', 'success', '完成！')
        }}
        autoPlay={false}
        showProgress={true}
        allowSkip={true}
      />
    </div>
  )
}

export default AnimationShowcase
