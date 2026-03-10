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

const ContentCarousel = ({
  title,
  subtitle,
  description,
  features,
  carouselItems,
  ctaText,
  ctaLink,
  reverse = false
}: ContentCarouselProps) => {
  const [currentIndex, setCurrentIndex] = useState(0)
  const [isAutoPlaying, setIsAutoPlaying] = useState(true)

  // 自动轮播
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
      viewport={{ once: false, margin: "-100px" }}
      transition={{ duration: 0.8 }}
      className="flex flex-col justify-center space-y-6"
    >
      {subtitle && (
        <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-gradient-to-r from-indigo-100 to-purple-100 dark:from-indigo-900/30 dark:to-purple-900/30 border border-indigo-200/50 dark:border-indigo-800/50 w-fit">
          <span className="text-sm font-semibold text-indigo-600 dark:text-indigo-400">{subtitle}</span>
        </div>
      )}
      
      <h2 className="text-4xl md:text-5xl font-bold text-gray-900 dark:text-white leading-tight">
        {title}
      </h2>
      
      <p className="text-lg text-gray-600 dark:text-gray-300 leading-relaxed">
        {description}
      </p>

      {features && features.length > 0 && (
        <ul className="space-y-3">
          {features.map((feature, index) => (
            <motion.li
              key={index}
              initial={{ opacity: 0, x: -20 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: false, margin: "-50px" }}
              transition={{ duration: 0.5, delay: index * 0.1 }}
              className="flex items-start gap-3"
            >
              <div className="w-6 h-6 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center flex-shrink-0 mt-0.5">
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <span className="text-gray-700 dark:text-gray-300">{feature}</span>
            </motion.li>
          ))}
        </ul>
      )}

      {ctaText && ctaLink && (
        <div>
          <a
            href={ctaLink}
            className="inline-flex items-center justify-center px-6 py-3 rounded-xl font-semibold shadow-lg shadow-indigo-500/20 bg-gradient-to-r from-indigo-500 via-purple-500 to-blue-500 hover:from-indigo-600 hover:via-purple-600 hover:to-blue-600 transition-all duration-300 hover:scale-105 hover:shadow-2xl hover:shadow-indigo-500/40 active:scale-95"
          >
            {ctaText}
          </a>
        </div>
      )}
    </motion.div>
  )

  const carouselSection = (
    <motion.div
      initial={{ opacity: 0, x: reverse ? -50 : 50 }}
      whileInView={{ opacity: 1, x: 0 }}
      viewport={{ once: false, margin: "-100px" }}
      transition={{ duration: 0.8 }}
      className="relative"
      onMouseEnter={() => setIsAutoPlaying(false)}
      onMouseLeave={() => setIsAutoPlaying(true)}
    >
      {/* 轮播容器 */}
      <div className="relative aspect-[16/9] rounded-2xl overflow-hidden shadow-2xl border border-gray-200/50 dark:border-gray-700/50">
        <AnimatePresence mode="wait">
          <motion.img
            key={currentIndex}
            src={carouselItems[currentIndex].image}
            alt={carouselItems[currentIndex].alt}
            initial={{ opacity: 0, scale: 1.1 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.95 }}
            transition={{ duration: 0.5 }}
            className="w-full h-full object-contain bg-white dark:bg-gray-900"
          />
        </AnimatePresence>

        {/* 导航按钮 */}
        {carouselItems.length > 1 && (
          <>
            <button
              onClick={goToPrevious}
              className="absolute left-4 top-1/2 -translate-y-1/2 w-10 h-10 rounded-full bg-white/90 dark:bg-gray-800/90 backdrop-blur-sm flex items-center justify-center shadow-lg hover:bg-white dark:hover:bg-gray-800 transition-all duration-300 hover:scale-110 group"
              aria-label="Previous slide"
            >
              <ChevronLeft className="w-5 h-5 text-gray-700 dark:text-gray-300 group-hover:text-indigo-600 dark:group-hover:text-indigo-400" />
            </button>
            <button
              onClick={goToNext}
              className="absolute right-4 top-1/2 -translate-y-1/2 w-10 h-10 rounded-full bg-white/90 dark:bg-gray-800/90 backdrop-blur-sm flex items-center justify-center shadow-lg hover:bg-white dark:hover:bg-gray-800 transition-all duration-300 hover:scale-110 group"
              aria-label="Next slide"
            >
              <ChevronRight className="w-5 h-5 text-gray-700 dark:text-gray-300 group-hover:text-indigo-600 dark:group-hover:text-indigo-400" />
            </button>
          </>
        )}
      </div>

      {/* 指示器 - 移到轮播图下方 */}
      {carouselItems.length > 1 && (
        <div className="flex justify-center gap-3 mt-6">
          {carouselItems.map((_, index) => (
            <button
              key={index}
              onClick={() => goToSlide(index)}
              className={`h-2 rounded-full transition-all duration-300 ${
                index === currentIndex
                  ? 'bg-gradient-to-r from-indigo-500 to-purple-600 w-12 shadow-lg shadow-indigo-500/30'
                  : 'bg-gray-300 dark:bg-gray-600 w-2 hover:bg-indigo-400 dark:hover:bg-indigo-500 hover:w-8'
              }`}
              aria-label={`Go to slide ${index + 1}`}
            />
          ))}
        </div>
      )}
    </motion.div>
  )

  return (
    <div className={`grid grid-cols-1 lg:grid-cols-5 gap-12 lg:gap-16 items-center ${reverse ? 'lg:grid-flow-dense' : ''}`}>
      {reverse ? (
        <>
          <div className="lg:col-start-4 lg:col-span-2">{contentSection}</div>
          <div className="lg:col-start-1 lg:col-span-3 lg:row-start-1">{carouselSection}</div>
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

export default ContentCarousel
