import { Link } from '@tanstack/react-router'
import type React from 'react'

import { HeaderLogo } from '@/components/layout/components/header-logo'
import { SolarSystemGalaxy } from '@/components/solar-system-galaxy'
import { Skeleton } from '@/components/ui/skeleton'
import { useSystemConfig } from '@/hooks/use-system-config'

type AuthLayoutProps = {
  children: React.ReactNode
  title?: React.ReactNode
  description?: React.ReactNode
}

export function AuthLayout({ children, title, description }: AuthLayoutProps) {
  const { systemName, logo, loading, logoLoaded } = useSystemConfig()

  return (
    <div className='relative w-full min-h-svh bg-background flex items-center justify-center overflow-hidden'>
      <div className='relative z-0 flex items-center justify-center'>
        <SolarSystemGalaxy stageClassName='h-svh' planeClassName='scale-75 md:scale-150 lg:scale-200' />
      </div>
      <div className="z-20 absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 w-full max-w-[92%] sm:max-w-md md:max-w-3/5 lg:max-w-2/5 p-2 sm:p-4">
        <div className='backdrop-blur-sm bg-white/10 border border-border/50 rounded-2xl px-6 py-8 sm:px-12 sm:py-10 flex flex-col items-center justify-center shadow-[0_0_20px_rgba(0,0,0,0.1)] shadow-foreground/10 transition-all duration-300'>
          {/* Header Row: 手机端上下居中排列，桌面端 (sm:) 保持同行 LOGO | 标题 */}
          <div className='mb-8 flex w-full flex-col items-center text-center sm:flex-row sm:items-center sm:text-left sm:gap-4'>
            {/* Logo */}
            <Link
              to='/'
              className='mb-4 flex shrink-0 items-center justify-center transition-opacity hover:opacity-80 sm:mb-0'
              aria-label={systemName}
            >
              <div className='flex h-9 max-h-9 max-w-[160px] items-center justify-center transition-all duration-300'>
                {loading ? (
                  <Skeleton className='h-8 w-28 rounded' />
                ) : (
                  <HeaderLogo
                    src={logo}
                    darkSrc='/logo.white.png'
                    loading={loading}
                    logoLoaded={logoLoaded}
                    className='h-8 max-h-8 w-auto max-w-[160px] rounded-none object-contain'
                  />
                )}
              </div>
            </Link>

            {/* 分割竖线（仅在桌面端显示） */}
            {(title || description) && (
              <div className='hidden h-8 w-[1px] shrink-0 bg-border/80 sm:block' />
            )}

            {/* 右侧/下方：主标题与描述文字 */}
            {(title || description) && (
              <div className='flex flex-col justify-center min-w-0 space-y-1 sm:space-y-0.5'>
                {title && (
                  <h2 className='text-base font-semibold tracking-tight text-foreground'>
                    {title}
                  </h2>
                )}
                {description && (
                  <div className='text-xs text-muted-foreground'>
                    {description}
                  </div>
                )}
              </div>
            )}
          </div>

          {children}
        </div>
      </div>
    </div>
  )
}


