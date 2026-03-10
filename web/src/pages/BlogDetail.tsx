import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { motion } from 'framer-motion'
import { Calendar, User, Tag, ArrowLeft, Clock, Share2 } from 'lucide-react'
import { useI18nStore } from '@/stores/i18nStore'
import PageSEO from '@/components/SEO/PageSEO'
import LoadingAnimation from '@/components/Animations/LoadingAnimation'
import MarkdownPreview from '@/components/UI/MarkdownPreview'
import blogs from '@/data/blogs.json'

interface BlogPost {
  id: string
  title: string
  titleZh?: string
  titleJa?: string
  excerpt: string
  excerptZh?: string
  excerptJa?: string
  content: string
  contentZh?: string
  contentJa?: string
  author: string
  date: string
  category: string
  categoryZh?: string
  categoryJa?: string
  image: string
  tags: string[]
  readTime?: number
}

const BlogDetail = () => {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t, language } = useI18nStore()
  const [blog, setBlog] = useState<BlogPost | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // 模拟加载
    setLoading(true)
    setTimeout(() => {
      const foundBlog = blogs.find((b) => b.id === id)
      setBlog(foundBlog as BlogPost || null)
      setLoading(false)
    }, 300)
  }, [id])

  const getBlogTitle = (blog: BlogPost) => {
    if (language === 'zh') return blog.titleZh || blog.title
    if (language === 'ja') return blog.titleJa || blog.title
    return blog.title
  }

  const getBlogContent = (blog: BlogPost) => {
    if (language === 'zh') return blog.contentZh || blog.content
    if (language === 'ja') return blog.contentJa || blog.content
    return blog.content
  }

  const getBlogCategory = (blog: BlogPost) => {
    if (language === 'zh') return blog.categoryZh || blog.category
    if (language === 'ja') return blog.categoryJa || blog.category
    return blog.category
  }

  const handleShare = () => {
    if (navigator.share) {
      navigator.share({
        title: blog ? getBlogTitle(blog) : '',
        url: window.location.href,
      })
    } else {
      navigator.clipboard.writeText(window.location.href)
      alert(t('blog.linkCopied') || 'Link copied to clipboard!')
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-900 dark:to-indigo-950/30">
        <div className="text-center">
          <LoadingAnimation type="progress" size="lg" className="mb-4" />
          <p className="text-gray-600 dark:text-gray-400">{t('blog.loading') || 'Loading blog post...'}</p>
        </div>
      </div>
    )
  }

  if (!blog) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-900 dark:to-indigo-950/30">
        <div className="text-center">
          <h1 className="text-4xl font-bold text-gray-900 dark:text-white mb-4">
            {t('blog.notFound') || 'Blog Post Not Found'}
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mb-8">
            {t('blog.notFoundDesc') || 'The blog post you are looking for does not exist.'}
          </p>
          <button
            onClick={() => navigate('/blog')}
            className="inline-flex items-center gap-2 px-6 py-3 bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            {t('blog.backToBlog') || 'Back to Blog'}
          </button>
        </div>
      </div>
    )
  }

  return (
    <>
      <PageSEO
        title={getBlogTitle(blog)}
        description={blog.excerpt}
      />
      
      <div className="min-h-screen bg-gradient-to-br from-indigo-50 via-white to-purple-50 dark:from-gray-900 dark:via-gray-900 dark:to-indigo-950/30">
        {/* Hero Section */}
        <section className="relative py-12 overflow-hidden">
          <div className="absolute inset-0 bg-gradient-to-br from-indigo-100/50 via-purple-100/30 to-blue-100/50 dark:from-indigo-950/30 dark:via-purple-950/20 dark:to-blue-950/30" />
          
          <div className="max-w-4xl mx-auto px-4 relative z-10">
            {/* Back Button */}
            <motion.button
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              onClick={() => navigate('/blog')}
              className="inline-flex items-center gap-2 text-gray-600 dark:text-gray-400 hover:text-indigo-600 dark:hover:text-indigo-400 mb-8 transition-colors"
            >
              <ArrowLeft className="w-4 h-4" />
              {t('blog.backToBlog') || 'Back to Blog'}
            </motion.button>

            {/* Category Badge */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.1 }}
              className="mb-4"
            >
              <span className="px-4 py-2 rounded-full text-sm font-semibold bg-gradient-to-r from-indigo-500 to-purple-600 shadow-lg">
                {getBlogCategory(blog)}
              </span>
            </motion.div>

            {/* Title */}
            <motion.h1
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.2 }}
              className="text-4xl md:text-5xl font-bold text-gray-900 dark:text-white mb-6 leading-tight"
            >
              {getBlogTitle(blog)}
            </motion.h1>

            {/* Meta Info */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.3 }}
              className="flex flex-wrap items-center gap-6 text-gray-600 dark:text-gray-400 mb-8"
            >
              <div className="flex items-center gap-2">
                <User className="w-5 h-5" />
                <span>{blog.author}</span>
              </div>
              <div className="flex items-center gap-2">
                <Calendar className="w-5 h-5" />
                <span>{blog.date}</span>
              </div>
              {blog.readTime && (
                <div className="flex items-center gap-2">
                  <Clock className="w-5 h-5" />
                  <span>{blog.readTime} {t('blog.minRead') || 'min read'}</span>
                </div>
              )}
              <button
                onClick={handleShare}
                className="flex items-center gap-2 hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors"
              >
                <Share2 className="w-5 h-5" />
                <span>{t('blog.share') || 'Share'}</span>
              </button>
            </motion.div>

            {/* Tags */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.4 }}
              className="flex flex-wrap gap-2"
            >
              {blog.tags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center gap-1 px-3 py-1 rounded-full text-sm bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 border border-gray-200 dark:border-gray-700"
                >
                  <Tag className="w-3 h-3" />
                  {tag}
                </span>
              ))}
            </motion.div>
          </div>
        </section>

        {/* Featured Image */}
        <section className="relative py-8">
          <div className="max-w-4xl mx-auto px-4">
            <motion.div
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ delay: 0.5 }}
              className="rounded-2xl overflow-hidden shadow-2xl"
            >
              <img
                src={blog.image}
                alt={getBlogTitle(blog)}
                className="w-full h-auto object-cover"
              />
            </motion.div>
          </div>
        </section>

        {/* Content */}
        <section className="relative py-12 pb-24">
          <div className="max-w-4xl mx-auto px-4">
            <motion.article
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.6 }}
              className="bg-white dark:bg-gray-800 rounded-2xl shadow-lg border border-gray-200/50 dark:border-gray-700/50 p-8 md:p-12"
            >
              <MarkdownPreview content={getBlogContent(blog)} />
            </motion.article>
          </div>
        </section>
      </div>
    </>
  )
}

export default BlogDetail
