import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ChevronDown, ArrowRight } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import products from '@/data/products.json'

export default function ProductNav() {
  const [isOpen, setIsOpen] = useState(false)

  const menuVariants = {
    hidden: {
      opacity: 0,
      y: -10,
    },
    visible: {
      opacity: 1,
      y: 0,
      transition: {
        duration: 0.3,
        ease: 'easeOut',
      },
    },
    exit: {
      opacity: 0,
      y: -10,
      transition: {
        duration: 0.2,
        ease: 'easeIn',
      },
    },
  }

  const containerVariants = {
    hidden: { opacity: 0 },
    visible: {
      opacity: 1,
      transition: {
        staggerChildren: 0.05,
        delayChildren: 0.1,
      },
    },
  }

  const itemVariants = {
    hidden: { opacity: 0, y: 10 },
    visible: {
      opacity: 1,
      y: 0,
      transition: {
        duration: 0.3,
        ease: 'easeOut',
      },
    },
  }

  return (
    <div className="relative">
      <button
        className="flex items-center gap-1 text-muted-foreground hover:text-foreground transition-colors py-2"
        onMouseEnter={() => setIsOpen(true)}
        onMouseLeave={() => setIsOpen(false)}
      >
        Products
        <motion.div
          animate={{ rotate: isOpen ? 180 : 0 }}
          transition={{ duration: 0.3 }}
        >
          <ChevronDown className="w-4 h-4" />
        </motion.div>
      </button>

      {/* Full-width Dropdown Menu with Animation */}
      <AnimatePresence>
        {isOpen && (
          <motion.div
            className="fixed left-0 right-0 top-16 w-screen bg-background/95 backdrop-blur-md border-b border-border shadow-2xl z-50"
            onMouseEnter={() => setIsOpen(true)}
            onMouseLeave={() => setIsOpen(false)}
            variants={menuVariants}
            initial="hidden"
            animate="visible"
            exit="exit"
          >
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
              <motion.div
                className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8"
                variants={containerVariants}
                initial="hidden"
                animate="visible"
              >
                {products.products.map((product) => (
                  <motion.div key={product.id} variants={itemVariants}>
                    <Link
                      to={`/product/${product.id}`}
                      className="group p-6 rounded-lg border border-border hover:border-indigo-500/50 hover:bg-accent/50 transition-all duration-300 block h-full"
                    >
                      <div className="flex items-start justify-between mb-4">
                        <div>
                          <h3 className="text-lg font-semibold text-foreground group-hover:text-indigo-400 transition-colors">
                            {product.name}
                          </h3>
                          <p className="text-sm text-muted-foreground mt-1">{product.shortDesc}</p>
                        </div>
                        <motion.div
                          className="flex-shrink-0"
                          whileHover={{ x: 4 }}
                          transition={{ duration: 0.2 }}
                        >
                          <ArrowRight className="w-5 h-5 text-muted-foreground group-hover:text-indigo-400 transition-colors" />
                        </motion.div>
                      </div>
                      
                      <p className="text-sm text-gray-400 mb-4">{product.description}</p>
                      
                      {product.features && (
                        <div className="space-y-2">
                          {product.features.slice(0, 3).map((feature, idx) => (
                            <motion.div
                              key={idx}
                              className="flex items-center gap-2 text-xs text-muted-foreground"
                              initial={{ opacity: 0, x: -10 }}
                              animate={{ opacity: 1, x: 0 }}
                              transition={{ delay: 0.15 + idx * 0.05 }}
                            >
                              <div className="w-1.5 h-1.5 rounded-full bg-indigo-400"></div>
                              {feature}
                            </motion.div>
                          ))}
                        </div>
                      )}
                    </Link>
                  </motion.div>
                ))}
              </motion.div>

              {/* Bottom section with CTA */}
              <motion.div
                className="mt-12 pt-8 border-t border-border"
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.3, duration: 0.3 }}
              >
                <div className="flex flex-col sm:flex-row items-center justify-between gap-6">
                  <div>
                    <h4 className="text-lg font-semibold mb-2">Explore All Products</h4>
                    <p className="text-sm text-muted-foreground">Discover our complete suite of solutions</p>
                  </div>
                  <motion.div
                    whileHover={{ scale: 1.05 }}
                    whileTap={{ scale: 0.95 }}
                  >
                    <Link
                      to="/products"
                      className="px-6 py-2 rounded-lg bg-indigo-500 hover:bg-indigo-600 text-white font-medium transition-colors whitespace-nowrap"
                    >
                      View All Products
                    </Link>
                  </motion.div>
                </div>
              </motion.div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
