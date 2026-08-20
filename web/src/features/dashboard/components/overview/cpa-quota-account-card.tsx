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
import {
  Calendar,
  Clock3,
  Fingerprint,
  Info,
  RefreshCcw,
  Sparkles,
  UserRound,
  X,
} from 'lucide-react'
import { useState, type ComponentType, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Progress } from '@/components/ui/progress'
import type {
  DashboardCPAQuotaAccount,
  DashboardCPAQuotaWindow,
} from '@/features/dashboard/types'
import dayjs from '@/lib/dayjs'
import { formatNumber, formatTimestampToDate } from '@/lib/format'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import { getCpaProviderTheme } from './cpa-quota-provider'

export function CpaQuotaAccountCard(props: {
  account: DashboardCPAQuotaAccount
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const account = props.account
  const theme = getCpaProviderTheme(account.provider)
  const displayName = account.email || account.account || account.name
  const primaryWindow =
    account.five_hour_window ?? account.weekly_window ?? account.monthly_window
  const resetAt =
    account.weekly_window?.reset_at ||
    account.monthly_window?.reset_at ||
    account.five_hour_window?.reset_at
  const isError = Boolean(account.error)
  const isDisabled = account.status?.toLowerCase() === 'disabled'
  const quotaLabel = resolveQuotaLabel(account, primaryWindow, t)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        className={cn(
          'bg-background/70 hover:bg-muted/30 h-full min-w-0 w-full rounded-xl border p-4 text-left shadow-xs transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring',
          open && 'ring-primary/40 ring-2'
        )}
        aria-label={t('View account details')}
      >
        <div className='text-muted-foreground text-[11px] font-medium'>
          {t('Account')}
        </div>
        <div className='mt-2 flex min-w-0 items-center gap-2.5'>
          <span
            className={cn(
              'flex size-8 shrink-0 items-center justify-center rounded-full',
              theme.avatar
            )}
          >
            <UserRound className='size-4' aria-hidden='true' />
          </span>
          <span className='truncate text-sm font-semibold'>
            {maskCpaEmail(displayName)}
          </span>
        </div>

        <div className='mt-4'>
          <div className='text-muted-foreground text-[11px] font-medium'>
            {t('Credential type')}
          </div>
          <span
            className={cn(
              'mt-2 inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium',
              theme.badge
            )}
          >
            <span
              className='flex size-3.5 shrink-0 items-center justify-center'
              aria-hidden='true'
            >
              {getLobeIcon(theme.iconKey, 14)}
            </span>
            {theme.label}
          </span>
        </div>

        <div className='mt-4'>
          <div className='text-muted-foreground text-[11px] font-medium'>
            {t('Quota')}
          </div>
          <div
            className={cn(
              'mt-1.5 text-lg font-semibold tabular-nums',
              !primaryWindow && 'text-muted-foreground text-sm font-medium'
            )}
          >
            {quotaLabel}
          </div>
          {primaryWindow ? (
            <Progress
              value={primaryWindow.remaining_percent}
              className={cn('mt-2 gap-0', theme.bar)}
            />
          ) : null}
        </div>

        <div className='mt-4'>
          <div className='text-muted-foreground text-[11px] font-medium'>
            {t('Reset date')}
          </div>
          <div className='mt-1.5 flex items-center gap-1.5 text-sm'>
            <Calendar
              className='text-muted-foreground size-3.5 shrink-0'
              aria-hidden='true'
            />
            <span>{formatResetDate(resetAt)}</span>
          </div>
        </div>
      </PopoverTrigger>

      <PopoverContent className='w-80 p-3' side='bottom' align='center'>
        <PopoverHeader className='flex flex-row items-center justify-between gap-2'>
          <PopoverTitle className='flex items-center gap-1.5'>
            <Info className='size-3.5' aria-hidden='true' />
            {t('Details')}
          </PopoverTitle>
          <Button
            type='button'
            variant='ghost'
            size='icon-xs'
            onClick={() => setOpen(false)}
            aria-label={t('Close')}
          >
            <X />
          </Button>
        </PopoverHeader>
        <dl className='mt-1 space-y-2 text-xs'>
          <DetailRow
            icon={UserRound}
            label={t('Account')}
            value={displayName}
          />
          <DetailRow
            icon={Sparkles}
            label={t('Plan type')}
            value={formatPlanType(account.plan_type, t)}
          />
          <DetailRow
            icon={RefreshCcw}
            label={t('Last refresh')}
            value={formatOptionalTime(account.last_refresh_at)}
          />
          {account.five_hour_window ? (
            <DetailRow
              icon={Clock3}
              label={t('5-hour quota')}
              value={formatWindowUsage(account.five_hour_window, t)}
            />
          ) : null}
          {account.weekly_window ? (
            <DetailRow
              icon={Calendar}
              label={t('Weekly quota')}
              value={formatWindowUsage(account.weekly_window, t)}
            />
          ) : null}
          {account.monthly_window ? (
            <DetailRow
              icon={Calendar}
              label={t('Monthly quota')}
              value={formatWindowUsage(account.monthly_window, t)}
            />
          ) : null}
          <DetailRow
            icon={Calendar}
            label={t('Reset time')}
            value={formatOptionalTime(resetAt)}
          />
          <DetailRow
            icon={Info}
            label={t('Status')}
            value={
              <span className='inline-flex items-center gap-1.5'>
                <span
                  className={cn(
                    'size-1.5 rounded-full',
                    isError || isDisabled ? 'bg-destructive' : 'bg-emerald-500'
                  )}
                  aria-hidden='true'
                />
                {formatAccountStatus(account, t)}
              </span>
            }
          />
          <DetailRow
            icon={Fingerprint}
            label={t('Auth ID')}
            value={
              account.auth_index
                ? t('Auth #{{index}}', { index: account.auth_index })
                : '-'
            }
          />
        </dl>
      </PopoverContent>
    </Popover>
  )
}

function DetailRow(props: {
  icon: ComponentType<{ className?: string }>
  label: string
  value: ReactNode
}) {
  const Icon = props.icon
  return (
    <div className='grid grid-cols-[minmax(0,7.5rem)_minmax(0,1fr)] items-start gap-2'>
      <dt className='text-muted-foreground flex items-center gap-1.5'>
        <Icon className='size-3.5 shrink-0' aria-hidden='true' />
        <span>{props.label}</span>
      </dt>
      <dd className='min-w-0 truncate text-right font-medium'>{props.value}</dd>
    </div>
  )
}

function maskCpaEmail(value: string): string {
  const trimmed = value.trim()
  if (!trimmed) return '-'
  const at = trimmed.indexOf('@')
  if (at <= 0) {
    return `${trimmed.slice(0, 3)}***`
  }
  return `${trimmed.slice(0, Math.min(3, at))}***${trimmed.slice(at)}`
}

function formatOptionalTime(value?: number): string {
  if (!value) return '-'
  return formatTimestampToDate(value)
}

function formatResetDate(value?: number): string {
  if (!value) return '-'
  return dayjs(value * 1000).format('YYYY-MM-DD')
}

function formatWindowUsage(
  window: DashboardCPAQuotaWindow | undefined,
  t: (key: string, params?: Record<string, unknown>) => string
): string {
  if (!window) return '-'
  return t('{{percent}}% used', { percent: formatNumber(window.used_percent) })
}

function resolveQuotaLabel(
  account: DashboardCPAQuotaAccount,
  primaryWindow: DashboardCPAQuotaWindow | undefined,
  t: (key: string, params?: Record<string, unknown>) => string
): string {
  if (primaryWindow) {
    return t('{{percent}}% remaining', {
      percent: formatNumber(primaryWindow.remaining_percent),
    })
  }
  if (account.error) return account.error
  if (account.plan_type) return formatPlanType(account.plan_type, t)
  return t('No quota data')
}

function formatPlanType(
  planType: string | undefined,
  t: (key: string) => string
): string {
  const trimmed = planType?.trim()
  if (!trimmed) return '-'
  if (trimmed === 'Paid plan') {
    return t('Paid plan')
  }
  return trimmed
}

function formatAccountStatus(
  account: DashboardCPAQuotaAccount,
  t: (key: string) => string
): string {
  if (account.error) return account.error
  if (account.status?.toLowerCase() === 'disabled') {
    return account.status_message || t('Disabled')
  }
  return t('Normal')
}
