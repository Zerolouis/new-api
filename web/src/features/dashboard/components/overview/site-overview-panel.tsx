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
  ChevronLeft,
  ChevronRight,
  Clock3,
  Hash,
  RadioTower,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatCompactNumber, formatNumber, formatPercent } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { getDashboardSiteOverview } from '@/features/dashboard/api'
import type {
  DashboardClientCacheHitSnapshot,
  DashboardModelDistributionItem,
  DashboardModelDistributionPeriod,
} from '@/features/dashboard/types'

const ICONS = [
  BarChart3,
  Hash,
  BarChart3,
  Hash,
  RadioTower,
  RadioTower,
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

function formatClientCacheRate(
  snapshot: DashboardClientCacheHitSnapshot | undefined
): string {
  if (!snapshot?.configured || snapshot.input_tokens <= 0) return '-'
  return formatPercent(snapshot.hit_rate)
}

function formatClientCacheHint(
  snapshot: DashboardClientCacheHitSnapshot | undefined,
  t: (key: string, options?: Record<string, unknown>) => string
): string {
  if (!snapshot?.configured) return t('Channel affinity rule not configured')
  return `${t('Cached tokens / input tokens')}: ${formatCompactNumber(snapshot.cached_tokens)} / ${formatCompactNumber(snapshot.input_tokens)}`
}

export function SiteOverviewPanel() {
  const { t } = useTranslation()
  const [modelPage, setModelPage] = useState(0)
  const [modelDistributionPeriod, setModelDistributionPeriod] =
    useState<DashboardModelDistributionPeriod>('today')
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
  const modelPageCount = Math.max(
    1,
    Math.ceil(distribution.length / MODEL_DISTRIBUTION_PAGE_SIZE)
  )
  const safeModelPage = Math.min(modelPage, modelPageCount - 1)
  const pagedDistribution = useMemo(
    () =>
      [...distribution]
        .sort((a, b) => b.token_used - a.token_used)
        .slice(
          safeModelPage * MODEL_DISTRIBUTION_PAGE_SIZE,
          (safeModelPage + 1) * MODEL_DISTRIBUTION_PAGE_SIZE
        ),
    [distribution, safeModelPage]
  )

  const cacheHitByClient = data?.cache_hit_24h_by_client

  const stats: Array<{
    label: string
    value: string
    hint: string
  }> = [
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
      label: t('24h Codex cache hit rate'),
      value: formatClientCacheRate(cacheHitByClient?.codex),
      hint: formatClientCacheHint(cacheHitByClient?.codex, t),
    },
    {
      label: t('24h Claude Code cache hit rate'),
      value: formatClientCacheRate(cacheHitByClient?.claude_code),
      hint: formatClientCacheHint(cacheHitByClient?.claude_code, t),
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

        <div className='space-y-3'>
          <div className='flex flex-wrap items-center justify-between gap-2'>
            <div>
              <div className='text-sm font-semibold'>
                {t('Model call distribution')}
              </div>
              <div className='text-muted-foreground text-xs'>
                {t('Top models ranked by token consumption')}
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
      </div>
    </section>
  )
}
