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
import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { createInstance } from 'i18next'
import { I18nextProvider, initReactI18next } from 'react-i18next'
import { describe, expect, test } from 'vitest'

import type { DashboardCPACodexForecast } from '@/features/dashboard/types'

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        '{{percent}}% · {{accounts}} full accounts':
          '{{percent}}% · {{accounts}} full accounts',
        'Based on {{ready}}/{{measured}} accounts':
          'Based on {{ready}}/{{measured}} accounts',
        'Can last until reset': 'Can last until reset',
        'Collecting quota samples': 'Collecting quota samples',
        'Estimated exhaustion': 'Estimated exhaustion',
        'Follow Tibo, meow~': 'Follow Tibo, meow~',
        'Forecast unavailable': 'Forecast unavailable',
        'Next reset: {{time}}': 'Next reset: {{time}}',
        'No measurable consumption': 'No measurable consumption',
        'Quota may run out before reset': 'Quota may run out before reset',
        'Reset outlook': 'Reset outlook',
        'Total remaining': 'Total remaining',
      },
    },
  },
})

import { CpaQuotaForecastFooter } from '../cpa-quota-forecast-footer'
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

async function renderFooter(forecast: DashboardCPACodexForecast) {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <CpaQuotaForecastFooter forecast={forecast} />
      </I18nextProvider>
    )
  })
  return { container, root }
}

async function unmountFooter(
  rendered: Awaited<ReturnType<typeof renderFooter>>
) {
  await act(async () => rendered.root.unmount())
  rendered.container.remove()
}

function estimatedForecast(
  canLastUntilReset: boolean
): DashboardCPACodexForecast {
  return {
    status: 'estimated',
    measured_accounts: 2,
    forecast_ready_accounts: 2,
    total_remaining_percent: 125,
    total_remaining_account_equivalents: 1.25,
    estimated_exhausted_at: 2_000_000_000,
    next_reset_at: 2_000_003_600,
    can_last_until_reset: canLastUntilReset,
    sampled_at: 1_999_999_000,
  }
}

describe('CPA quota forecast footer', () => {
  test('renders the requested Tibo link as a safe new-tab action', async () => {
    const rendered = await renderFooter(estimatedForecast(true))
    const link = rendered.container.querySelector<HTMLAnchorElement>(
      'a[href="https://x.com/thsottiaux"]'
    )

    expect(link).not.toBeNull()
    expect(link?.textContent).toContain('Follow Tibo, meow~')
    expect(link?.target).toBe('_blank')
    expect(link?.rel).toBe('noopener noreferrer')
    expect(rendered.container.textContent).toContain(
      '125% · 1.25 full accounts'
    )

    await unmountFooter(rendered)
  })

  test('marks the exhaustion time as a warning only when quota cannot reach reset', async () => {
    const insufficient = await renderFooter(estimatedForecast(false))
    expect(
      insufficient.container.querySelector('[data-quota-warning="true"]')
    ).not.toBeNull()
    await unmountFooter(insufficient)

    const sufficient = await renderFooter(estimatedForecast(true))
    expect(
      sufficient.container.querySelector('[data-quota-warning="true"]')
    ).toBeNull()
    await unmountFooter(sufficient)
  })
})
