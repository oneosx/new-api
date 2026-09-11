/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useCallback, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { Footer } from '@/components/layout/components/footer'
import { RichContent } from '@/components/rich-content'
import { useTheme } from '@/context/theme-provider'
import { isLikelyHtml } from '@/lib/content-format'
import { useAuthStore } from '@/stores/auth-store'

import { CodingAgentHero } from './components/coding-agent-hero'
import { IntegrationSection } from './components/integration-section'
import { EfficiencyCtaSection } from './components/efficiency-cta-section'
import { useHomePageContent } from './hooks'

export function Home() {
  const { i18n, t } = useTranslation()
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const { resolvedTheme } = useTheme()
  const { auth } = useAuthStore()
  const isAuthenticated = !!auth.user
  const { content, isLoaded, isUrl } = useHomePageContent()

  const syncIframePreferences = useCallback(() => {
    try {
      iframeRef.current?.contentWindow?.postMessage(
        { themeMode: resolvedTheme },
        '*'
      )
      iframeRef.current?.contentWindow?.postMessage(
        { lang: i18n.language },
        '*'
      )
    } catch {
      // Cross-origin frames may reject access while navigating.
    }
  }, [i18n.language, resolvedTheme])

  useEffect(() => {
    if (isUrl) {
      syncIframePreferences()
    }
  }, [isUrl, syncIframePreferences])

  if (!isLoaded) {
    return (
      <PublicLayout showMainContainer={false}>
        <main className='flex min-h-screen items-center justify-center'>
          <div className='text-muted-foreground'>{t('Loading...')}</div>
        </main>
      </PublicLayout>
    )
  }

  // If admin configured an external URL, use dedicated full-screen iframe
  if (content && isUrl) {
    return (
      <PublicLayout showMainContainer={false}>
        <iframe
          ref={iframeRef}
          src={content}
          className='h-screen w-full border-none'
          title={t('Custom Home Page')}
          sandbox='allow-forms allow-popups allow-popups-to-escape-sandbox allow-scripts allow-top-navigation-by-user-activation'
          onLoad={syncIframePreferences}
        />
      </PublicLayout>
    )
  }

  const contentIsHtml = Boolean(content && isLikelyHtml(content))

  return (
    <PublicLayout showMainContainer={false}>
      <main className='flex flex-1 flex-col items-center w-full'>
        <CodingAgentHero />
        {/* Subtle gradient divider between Hero and Integration */}
        <div className='w-full max-w-7xl px-2'>
          <div className='h-px w-full bg-gradient-to-r from-transparent via-border/80 to-transparent' />
        </div>
        <IntegrationSection />

        {/* Admin-configured custom home content (Markdown/HTML) rendered right below Integration */}
        {content && (
          <>
            <div className='w-full max-w-7xl px-2'>
              <div className='h-px w-full bg-gradient-to-r from-transparent via-border/80 to-transparent' />
            </div>
            <section className='w-full max-w-7xl p-2 my-16'>
              <div className='rounded border border-border/50 bg-card/60 p-6 sm:p-8 backdrop-blur-xs'>
                <RichContent
                  mode={contentIsHtml ? 'html' : 'markdown'}
                  htmlVariant={contentIsHtml ? 'isolated' : undefined}
                  content={content}
                  className='custom-home-content prose-neutral dark:prose-invert max-w-none'
                />
              </div>
            </section>
          </>
        )}

        {/* Subtle gradient divider before Efficiency Section */}
        <div className='w-full max-w-7xl px-2'>
          <div className='h-px w-full bg-gradient-to-r from-transparent via-border/80 to-transparent' />
        </div>
        <EfficiencyCtaSection />
      </main>
      <Footer simple />
    </PublicLayout>
  )
}
