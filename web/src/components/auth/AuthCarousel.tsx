import { useEffect, useMemo, useRef } from 'react'
import { useSiteConfig } from '@/contexts/siteConfig'
import { resolveLogoUrl } from '@/config/brandConfig'
import { useTranslation } from '@/i18n'

const ICONS = [
  '/icons/icons/openai_icon_svg.svg',
  '/icons/icons/aws_bedrock_icon_svg.svg',
  '/icons/icons/azure_icon_svg.svg',
  '/icons/icons/gemini_icon_svg.svg',
  '/icons/icons/tencent_cloud_icon_svg.svg',
  '/icons/icons/deepseek_icon_svg.svg',
  '/icons/icons/zhipu_icon_svg.svg',
  '/icons/icons/kimi_icon_svg.svg',
  '/icons/icons/volcanic_engine_icon_svg.svg',
  '/icons/icons/ollama_icon_svg.svg',
  '/icons/icons/local_icon_svg.svg',
  '/icons/icons/tencent_icon_svg.svg',
  '/icons/icons/minimax_icon_svg.svg',
  '/icons/icons/docker_ai_icon_svg.svg',
  '/icons/icons/wenxin_icon_svg.svg',
  '/icons/icons/vllm_icon_svg.svg',
  '/icons/icons/xinference_icon_svg.svg',
  '/icons/icons/anthropic_icon_svg.svg',
] as const

// 修改这里即可调轨道半径与转速
const ORBITS = [
  { size: 300, speed: 0.0012, iconCount: 4 },
  { size: 520, speed: -0.0026, iconCount: 6 },
  { size: 740, speed: 0.0036, iconCount: 8 },
] as const

// 修改这里即可调粒子数量与速度区间
const PARTICLE_TOTAL = 140
const PARTICLE_SPEED_RANGE = 0.018
const PARTICLE_RADIUS_RANGE: [number, number] = [60, 500]

export default function AuthCarousel() {
  const { t } = useTranslation()
  const { config } = useSiteConfig()
  const logoUrl = resolveLogoUrl(config.SITE_LOGO_URL)
  const orbitRefs = useRef<(HTMLDivElement | null)[]>([])
  const particleRefs = useRef<(HTMLSpanElement | null)[]>([])
  const frameRef = useRef<number | null>(null)
  const stateRef = useRef({
    orbitRotate: [0, 0, 0],
    particles: [] as Array<{ angle: number; radius: number; speed: number }>,
  })

  const totalIcons = useMemo(() => ORBITS.reduce((sum, orbit) => sum + orbit.iconCount, 0), [])

  useEffect(() => {
    stateRef.current.particles = Array.from({ length: PARTICLE_TOTAL }, () => ({
      angle: Math.random() * Math.PI * 2,
      radius: PARTICLE_RADIUS_RANGE[0] + Math.random() * (PARTICLE_RADIUS_RANGE[1] - PARTICLE_RADIUS_RANGE[0]),
      speed: (Math.random() - 0.5) * PARTICLE_SPEED_RANGE,
    }))

    const animate = () => {
      const state = stateRef.current

      ORBITS.forEach((orbit, index) => {
        state.orbitRotate[index] += orbit.speed
        const el = orbitRefs.current[index]
        if (el) {
          el.style.transform = `translate(-50%, -50%) rotate(${state.orbitRotate[index]}rad)`
        }
      })

      particleRefs.current.forEach((el, index) => {
        if (!el) return
        const particle = state.particles[index]
        particle.angle += particle.speed
        const x = Math.cos(particle.angle) * particle.radius
        const y = Math.sin(particle.angle) * particle.radius
        el.style.transform = `translate(-50%, -50%) translate(${x}px, ${y}px)`
      })

      frameRef.current = window.requestAnimationFrame(animate)
    }

    frameRef.current = window.requestAnimationFrame(animate)
    return () => {
      if (frameRef.current) window.cancelAnimationFrame(frameRef.current)
    }
  }, [totalIcons])

  return (
    <div className="relative h-full w-full overflow-hidden rounded-[28px] bg-[linear-gradient(180deg,#bfdcff_0%,#d6eaff_50%,#ffffff_100%)]">
      <div className="absolute left-0 right-0 bottom-0 z-50 h-[min(82vh,900px)] bg-[linear-gradient(0deg,rgba(255,255,255,1)_0%,rgba(255,255,255,0.99)_5%,rgba(255,255,255,0.98)_12%,rgba(255,255,255,0.94)_20%,rgba(255,255,255,0.88)_30%,rgba(255,255,255,0.76)_42%,rgba(255,255,255,0.56)_58%,rgba(255,255,255,0.28)_78%,rgba(255,255,255,0.06)_92%,rgba(255,255,255,0)_100%)]" />

      <div className="pointer-events-none absolute left-1/2 bottom-[clamp(36px,5vh,56px)] z-[60] w-[min(88%,520px)] -translate-x-1/2 text-center">
        <div className="text-[clamp(17px,1.35vw,22px)] font-semibold leading-snug tracking-tight text-neutral-900">
          {t('auth.carouselTitle')}
        </div>
        <div className="mt-2 text-[clamp(12px,0.85vw,15px)] leading-relaxed text-neutral-500">
          {t('auth.carouselSubtitle')}
        </div>
      </div>

      <div className="absolute left-1/2 top-1/2 z-10 h-[900px] w-[900px] -translate-x-1/2 -translate-y-1/2 overflow-visible box-content">
        {ORBITS.map((orbit, ringIndex) => (
          <div
            key={orbit.size}
            className="absolute left-1/2 top-1/2 rounded-full border box-content"
            style={{
              width: orbit.size,
              height: orbit.size,
              transform: 'translate(-50%, -50%)',
              borderColor: `rgba(45,174,240,${0.16 + ringIndex * 0.05})`,
              boxShadow: `0 0 24px rgba(45,174,240,${0.04 + ringIndex * 0.02}) inset`,
            }}
          />
        ))}

        <div className="absolute left-1/2 top-1/2 z-20 flex h-[110px] w-[110px] -translate-x-1/2 -translate-y-1/2 items-center justify-center overflow-hidden rounded-full bg-white shadow-[0_0_0_14px_rgba(255,255,255,0.85),0_0_34px_rgba(0,114,255,0.45)]">
          <img src={logoUrl} alt="" aria-hidden="true" draggable={false} className="h-full w-full object-contain p-4" />
        </div>

        {ORBITS.map((orbit, orbitIndex) => {
          const startIndex = ORBITS.slice(0, orbitIndex).reduce((sum, item) => sum + item.iconCount, 0)
          const icons = ICONS.slice(startIndex, startIndex + orbit.iconCount)
          return (
            <div
              key={orbit.size}
              ref={(el) => {
                orbitRefs.current[orbitIndex] = el
              }}
              className="absolute left-1/2 top-1/2 h-[900px] w-[900px] -translate-x-1/2 -translate-y-1/2"
            >
              {icons.map((icon, iconIndex) => {
                const angle = (Math.PI * 2 * iconIndex) / orbit.iconCount
                return (
                  <div
                    key={icon}
                    className="absolute left-1/2 top-1/2 grid h-[44px] w-[44px] -translate-x-1/2 -translate-y-1/2 place-items-center rounded-full bg-white/88 shadow-[0_10px_24px_rgba(0,0,0,0.08)] backdrop-blur-xl transition-all duration-300 hover:scale-110 hover:bg-white hover:shadow-[0_14px_28px_rgba(0,114,255,0.22)]"
                    style={{ transform: `translate(-50%, -50%) rotate(${angle}rad) translateY(-${orbit.size / 2}px)` }}
                  >
                    <span className="absolute left-1/2 top-1/2 h-[calc(100%+18px)] w-[calc(100%+18px)] -translate-x-1/2 -translate-y-1/2 rounded-full bg-[radial-gradient(circle,rgba(255,255,255,0.96)_0%,rgba(255,255,255,0.66)_45%,rgba(255,255,255,0)_75%)] blur-[3px]" />
                    <img
                      src={icon}
                      alt=""
                      aria-hidden="true"
                      draggable={false}
                      className="relative z-10 h-[70%] w-[70%] object-contain opacity-95 mix-blend-multiply"
                    />
                  </div>
                )
              })}
            </div>
          )
        })}

        <div className="absolute left-1/2 top-1/2 h-[900px] w-[900px] -translate-x-1/2 -translate-y-1/2 overflow-visible pointer-events-none">
          {Array.from({ length: PARTICLE_TOTAL }).map((_, index) => (
            <span
              key={`particle-${index}`}
              ref={(el) => {
                particleRefs.current[index] = el
              }}
              className="absolute left-1/2 top-1/2 h-[4px] w-[4px] rounded-full bg-[rgba(126,194,255,0.58)] shadow-[0_0_12px_rgba(126,194,255,0.34)]"
              style={{ transform: 'translate(-50%, -50%)' }}
            />
          ))}
        </div>
      </div>
    </div>
  )
}
