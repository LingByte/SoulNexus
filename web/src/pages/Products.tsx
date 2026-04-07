import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { ArrowLeft, ArrowRight, Github, ExternalLink, Home } from 'lucide-react'
import Footer from '@/components/Layout/Footer'
import LoadingAnimation from '@/components/Animations/LoadingAnimation'
import { useI18nStore } from '@/stores/i18nStore'

interface Product {
  id: string
  name: string
  title: string
  description: string
  icon?: string
  logo?: string
  color?: string
  links?: {
    github?: string
    demo?: string
  }
}

export default function Products() {
  const navigate = useNavigate()
  const { t } = useI18nStore()
  const [products, setProducts] = useState<Product[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const loadProducts = async () => {
      try {
        const response = await import('@/data/products.json')
        setProducts(response.default.products || [])
      } catch (err) {
        console.error('Failed to load products:', err)
      } finally {
        setLoading(false)
      }
    }

    loadProducts()
  }, [])

  const containerVariants = {
    hidden: { opacity: 0 },
    visible: {
      opacity: 1,
      transition: {
        staggerChildren: 0.1,
        delayChildren: 0.2,
      },
    },
  }

  const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: {
      opacity: 1,
      y: 0,
      transition: { duration: 0.5 },
    },
  }

  return (
    <div className="min-h-screen relative text-slate-100">
      {/* Background: deep slate base, subtle accents (avoid washed-out purple) */}
      <div className="pointer-events-none absolute inset-0 -z-20">
        <div className="absolute inset-0 bg-gradient-to-b from-slate-950 via-slate-900 to-slate-950" />
        <div className="absolute inset-0 bg-[radial-gradient(900px_480px_at_50%_-5%,rgba(56,189,248,0.12),transparent),radial-gradient(700px_420px_at_100%_15%,rgba(99,102,241,0.10),transparent)]" />
        <div className="absolute inset-0 opacity-25 [background-image:linear-gradient(to_right,rgba(148,163,184,0.08)_1px,transparent_1px),linear-gradient(to_bottom,rgba(148,163,184,0.08)_1px,transparent_1px)] [background-size:28px_28px]" />
      </div>

      <div className="relative z-10 max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 pt-24 pb-6">
        <button
          type="button"
          onClick={() => navigate('/')}
          className="inline-flex items-center gap-2 rounded-lg border border-slate-600 bg-slate-900/80 px-4 py-2.5 text-sm font-medium text-slate-100 shadow-sm hover:bg-slate-800 hover:border-slate-500 transition-colors"
        >
          <ArrowLeft className="w-4 h-4 shrink-0 opacity-90" aria-hidden />
          <Home className="w-4 h-4 shrink-0 opacity-90" aria-hidden />
          {t('products.backHome')}
        </button>
      </div>

      {/* Hero Section */}
      <section className="relative px-4 sm:px-6 lg:px-8 pb-16">
        <div className="max-w-6xl mx-auto">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
            className="text-center mb-14"
          >
            <h1 className="text-4xl sm:text-5xl md:text-6xl font-bold mb-4 ">
              {t('products.heroTitle')}
            </h1>
            <p className="text-lg sm:text-xl text-slate-300 max-w-2xl mx-auto leading-relaxed">
              {t('products.heroSubtitle')}
            </p>
          </motion.div>

          {/* Products Grid */}
          {loading ? (
            <div className="flex items-center justify-center py-20">
              <div className="text-center">
                <LoadingAnimation type="progress" size="lg" className="mb-4" />
                <p className="text-slate-300">{t('common.loading')}</p>
              </div>
            </div>
          ) : (
            <motion.div
              variants={containerVariants}
              initial="hidden"
              animate="visible"
              className="grid grid-cols-1 md:grid-cols-2 gap-8 max-w-6xl mx-auto"
            >
              {products.map((product) => (
                <motion.div
                  key={product.id}
                  variants={itemVariants}
                  className="group relative"
                >
                  <div className="absolute inset-0 rounded-2xl bg-gradient-to-r from-sky-500/15 to-indigo-500/15 blur-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-500" />

                  <div className="relative rounded-2xl border border-slate-600/80 bg-slate-900/90 backdrop-blur-md p-8 shadow-lg shadow-black/20 hover:border-slate-500 transition-colors duration-300 h-full flex flex-col">
                    {/* Header */}
                    <div className="mb-6">
                      {product.logo ? (
                        <img src={product.logo} alt={product.name} className="w-10 mb-4" />
                      ) : (
                        <div className="inline-flex items-center justify-center w-12 h-12 rounded-lg bg-slate-800 border border-slate-600 mb-4">
                          <span className="text-xl font-bold">{product.name.charAt(0)}</span>
                        </div>
                      )}
                      <h3 className="text-2xl font-bold mb-3">{product.title}</h3>
                      <p className="text-slate-300 leading-relaxed text-[15px]">{product.description}</p>
                    </div>

                    {/* Links */}
                    {product.links && (
                      <div className="flex gap-3 mb-6 flex-wrap">
                        {product.links.github && (
                          <a
                            href={product.links.github}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-2 px-3 py-2 rounded-lg border border-slate-600 bg-slate-800/90 text-sm text-slate-100 hover:bg-slate-700 hover:border-slate-500 transition-colors"
                          >
                            <Github className="w-4 h-4" />
                            GitHub
                          </a>
                        )}
                        {product.links.demo && (
                          <a
                            href={product.links.demo}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-2 px-3 py-2 rounded-lg border border-slate-600 bg-slate-800/90 text-sm text-slate-100 hover:bg-slate-700 hover:border-slate-500 transition-colors"
                          >
                            <ExternalLink className="w-4 h-4" />
                            Demo
                          </a>
                        )}
                      </div>
                    )}

                    {/* CTA Button */}
                    <div className="mt-auto">
                      <button
                        type="button"
                        onClick={() => navigate(`/product/${product.id}`)}
                        className="w-full inline-flex items-center justify-center gap-2 px-4 py-3 rounded-lg bg-gradient-to-r from-sky-600 to-indigo-600 hover:from-sky-500 hover:to-indigo-500 font-semibold shadow-md shadow-sky-900/30 transition-all duration-300 group/btn"
                      >
                        Learn More
                        <ArrowRight className="w-4 h-4 group-hover/btn:translate-x-1 transition-transform" />
                      </button>
                    </div>
                  </div>
                </motion.div>
              ))}
            </motion.div>
          )}
        </div>
      </section>

      <Footer />
    </div>
  )
}
