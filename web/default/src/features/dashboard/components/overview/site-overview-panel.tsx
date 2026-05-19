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
import { useQuery } from '@tanstack/react-query'
import { Activity, BarChart3, Clock3, Hash, Timer, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatCompactNumber, formatNumber, formatPercent } from '@/lib/format'
import { cn } from '@/lib/utils'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { getDashboardSiteOverview } from '@/features/dashboard/api'

const ICONS = [BarChart3, Hash, Clock3, Zap] as const

function modelHealthClass(rate: number): string {
  if (rate >= 99.9) return 'text-success'
  if (rate >= 99) return 'text-warning'
  return 'text-destructive'
}

export function SiteOverviewPanel() {
  const { t } = useTranslation()
  const overviewQuery = useQuery({
    queryKey: ['dashboard', 'site-overview'],
    queryFn: getDashboardSiteOverview,
    staleTime: 60 * 1000,
    retry: false,
  })

  const data = overviewQuery.data?.data
  const distribution = data?.model_distribution ?? []
  const topHealthModels = data?.health.top_models ?? []

  const stats = [
    {
      label: t('Total tokens'),
      value: formatCompactNumber(data?.total_tokens),
      hint: t('Historical total'),
    },
    {
      label: t('Call count'),
      value: formatCompactNumber(data?.total_requests),
      hint: t('Historical requests'),
    },
    {
      label: t('Average RPM'),
      value: formatNumber(data?.avg_rpm),
      hint: t('Last {{hours}} hours', { hours: data?.window_hours ?? 24 }),
    },
    {
      label: t('Average TPM'),
      value: formatCompactNumber(data?.avg_tpm),
      hint: t('Last {{hours}} hours', { hours: data?.window_hours ?? 24 }),
    },
  ]

  return (
    <section className='bg-card h-full overflow-hidden rounded-2xl border shadow-xs'>
      <div className='flex items-center gap-2 border-b px-4 py-3 sm:px-5'>
        <Activity className='text-muted-foreground/60 size-4 shrink-0' aria-hidden='true' />
        <h3 className='text-sm font-semibold'>{t('Site-wide overview')}</h3>
        <span className='text-muted-foreground ml-auto text-xs'>
          {t('Shared dashboard metrics for all signed-in users')}
        </span>
      </div>

      <div className='space-y-5 p-4 sm:p-5'>
        <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
          {stats.map((item, index) => {
            const Icon = ICONS[index] ?? Activity
            return (
              <div key={item.label} className='bg-muted/35 rounded-xl border px-3 py-3'>
                <div className='text-muted-foreground flex items-center gap-1.5 text-[11px] font-medium'>
                  <Icon className='size-3.5 shrink-0' aria-hidden='true' />
                  <span>{item.label}</span>
                </div>
                {overviewQuery.isLoading ? (
                  <Skeleton className='mt-2 h-6 w-20' />
                ) : (
                  <div className='mt-2 font-mono text-lg font-semibold tabular-nums'>
                    {item.value}
                  </div>
                )}
                <div className='text-muted-foreground mt-1 text-xs'>
                  {item.hint}
                </div>
              </div>
            )
          })}
        </div>

        <div className='grid gap-4 xl:grid-cols-[minmax(0,1fr)_19rem]'>
          <div className='space-y-3'>
            <div className='flex items-center justify-between gap-2'>
              <div>
                <div className='text-sm font-semibold'>
                  {t('Model call distribution')}
                </div>
                <div className='text-muted-foreground text-xs'>
                  {t('Top models ranked by historical request share')}
                </div>
              </div>
              {!overviewQuery.isLoading && distribution.length > 0 && (
                <span className='text-muted-foreground text-xs'>
                  {t('{{count}} active models', { count: distribution.length })}
                </span>
              )}
            </div>

            {overviewQuery.isLoading ? (
              <div className='space-y-2'>
                {Array.from({ length: 5 }).map((_, index) => (
                  <Skeleton key={index} className='h-11 w-full rounded-xl' />
                ))}
              </div>
            ) : distribution.length === 0 ? (
              <div className='text-muted-foreground bg-muted/25 rounded-xl border border-dashed px-4 py-6 text-sm'>
                {t('No aggregated model traffic is available yet')}
              </div>
            ) : (
              <div className='space-y-2'>
                {distribution.map((item) => (
                  <div key={item.model_name} className='rounded-xl border px-3 py-3'>
                    <div className='flex items-center justify-between gap-3'>
                      <span className='min-w-0 truncate font-mono text-sm'>
                        {item.model_name}
                      </span>
                      <span className='text-muted-foreground shrink-0 font-mono text-xs tabular-nums'>
                        {formatPercent(item.percentage)}
                      </span>
                    </div>
                    <Progress
                      value={item.percentage}
                      className='mt-2 gap-0'
                    />
                    <div className='text-muted-foreground mt-2 flex items-center justify-between gap-3 text-xs'>
                      <span>
                        {t('{{count}} requests', {
                          count: formatCompactNumber(item.request_count),
                        })}
                      </span>
                      <span>{formatCompactNumber(item.token_used)} tokens</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className='space-y-3'>
            <div>
              <div className='text-sm font-semibold'>{t('Health snapshot')}</div>
              <div className='text-muted-foreground text-xs'>
                {t('Reuses the current performance summary window')}
              </div>
            </div>

            <div className='grid gap-2 sm:grid-cols-3 xl:grid-cols-1'>
              <HealthCell
                icon={Timer}
                label={t('Success rate')}
                value={formatPercent(data?.health.success_rate)}
                loading={overviewQuery.isLoading}
              />
              <HealthCell
                icon={Clock3}
                label={t('Average latency')}
                value={data?.health.avg_latency_ms ? `${formatNumber(data.health.avg_latency_ms)} ms` : '-'}
                loading={overviewQuery.isLoading}
              />
              <HealthCell
                icon={Zap}
                label={t('Throughput')}
                value={data?.health.avg_tps ? `${formatNumber(data.health.avg_tps)} tok/s` : '-'}
                loading={overviewQuery.isLoading}
              />
            </div>

            {!overviewQuery.isLoading && topHealthModels.length > 0 && (
              <div className='rounded-xl border px-3 py-3'>
                <div className='mb-2 text-xs font-medium'>
                  {t('Top models by traffic')}
                </div>
                <div className='space-y-2'>
                  {topHealthModels.map((item) => (
                    <div
                      key={item.model_name}
                      className='flex items-center justify-between gap-3'
                    >
                      <span className='min-w-0 truncate font-mono text-xs'>
                        {item.model_name}
                      </span>
                      <span
                        className={cn(
                          'shrink-0 font-mono text-xs font-semibold tabular-nums',
                          modelHealthClass(item.success_rate)
                        )}
                      >
                        {formatPercent(item.success_rate)}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </section>
  )
}

function HealthCell(props: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: string
  loading: boolean
}) {
  const Icon = props.icon
  return (
    <div className='bg-muted/35 rounded-xl border px-3 py-3'>
      <div className='text-muted-foreground flex items-center gap-1.5 text-[11px] font-medium'>
        <Icon className='size-3.5 shrink-0' aria-hidden='true' />
        <span>{props.label}</span>
      </div>
      {props.loading ? (
        <Skeleton className='mt-2 h-5 w-16' />
      ) : (
        <div className='mt-2 font-mono text-sm font-semibold tabular-nums'>
          {props.value}
        </div>
      )}
    </div>
  )
}
