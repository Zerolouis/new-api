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
import { useEffect, useMemo, useRef } from 'react'
import * as z from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { MultiSelect } from '@/components/multi-select'
import { getChannels } from '@/features/channels/api'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'
import { removeTrailingSlash } from './utils'

const createCPASchema = (t: (key: string) => string) =>
  z.object({
    console_setting: z.object({
      cpa_base_url: z.string().refine((value) => {
        const trimmed = value.trim()
        if (!trimmed) return true
        return /^https?:\/\//.test(trimmed)
      }, t('Provide a valid URL starting with http:// or https://')),
      cpa_management_key: z.string(),
    }),
    cpaChannelIds: z.array(z.string()),
  })

type CPAFormValues = z.infer<ReturnType<typeof createCPASchema>>

type CPASettingsSectionProps = {
  defaultValues: {
    'console_setting.cpa_base_url': string
    'console_setting.cpa_management_key': string
    'console_setting.cpa_channel_ids': string
  }
}

type NormalizedCPAValues = {
  'console_setting.cpa_base_url': string
  'console_setting.cpa_management_key': string
  'console_setting.cpa_channel_ids': string
}

function parseChannelIds(raw: string): string[] {
  const trimmed = raw.trim()
  if (!trimmed) return []
  try {
    const parsed = JSON.parse(trimmed)
    if (!Array.isArray(parsed)) return []
    return parsed
      .map((value) => String(value).trim())
      .filter((value) => value.length > 0)
  } catch {
    return []
  }
}

function normalizeCPAValues(values: CPAFormValues): NormalizedCPAValues {
  return {
    'console_setting.cpa_base_url': removeTrailingSlash(
      values.console_setting.cpa_base_url
    ),
    'console_setting.cpa_management_key':
      values.console_setting.cpa_management_key.trim(),
    'console_setting.cpa_channel_ids': JSON.stringify(
      values.cpaChannelIds
        .map((value) => Number(value))
        .filter((value) => Number.isInteger(value) && value > 0)
    ),
  }
}

export function CPASettingsSection({
  defaultValues,
}: CPASettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const schema = createCPASchema(t)
  const formDefaults = useMemo<CPAFormValues>(
    () => ({
      console_setting: {
        cpa_base_url: defaultValues['console_setting.cpa_base_url'],
        cpa_management_key:
          defaultValues['console_setting.cpa_management_key'],
      },
      cpaChannelIds: parseChannelIds(
        defaultValues['console_setting.cpa_channel_ids']
      ),
    }),
    [defaultValues]
  )
  const normalizedDefaults = useMemo(
    () => normalizeCPAValues(formDefaults),
    [formDefaults]
  )
  const baselineRef = useRef<NormalizedCPAValues>(normalizedDefaults)

  useEffect(() => {
    baselineRef.current = normalizedDefaults
  }, [normalizedDefaults])

  const form = useForm<CPAFormValues>({
    resolver: zodResolver(schema),
    defaultValues: formDefaults,
  })

  useResetForm(form, formDefaults)

  const channelsQuery = useQuery({
    queryKey: ['system-settings', 'cpa-channels'],
    queryFn: async () => {
      const result = await getChannels({ p: 1, page_size: 1000 })
      return result.data?.items ?? []
    },
    staleTime: 60 * 1000,
  })

  const channelOptions = useMemo(
    () =>
      (channelsQuery.data ?? []).map((channel) => ({
        value: String(channel.id),
        label: `${channel.name} (#${channel.id})`,
      })),
    [channelsQuery.data]
  )

  const onSubmit = async (values: CPAFormValues) => {
    const normalized = normalizeCPAValues(values)
    const updates: Array<{ key: string; value: string }> = []

    if (
      normalized['console_setting.cpa_base_url'] !==
      baselineRef.current['console_setting.cpa_base_url']
    ) {
      updates.push({
        key: 'console_setting.cpa_base_url',
        value: normalized['console_setting.cpa_base_url'],
      })
    }

    if (
      normalized['console_setting.cpa_management_key'] !==
      baselineRef.current['console_setting.cpa_management_key']
    ) {
      updates.push({
        key: 'console_setting.cpa_management_key',
        value: normalized['console_setting.cpa_management_key'],
      })
    }

    if (
      normalized['console_setting.cpa_channel_ids'] !==
      baselineRef.current['console_setting.cpa_channel_ids']
    ) {
      updates.push({
        key: 'console_setting.cpa_channel_ids',
        value: normalized['console_setting.cpa_channel_ids'],
      })
    }

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }

    baselineRef.current = normalized
  }

  const onInvalidSubmit = () => {
    toast.error(t('Please fix the highlighted fields before saving'))
  }

  // eslint-disable-next-line react-hooks/refs
  const handleSubmit = form.handleSubmit(onSubmit, onInvalidSubmit)

  return (
    <SettingsSection
      title={t('CPA Integration')}
      description={t(
        'Configure CLI Proxy API management access and mark which channels belong to CPA'
      )}
    >
      <Form {...form}>
        <form
          onSubmit={handleSubmit}
          autoComplete='off'
          noValidate
          className='space-y-6'
        >
          <FormField
            control={form.control}
            name='console_setting.cpa_base_url'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('CPA Base URL')}</FormLabel>
                <FormControl>
                  <Input
                    type='text'
                    inputMode='url'
                    placeholder={t('https://cpamc.example.com')}
                    autoComplete='off'
                    spellCheck={false}
                    {...field}
                    onChange={(event) => field.onChange(event.target.value)}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Used for CPA management endpoints such as auth-files and quota previews'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='console_setting.cpa_management_key'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('CPA Management Key')}</FormLabel>
                <FormControl>
                  <Input
                    type='password'
                    autoComplete='new-password'
                    placeholder={t('Enter CPA management key')}
                    {...field}
                    onChange={(event) => field.onChange(event.target.value)}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Sent as the management bearer token when loading CPA account quotas'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='cpaChannelIds'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('CPA Channels')}</FormLabel>
                <FormControl>
                  <MultiSelect
                    options={channelOptions}
                    selected={field.value}
                    onChange={field.onChange}
                    placeholder={t('Select CPA channels...')}
                  />
                </FormControl>
                <FormDescription>
                  {channelsQuery.isLoading
                    ? t('Loading channel list...')
                    : t(
                        'Optional. Leave empty if CPA quotas should be visible but no channels are marked yet'
                      )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <Button type='submit' disabled={updateOption.isPending}>
            {updateOption.isPending
              ? t('Saving...')
              : t('Save CPA settings')}
          </Button>
        </form>
      </Form>
    </SettingsSection>
  )
}
