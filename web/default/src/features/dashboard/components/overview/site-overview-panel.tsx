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
import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Activity,
  BarChart3,
  CalendarClock,
  ChevronLeft,
  ChevronRight,
  Clock3,
  Hash,
  RadioTower,
  Timer,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatCompactNumber, formatNumber, formatPercent } from '@/lib/format'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { getDashboardSiteOverview } from '@/features/dashboard/api'
import type {
  DashboardModelDistributionItem,
  DashboardModelDistributionPeriod,
} from '@/features/dashboard/types'

const ICONS = [
  BarChart3,
  Hash,
  BarChart3,
  Hash,
  RadioTower,
  CalendarClock,
  Clock3,
  Zap,
] as const
const MODEL_DISTRIBUTION_PAGE_SIZE = 5
const EMPTY_MODEL_DISTRIBUTION: DashboardModelDistributionItem[] = []

const MODEL_DISTRIBUTION_PERIODS: Array<{
  key: DashboardModelDistributionPeriod
  labelKey: string
}> = [
  { key: 'today', labelKey: '24h' },
  { key: 'week', labelKey: '7 Days' },
  { key: 'all', labelKey: 'History' },
]

function modelHealthClass(rate: number): string {
  if (rate >= 99.9) return 'text-success'
  if (rate >= 99) return 'text-warning'
  return 'text-destructive'
}

function formatUptimeDuration(
  seconds: number | null | undefined,
  t: (key: string, options?: Record<string, unknown>) => string
): string {
  if (seconds == null || Number.isNaN(seconds) || seconds < 0) return '-'
  if (seconds < 60) {
    return t('{{value}}s', { value: Math.floor(seconds) })
  }
  const totalMinutes = Math.floor(seconds / 60)
  const days = Math.floor(totalMinutes / 1440)
  const hours = Math.floor((totalMinutes % 1440) / 60)
  const minutes = totalMinutes % 60

  if (days > 0) {
    return t('{{days}}d {{hours}}h', { days, hours })
  }
  if (hours > 0) {
    return t('{{hours}}h {{minutes}}m', { hours, minutes })
  }
  return t('{{minutes}}m', { minutes })
}

export function SiteOverviewPanel() {
  const { t } = useTranslation()
  const [modelPage, setModelPage] = useState(0)
  const [modelDistributionPeriod, setModelDistributionPeriod] =
    useState<DashboardModelDistributionPeriod>('all')
  const overviewQuery = useQuery({
    queryKey: ['dashboard', 'site-overview', modelDistributionPeriod],
    queryFn: () =>
      getDashboardSiteOverview({
        model_distribution_period: modelDistributionPeriod,
      }),
    staleTime: 60 * 1000,
    retry: false,
  })

  const data = overviewQuery.data?.data
  const distribution = data?.model_distribution ?? EMPTY_MODEL_DISTRIBUTION
  const topHealthModels = data?.health.top_models ?? []
  const modelPageCount = Math.max(
    1,
    Math.ceil(distribution.length / MODEL_DISTRIBUTION_PAGE_SIZE)
  )
  const safeModelPage = Math.min(modelPage, modelPageCount - 1)
  const pagedDistribution = useMemo(
    () =>
      distribution.slice(
        safeModelPage * MODEL_DISTRIBUTION_PAGE_SIZE,
        (safeModelPage + 1) * MODEL_DISTRIBUTION_PAGE_SIZE
      ),
    [distribution, safeModelPage]
  )

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
      label: t('24h tokens'),
      value: formatCompactNumber(data?.recent_tokens),
      hint: t('Last {{hours}} hours', { hours: data?.window_hours ?? 24 }),
    },
    {
      label: t('24h requests'),
      value: formatCompactNumber(data?.recent_requests),
      hint: t('Last {{hours}} hours', { hours: data?.window_hours ?? 24 }),
    },
    {
      label: t('24h cache hit rate'),
      value: formatPercent(data?.cache_hit_24h?.hit_rate),
      hint: `${t('Cached tokens / total tokens')}: ${formatCompactNumber(data?.cache_hit_24h?.cached_tokens)} / ${formatCompactNumber(data?.cache_hit_24h?.total_tokens)}`,
    },
    {
      label: t('Site uptime'),
      value: formatUptimeDuration(data?.site_uptime_seconds, t),
      hint: t('Since process start'),
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
        <Activity
          className='text-muted-foreground/60 size-4 shrink-0'
          aria-hidden='true'
        />
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
              <div
                key={item.label}
                className='bg-muted/35 rounded-xl border px-3 py-3'
              >
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
            <div className='flex flex-wrap items-center justify-between gap-2'>
              <div>
                <div className='text-sm font-semibold'>
                  {t('Model call distribution')}
                </div>
                <div className='text-muted-foreground text-xs'>
                  {t('Top models ranked by selected request share')}
                </div>
              </div>
              <div className='flex shrink-0 flex-wrap items-center justify-end gap-2'>
                {MODEL_DISTRIBUTION_PERIODS.map((item) => (
                  <Button
                    key={item.key}
                    type='button'
                    size='sm'
                    variant={
                      modelDistributionPeriod === item.key
                        ? 'default'
                        : 'outline'
                    }
                    onClick={() => {
                      setModelDistributionPeriod(item.key)
                      setModelPage(0)
                    }}
                  >
                    {t(item.labelKey)}
                  </Button>
                ))}
                {!overviewQuery.isLoading && distribution.length > 0 && (
                  <span className='text-muted-foreground text-xs'>
                    {t('{{count}} active models', {
                      count: distribution.length,
                    })}
                  </span>
                )}
              </div>
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
              <>
                <div className='space-y-2'>
                  {pagedDistribution.map((item) => (
                    <div
                      key={item.model_name}
                      className='rounded-xl border px-3 py-3'
                    >
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
                        <span>
                          {formatCompactNumber(item.token_used)} tokens
                        </span>
                      </div>
                    </div>
                  ))}
                </div>

                {modelPageCount > 1 ? (
                  <div className='flex items-center justify-between gap-3 pt-1'>
                    <Button
                      type='button'
                      variant='outline'
                      size='icon-sm'
                      onClick={() =>
                        setModelPage((page) => Math.max(0, page - 1))
                      }
                      disabled={safeModelPage === 0}
                      aria-label={t('Previous')}
                    >
                      <ChevronLeft className='size-4' aria-hidden='true' />
                    </Button>
                    <span className='text-muted-foreground text-xs'>
                      {t('Page {{current}} of {{total}}', {
                        current: safeModelPage + 1,
                        total: modelPageCount,
                      })}
                    </span>
                    <Button
                      type='button'
                      variant='outline'
                      size='icon-sm'
                      onClick={() =>
                        setModelPage((page) =>
                          Math.min(modelPageCount - 1, page + 1)
                        )
                      }
                      disabled={safeModelPage >= modelPageCount - 1}
                      aria-label={t('Next')}
                    >
                      <ChevronRight className='size-4' aria-hidden='true' />
                    </Button>
                  </div>
                ) : null}
              </>
            )}
          </div>

          <div className='space-y-3'>
            <div>
              <div className='text-sm font-semibold'>
                {t('Health snapshot')}
              </div>
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
                value={
                  data?.health.avg_latency_ms
                    ? `${formatNumber(data.health.avg_latency_ms)} ms`
                    : '-'
                }
                loading={overviewQuery.isLoading}
              />
              <HealthCell
                icon={Zap}
                label={t('Throughput')}
                value={
                  data?.health.avg_tps
                    ? `${formatNumber(data.health.avg_tps)} tok/s`
                    : '-'
                }
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
