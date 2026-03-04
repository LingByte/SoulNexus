import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import {
  Zap,
  Mic,
  BookOpen,
  Activity,
  Phone,
  User,
  Settings,
  Check,
  ArrowLeft,
  Code,
  Layers,
  Zap as ZapIcon,
  Shield,
  Gauge,
  Cloud,
  Database,
  Lock,
  Cpu,
  Network,
  Workflow,
  Github,
  ExternalLink
} from 'lucide-react'
import Button from '@/components/UI/Button'
import Footer from '@/components/Layout/Footer'

const iconMap: Record<string, any> = {
  Zap,
  Mic,
  BookOpen,
  Activity,
  Phone,
  User,
  Settings,
  Code,
  Layers,
  ZapIcon,
  Shield,
  Gauge,
  Cloud,
  Database,
  Lock,
  Cpu,
  Network,
  Workflow,
  Key: Lock,
}

interface ProductData {
  id: string
  name: string
  title: string
  description: string
  hero: {
    title: string
    subtitle: string
    description: string
    image: string
  }
  sections: Array<{
    id: string
    title: string
    content?: string
    features?: Array<{
      title: string
      description: string
      icon: string
    }>
    subsections?: Array<{
      title: string
      description?: string
      details?: string[]
      image?: string
      technologies?: Array<{
        name: string
        version: string
        description: string
      }>
    }>
    useCases?: Array<{
      title: string
      description: string
      icon: string
    }>
    steps?: Array<{
      number: number
      title: string
      description: string
    }>
    plans?: Array<{
      name: string
      price: string
      period: string
      description: string
      features: string[]
      popular?: boolean
    }>
  }>
}

export default function ProductDetail() {
  const { productId } = useParams<{ productId: string }>()
  const navigate = useNavigate()
  const [productData, setProductData] = useState<ProductData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const loadProduct = async () => {
      try {
        setLoading(true)
        const response = await fetch(`/docs/products/${productId}.json`)
        if (!response.ok) {
          throw new Error('Product not found')
        }
        const data = await response.json()
        setProductData(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load product')
      } finally {
        setLoading(false)
      }
    }

    if (productId) {
      loadProduct()
    }
  }, [productId])

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-500 mx-auto mb-4"></div>
          <p className="text-muted-foreground">Loading product information...</p>
        </div>
      </div>
    )
  }

  if (error || !productData) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <p className="text-red-500 mb-4">{error || 'Product not found'}</p>
          <Button variant="primary" onClick={() => navigate('/products')}>
            Back to Products
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen relative">
      {/* Background */}
      <div className="pointer-events-none absolute inset-0 -z-20">
        <div className="absolute inset-0 bg-gradient-to-br from-indigo-950 via-purple-950 to-pink-950" />
        <div className="absolute inset-0 bg-[radial-gradient(1200px_600px_at_50%_-10%,rgba(99,102,241,0.4),transparent),radial-gradient(1000px_500px_at_100%_20%,rgba(236,72,153,0.3),transparent)]" />
        <div className="absolute inset-0 opacity-20 [background-image:linear-gradient(to_right,rgba(255,255,255,0.08)_1px,transparent_1px),linear-gradient(to_bottom,rgba(255,255,255,0.08)_1px,transparent_1px)] [background-size:26px_26px]" />
      </div>

      {/* Header with Back Button */}
      <div className="sticky top-0 z-40 backdrop-blur-md bg-gradient-to-r from-indigo-900/80 via-purple-900/80 to-pink-900/80 border-b border-indigo-500/30">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-4 flex items-center justify-between">
          <button
            onClick={() => navigate('/products')}
            className="inline-flex items-center gap-2 text-gray-300 hover:text-white transition-colors"
          >
            <ArrowLeft className="w-5 h-5" />
            <span>Back to Products</span>
          </button>
          <h1 className="text-2xl font-bold">{productData.name}</h1>
          <div className="w-24" />
        </div>
      </div>

      {/* Hero Section */}
      <section className="relative py-20 px-4 sm:px-6 lg:px-8 overflow-hidden">
        {/* Vibrant gradient background */}
        <div className="absolute inset-0 -z-10">
          <div className="absolute inset-0 bg-gradient-to-br from-indigo-600/30 via-purple-600/30 to-pink-600/30 blur-3xl" />
        </div>
        
        <div className="max-w-6xl mx-auto">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
            className="text-center mb-12"
          >
            {/* Logo */}
            {productData.hero.image && (
              <motion.div
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ duration: 0.6, delay: 0.1 }}
                className="mb-8 flex justify-center"
              >
                <img 
                  src={productData.hero.image} 
                  alt={productData.hero.title}
                  className="h-24 w-auto object-contain"
                />
              </motion.div>
            )}
            
            <h2 className="text-5xl md:text-6xl font-bold mb-6 bg-gradient-to-r from-indigo-300 via-purple-300 to-pink-300 bg-clip-text">
              {productData.hero.title}
            </h2>
            <p className="text-xl text-gray-200 mb-6 max-w-3xl mx-auto leading-relaxed">
              {productData.hero.subtitle}
            </p>
            <p className="text-lg text-gray-300 mb-8 max-w-4xl mx-auto leading-relaxed">
              {productData.hero.description}
            </p>
            
            {/* Links */}
            <div className="flex flex-col sm:flex-row gap-4 justify-center">
              {/* GitHub Link */}
              <a
                href={`https://github.com/LingByte/${productData.name}`}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-2 px-6 py-3 rounded-lg bg-indigo-800/50 hover:bg-indigo-700/50 text-white font-semibold transition-all duration-300 border border-indigo-500/30 hover:border-indigo-500/50"
              >
                <Github className="w-5 h-5" />
                GitHub
              </a>
              
              {/* Live Demo Link */}
              <a
                href="https://lingecho.com"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-2 px-6 py-3 rounded-lg bg-gradient-to-r from-indigo-500 to-purple-500 hover:from-indigo-600 hover:to-purple-600 font-semibold transition-all duration-300"
              >
                <ExternalLink className="w-5 h-5" />
                Live Demo
              </a>
            </div>
          </motion.div>
        </div>
      </section>

      {/* Sections */}
      <div className="relative space-y-24 px-4 sm:px-6 lg:px-8 pb-20">
        {productData.sections.map((section) => (
          <motion.section
            key={section.id}
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
            viewport={{ once: true }}
            className="max-w-6xl mx-auto"
          >
            {/* Section Title */}
            <div className="mb-16">
              <h2 className="text-4xl md:text-5xl font-bold mb-4">{section.title}</h2>
              <div className="h-1 w-20 bg-gradient-to-r from-indigo-500 to-purple-500 rounded-full" />
            </div>

            {/* Overview Section */}
            {section.id === 'overview' && section.content && (
              <div className="space-y-8">
                <p className="text-lg text-gray-300 leading-relaxed">{section.content}</p>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  {section.features?.map((feature, i) => {
                    const Icon = iconMap[feature.icon]
                    return (
                      <motion.div
                        key={i}
                        initial={{ opacity: 0, y: 10 }}
                        whileInView={{ opacity: 1, y: 0 }}
                        transition={{ delay: i * 0.1 }}
                        viewport={{ once: true }}
                      >
                        <div className="group relative">
                          <div className="absolute inset-0 bg-gradient-to-r from-indigo-500/10 to-purple-500/10 rounded-xl blur opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                      <div className="relative bg-gradient-to-br from-indigo-900/40 to-purple-900/40 backdrop-blur border border-indigo-500/30 rounded-xl p-6 hover:border-indigo-400/50 transition-all duration-300">
                            <div className="flex items-start gap-4">
                              {Icon && (
                                <div className="flex-shrink-0 w-12 h-12 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-500 flex items-center justify-center">
                                  <Icon className="w-6 h-6 text-white" />
                                </div>
                              )}
                              <div className="flex-1">
                                <h3 className="font-semibold text-lg mb-2">{feature.title}</h3>
                                <p className="text-gray-300">{feature.description}</p>
                              </div>
                            </div>
                          </div>
                        </div>
                      </motion.div>
                    )
                  })}
                </div>
              </div>
            )}

            {/* Features Section */}
            {section.id === 'features' && section.subsections && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {section.subsections.map((subsection, i) => (
                  <motion.div
                    key={i}
                    initial={{ opacity: 0, y: 10 }}
                    whileInView={{ opacity: 1, y: 0 }}
                    transition={{ delay: i * 0.1 }}
                    viewport={{ once: true }}
                  >
                    <div className="group relative h-full">
                      <div className="absolute inset-0 bg-gradient-to-r from-indigo-500/10 to-purple-500/10 rounded-xl blur opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                      <div className="relative bg-indigo-900/40 backdrop-blur border border-indigo-500/30 rounded-xl p-6 hover:border-indigo-500/50 transition-all duration-300 h-full">
                        <h3 className="font-semibold text-lg mb-2">{subsection.title}</h3>
                        <p className="text-sm text-gray-300 mb-4">{subsection.description}</p>
                        <ul className="space-y-3">
                          {subsection.details?.map((detail, j) => (
                            <li key={j} className="flex items-start gap-3">
                              <Check className="w-5 h-5 text-green-400 flex-shrink-0 mt-0.5" />
                              <span className="text-sm text-gray-300">{detail}</span>
                            </li>
                          ))}
                        </ul>
                      </div>
                    </div>
                  </motion.div>
                ))}
              </div>
            )}

            {/* Technology Stack */}
            {section.id === 'technology' && section.subsections && (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                {section.subsections.map((subsection, i) => (
                  <motion.div
                    key={i}
                    initial={{ opacity: 0, y: 10 }}
                    whileInView={{ opacity: 1, y: 0 }}
                    transition={{ delay: i * 0.1 }}
                    viewport={{ once: true }}
                  >
                    <div className="group relative h-full">
                      <div className="absolute inset-0 bg-gradient-to-r from-indigo-500/10 to-purple-500/10 rounded-xl blur opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                      <div className="relative bg-indigo-900/40 backdrop-blur border border-indigo-500/30 rounded-xl p-6 hover:border-indigo-500/50 transition-all duration-300 h-full">
                        <h3 className="font-semibold text-lg mb-6">{subsection.title}</h3>
                        <div className="space-y-4">
                          {subsection.technologies?.map((tech, j) => (
                            <div key={j} className="pb-4 border-b border-indigo-500/30 last:border-0 last:pb-0">
                              <div className="font-semibold text-sm text-indigo-400 mb-1">{tech.name}</div>
                              <div className="text-xs text-gray-500 mb-2">{tech.version}</div>
                              <div className="text-xs text-gray-300">{tech.description}</div>
                            </div>
                          ))}
                        </div>
                      </div>
                    </div>
                  </motion.div>
                ))}
              </div>
            )}

            {/* Use Cases */}
            {section.id === 'use-cases' && section.useCases && (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {section.useCases.map((useCase, i) => {
                  const Icon = iconMap[useCase.icon]
                  return (
                    <motion.div
                      key={i}
                      initial={{ opacity: 0, y: 10 }}
                      whileInView={{ opacity: 1, y: 0 }}
                      transition={{ delay: i * 0.1 }}
                      viewport={{ once: true }}
                    >
                      <div className="group relative h-full">
                        <div className="absolute inset-0 bg-gradient-to-r from-indigo-500/10 to-purple-500/10 rounded-xl blur opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                        <div className="relative bg-indigo-900/40 backdrop-blur border border-indigo-500/30 rounded-xl p-6 hover:border-indigo-500/50 transition-all duration-300 h-full">
                          <div className="flex items-start gap-4 mb-4">
                            {Icon && (
                              <div className="flex-shrink-0 w-10 h-10 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-500 flex items-center justify-center">
                                <Icon className="w-5 h-5 text-white" />
                              </div>
                            )}
                            <h3 className="font-semibold text-lg">{useCase.title}</h3>
                          </div>
                          <p className="text-gray-300">{useCase.description}</p>
                        </div>
                      </div>
                    </motion.div>
                  )
                })}
              </div>
            )}

            {/* Getting Started */}
            {section.id === 'getting-started' && section.steps && (
              <div className="space-y-6">
                {section.steps.map((step, i) => (
                  <motion.div
                    key={i}
                    initial={{ opacity: 0, x: -20 }}
                    whileInView={{ opacity: 1, x: 0 }}
                    transition={{ delay: i * 0.1 }}
                    viewport={{ once: true }}
                    className="flex gap-6"
                  >
                    <div className="flex-shrink-0">
                      <div className="flex items-center justify-center h-12 w-12 rounded-full bg-gradient-to-br from-indigo-500 to-purple-500 text-white font-bold text-lg">
                        {step.number}
                      </div>
                    </div>
                    <div className="flex-1 pt-1">
                      <h3 className="font-semibold text-lg mb-2">{step.title}</h3>
                      <p className="text-gray-300">{step.description}</p>
                    </div>
                  </motion.div>
                ))}
              </div>
            )}

            {/* Pricing */}
            {section.id === 'pricing' && section.plans && (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
                {section.plans.map((plan, i) => (
                  <motion.div
                    key={i}
                    initial={{ opacity: 0, y: 20 }}
                    whileInView={{ opacity: 1, y: 0 }}
                    transition={{ delay: i * 0.1 }}
                    viewport={{ once: true }}
                    className="relative group"
                  >
                    {plan.popular && (
                      <div className="absolute -top-4 left-1/2 transform -translate-x-1/2 z-10">
                        <span className="bg-gradient-to-r from-indigo-500 to-purple-500 text-white px-4 py-1 rounded-full text-sm font-semibold">
                          Most Popular
                        </span>
                      </div>
                    )}
                    <div className={`relative h-full rounded-xl backdrop-blur border transition-all duration-300 p-8 ${
                      plan.popular
                        ? 'bg-gradient-to-br from-indigo-500/20 to-purple-500/20 border-indigo-500/50 shadow-lg shadow-indigo-500/20'
                        : 'bg-indigo-900/40 border-indigo-500/30 hover:border-indigo-500/50'
                    }`}>
                      <h3 className="font-semibold text-xl mb-2">{plan.name}</h3>
                      <p className="text-sm text-gray-300 mb-6">{plan.description}</p>
                      <ul className="space-y-3 mb-8">
                        {plan.features.map((feature, j) => (
                          <li key={j} className="flex items-start gap-3">
                            <Check className="w-5 h-5 text-green-400 flex-shrink-0 mt-0.5" />
                            <span className="text-sm text-gray-300">{feature}</span>
                          </li>
                        ))}
                      </ul>
                      <Button
                        variant={plan.popular ? 'primary' : 'outline'}
                        className="w-full"
                      >
                        Get Started
                      </Button>
                    </div>
                  </motion.div>
                ))}
              </div>
            )}

            {/* Features Showcase with Images */}
            {section.id === 'showcase' && section.subsections && (
              <div className="space-y-16">
                {section.subsections.map((showcase, i) => (
                  <motion.div
                    key={i}
                    initial={{ opacity: 0, y: 20 }}
                    whileInView={{ opacity: 1, y: 0 }}
                    transition={{ delay: i * 0.1 }}
                    viewport={{ once: true }}
                    className={`flex flex-col ${i % 2 === 0 ? 'md:flex-row' : 'md:flex-row-reverse'} gap-8 items-center`}
                  >
                    {/* Image */}
                    <div className="flex-1">
                      <div className="group relative rounded-xl overflow-hidden">
                        <div className="absolute inset-0 bg-gradient-to-r from-indigo-500/20 to-purple-500/20 rounded-xl blur opacity-0 group-hover:opacity-100 transition-opacity duration-300 z-10" />
                        <img
                          src={showcase.image}
                          alt={showcase.title}
                          className="w-full h-auto rounded-xl border border-indigo-500/30 hover:border-indigo-500/50 transition-all duration-300"
                        />
                      </div>
                    </div>
                    
                    {/* Content */}
                    <div className="flex-1">
                      <h3 className="text-2xl font-bold mb-3">{showcase.title}</h3>
                      <p className="text-gray-300 mb-6">{showcase.description}</p>
                      <ul className="space-y-3">
                        {showcase.details?.map((detail, j) => (
                          <li key={j} className="flex items-start gap-3">
                            <Check className="w-5 h-5 text-green-400 flex-shrink-0 mt-0.5" />
                            <span className="text-sm text-gray-300">{detail}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  </motion.div>
                ))}
              </div>
            )}

            {/* Architecture & Design, API Examples, Resources */}
            {(section.id === 'architecture' || section.id === 'api-examples' || section.id === 'resources') && section.subsections && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {section.subsections.map((subsection, i) => (
                  <motion.div
                    key={i}
                    initial={{ opacity: 0, y: 10 }}
                    whileInView={{ opacity: 1, y: 0 }}
                    transition={{ delay: i * 0.1 }}
                    viewport={{ once: true }}
                  >
                    <div className="group relative h-full">
                      <div className="absolute inset-0 bg-gradient-to-r from-indigo-500/10 to-purple-500/10 rounded-xl blur opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
                      <div className="relative bg-indigo-900/40 backdrop-blur border border-indigo-500/30 rounded-xl p-6 hover:border-indigo-500/50 transition-all duration-300 h-full">
                        <h3 className="font-semibold text-lg mb-2">{subsection.title}</h3>
                        <p className="text-sm text-gray-300 mb-4">{subsection.description}</p>
                        <ul className="space-y-3">
                          {subsection.details?.map((detail, j) => (
                            <li key={j} className="flex items-start gap-3">
                              <Check className="w-5 h-5 text-green-400 flex-shrink-0 mt-0.5" />
                              <span className="text-sm text-gray-300">{detail}</span>
                            </li>
                          ))}
                        </ul>
                      </div>
                    </div>
                  </motion.div>
                ))}
              </div>
            )}
          </motion.section>
        ))}
      </div>

      <Footer />
    </div>
  )
}
