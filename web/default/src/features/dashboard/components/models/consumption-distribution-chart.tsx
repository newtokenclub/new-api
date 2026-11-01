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
import { VChart } from '@visactor/react-vchart'
import {
  AreaChart,
  BarChart3,
  Eye,
  EyeOff,
  RefreshCw,
  WalletCards,
} from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useThemeCustomization } from '@/context/theme-customization-provider'
import { useTheme } from '@/context/theme-provider'
import {
  CONSUMPTION_DISTRIBUTION_CHART_OPTIONS,
  DEFAULT_TIME_GRANULARITY,
} from '@/features/dashboard/constants'
import { processChartData } from '@/features/dashboard/lib'
import type {
  ConsumptionDistributionChartType,
  QuotaDataItem,
} from '@/features/dashboard/types'
import { useThemeRadiusPx } from '@/lib/theme-radius'
import type { TimeGranularity } from '@/lib/time'
import { cn } from '@/lib/utils'
import { VCHART_OPTION } from '@/lib/vchart'

let themeManagerPromise: Promise<
  (typeof import('@visactor/vchart'))['ThemeManager']
> | null = null

interface ConsumptionDistributionChartProps {
  data: QuotaDataItem[]
  loading?: boolean
  timeGranularity?: TimeGranularity
  defaultChartType?: ConsumptionDistributionChartType
  onRefresh?: () => void
}

const CONSUMPTION_VISIBILITY_KEY = 'dashboard_consumption_distribution_hidden'

const CHART_TYPE_ICONS: Record<
  ConsumptionDistributionChartType,
  typeof BarChart3
> = {
  bar: BarChart3,
  area: AreaChart,
}

export function ConsumptionDistributionChart(
  props: ConsumptionDistributionChartProps
) {
  const { t } = useTranslation()
  const { resolvedTheme } = useTheme()
  const { customization } = useThemeCustomization()
  const chartRadius = useThemeRadiusPx(
    '--radius-md',
    `${customization.preset}:${customization.radius}`
  )
  const [chartType, setChartType] = useState<ConsumptionDistributionChartType>(
    props.defaultChartType ?? 'bar'
  )
  const [groupMode, setGroupMode] = useState(false)
  const [themeReady, setThemeReady] = useState(false)
  const [hidden, setHidden] = useState(
    () => localStorage.getItem(CONSUMPTION_VISIBILITY_KEY) === 'true'
  )
  const themeManagerRef = useRef<
    (typeof import('@visactor/vchart'))['ThemeManager'] | null
  >(null)
  const timeGranularity = props.timeGranularity ?? DEFAULT_TIME_GRANULARITY

  useEffect(() => {
    if (props.defaultChartType) setChartType(props.defaultChartType)
  }, [props.defaultChartType])

  useEffect(() => {
    const updateTheme = async () => {
      setThemeReady(false)

      if (!themeManagerPromise) {
        themeManagerPromise = import('@visactor/vchart').then(
          (m) => m.ThemeManager
        )
      }

      const ThemeManager = await themeManagerPromise
      themeManagerRef.current = ThemeManager
      ThemeManager.setCurrentTheme(resolvedTheme === 'dark' ? 'dark' : 'light')
      setThemeReady(true)
    }

    updateTheme()
  }, [resolvedTheme])

  const toggleHidden = () => {
    setHidden((current) => {
      const next = !current
      localStorage.setItem(CONSUMPTION_VISIBILITY_KEY, String(next))
      return next
    })
  }

  const chartData = useMemo(
    () =>
      processChartData(
        props.loading ? [] : props.data,
        timeGranularity,
        t,
        chartRadius
      ),
    [props.data, props.loading, timeGranularity, t, chartRadius]
  )
  const spec = groupMode
    ? chartData.spec_group
    : chartType === 'bar'
      ? chartData.spec_line
      : chartData.spec_area
  const specType = typeof spec?.type === 'string' ? spec.type : chartType
  const chartKey = [
    chartType,
    groupMode ? 'group' : 'model',
    specType,
    props.loading ? 'loading' : 'ready',
    props.data.length,
    resolvedTheme,
    customization.preset,
  ].join('-')

  return (
    <div className='overflow-hidden rounded-lg border'>
      <div className='flex w-full flex-col gap-1.5 border-b px-3 py-2 sm:gap-3 sm:px-5 sm:py-3 lg:flex-row lg:items-center lg:justify-between'>
        <div className='flex min-w-0 items-center gap-2'>
          <WalletCards className='text-muted-foreground/60 size-4' />
          <div className='text-sm font-semibold'>{t('Quota Distribution')}</div>
          <span
            className={cn(
              'text-muted-foreground text-xs transition-[filter]',
              hidden && 'pointer-events-none select-none blur-sm'
            )}
          >
            {t('Total:')} {chartData.totalQuotaDisplay}
          </span>
        </div>

        <div className='flex min-w-0 items-center gap-1.5'>
          <div className='bg-muted/60 inline-flex h-7 min-w-0 flex-1 overflow-x-auto rounded-lg border p-0.5 sm:h-8 sm:w-auto sm:flex-none'>
            {CONSUMPTION_DISTRIBUTION_CHART_OPTIONS.map((item) => {
              const Icon = CHART_TYPE_ICONS[item.value]
              return (
                <button
                  key={item.value}
                  type='button'
                  onClick={() => {
                    setGroupMode(false)
                    setChartType(item.value)
                  }}
                  className={`inline-flex shrink-0 items-center gap-1.5 rounded-md px-3 text-xs font-medium transition-colors ${
                    chartType === item.value
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground'
                  }`}
                >
                  <Icon className='size-3.5' />
                  {t(item.labelKey)}
                </button>
              )
            })}
            <button
              type='button'
              onClick={() => setGroupMode(true)}
              className={`shrink-0 rounded-md px-3 text-xs font-medium transition-colors ${groupMode ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}`}
            >
              {t('Group Amounts')}
            </button>
          </div>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant='ghost'
                  size='icon-sm'
                  onClick={toggleHidden}
                  aria-label={
                    hidden ? t('Show sensitive data') : t('Hide sensitive data')
                  }
                />
              }
            >
              {hidden ? <EyeOff /> : <Eye />}
            </TooltipTrigger>
            <TooltipContent>
              {hidden ? t('Show sensitive data') : t('Hide sensitive data')}
            </TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant='ghost'
                  size='icon-sm'
                  onClick={props.onRefresh}
                  disabled={props.loading || !props.onRefresh}
                  aria-label={t('Refresh')}
                />
              }
            >
              <RefreshCw className={cn(props.loading && 'animate-spin')} />
            </TooltipTrigger>
            <TooltipContent>{t('Refresh')}</TooltipContent>
          </Tooltip>
        </div>
      </div>

      <div className='relative h-[300px] overflow-hidden p-1.5 sm:h-96 sm:p-2'>
        <div
          className={cn(
            'size-full transition-[filter,opacity] duration-200',
            hidden && 'pointer-events-none select-none blur-xl'
          )}
          aria-hidden={hidden}
        >
          {themeReady && spec && (
            <VChart
              key={chartKey}
              spec={{
                ...spec,
                theme: resolvedTheme === 'dark' ? 'dark' : 'light',
                background: 'transparent',
              }}
              option={VCHART_OPTION}
            />
          )}
        </div>
        {hidden && (
          <button
            type='button'
            className='bg-background/75 text-muted-foreground absolute inset-0 flex items-center justify-center text-sm font-medium backdrop-blur-sm'
            onClick={toggleHidden}
          >
            <EyeOff className='mr-2 size-4' />
            {t('Consumption data hidden')}
          </button>
        )}
      </div>
    </div>
  )
}
