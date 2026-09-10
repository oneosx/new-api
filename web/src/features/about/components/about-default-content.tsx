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
import {
  AlertCircle,
  ArrowRight,
  CheckCircle2,
  Cpu,
  FileText,
  HelpCircle,
  Key,
  Layers,
  Link2,
  ShieldCheck,
  Terminal,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const SUPPORTED_TOOLS = [
  'Claude Code',
  'Codex',
  'OpenCode',
  'Pi',
  'Qwen Code',
  'Openclaw',
  'Hermes',
]

export function AboutDefaultContent() {
  const { t } = useTranslation()

  const serviceTerms = [
    {
      id: 'T1',
      title: t('about.terms.t1.title'),
      desc: t('about.terms.t1.desc'),
    },
    {
      id: 'T2',
      title: t('about.terms.t2.title'),
      desc: t('about.terms.t2.desc'),
    },
    {
      id: 'T3',
      title: t('about.terms.t3.title'),
      desc: t('about.terms.t3.desc'),
    },
    {
      id: 'T4',
      title: t('about.terms.t4.title'),
      desc: t('about.terms.t4.desc'),
    },
    {
      id: 'T5',
      title: t('about.terms.t5.title'),
      desc: t('about.terms.t5.desc'),
    },
    {
      id: 'T6',
      title: t('about.terms.t6.title'),
      desc: t('about.terms.t6.desc'),
    },
    {
      id: 'T7',
      title: t('about.terms.t7.title'),
      desc: t('about.terms.t7.desc'),
    },
    {
      id: 'T8',
      title: t('about.terms.t8.title'),
      desc: t('about.terms.t8.desc'),
    },
  ]

  const statusCodes = [
    {
      code: '401',
      meaning: t('about.help.status.401.meaning'),
      action: t('about.help.status.401.action'),
    },
    {
      code: '402',
      meaning: t('about.help.status.402.meaning'),
      action: t('about.help.status.402.action'),
    },
    {
      code: '429',
      meaning: t('about.help.status.429.meaning'),
      action: t('about.help.status.429.action'),
    },
    {
      code: '5xx',
      meaning: t('about.help.status.5xx.meaning'),
      action: t('about.help.status.5xx.action'),
    },
  ]

  const workflowSteps = [
    {
      step: '1',
      title: t('about.help.step1.title'),
      desc: t('about.help.step1.desc'),
      icon: CheckCircle2,
    },
    {
      step: '2',
      title: t('about.help.step2.title'),
      desc: t('about.help.step2.desc'),
      icon: Key,
    },
    {
      step: '3',
      title: t('about.help.step3.title'),
      desc: t('about.help.step3.desc'),
      icon: Link2,
      code1: 'https://get.svip.app',
      code2: 'https://get.svip.app/v1',
    },
    {
      step: '4',
      title: t('about.help.step4.title'),
      desc: t('about.help.step4.desc'),
      icon: Terminal,
    },
  ]

  return (
    <div className='mx-auto max-w-5xl px-4 py-8 md:py-12 space-y-12 sm:space-y-16'>
      {/* Hero Header */}
      <div className='space-y-4 text-center sm:text-left border-b border-border/40 pb-8'>
        <div className='inline-flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-medium bg-primary/10 text-primary'>
          <Cpu className='size-3.5' />
          <span>get.svip.app</span>
        </div>
        <h1 className='text-3xl sm:text-4xl md:text-5xl font-bold tracking-tight text-foreground'>
          {t('about.hero.title')}
        </h1>
        <p className='text-muted-foreground text-sm sm:text-base md:text-lg max-w-2xl leading-relaxed'>
          {t('about.hero.subtitle')}
        </p>
      </div>

      {/* Section 1: What We Do */}
      <section className='space-y-6'>
        <div className='flex items-center gap-2.5'>
          <div className='size-8 rounded bg-primary/10 text-primary flex items-center justify-center shrink-0'>
            <Layers className='size-4' />
          </div>
          <h2 className='text-xl sm:text-2xl font-semibold tracking-tight text-foreground'>
            {t('about.whatWeDo.title')}
          </h2>
        </div>

        <Card className='rounded border border-border/60 bg-card/60 shadow-xs'>
          <CardHeader className='pb-4'>
            <CardDescription className='text-sm sm:text-base text-foreground/90 leading-relaxed'>
              {t('about.whatWeDo.description')}
            </CardDescription>
          </CardHeader>
          <CardContent className='space-y-6'>
            {/* Compatible Tools Tags */}
            <div className='space-y-2.5'>
              <div className='text-xs font-medium uppercase tracking-wider text-muted-foreground'>
                {t('about.whatWeDo.supportedClients')}
              </div>
              <div className='flex flex-wrap gap-2'>
                {SUPPORTED_TOOLS.map((tool) => (
                  <span
                    key={tool}
                    className='inline-flex items-center px-3 py-1 rounded text-xs font-medium bg-muted/60 text-foreground border border-border/40 hover:border-primary/40 transition-colors'
                  >
                    {tool}
                  </span>
                ))}
              </div>
            </div>

            <div className='grid grid-cols-1 md:grid-cols-2 gap-4 pt-2'>
              <div className='p-4 rounded border border-border/40 bg-muted/20 space-y-1.5'>
                <div className='text-sm font-semibold text-foreground'>
                  {t('about.whatWeDo.plansTitle')}
                </div>
                <p className='text-xs sm:text-sm text-muted-foreground leading-relaxed'>
                  {t('about.whatWeDo.plansDesc')}
                </p>
              </div>
              <div className='p-4 rounded border border-border/40 bg-muted/20 space-y-1.5'>
                <div className='text-sm font-semibold text-foreground'>
                  {t('about.whatWeDo.pricingTitle')}
                </div>
                <p className='text-xs sm:text-sm text-muted-foreground leading-relaxed'>
                  {t('about.whatWeDo.pricingDesc')}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </section>

      {/* Section 2: Terms of Service */}
      <section className='space-y-6'>
        <div className='flex flex-col sm:flex-row sm:items-baseline justify-between gap-2 border-b border-border/40 pb-3'>
          <div className='flex items-center gap-2.5'>
            <div className='size-8 rounded bg-primary/10 text-primary flex items-center justify-center shrink-0'>
              <FileText className='size-4' />
            </div>
            <h2 className='text-xl sm:text-2xl font-semibold tracking-tight text-foreground'>
              {t('about.terms.sectionTitle')}
            </h2>
          </div>
          <span className='text-xs text-muted-foreground'>
            {t('about.terms.notice')}
          </span>
        </div>

        <div className='grid grid-cols-1 md:grid-cols-2 gap-4'>
          {serviceTerms.map((term) => (
            <div
              key={term.id}
              className='p-4 sm:p-5 rounded border border-border/50 bg-card/60 hover:bg-card transition-colors flex flex-col justify-between space-y-2'
            >
              <div className='space-y-2'>
                <div className='flex items-center gap-2'>
                  <span className='px-1.5 py-0.5 rounded text-[11px] font-semibold bg-primary/15 text-primary tracking-wide'>
                    {term.id}
                  </span>
                  <h3 className='text-sm sm:text-base font-semibold text-foreground'>
                    {term.title}
                  </h3>
                </div>
                <p className='text-xs sm:text-sm text-muted-foreground leading-relaxed'>
                  {term.desc}
                </p>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* Section 3: Help & Guide */}
      <section className='space-y-6'>
        <div className='flex flex-col sm:flex-row sm:items-baseline justify-between gap-2 border-b border-border/40 pb-3'>
          <div className='flex items-center gap-2.5'>
            <div className='size-8 rounded bg-primary/10 text-primary flex items-center justify-center shrink-0'>
              <HelpCircle className='size-4' />
            </div>
            <h2 className='text-xl sm:text-2xl font-semibold tracking-tight text-foreground'>
              {t('about.help.title')}
            </h2>
          </div>
          <span className='text-xs text-muted-foreground'>
            {t('about.help.contactHint')}
          </span>
        </div>

        {/* 4 Steps */}
        <div className='grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4'>
          {workflowSteps.map((step) => {
            const Icon = step.icon
            return (
              <div
                key={step.step}
                className='p-4 rounded border border-border/50 bg-card/60 flex flex-col justify-between space-y-3'
              >
                <div className='space-y-2'>
                  <div className='flex items-center justify-between'>
                    <span className='size-6 rounded-full bg-primary/15 text-primary text-xs font-bold flex items-center justify-center'>
                      {step.step}
                    </span>
                    <Icon className='size-4 text-muted-foreground' />
                  </div>
                  <h4 className='text-sm font-semibold text-foreground'>
                    {step.title}
                  </h4>
                  <p className='text-xs text-muted-foreground leading-relaxed'>
                    {step.desc}
                  </p>
                </div>

                {step.code1 && (
                  <div className='space-y-1.5 pt-1 text-[11px] font-mono'>
                    <div className='flex items-center justify-between p-1.5 rounded bg-muted/50 border border-border/40'>
                      <span className='truncate text-foreground/80'>{step.code1}</span>
                      <CopyButton text={step.code1} />
                    </div>
                    <div className='flex items-center justify-between p-1.5 rounded bg-muted/50 border border-border/40'>
                      <span className='truncate text-foreground/80'>{step.code2}</span>
                      <CopyButton text={step.code2} />
                    </div>
                  </div>
                )}
              </div>
            )
          })}
        </div>

        {/* Status Code Reference: Mobile Card List + Desktop Table */}
        <div className='space-y-3 pt-2'>
          <div className='flex items-center gap-2'>
            <AlertCircle className='size-4 text-primary' />
            <h3 className='text-sm sm:text-base font-semibold text-foreground'>
              {t('about.help.statusTableTitle')}
            </h3>
          </div>

          {/* Mobile View: Vertical Cards */}
          <div className='grid grid-cols-1 gap-2.5 sm:hidden'>
            {statusCodes.map((row) => (
              <div
                key={row.code}
                className='p-3.5 rounded border border-border/50 bg-card/60 space-y-2'
              >
                <div className='flex items-center justify-between gap-2'>
                  <span className='px-2 py-0.5 rounded text-xs font-mono font-bold bg-primary/15 text-primary'>
                    {row.code}
                  </span>
                  <span className='text-xs font-medium text-foreground text-right'>
                    {row.meaning}
                  </span>
                </div>
                <div className='p-2 rounded bg-muted/40 text-xs text-muted-foreground leading-relaxed flex items-start gap-1.5'>
                  <span className='font-medium text-foreground/80 shrink-0'>{t('about.help.thAction')}:</span>
                  <span>{row.action}</span>
                </div>
              </div>
            ))}
          </div>

          {/* Tablet & Desktop View: Table */}
          <div className='hidden sm:block rounded border border-border/50 overflow-hidden'>
            <div className='overflow-x-auto'>
              <table className='w-full text-left text-xs sm:text-sm'>
                <thead className='bg-muted/40 border-b border-border/40 text-muted-foreground font-medium'>
                  <tr>
                    <th className='px-4 py-2.5 w-24 sm:w-28'>{t('about.help.thCode')}</th>
                    <th className='px-4 py-2.5 w-44 sm:w-56'>{t('about.help.thMeaning')}</th>
                    <th className='px-4 py-2.5'>{t('about.help.thAction')}</th>
                  </tr>
                </thead>
                <tbody className='divide-y divide-border/30 bg-card/40'>
                  {statusCodes.map((row) => (
                    <tr key={row.code} className='hover:bg-muted/20 transition-colors'>
                      <td className='px-4 py-2.5 font-mono font-semibold text-primary'>
                        {row.code}
                      </td>
                      <td className='px-4 py-2.5 font-medium text-foreground'>
                        {row.meaning}
                      </td>
                      <td className='px-4 py-2.5 text-muted-foreground leading-relaxed'>
                        {row.action}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>

        {/* FAQ Cards */}
        <div className='grid grid-cols-1 md:grid-cols-2 gap-4 pt-2'>
          <Card className='rounded border border-border/50 bg-card/60'>
            <CardHeader className='pb-2'>
              <CardTitle className='text-sm sm:text-base flex items-center gap-2'>
                <ShieldCheck className='size-4 text-primary' />
                <span>{t('about.help.faq1.question')}</span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className='text-xs sm:text-sm text-muted-foreground leading-relaxed'>
                {t('about.help.faq1.answer')}
              </p>
              <div className='mt-3'>
                <Link
                  to='/dashboard'
                  className='inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline'
                >
                  <span>{t('about.help.goToWallet')}</span>
                  <ArrowRight className='size-3' />
                </Link>
              </div>
            </CardContent>
          </Card>

          <Card className='rounded border border-border/50 bg-card/60'>
            <CardHeader className='pb-2'>
              <CardTitle className='text-sm sm:text-base flex items-center gap-2'>
                <Key className='size-4 text-primary' />
                <span>{t('about.help.faq2.question')}</span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className='text-xs sm:text-sm text-muted-foreground leading-relaxed'>
                {t('about.help.faq2.answer')}
              </p>
              <div className='mt-3'>
                <Link
                  to='/usage-logs/common'
                  className='inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline'
                >
                  <span>{t('about.help.goToLogs')}</span>
                  <ArrowRight className='size-3' />
                </Link>
              </div>
            </CardContent>
          </Card>
        </div>
      </section>
    </div>
  )
}
