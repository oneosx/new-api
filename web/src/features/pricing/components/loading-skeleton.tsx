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
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

import { VIEW_MODES, type ViewMode } from '../constants'

export interface LoadingSkeletonProps {
  viewMode?: ViewMode
}

export function LoadingSkeleton(props: LoadingSkeletonProps) {
  return (
    <div aria-busy='true'>
      <div className='grid gap-4 xl:grid-cols-[280px_minmax(0,1fr)]'>
        {/* 左侧侧边栏 Skeleton */}
        <div className='hidden rounded-xl border p-3 xl:block'>
          <Skeleton className='mb-4 h-5 w-24' />
          {Array.from({ length: 5 }, (_, index) => (
            <div
              key={index}
              className='flex flex-col gap-3 border-b py-4 last:border-0'
            >
              <Skeleton className='h-4 w-28' />
              <div className='flex flex-wrap gap-2'>
                <Skeleton className='h-7 w-24' />
                <Skeleton className='h-7 w-20' />
                <Skeleton className='h-7 w-28' />
              </div>
            </div>
          ))}
        </div>

        {/* 右侧内容区 Skeleton */}
        <div className='flex min-w-0 flex-col gap-4'>
          {/* 工具栏 Skeleton：对齐搜索框与各控制按钮 */}
          <div className='bg-card flex flex-wrap items-center justify-between gap-3 rounded-xl border p-2.5 sm:p-3'>
            <div className='flex w-full items-center gap-2 sm:w-auto sm:flex-1 sm:min-w-0 sm:gap-3'>
              {/* 手机端筛选按钮占位 */}
              <Skeleton className='h-9 w-16 shrink-0 xl:hidden' />
              {/* 搜索框占位 */}
              <Skeleton className='h-9 w-full sm:h-8.5 sm:max-w-xs md:max-w-sm rounded-lg' />
              {/* 模型计数占位 */}
              <Skeleton className='h-5 w-16 shrink-0' />
            </div>

            {/* 右侧按钮组占位 */}
            <div className='grid grid-cols-4 w-full gap-1.5 sm:flex sm:w-auto sm:items-center sm:gap-2'>
              <Skeleton className='h-8 w-full sm:w-28 rounded-lg' />
              <Skeleton className='h-8 w-full sm:w-24 rounded-lg' />
              <Skeleton className='h-8 w-full sm:w-20 rounded-lg' />
              <Skeleton className='h-8 w-full sm:w-16 rounded-lg' />
            </div>
          </div>
          {props.viewMode === VIEW_MODES.TABLE ? (
            <div className='overflow-hidden rounded-xl border'>
              {Array.from({ length: 10 }, (_, index) => (
                <div
                  key={index}
                  className='flex gap-4 border-b p-4 last:border-0'
                >
                  <Skeleton className='h-5 w-40 max-w-full' />
                  <Skeleton className='h-5 flex-1' />
                  <Skeleton className='h-5 w-20' />
                </div>
              ))}
            </div>
          ) : (
            <div className='grid grid-cols-1 gap-3 sm:gap-4 md:grid-cols-2 xl:grid-cols-3'>
              {Array.from({ length: 6 }, (_, index) => (
                <Card key={index} className='gap-3'>
                  <CardHeader className='flex flex-row gap-3'>
                    <Skeleton className='size-10 shrink-0' />
                    <div className='flex min-w-0 flex-1 flex-col gap-2'>
                      <Skeleton className='h-5 w-40 max-w-full' />
                      <Skeleton className='h-3 w-20' />
                    </div>
                    <Skeleton className='size-7 shrink-0' />
                  </CardHeader>
                  <CardContent className='flex flex-1 flex-col gap-3'>
                    <div className='flex flex-col gap-2'>
                      <Skeleton className='h-3.5 w-full' />
                      <Skeleton className='h-3.5 w-4/5' />
                    </div>
                    <div className='mt-auto flex flex-col gap-1.5'>
                      <Skeleton className='h-4 w-16' />
                      <div className='grid grid-cols-3 gap-3'>
                        <Skeleton className='h-10' />
                        <Skeleton className='h-10' />
                        <Skeleton className='h-10' />
                      </div>
                    </div>
                    <div className='grid grid-cols-2 gap-3'>
                      <Skeleton className='h-4 w-28 max-w-full' />
                      <Skeleton className='h-4 w-28 max-w-full' />
                    </div>
                  </CardContent>
                  <CardFooter className='border-0 bg-transparent pt-0'>
                    <div className='border-border/60 flex w-full items-center justify-between gap-3 border-t pt-2'>
                      <div className='flex items-start gap-5'>
                        <div className='flex w-24 shrink-0 flex-col gap-1'>
                          <Skeleton className='h-4 w-10' />
                          <div className='flex h-3 items-center justify-between'>
                            {Array.from({ length: 24 }, (_, bar) => (
                              <Skeleton
                                key={bar}
                                className='h-full w-[3px] rounded-xs'
                              />
                            ))}
                          </div>
                        </div>
                        <Skeleton className='h-8 w-6' />
                        <Skeleton className='h-8 w-8' />
                      </div>
                      <Skeleton className='h-7 w-12' />
                    </div>
                  </CardFooter>
                </Card>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
