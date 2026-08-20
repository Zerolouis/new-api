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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  CPA_QUOTA_ACCOUNT_CARD_MIN_WIDTH,
  cpaQuotaAccountGridClassName,
} from '../cpa-quota-layout'

describe('cpa quota account grid layout', () => {
  test('fills extra columns instead of stretching cards on wide screens', () => {
    assert.match(cpaQuotaAccountGridClassName, /auto-fill/)
    assert.doesNotMatch(cpaQuotaAccountGridClassName, /auto-fit/)
    assert.match(
      cpaQuotaAccountGridClassName,
      new RegExp(
        `minmax\\(min\\(100%,${CPA_QUOTA_ACCOUNT_CARD_MIN_WIDTH}\\),1fr\\)`
      )
    )
  })

  test('keeps a single column from overflowing on narrow phones', () => {
    assert.match(
      cpaQuotaAccountGridClassName,
      /minmax\(min\(100%,16rem\),1fr\)/
    )
  })
})
