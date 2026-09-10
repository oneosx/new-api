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

import { HeaderLogo } from '@/components/layout/components/header-logo'
import { Skeleton } from '@/components/ui/skeleton'
import { useSystemConfig } from '@/hooks/use-system-config'

type AuthLayoutProps = {
  children: React.ReactNode
}

export function AuthLayout({ children }: AuthLayoutProps) {
  const { systemName, logo, loading, logoLoaded } = useSystemConfig()

  return (
    <div className='relative grid h-svh max-w-none'>
      <Link
        to='/'
        className='absolute top-4 left-4 z-10 flex items-center transition-opacity hover:opacity-80 sm:top-8 sm:left-8'
        aria-label={systemName}
      >
        <div className='flex h-7 max-h-7 max-w-[160px] items-center justify-center transition-all duration-300'>
          {loading ? (
            <Skeleton className='h-7 w-28 rounded' />
          ) : (
            <HeaderLogo
              src={logo}
              darkSrc='/logo.white.png'
              loading={loading}
              logoLoaded={logoLoaded}
              className='h-7 max-h-7 w-auto max-w-[160px] rounded-none object-contain'
            />
          )}
        </div>
      </Link>
      <div className='container flex items-center pt-16 sm:pt-0'>
        <div className='mx-auto flex w-full flex-col justify-center space-y-2 px-4 py-8 sm:w-[480px] sm:p-8'>
          {children}
        </div>
      </div>
    </div>
  )
}
