import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { ArrowRight, Github, ExternalLink } from 'lucide-react'
import Footer from '@/components/Layout/Footer'

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
    <div className="min-h-screen relative">
      {/* Background */}
      <div className="pointer-events-none absolute inset-0 -z-20">
        <div className="absolute inset-0 bg-[radial-gradient(1200px_600px_at_50%_-10%,rgba(59,130,246,0.25),transparent),radial-gradient(1000px_500px_at_100%_20%,rgba(147,51,234,0.22),transparent),linear-gradient(180deg,#0B1020, #0E1224_40%, #0B1020)] dark:bg-[radial-gradient(1200px_600px_at_50%_-10%,rgba(59,130,246,0.15),transparent),radial-gradient(1000px_500px_at_100%_20%,rgba(147,51,234,0.12),transparent),linear-gradient(180deg,#1a1a2e, #2d2d44_40%, #1a1a2e)]" />
        <div className="absolute inset-0 opacity-30 [background-image:linear-gradient(to_right,rgba(255,255,255,0.06)_1px,transparent_1px),linear-gradient(to_bottom,rgba(255,255,255,0.06)_1px,transparent_1px)] [background-size:26px_26px]" />
      </div>

      {/* Hero Section */}
      <section className="relative py-20 px-4 sm:px-6 lg:px-8 pt-32">
        <div className="max-w-6xl mx-auto">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
            className="text-center mb-16"
          >
            <h1 className="text-5xl md:text-6xl font-bold mb-4 bg-gradient-to-r from-indigo-400 via-purple-400 to-blue-400 bg-clip-text">
              Our Products
            </h1>
            <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
              Powerful solutions for voice, communication, and data management
            </p>
          </motion.div>

          {/* Products Grid */}
          {loading ? (
            <div className="flex items-center justify-center py-20">
              <div className="text-center">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-500 mx-auto mb-4"></div>
                <p className="text-muted-foreground">Loading products...</p>
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
                  <div className="absolute inset-0 bg-gradient-to-r from-indigo-500/20 to-purple-500/20 rounded-2xl blur-xl opacity-0 group-hover:opacity-100 transition-opacity duration-500" />
                  
                  <div className="relative bg-gradient-to-br from-indigo-900/40 to-purple-900/40 backdrop-blur-xl border border-indigo-500/30 rounded-2xl p-8 hover:border-indigo-500/50 transition-all duration-300 h-full flex flex-col">
                    {/* Header */}
                    <div className="mb-6">
                      {product.logo ? (
                        <img 
                          src={product.logo} 
                          alt={product.name}
                          className="w-10 mb-4"
                        />
                      ) : (
                        <div className="inline-flex items-center justify-center w-12 h-12 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-500 mb-4">
                          <span className="text-xl font-bold">{product.name.charAt(0)}</span>
                        </div>
                      )}
                      <h3 className="text-2xl font-bold mb-2">{product.title}</h3>
                      <p className="text-gray-300">{product.description}</p>
                    </div>

                    {/* Links */}
                    {product.links && (
                      <div className="flex gap-3 mb-6 flex-wrap">
                        {product.links.github && (
                          <a
                            href={product.links.github}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-indigo-800/50 hover:bg-indigo-700/50 text-sm text-gray-200 hover:text-white transition-colors"
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
                            className="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-indigo-800/50 hover:bg-indigo-700/50 text-sm text-gray-200 hover:text-white transition-colors"
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
                        onClick={() => navigate(`/product/${product.id}`)}
                        className="w-full inline-flex items-center justify-center gap-2 px-4 py-3 rounded-lg bg-gradient-to-r from-indigo-500 to-purple-500 hover:from-indigo-600 hover:to-purple-600 font-semibold transition-all duration-300 group/btn"
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
