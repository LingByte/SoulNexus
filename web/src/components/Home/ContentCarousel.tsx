import { motion, AnimatePresence } from 'framer-motion'
import { useState, useEffect } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'

interface CarouselItem {
  image: string
  alt: string
}

interface ContentCarouselProps {
  title: string
  subtitle?: string
  description: string
  features?: string[]
  carouselItems: CarouselItem[]
  ctaText?: string
  ctaLink?: string
  reverse?: boolean
}

export default function ContentCarousel({
  title,
  subtitle,
  description,
  features,
  carouselItems,
  ctaText,
  ctaLink,
  reverse = false,
}: ContentCarouselProps) {
  const [currentIndex, setCurrentIndex] = useState(0)
  const [isAutoPlaying, setIsAutoPlaying] = useState(true)

  useEffect(() => {
    if (!isAutoPlaying || carouselItems.length <= 1) return
    const interval = setInterval(() => {
      setCurrentIndex((prev) => (prev + 1) % carouselItems.length)
    }, 4000)
    return () => clearInterval(interval)
  }, [isAutoPlaying, carouselItems.length])

  const goToPrevious = () => {
    setCurrentIndex((prev) => (prev - 1 + carouselItems.length) % carouselItems.length)
    setIsAutoPlaying(false)
  }

  const goToNext = () => {
    setCurrentIndex((prev) => (prev + 1) % carouselItems.length)
    setIsAutoPlaying(false)
  }

  const goToSlide = (index: number) => {
    setCurrentIndex(index)
    setIsAutoPlaying(false)
  }

  const contentSection = (
    <motion.div
      initial={{ opacity: 0, x: reverse ? 50 : -50 }}
      whileInView={{ opacity: 1, x: 0 }}
      viewport={{ once: true, margin: '-80px' }}
      transition={{ duration: 0.7 }}
      className="flex flex-col justify-center space-y-5"
    >
      {subtitle ? (
        <div className="inline-flex w-fit items-center gap-2 rounded-full border border-violet-200/60 bg-gradient-to-r from-violet-100 to-purple-100 px-4 py-2 dark:border-violet-800/50 dark:from-violet-900/30 dark:to-purple-900/30">
          <span className="text-sm font-semibold text-violet-600 dark:text-violet-300">{subtitle}</span>
        </div>
      ) : null}

      <h2 className="font-display text-3xl font-bold leading-tight tracking-tight text-[hsl(var(--foreground))] md:text-4xl">
        {title}
      </h2>

      <p className="text-base leading-relaxed text-[hsl(var(--muted-foreground))] md:text-lg">{description}</p>

      {features && features.length > 0 ? (
        <ul className="space-y-2.5">
          {features.map((feature, index) => (
            <motion.li
              key={feature}
              initial={{ opacity: 0, x: -16 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.45, delay: index * 0.08 }}
              className="flex items-start gap-3"
            >
              <div className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-violet-500 to-purple-600">
                <svg className="h-3.5 w-3.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <span className="text-sm text-[hsl(var(--foreground)/0.85)] md:text-base">{feature}</span>
            </motion.li>
          ))}
        </ul>
      ) : null}

      {ctaText && ctaLink ? (
        <div>
          <a
            href={ctaLink}
            target={ctaLink.startsWith('http') ? '_blank' : undefined}
            rel={ctaLink.startsWith('http') ? 'noopener noreferrer' : undefined}
            className="inline-flex items-center justify-center rounded-xl bg-gradient-to-r from-violet-600 via-purple-600 to-indigo-600 px-6 py-3 text-sm font-semibold text-white shadow-lg shadow-violet-500/25 transition hover:scale-[1.02] hover:shadow-xl active:scale-[0.98]"
          >
            {ctaText}
          </a>
        </div>
      ) : null}
    </motion.div>
  )

  const carouselSection = (
    <motion.div
      initial={{ opacity: 0, x: reverse ? -50 : 50 }}
      whileInView={{ opacity: 1, x: 0 }}
      viewport={{ once: true, margin: '-80px' }}
      transition={{ duration: 0.7 }}
      className="relative"
      onMouseEnter={() => setIsAutoPlaying(false)}
      onMouseLeave={() => setIsAutoPlaying(true)}
    >
      <div className="relative min-h-[280px] overflow-hidden rounded-2xl border border-[hsl(var(--border))] shadow-2xl shadow-violet-500/10 sm:min-h-[360px] lg:min-h-[420px]">
        <AnimatePresence mode="wait">
          <motion.img
            key={currentIndex}
            src={carouselItems[currentIndex].image}
            alt={carouselItems[currentIndex].alt}
            initial={{ opacity: 0, scale: 1.04 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.98 }}
            transition={{ duration: 0.45 }}
            className="h-full min-h-[280px] w-full bg-[hsl(var(--card))] object-contain object-top sm:min-h-[360px] lg:min-h-[420px]"
          />
        </AnimatePresence>

        {carouselItems.length > 1 ? (
          <>
            <button
              type="button"
              onClick={goToPrevious}
              className="absolute left-3 top-1/2 flex h-10 w-10 -translate-y-1/2 items-center justify-center rounded-full bg-white/90 shadow-lg backdrop-blur-sm transition hover:scale-110 dark:bg-gray-800/90"
              aria-label="Previous slide"
            >
              <ChevronLeft className="h-5 w-5 text-violet-700 dark:text-violet-300" />
            </button>
            <button
              type="button"
              onClick={goToNext}
              className="absolute right-3 top-1/2 flex h-10 w-10 -translate-y-1/2 items-center justify-center rounded-full bg-white/90 shadow-lg backdrop-blur-sm transition hover:scale-110 dark:bg-gray-800/90"
              aria-label="Next slide"
            >
              <ChevronRight className="h-5 w-5 text-violet-700 dark:text-violet-300" />
            </button>
          </>
        ) : null}
      </div>

      {carouselItems.length > 1 ? (
        <div className="mt-5 flex justify-center gap-2">
          {carouselItems.map((_, index) => (
            <button
              key={index}
              type="button"
              onClick={() => goToSlide(index)}
              className={`h-2 rounded-full transition-all duration-300 ${
                index === currentIndex
                  ? 'w-10 bg-gradient-to-r from-violet-500 to-purple-600 shadow-md shadow-violet-500/30'
                  : 'w-2 bg-gray-300 hover:w-6 hover:bg-violet-400 dark:bg-gray-600'
              }`}
              aria-label={`Slide ${index + 1}`}
            />
          ))}
        </div>
      ) : null}
    </motion.div>
  )

  return (
    <div
      className={`grid grid-cols-1 items-center gap-10 lg:grid-cols-5 lg:gap-14 ${reverse ? 'lg:grid-flow-dense' : ''}`}
    >
      {reverse ? (
        <>
          <div className="lg:col-span-2 lg:col-start-4">{contentSection}</div>
          <div className="lg:col-span-3 lg:col-start-1 lg:row-start-1">{carouselSection}</div>
        </>
      ) : (
        <>
          <div className="lg:col-span-2">{contentSection}</div>
          <div className="lg:col-span-3">{carouselSection}</div>
        </>
      )}
    </div>
  )
}
