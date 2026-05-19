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
import { AlertCircle, Clock3, KeyRound, RefreshCcw, ServerCog } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { formatCompactNumber, formatNumber, formatTimestampToDate } from '@/lib/format'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import { getDashboardCPAQuotas } from '@/features/dashboard/api'
import type {
  DashboardCPAQuotaAccount,
  DashboardCPAQuotaWindow,
} from '@/features/dashboard/types'

export function CpaQuotaPanel() {
  const { t } = useTranslation()
  const quotaQuery = useQuery({
    queryKey: ['dashboard', 'cpa-quotas'],
    queryFn: getDashboardCPAQuotas,
    staleTime: 60 * 1000,
    retry: false,
  })

  const data = quotaQuery.data?.data

  return (
    <section className='bg-card overflow-hidden rounded-2xl border shadow-xs'>
      <div className='flex items-center gap-2 border-b px-4 py-3 sm:px-5'>
        <ServerCog className='text-muted-foreground/60 size-4 shrink-0' aria-hidden='true' />
        <h3 className='text-sm font-semibold'>{t('CLI Proxy API quota preview')}</h3>
        <span className='text-muted-foreground ml-auto text-xs'>
          {t('Codex account windows from CPA management')}
        </span>
      </div>

      <div className='space-y-4 p-4 sm:p-5'>
        {quotaQuery.isLoading ? (
          <div className='space-y-3'>
            <div className='grid gap-3 md:grid-cols-4'>
              {Array.from({ length: 4 }).map((_, index) => (
                <Skeleton key={index} className='h-20 w-full rounded-xl' />
              ))}
            </div>
            <Skeleton className='h-72 w-full rounded-xl' />
          </div>
        ) : !data?.configured ? (
          <EmptyState
            icon={KeyRound}
            title={t('CPA is not configured')}
            description={
              data?.message || t('Set the CPA base URL and management key in system settings first')
            }
          />
        ) : data.accounts.length === 0 ? (
          <EmptyState
            icon={AlertCircle}
            title={t('No Codex accounts were found')}
            description={data.message || t('CPA returned no usable Codex auth files')}
          />
        ) : (
          <>
            <div className='grid gap-3 md:grid-cols-4'>
              <SummaryCard
                label={t('Accounts')}
                value={formatCompactNumber(data.summary.total_accounts)}
                hint={t('Total Codex accounts')}
              />
              <SummaryCard
                label={t('Available')}
                value={formatCompactNumber(data.summary.available_accounts)}
                hint={t('Accounts with remaining quota')}
              />
              <SummaryCard
                label={t('Exhausted')}
                value={formatCompactNumber(data.summary.exhausted_accounts)}
                hint={t('Reached at least one quota window')}
              />
              <SummaryCard
                label={t('Errors')}
                value={formatCompactNumber(data.summary.error_accounts)}
                hint={t('Accounts that could not be inspected')}
              />
            </div>

            <div className='flex flex-wrap items-center gap-2'>
              <Badge variant='outline'>
                {data.channels_configured
                  ? t('CPA channels: {{count}}', { count: data.channels.length })
                  : t('CPA channels: Unset')}
              </Badge>
              {data.channels.map((channel) => (
                <Badge key={channel.id} variant='secondary'>
                  {channel.name}
                </Badge>
              ))}
              {data.message && (
                <span className='text-muted-foreground text-xs'>{data.message}</span>
              )}
            </div>

            <ScrollArea className='h-[30rem] pe-3'>
              <div className='grid gap-3 lg:grid-cols-2'>
                {data.accounts.map((account) => (
                  <QuotaAccountCard key={account.name} account={account} />
                ))}
              </div>
            </ScrollArea>
          </>
        )}
      </div>
    </section>
  )
}

function QuotaAccountCard(props: { account: DashboardCPAQuotaAccount }) {
  const { t } = useTranslation()
  const { account } = props
  const isError = Boolean(account.error)

  return (
    <div className='bg-background/70 space-y-3 rounded-xl border p-4'>
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0'>
          <div className='truncate text-sm font-semibold'>{account.email || account.name}</div>
          <div className='text-muted-foreground truncate text-xs'>
            {account.account || account.name}
          </div>
        </div>
        <Badge variant={isError ? 'destructive' : 'outline'}>
          {account.plan_type || t('Unknown')}
        </Badge>
      </div>

      <div className='grid gap-2 sm:grid-cols-2'>
        <InfoRow
          icon={RefreshCcw}
          label={t('Last refresh')}
          value={formatOptionalTime(account.last_refresh_at)}
        />
        <InfoRow
          icon={Clock3}
          label={t('Account remaining')}
          value={formatRemainingDuration(account.account_remaining_seconds, t)}
        />
      </div>

      {account.error ? (
        <div className='text-destructive bg-destructive/5 rounded-xl border border-dashed px-3 py-3 text-sm'>
          {account.error}
        </div>
      ) : (
        <div className='grid gap-3 sm:grid-cols-2'>
          <QuotaWindowCard
            title={t('5-hour remaining')}
            window={account.five_hour_window}
          />
          <QuotaWindowCard
            title={t('Weekly remaining')}
            window={account.weekly_window}
          />
        </div>
      )}

      <div className='text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1 text-xs'>
        {account.auth_index ? (
          <span>{t('Auth #{{index}}', { index: account.auth_index })}</span>
        ) : null}
        {account.next_retry_after ? (
          <span>{t('Retry at {{time}}', { time: formatOptionalTime(account.next_retry_after) })}</span>
        ) : null}
        {account.status_message ? <span>{account.status_message}</span> : null}
      </div>
    </div>
  )
}

function QuotaWindowCard(props: {
  title: string
  window?: DashboardCPAQuotaWindow
}) {
  const { t } = useTranslation()
  if (!props.window) {
    return (
      <div className='rounded-xl border border-dashed px-3 py-3'>
        <div className='text-sm font-medium'>{props.title}</div>
        <div className='text-muted-foreground mt-2 text-xs'>
          {t('No quota data')}
        </div>
      </div>
    )
  }

  return (
    <div className='rounded-xl border px-3 py-3'>
      <div className='flex items-center justify-between gap-3'>
        <div className='text-sm font-medium'>{props.title}</div>
        <span className='font-mono text-xs font-semibold tabular-nums'>
          {formatNumber(props.window.remaining_percent)}%
        </span>
      </div>
      <Progress
        value={props.window.remaining_percent}
        className='mt-2 gap-0'
      />
      <div className='text-muted-foreground mt-2 space-y-1 text-xs'>
        <div>
          {t('Refresh time')}: {formatOptionalTime(props.window.reset_at)}
        </div>
        <div>
          {t('Remaining window')}: {formatRemainingDuration(props.window.reset_after_seconds, t)}
        </div>
      </div>
    </div>
  )
}

function SummaryCard(props: { label: string; value: string; hint: string }) {
  return (
    <div className='bg-muted/35 rounded-xl border px-3 py-3'>
      <div className='text-muted-foreground text-[11px] font-medium'>
        {props.label}
      </div>
      <div className='mt-2 font-mono text-lg font-semibold tabular-nums'>
        {props.value}
      </div>
      <div className='text-muted-foreground mt-1 text-xs'>{props.hint}</div>
    </div>
  )
}

function InfoRow(props: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: string
}) {
  const Icon = props.icon
  return (
    <div className='bg-muted/25 rounded-xl px-3 py-3'>
      <div className='text-muted-foreground flex items-center gap-1.5 text-[11px] font-medium'>
        <Icon className='size-3.5 shrink-0' aria-hidden='true' />
        <span>{props.label}</span>
      </div>
      <div className='mt-1.5 text-sm'>{props.value}</div>
    </div>
  )
}

function EmptyState(props: {
  icon: React.ComponentType<{ className?: string }>
  title: string
  description: string
}) {
  const Icon = props.icon
  return (
    <div className='text-muted-foreground flex min-h-56 flex-col items-center justify-center rounded-xl border border-dashed px-4 py-8 text-center'>
      <span className='bg-muted mb-3 flex size-10 items-center justify-center rounded-xl'>
        <Icon className='size-5' aria-hidden='true' />
      </span>
      <div className='text-foreground text-sm font-semibold'>{props.title}</div>
      <div className='mt-1 max-w-xl text-sm'>{props.description}</div>
    </div>
  )
}

function formatOptionalTime(value?: number): string {
  if (!value) return '-'
  return formatTimestampToDate(value)
}

function formatRemainingDuration(
  seconds: number | undefined,
  t: (key: string, params?: Record<string, unknown>) => string
): string {
  if (!seconds || seconds <= 0) return '-'
  const total = Math.floor(seconds)
  const days = Math.floor(total / 86400)
  const hours = Math.floor((total % 86400) / 3600)
  const minutes = Math.floor((total % 3600) / 60)

  if (days > 0) return t('{{days}}d {{hours}}h', { days, hours })
  if (hours > 0) return t('{{hours}}h {{minutes}}m', { hours, minutes })
  return t('{{minutes}}m', { minutes: Math.max(minutes, 1) })
}
