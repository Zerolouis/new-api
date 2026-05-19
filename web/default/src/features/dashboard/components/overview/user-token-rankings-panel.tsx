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
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Medal, TrendingUp } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatCompactNumber } from '@/lib/format'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { getDashboardUserRankings } from '@/features/dashboard/api'

type RankingPeriod = 'all' | 'today' | 'week'

export function UserTokenRankingsPanel() {
  const { t } = useTranslation()
  const [period, setPeriod] = useState<RankingPeriod>('all')

  const rankingsQuery = useQuery({
    queryKey: ['dashboard', 'user-rankings', period],
    queryFn: () => getDashboardUserRankings(period),
    staleTime: 60 * 1000,
    retry: false,
  })

  const items = rankingsQuery.data?.data ?? []

  const periods: Array<{ key: RankingPeriod; label: string }> = [
    { key: 'all', label: t('History') },
    { key: 'today', label: t('1 Day') },
    { key: 'week', label: t('7 Days') },
  ]

  return (
    <section className='bg-card h-full overflow-hidden rounded-2xl border shadow-xs'>
      <div className='flex items-center gap-2 border-b px-4 py-3 sm:px-5'>
        <Medal className='text-muted-foreground/60 size-4 shrink-0' aria-hidden='true' />
        <h3 className='text-sm font-semibold'>{t('User token leaderboard')}</h3>
      </div>

      <div className='space-y-4 p-4 sm:p-5'>
        <div className='flex flex-wrap items-center gap-2'>
          {periods.map((item) => (
            <Button
              key={item.key}
              size='sm'
              variant={period === item.key ? 'default' : 'outline'}
              onClick={() => setPeriod(item.key)}
            >
              {item.label}
            </Button>
          ))}
        </div>

        {rankingsQuery.isLoading ? (
          <div className='space-y-2'>
            {Array.from({ length: 8 }).map((_, index) => (
              <Skeleton key={index} className='h-12 w-full rounded-xl' />
            ))}
          </div>
        ) : items.length === 0 ? (
          <div className='text-muted-foreground bg-muted/25 rounded-xl border border-dashed px-4 py-6 text-sm'>
            {t('No token consumption data is available for this time range')}
          </div>
        ) : (
          <ScrollArea className='h-[28rem] pe-3'>
            <div className='space-y-2'>
              {items.map((item) => (
                <div
                  key={`${period}-${item.rank}-${item.username}`}
                  className='bg-background/70 flex items-center gap-3 rounded-xl border px-3 py-3'
                >
                  <div
                    className={cn(
                      'bg-muted flex size-8 shrink-0 items-center justify-center rounded-lg font-mono text-sm font-semibold',
                      item.rank <= 3 && 'bg-primary/10 text-primary'
                    )}
                  >
                    {item.rank}
                  </div>

                  <div className='min-w-0 flex-1'>
                    <div className='truncate text-sm font-medium'>
                      {item.display_name}
                    </div>
                    <div className='text-muted-foreground text-xs'>
                      {t('{{count}} requests', {
                        count: formatCompactNumber(item.request_count),
                      })}
                    </div>
                  </div>

                  <div className='text-right'>
                    <div className='font-mono text-sm font-semibold tabular-nums'>
                      {formatCompactNumber(item.total_tokens)}
                    </div>
                    <div className='text-muted-foreground text-xs'>
                      {t('tokens')}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </ScrollArea>
        )}

        <div className='text-muted-foreground flex items-center gap-2 text-xs'>
          <TrendingUp className='size-3.5 shrink-0' aria-hidden='true' />
          <span>
            {t('Usernames are masked and rankings are sorted by token consumption')}
          </span>
        </div>
      </div>
    </section>
  )
}
