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
import { describe, expect, test } from 'vitest'

import { getCpaProviderTheme } from '../cpa-quota-provider'

describe('cpa quota provider icons', () => {
  test('maps Codex accounts to the Codex LobeHub icon', () => {
    const theme = getCpaProviderTheme('codex')

    expect(theme.label).toBe('Codex')
    expect(theme.iconKey).toBe('Codex.Color')
  })

  test('maps Grok and xAI accounts to the Grok LobeHub icon', () => {
    const grokTheme = getCpaProviderTheme('grok')
    const xaiTheme = getCpaProviderTheme('xai')

    expect(grokTheme.label).toBe('Grok')
    expect(grokTheme.iconKey).toBe('Grok')
    expect(xaiTheme.iconKey).toBe('Grok')
  })

  test('maps Claude accounts to the Claude LobeHub icon', () => {
    const theme = getCpaProviderTheme('claude')

    expect(theme.label).toBe('Claude')
    expect(theme.iconKey).toBe('Claude.Color')
  })
})
