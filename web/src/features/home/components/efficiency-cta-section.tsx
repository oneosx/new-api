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
import { ArrowRight, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'

export function EfficiencyCtaSection() {
  const { t } = useTranslation()

  return (
    <section className='w-full max-w-7xl mx-auto my-16 p-4'>
      <div className='relative overflow-hidden rounded border border-border/60 bg-card/60 px-6 py-10 sm:px-12 sm:py-14 text-center md:text-left flex flex-col md:flex-row md:items-center md:justify-between gap-8 backdrop-blur-xs'>
        {/* Subtle decorative background glow */}
        <div
          aria-hidden='true'
          className='pointer-events-none absolute -right-20 -top-20 size-72 rounded-full bg-primary/10 blur-3xl'
        />
        <div
          aria-hidden='true'
          className='pointer-events-none absolute -left-20 -bottom-20 size-72 rounded-full bg-primary/5 blur-3xl'
        />

        {/* Content */}
        <div className='relative z-10 max-w-2xl space-y-3 sm:space-y-4'>
          <div className='inline-flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-medium bg-primary/10 text-primary'>
            <Zap className='size-3.5' />
            <span>{t('home.efficiency.badge')}</span>
          </div>
          <h2 className='text-2xl sm:text-3xl md:text-4xl font-bold tracking-tight text-foreground'>
            {t('home.efficiency.title')}
          </h2>
          <p className='text-xs sm:text-sm md:text-base text-muted-foreground leading-relaxed'>
            {t('home.efficiency.description')}
          </p>
        </div>

        {/* Action Button */}
        <div className='relative z-10 shrink-0 flex justify-center md:justify-end'>
          <Link
            to='/dashboard'
            className='group inline-flex items-center gap-2 rounded-full bg-primary px-7 py-3 text-sm font-medium text-primary-foreground shadow-xs transition-all duration-200 hover:opacity-90 hover:shadow-md active:scale-[0.98]'
          >
            <span>{t('home.efficiency.button')}</span>
            <ArrowRight className='size-4 transition-transform duration-200 group-hover:translate-x-0.5' />
          </Link>
        </div>
      </div>
    </section>
  )
}
