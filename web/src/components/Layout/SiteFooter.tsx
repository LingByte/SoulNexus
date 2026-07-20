import { Link } from 'react-router-dom'
import { BookOpen, FileText, Github, Mail, Shield } from 'lucide-react'
import { useSiteConfig } from '@/contexts/siteConfig'
import { useTranslation } from '@/i18n'
import {
  contactEmailFromEnv,
  githubUrlFromEnv,
  icpLinkFromEnv,
  icpNumberFromEnv,
  privacyUrlFromEnv,
  publicSecurityLinkFromEnv,
  publicSecurityRecordFromEnv,
  resolveCopyrightCompany,
  resolveLogoUrl,
  termsUrlFromEnv,
} from '@/config/brandConfig'
import { BRAND_LOGO_SRC } from '@/utils/brandLogo'
import LandingSectionLink from '@/components/Home/LandingSectionLink'

type ProductLink = { name: string; sectionId: string }
type ExternalLink = { name: string; href: string }

export default function SiteFooter() {
  const { t } = useTranslation()
  const { config } = useSiteConfig()
  const year = new Date().getFullYear()
  const company = resolveCopyrightCompany(config.SITE_NAME, t('layout.siteName'))
  const logoUrl = resolveLogoUrl(config.SITE_LOGO_URL, BRAND_LOGO_SRC)

  const icpNumber = icpNumberFromEnv()
  const icpLink = icpLinkFromEnv()
  const psRecord = publicSecurityRecordFromEnv()
  const psLink = publicSecurityLinkFromEnv()
  const contactEmail = contactEmailFromEnv()
  const githubUrl = githubUrlFromEnv()
  const privacyUrl = privacyUrlFromEnv()
  const termsUrl = termsUrlFromEnv()

  const productLinks: ProductLink[] = [
    { name: t('footer.linkShowcase'), sectionId: 'platform-showcase' },
    { name: t('footer.linkFeatures'), sectionId: 'features' },
    { name: t('footer.linkKnowledge'), sectionId: 'knowledge' },
    { name: t('footer.linkCapabilities'), sectionId: 'more' },
  ]

  const resourceLinks: ExternalLink[] = [{ name: t('nav.docsCenter'), href: 'https://docs.lingecho.com/' }]

  const legalLinks: ExternalLink[] = [
    ...(privacyUrl ? [{ name: t('footer.privacy'), href: privacyUrl }] : []),
    ...(termsUrl ? [{ name: t('footer.terms'), href: termsUrl }] : []),
  ]

  const socialLinks = [
    ...(githubUrl ? [{ name: 'GitHub', href: githubUrl, icon: Github }] : []),
    ...(contactEmail ? [{ name: t('footer.contact'), href: `mailto:${contactEmail}`, icon: Mail }] : []),
  ]

  const linkClass =
    'text-sm text-[hsl(var(--muted-foreground))] transition-colors hover:text-violet-600 dark:hover:text-violet-400'

  return (
    <footer className="relative border-t border-[hsl(var(--border))] bg-[hsl(var(--background))]">
      <div className="pointer-events-none absolute inset-0 bg-gradient-to-b from-violet-50/40 to-transparent dark:from-violet-950/20" aria-hidden />

      <div className="relative mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="grid grid-cols-1 gap-8 py-12 md:grid-cols-2 lg:grid-cols-4">
          <div className="space-y-4">
            <Link to="/" className="group flex items-center gap-2.5">
              <img
                src={logoUrl}
                alt={t('layout.siteName')}
                className="logo-brand h-9 w-9 rounded-lg object-contain transition-transform group-hover:scale-105"
              />
              <span className="font-display text-lg font-bold text-violet-700 dark:text-violet-300">{t('layout.siteName')}</span>
            </Link>
            <p className="text-sm leading-relaxed text-[hsl(var(--muted-foreground))]">{t('footer.description')}</p>
            {socialLinks.length > 0 ? (
              <div className="flex items-center gap-2">
                {socialLinks.map((link) => {
                  const Icon = link.icon
                  return (
                    <a
                      key={link.name}
                      href={link.href}
                      target={link.href.startsWith('mailto:') ? undefined : '_blank'}
                      rel={link.href.startsWith('mailto:') ? undefined : 'noopener noreferrer'}
                      className="flex h-9 w-9 items-center justify-center rounded-lg bg-[hsl(var(--muted))] text-[hsl(var(--muted-foreground))] transition hover:bg-violet-100 hover:text-violet-700 dark:hover:bg-violet-950 dark:hover:text-violet-300"
                      title={link.name}
                    >
                      <Icon className="h-4 w-4" />
                    </a>
                  )
                })}
              </div>
            ) : null}
          </div>

          <div>
            <h3 className="mb-4 text-xs font-semibold uppercase tracking-wider text-[hsl(var(--foreground))]">
              {t('footer.products')}
            </h3>
            <ul className="space-y-2.5">
              {productLinks.map((link) => (
                <li key={link.sectionId}>
                  <LandingSectionLink sectionId={link.sectionId} className={linkClass}>
                    {link.name}
                  </LandingSectionLink>
                </li>
              ))}
            </ul>
          </div>

          <div>
            <h3 className="mb-4 text-xs font-semibold uppercase tracking-wider text-[hsl(var(--foreground))]">
              {t('footer.resources')}
            </h3>
            <ul className="space-y-2.5">
              {resourceLinks.map((link) => (
                <li key={link.href} className="flex items-center gap-2">
                  <BookOpen className="h-4 w-4 opacity-50" />
                  <a href={link.href} target="_blank" rel="noopener noreferrer" className={linkClass}>
                    {link.name}
                  </a>
                </li>
              ))}
            </ul>
          </div>

          <div>
            <h3 className="mb-4 text-xs font-semibold uppercase tracking-wider text-[hsl(var(--foreground))]">
              {t('footer.legal')}
            </h3>
            {legalLinks.length > 0 ? (
              <ul className="space-y-2.5">
                {legalLinks.map((link) => (
                  <li key={link.href} className="flex items-center gap-2">
                    {link.name === t('footer.privacy') ? (
                      <Shield className="h-4 w-4 opacity-50" />
                    ) : (
                      <FileText className="h-4 w-4 opacity-50" />
                    )}
                    <a href={link.href} target="_blank" rel="noopener noreferrer" className={linkClass}>
                      {link.name}
                    </a>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-[hsl(var(--muted-foreground))]">{t('footer.legalPlaceholder')}</p>
            )}
          </div>
        </div>

        <div className="border-t border-[hsl(var(--border))]" />

        <div className="flex flex-col items-center justify-between gap-3 py-6 md:flex-row">
          <div className="flex flex-col items-center gap-2 text-center text-xs text-[hsl(var(--muted-foreground))] md:flex-row md:text-left">
            <span>
              © {year} {company}
            </span>
            {icpNumber ? (
              <>
                <span className="hidden md:inline" aria-hidden>
                  ·
                </span>
                <a href={icpLink} target="_blank" rel="noopener noreferrer" className="hover:text-violet-600 dark:hover:text-violet-400">
                  {icpNumber}
                </a>
              </>
            ) : null}
            {psRecord ? (
              <>
                <span className="hidden md:inline" aria-hidden>
                  ·
                </span>
                {psLink ? (
                  <a href={psLink} target="_blank" rel="noopener noreferrer" className="hover:text-violet-600 dark:hover:text-violet-400">
                    {psRecord}
                  </a>
                ) : (
                  <span>{psRecord}</span>
                )}
              </>
            ) : null}
          </div>
          <p className="text-xs text-[hsl(var(--muted-foreground))]">
            {t('footer.madeWith')} <span className="text-red-500">♥</span> {t('footer.by')} {t('footer.team')}
          </p>
        </div>
      </div>
    </footer>
  )
}
