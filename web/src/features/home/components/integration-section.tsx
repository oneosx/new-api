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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'

export function IntegrationSection() {
  const { t } = useTranslation()

  const claudeComment = t('home.integration.claudeCodeComment')
  const openaiComment = t('home.integration.openaiComment')
  const tokenPlaceholder = t('home.integration.tokenPlaceholder')

  const codeSnippet = `${claudeComment}
export ANTHROPIC_BASE_URL="https://get.svip.app"
export ANTHROPIC_AUTH_TOKEN="${tokenPlaceholder}"

${openaiComment}
export OPENAI_BASE_URL="https://get.svip.app/v1"
export OPENAI_API_KEY="${tokenPlaceholder}"`

  return (
    <section className='w-full max-w-7xl mx-auto px-6 py-16 lg:py-24'>
      <div className='grid grid-cols-1 lg:grid-cols-12 gap-12 lg:gap-16 items-start'>
        {/* Left Column: Heading & Integration Table */}
        <div className='lg:col-span-6 flex flex-col'>
          <h2 className='text-3xl sm:text-4xl font-medium tracking-tight text-foreground'>
            {t('home.integration.title')}
          </h2>
          <p className='mt-3 text-base text-muted-foreground'>
            {t('home.integration.subtitle')}
          </p>

          {/* Protocols & Tools Table */}
          <div className='mt-10 divide-y divide-border/60 border-t border-b border-border/60'>
            {/* Row 1: Anthropic */}
            <div className='py-5 grid grid-cols-12 gap-4 items-center'>
              <div className='col-span-4 sm:col-span-3 text-base font-medium text-foreground'>
                {t('home.integration.anthropicProtocol')}
              </div>
              <div className='col-span-8 sm:col-span-9 text-sm sm:text-base text-muted-foreground'>
                Claude Code
              </div>
            </div>

            {/* Row 2: OpenAI */}
            <div className='py-5 grid grid-cols-12 gap-4 items-center'>
              <div className='col-span-4 sm:col-span-3 text-base font-medium text-foreground'>
                {t('home.integration.openaiProtocol')}
              </div>
              <div className='col-span-8 sm:col-span-9 text-sm sm:text-base text-muted-foreground'>
                Codex · OpenCode · Pi · Opencode · Cline...
              </div>
            </div>

            {/* Row 3: SDK */}
            <div className='py-5 grid grid-cols-12 gap-4 items-center'>
              <div className='col-span-4 sm:col-span-3 text-base font-medium text-foreground'>
                {t('home.integration.sdks')}
              </div>
              <div className='col-span-8 sm:col-span-9 text-sm sm:text-base text-muted-foreground'>
                {t('home.integration.officialSdks')}
              </div>
            </div>
          </div>
        </div>

        {/* Right Column: Terminal Code Card & Footer Note */}
        <div className='lg:col-span-6 flex flex-col'>
          {/* Terminal Window Card */}
          <div className='relative rounded-xl overflow-hidden bg-[#1e1c18] dark:bg-[#07242c] text-[#eee8d5] shadow-xl border border-black/10 dark:border-white/10'>
            {/* Top Bar with Copy Button */}
            <div className='flex items-center justify-between px-4 py-2.5 border-b border-white/5 bg-black/20'>
              <div className='flex items-center gap-1.5' aria-hidden='true'>
                <span className='size-2.5 rounded-full bg-red-500/70' />
                <span className='size-2.5 rounded-full bg-yellow-500/70' />
                <span className='size-2.5 rounded-full bg-green-500/70' />
              </div>
              <CopyButton
                value={codeSnippet}
                className='text-white/60 hover:text-white hover:bg-white/10 size-7'
              />
            </div>

            {/* Terminal Code Body */}
            <pre className='p-5 sm:p-6 text-[13px] sm:text-[14px] font-mono leading-relaxed overflow-x-auto whitespace-pre'>
              <code>
                <span className='text-muted-foreground/60'>{claudeComment}</span>{'\n'}
                <span className='text-[#cb4b16]'>export</span> ANTHROPIC_BASE_URL=<span className='text-[#b58900]'>&quot;https://get.svip.app&quot;</span>{'\n'}
                <span className='text-[#cb4b16]'>export</span> ANTHROPIC_AUTH_TOKEN=<span className='text-[#b58900]'>&quot;{tokenPlaceholder}&quot;</span>{'\n'}
                {'\n'}
                <span className='text-muted-foreground/60'>{openaiComment}</span>{'\n'}
                <span className='text-[#cb4b16]'>export</span> OPENAI_BASE_URL=<span className='text-[#b58900]'>&quot;https://get.svip.app/v1&quot;</span>{'\n'}
                <span className='text-[#cb4b16]'>export</span> OPENAI_API_KEY=<span className='text-[#b58900]'>&quot;{tokenPlaceholder}&quot;</span>
              </code>
            </pre>
          </div>

          {/* Under-Terminal Documentation Footnote */}
          <p className='mt-4 text-xs sm:text-sm text-muted-foreground'>
            {t('home.integration.footnotePrefix')}
            <Link
              to='/about'
              className='text-foreground underline underline-offset-4 hover:text-primary transition-colors ml-1'
            >
              {t('home.integration.footnoteDocs')}
            </Link>
            。
          </p>
        </div>
      </div>
    </section>
  )
}
