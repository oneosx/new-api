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
import { ArrowUpDown, Check, Filter, Search, X } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTableViewModeToggle } from '@/components/data-table'
import {
  sideDrawerContentClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { cn } from '@/lib/utils'

import { getSortLabels, type SortOption, type ViewMode } from '../constants'
import type { PricingModel, PricingVendor, TokenUnit } from '../types'
import { PricingSidebar } from './pricing-sidebar'

export interface PricingToolbarProps {
  filteredCount: number
  totalCount?: number
  searchValue: string
  onSearchChange: (value: string) => void
  onSearchClear: () => void
  sortBy: string
  onSortChange: (value: string) => void
  tokenUnit: TokenUnit
  onTokenUnitChange: (value: TokenUnit) => void
  showRechargePrice: boolean
  onRechargePriceChange: (value: boolean) => void
  viewMode: ViewMode
  onViewModeChange: (value: ViewMode) => void
  quotaTypeFilter: string
  endpointTypeFilter: string
  vendorFilter: string
  groupFilter: string
  tagFilter: string
  onQuotaTypeChange: (value: string) => void
  onEndpointTypeChange: (value: string) => void
  onVendorChange: (value: string) => void
  onGroupChange: (value: string) => void
  onTagChange: (value: string) => void
  vendors: PricingVendor[]
  groups: string[]
  groupRatios?: Record<string, number>
  tags: string[]
  models: PricingModel[]
  hasActiveFilters: boolean
  activeFilterCount: number
  onClearFilters: () => void
}

export function PricingToolbar(props: PricingToolbarProps) {
  const { t } = useTranslation()
  const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false)
  const sortLabels = getSortLabels(t)

  return (
    <div className='bg-card rounded-xl border p-2.5 sm:p-3'>
      <div className='flex flex-col gap-2.5 sm:flex-row sm:items-center sm:justify-between sm:gap-3'>
        {/* 第一行（手机端）：筛选 + 搜索框 + xx个模型；桌面端：左半区 */}
        <div className='flex w-full items-center gap-2 sm:w-auto sm:flex-1 sm:min-w-0 sm:gap-3'>
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={() => setMobileFiltersOpen(true)}
            className='gap-1 shrink-0 h-9 px-2.5 text-xs sm:h-8.5 sm:px-3 sm:text-xs xl:hidden'
          >
            <Filter className='size-3.5 sm:size-4' />
            <span>{t('Filter')}</span>
            {props.activeFilterCount > 0 && (
              <Badge className='size-4 justify-center p-0 text-[9px] leading-none sm:ml-0.5 sm:size-5 sm:text-[10px]'>
                {props.activeFilterCount}
              </Badge>
            )}
          </Button>

          <div className='relative flex-1 min-w-0 sm:max-w-xs md:max-w-sm'>
            <input
              type='text'
              placeholder={t('Search model name, provider, endpoint, or tag...')}
              value={props.searchValue}
              onChange={(e) => props.onSearchChange(e.target.value)}
              className={cn(
                'border-border/60 bg-background placeholder:text-muted-foreground/50',
                'hover:border-border',
                'focus:border-primary/50 focus:ring-primary/20 focus:ring-2',
                'h-9 sm:h-8.5 w-full rounded-lg border pr-7 sm:pr-14 pl-7 sm:pl-8 text-xs sm:text-sm transition-all outline-none'
              )}
              aria-label={t('Search models')}
            />
            <Search className='text-muted-foreground/60 pointer-events-none absolute top-1/2 left-2 sm:left-2.5 size-3.5 -translate-y-1/2' />
            <div className='absolute top-1/2 right-1.5 sm:right-2 flex -translate-y-1/2 items-center gap-1'>
              {props.searchValue ? (
                <button
                  type='button'
                  onClick={props.onSearchClear}
                  className='text-muted-foreground/60 hover:text-foreground rounded p-0.5 transition-colors'
                  aria-label={t('Clear search')}
                >
                  <X className='size-3.5' />
                </button>
              ) : (
                <kbd className='bg-muted/80 text-muted-foreground/70 hidden rounded px-1.5 py-0.5 font-mono text-[10px] sm:inline-block'>
                  ⌘K
                </kbd>
              )}
            </div>
          </div>

          <div className='text-muted-foreground flex shrink-0 items-baseline gap-0.5 sm:gap-1 text-xs sm:text-sm whitespace-nowrap'>
            <span className='text-foreground font-semibold tabular-nums'>
              {props.filteredCount.toLocaleString()}
            </span>
            <span>{props.filteredCount === 1 ? t('model') : t('models')}</span>
            {props.totalCount != null &&
              props.filteredCount !== props.totalCount && (
                <span className='text-muted-foreground/60 text-[10px] sm:text-xs'>
                  / {props.totalCount.toLocaleString()}
                </span>
              )}
          </div>
        </div>

        {/* 第二行（手机端）：4列均分吃完100%整行宽度；桌面端：右侧常规排列 */}
        <div className='grid grid-cols-4 sm:flex sm:items-center w-full sm:w-auto gap-1.5 sm:gap-2'>
          <ToggleGroup
            value={[props.showRechargePrice ? 'recharge' : 'standard']}
            onValueChange={(values) => {
              if (values.length > 0) {
                props.onRechargePriceChange(values[0] === 'recharge')
              }
            }}
            variant='outline'
            size='sm'
            aria-label={t('Price display mode')}
            className='w-full justify-center [&>*]:flex-1 sm:[&>*]:flex-none'
          >
            <ToggleGroupItem value='standard'>{t('Standard')}</ToggleGroupItem>
            <ToggleGroupItem value='recharge'>{t('Recharge')}</ToggleGroupItem>
          </ToggleGroup>

          <ToggleGroup
            value={[props.tokenUnit]}
            onValueChange={(values) => {
              if (values[0] === 'M' || values[0] === 'K') {
                props.onTokenUnitChange(values[0])
              }
            }}
            variant='outline'
            size='sm'
            aria-label={t('Token unit')}
            className='w-full justify-center [&>*]:flex-1 sm:[&>*]:flex-none'
          >
            <ToggleGroupItem value='M'>/1M</ToggleGroupItem>
            <ToggleGroupItem value='K'>/1K</ToggleGroupItem>
          </ToggleGroup>

          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  className='h-8 w-full gap-1 px-1.5 text-xs justify-center sm:w-auto sm:px-3 sm:gap-1.5'
                />
              }
            >
              <ArrowUpDown className='size-3.5 shrink-0' />
              <span className='truncate'>{sortLabels[props.sortBy as SortOption] || t('Sort')}</span>
            </DropdownMenuTrigger>
            <DropdownMenuContent align='end' className='w-44'>
              <DropdownMenuGroup>
                {Object.entries(sortLabels).map(([value, label]) => (
                  <DropdownMenuItem
                    key={value}
                    onClick={() => props.onSortChange(value)}
                    className='gap-2'
                  >
                    <Check
                      className={cn(
                        'size-4 shrink-0',
                        props.sortBy === value ? 'opacity-100' : 'opacity-0'
                      )}
                    />
                    {label}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>

          <div className='w-full flex justify-center sm:w-auto'>
            <DataTableViewModeToggle
              value={props.viewMode}
              onChange={props.onViewModeChange}
              className='w-full justify-center [&>*]:flex-1 sm:[&>*]:flex-none'
            />
          </div>
        </div>
      </div>

      <Sheet open={mobileFiltersOpen} onOpenChange={setMobileFiltersOpen}>
        <SheetContent
          side='left'
          className={sideDrawerContentClassName('sm:max-w-md')}
        >
          <SheetHeader className={sideDrawerHeaderClassName()}>
            <SheetTitle>{t('Filter')}</SheetTitle>
            <SheetDescription>
              {t('Filter models by provider, group, type, endpoint, and tags.')}
            </SheetDescription>
          </SheetHeader>
          <div className={sideDrawerFormClassName('gap-0')}>
            <PricingSidebar
              quotaTypeFilter={props.quotaTypeFilter}
              endpointTypeFilter={props.endpointTypeFilter}
              vendorFilter={props.vendorFilter}
              groupFilter={props.groupFilter}
              tagFilter={props.tagFilter}
              onQuotaTypeChange={props.onQuotaTypeChange}
              onEndpointTypeChange={props.onEndpointTypeChange}
              onVendorChange={props.onVendorChange}
              onGroupChange={props.onGroupChange}
              onTagChange={props.onTagChange}
              vendors={props.vendors}
              groups={props.groups}
              groupRatios={props.groupRatios}
              tags={props.tags}
              models={props.models}
              hasActiveFilters={props.hasActiveFilters}
              onClearFilters={props.onClearFilters}
              className='border-0 bg-transparent p-0 shadow-none'
            />
          </div>
        </SheetContent>
      </Sheet>
    </div>
  )
}
