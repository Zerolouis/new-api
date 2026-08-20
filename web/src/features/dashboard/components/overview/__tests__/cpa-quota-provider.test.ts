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
import * as LobeIcons from '@lobehub/icons'
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { getCpaProviderTheme } from '../cpa-quota-provider'

function hasLobeIcon(iconKey: string): boolean {
  const [baseKey, variant] = iconKey.split('.')
  const baseIcon = (LobeIcons as Record<string, unknown>)[baseKey]
  if (!baseIcon) return false
  if (!variant) return true
  return Boolean((baseIcon as Record<string, unknown>)[variant])
}

describe('cpa quota provider icons', () => {
  test('maps Codex accounts to the Codex LobeHub icon', () => {
    const theme = getCpaProviderTheme('codex')

    assert.equal(theme.label, 'Codex')
    assert.equal(theme.iconKey, 'Codex.Color')
    assert.equal(hasLobeIcon(theme.iconKey), true)
  })

  test('maps Grok and xAI accounts to the Grok LobeHub icon', () => {
    const grokTheme = getCpaProviderTheme('grok')
    const xaiTheme = getCpaProviderTheme('xai')

    assert.equal(grokTheme.label, 'Grok')
    assert.equal(grokTheme.iconKey, 'Grok')
    assert.equal(xaiTheme.iconKey, 'Grok')
    assert.equal(hasLobeIcon(grokTheme.iconKey), true)
  })

  test('maps Claude accounts to the Claude LobeHub icon', () => {
    const theme = getCpaProviderTheme('claude')

    assert.equal(theme.label, 'Claude')
    assert.equal(theme.iconKey, 'Claude.Color')
    assert.equal(hasLobeIcon(theme.iconKey), true)
  })
})
