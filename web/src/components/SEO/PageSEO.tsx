import { useEffect } from 'react'

interface PageSEOProps {
    title: string
    description: string
    keywords?: string
    ogImage?: string
    canonical?: string
    structuredData?: object
}

/**
 * PageSEO组件 - 用于动态设置页面SEO元数据
 * 
 * @param title - 页面标题
 * @param description - 页面描述
 * @param keywords - 关键词（可选）
 * @param ogImage - Open Graph图片（可选）
 * @param canonical - 规范链接（可选）
 * @param structuredData - 结构化数据（可选）
 */
export const PageSEO = ({
    title,
    description,
    keywords,
    ogImage,
    canonical,
    structuredData
}: PageSEOProps) => {
    useEffect(() => {
        // 设置页面标题
        document.title = title

        // 更新meta标签
        const updateMetaTag = (name: string, content: string, isProperty = false) => {
            const attribute = isProperty ? 'property' : 'name'
            let meta = document.querySelector(`meta[${attribute}="${name}"]`)
            
            if (!meta) {
                meta = document.createElement('meta')
                meta.setAttribute(attribute, name)
                document.head.appendChild(meta)
            }
            
            meta.setAttribute('content', content)
        }

        // 基础meta标签
        updateMetaTag('description', description)
        if (keywords) {
            updateMetaTag('keywords', keywords)
        }

        // Open Graph标签
        updateMetaTag('og:title', title, true)
        updateMetaTag('og:description', description, true)
        if (ogImage) {
            updateMetaTag('og:image', ogImage, true)
        }

        // Twitter Card标签
        updateMetaTag('twitter:title', title)
        updateMetaTag('twitter:description', description)
        if (ogImage) {
            updateMetaTag('twitter:image', ogImage)
        }

        // Canonical链接
        if (canonical) {
            let link = document.querySelector('link[rel="canonical"]')
            if (!link) {
                link = document.createElement('link')
                link.setAttribute('rel', 'canonical')
                document.head.appendChild(link)
            }
            link.setAttribute('href', canonical)
        }

        // 结构化数据
        if (structuredData) {
            let script = document.querySelector('script[type="application/ld+json"][data-page-seo]')
            if (!script) {
                script = document.createElement('script')
                script.setAttribute('type', 'application/ld+json')
                script.setAttribute('data-page-seo', 'true')
                document.head.appendChild(script)
            }
            script.textContent = JSON.stringify(structuredData)
        }

        // 清理函数
        return () => {
            // 可选：在组件卸载时清理动态添加的标签
        }
    }, [title, description, keywords, ogImage, canonical, structuredData])

    return null // 这个组件不渲染任何内容
}

export default PageSEO
