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
import { Clock3, ExternalLink, Gauge, RotateCcw, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import type { DashboardCPACodexForecast } from '@/features/dashboard/types'
import { formatNumber, formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

export function CpaQuotaForecastFooter(props: {
  forecast: DashboardCPACodexForecast
}) {
  const { t } = useTranslation()
  const forecast = props.forecast
  const isInsufficient = forecast.can_last_until_reset === false
  const totalRemaining =
    forecast.measured_accounts > 0
      ? t('{{percent}}% · {{accounts}} full accounts', {
          percent: formatNumber(forecast.total_remaining_percent),
          accounts: formatNumber(forecast.total_remaining_account_equivalents),
        })
      : '-'

  return (
    <>
      <Separator className='my-5' />
      <div className='flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between'>
        <div className='grid min-w-0 flex-1 gap-3 sm:grid-cols-3'>
          <ForecastMetric
            icon={Gauge}
            label={t('Total remaining')}
            value={totalRemaining}
          />
          <ForecastMetric
            icon={Clock3}
            label={t('Estimated exhaustion')}
            value={formatForecastExhaustion(forecast, t)}
            warning={isInsufficient}
          />
          <div className='bg-muted/30 min-w-0 rounded-xl border p-3'>
            <div className='text-muted-foreground flex items-center gap-1.5 text-xs font-medium'>
              <RotateCcw className='size-3.5 shrink-0' aria-hidden='true' />
              {t('Reset outlook')}
            </div>
            <div className='mt-2 flex flex-wrap items-center gap-2'>
              <Badge variant={isInsufficient ? 'warning' : 'secondary'}>
                {formatResetOutlook(forecast, t)}
              </Badge>
              {forecast.next_reset_at ? (
                <span className='text-muted-foreground text-xs tabular-nums'>
                  {t('Next reset: {{time}}', {
                    time: formatTimestampToDate(forecast.next_reset_at),
                  })}
                </span>
              ) : null}
            </div>
            {forecast.measured_accounts > 0 ? (
              <div className='text-muted-foreground mt-2 text-[11px]'>
                {t('Based on {{ready}}/{{measured}} accounts', {
                  ready: forecast.forecast_ready_accounts,
                  measured: forecast.measured_accounts,
                })}
              </div>
            ) : null}
          </div>
        </div>

        <span className='cpa-tibo-button-frame inline-flex rounded-lg p-px'>
          <Button
            variant='outline'
            size='lg'
            className='w-full lg:w-auto'
            render={
              <a
                href='https://x.com/thsottiaux'
                target='_blank'
                rel='noopener noreferrer'
              />
            }
          >
            <Sparkles data-icon='inline-start' aria-hidden='true' />
            {t('Follow Tibo, meow~')}
            <ExternalLink data-icon='inline-end' aria-hidden='true' />
          </Button>
        </span>
      </div>
    </>
  )
}

function ForecastMetric(props: {
  icon: typeof Gauge
  label: string
  value: string
  warning?: boolean
}) {
  const Icon = props.icon
  return (
    <div className='bg-muted/30 min-w-0 rounded-xl border p-3'>
      <div className='text-muted-foreground flex items-center gap-1.5 text-xs font-medium'>
        <Icon className='size-3.5 shrink-0' aria-hidden='true' />
        {props.label}
      </div>
      <div
        className={cn(
          'mt-2 truncate text-sm font-semibold tabular-nums',
          props.warning && 'text-warning'
        )}
        data-quota-warning={props.warning ? 'true' : undefined}
        title={props.value}
      >
        {props.value}
      </div>
    </div>
  )
}

function formatForecastExhaustion(
  forecast: DashboardCPACodexForecast,
  t: (key: string) => string
): string {
  if (forecast.status === 'collecting') return t('Collecting quota samples')
  if (forecast.status === 'stable') return t('No measurable consumption')
  if (forecast.status === 'estimated' && forecast.estimated_exhausted_at) {
    return formatTimestampToDate(forecast.estimated_exhausted_at)
  }
  return t('Forecast unavailable')
}

function formatResetOutlook(
  forecast: DashboardCPACodexForecast,
  t: (key: string) => string
): string {
  if (forecast.can_last_until_reset === false) {
    return t('Quota may run out before reset')
  }
  if (forecast.can_last_until_reset === true) {
    return t('Can last until reset')
  }
  if (forecast.status === 'collecting') return t('Collecting quota samples')
  return t('Forecast unavailable')
}
