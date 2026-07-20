import type { LucideIcon } from 'lucide-react'
import { motion } from 'framer-motion'
import { cn } from '@/utils/cn'

const TAG_GRADIENTS = [
  'from-violet-500/20 via-violet-500/10 to-transparent text-violet-800 dark:text-violet-100 border-violet-400/50 dark:border-violet-400/30',
  'from-purple-500/20 via-purple-500/10 to-transparent text-purple-800 dark:text-purple-100 border-purple-400/50 dark:border-purple-400/30',
  'from-fuchsia-500/20 via-fuchsia-500/10 to-transparent text-fuchsia-800 dark:text-fuchsia-100 border-fuchsia-400/50 dark:border-fuchsia-400/30',
  'from-cyan-500/20 via-cyan-500/10 to-transparent text-cyan-800 dark:text-cyan-100 border-cyan-400/50 dark:border-cyan-400/30',
  'from-emerald-500/20 via-emerald-500/10 to-transparent text-emerald-800 dark:text-emerald-100 border-emerald-400/50 dark:border-emerald-400/30',
] as const

type FeatureGridCardProps = {
  icon: LucideIcon
  title: string
  description: string
  tags?: string[]
  index?: number
  tall?: boolean
}

export default function FeatureGridCard({ icon: Icon, title, description, tags, index = 0, tall = true }: FeatureGridCardProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 24 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: '-60px' }}
      transition={{ duration: 0.5, delay: index * 0.06 }}
      className="group relative h-full rounded-2xl bg-gradient-to-br from-violet-500/35 via-purple-500/20 to-transparent p-[1px]"
    >
      <div
        className={cn(
          'relative flex h-full flex-col overflow-hidden rounded-2xl border border-[hsl(var(--border)/0.8)] bg-[hsl(var(--card)/0.92)] p-6 shadow-[0_10px_40px_-12px_rgba(124,58,237,0.25)] backdrop-blur-md transition duration-300 group-hover:border-violet-400/40 group-hover:shadow-[0_16px_48px_-12px_rgba(124,58,237,0.35)]',
          tall ? 'min-h-[320px]' : 'min-h-[240px]',
        )}
      >
        <div
          className="pointer-events-none absolute -inset-24 opacity-60 bg-[radial-gradient(circle_at_20%_20%,rgba(139,92,246,0.12),transparent_42%),radial-gradient(circle_at_80%_120%,rgba(168,85,247,0.1),transparent_40%)]"
          aria-hidden
        />
        <div className="relative flex-1">
          <div className="mb-4 inline-flex h-11 w-11 items-center justify-center rounded-xl border border-violet-200/50 bg-gradient-to-br from-violet-500/25 to-purple-500/25 text-violet-700 dark:border-violet-800/50 dark:text-violet-200">
            <Icon className="h-5 w-5" />
          </div>
          <h3 className="text-lg font-semibold tracking-tight">{title}</h3>
          <p className="mt-2 text-sm leading-relaxed text-[hsl(var(--muted-foreground))]">{description}</p>
        </div>
        {tags && tags.length > 0 ? (
          <div className="relative mt-4 flex flex-wrap gap-2">
            {tags.map((tag, i) => (
              <span
                key={tag}
                className={cn('rounded-full border px-2.5 py-1 text-xs bg-gradient-to-r', TAG_GRADIENTS[i % TAG_GRADIENTS.length])}
              >
                {tag}
              </span>
            ))}
          </div>
        ) : null}
      </div>
    </motion.div>
  )
}
