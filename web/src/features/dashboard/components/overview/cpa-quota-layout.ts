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

/** Minimum card track width used by the CPA account grid. */
export const CPA_QUOTA_ACCOUNT_CARD_MIN_WIDTH = '16rem'

/**
 * Fluid card grid from phone to ultrawide.
 * auto-fill keeps empty tracks so a few cards do not stretch across the row.
 * min(100%, 16rem) keeps a single column from overflowing on narrow screens.
 */
export const cpaQuotaAccountGridClassName =
  'grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(min(100%,16rem),1fr))]'
