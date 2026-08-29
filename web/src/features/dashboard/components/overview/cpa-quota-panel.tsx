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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertCircle, Clock3, KeyRound, RefreshCcw } from 'lucide-react'
import type { ComponentType } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  getDashboardCPAQuotas,
  refreshDashboardCPAQuotaStatus,
} from '@/features/dashboard/api'
import type { DashboardCPAQuotaAccount } from '@/features/dashboard/types'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

import { CpaQuotaAccountCard } from './cpa-quota-account-card'
import { CpaQuotaForecastFooter } from './cpa-quota-forecast-footer'
import { cpaQuotaAccountGridClassName } from './cpa-quota-layout'

export function CpaQuotaPanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const quotaQuery = useQuery({
    queryKey: ['dashboard', 'cpa-quotas'],
    queryFn: getDashboardCPAQuotas,
    staleTime: 60 * 1000,
    retry: false,
  })
  const refreshMutation = useMutation({
    mutationFn: refreshDashboardCPAQuotaStatus,
    onSuccess: (res) => {
      queryClient.setQueryData(['dashboard', 'cpa-quotas'], res)
      toast.success(t('CPA status refreshed'))
    },
    onError: (error) => {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to refresh CPA status')
      )
    },
  })

  const data = quotaQuery.data?.data
  const refreshing = refreshMutation.isPending
  const lastUpdated =
    quotaQuery.dataUpdatedAt > 0
      ? dayjs(quotaQuery.dataUpdatedAt).format('YYYY-MM-DD HH:mm')
      : ''

  return (
    <section className='bg-card overflow-hidden rounded-2xl border shadow-xs'>
      <div className='flex flex-wrap items-start justify-between gap-3 border-b px-4 py-3 sm:px-5'>
        <div className='min-w-0'>
          <h3 className='text-sm font-semibold'>{t('CPA monitoring')}</h3>
          <p className='text-muted-foreground mt-0.5 text-xs'>
            {t('Overview of account quota usage and reset times')}
          </p>
        </div>
        <div className='flex items-center gap-2'>
          {lastUpdated ? (
            <span className='text-muted-foreground flex items-center gap-1.5 text-xs'>
              <Clock3 className='size-3.5 shrink-0' aria-hidden='true' />
              {t('Last updated: {{time}}', { time: lastUpdated })}
            </span>
          ) : null}
          <Button
            type='button'
            variant='outline'
            size='icon-sm'
            onClick={() => refreshMutation.mutate()}
            disabled={
              refreshing || quotaQuery.isLoading || data?.configured === false
            }
            aria-label={t('Refresh CPA status')}
          >
            <RefreshCcw
              className={cn('size-3.5', refreshing && 'animate-spin')}
              aria-hidden='true'
            />
          </Button>
        </div>
      </div>

      <div className='p-4 sm:p-5'>
        <PanelBody
          loading={quotaQuery.isLoading}
          configured={data?.configured}
          message={data?.message}
          accounts={data?.accounts}
        />
        {data?.configured && data.accounts.length > 0 ? (
          <CpaQuotaForecastFooter forecast={data.codex_forecast} />
        ) : null}
      </div>
    </section>
  )
}

function PanelBody(props: {
  loading: boolean
  configured?: boolean
  message?: string
  accounts?: DashboardCPAQuotaAccount[]
}) {
  const { t } = useTranslation()

  if (props.loading) {
    return (
      <div className={cpaQuotaAccountGridClassName}>
        {['cpa-skeleton-1', 'cpa-skeleton-2', 'cpa-skeleton-3'].map((key) => (
          <Skeleton key={key} className='h-56 w-full rounded-xl' />
        ))}
      </div>
    )
  }

  if (!props.configured) {
    return (
      <EmptyState
        icon={KeyRound}
        title={t('CPA is not configured')}
        description={
          props.message ||
          t('Set the CPA base URL and management key in system settings first')
        }
      />
    )
  }

  if (!props.accounts?.length) {
    return (
      <EmptyState
        icon={AlertCircle}
        title={t('No Codex or Grok accounts were found')}
        description={
          props.message || t('CPA returned no usable Codex or Grok auth files')
        }
      />
    )
  }

  return (
    <div className={cpaQuotaAccountGridClassName}>
      {props.accounts.map((account) => (
        <CpaQuotaAccountCard key={account.name} account={account} />
      ))}
    </div>
  )
}

function EmptyState(props: {
  icon: ComponentType<{ className?: string }>
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
