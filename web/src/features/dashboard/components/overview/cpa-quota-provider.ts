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

export type CpaProviderTheme = {
  label: string
  iconKey: string
  avatar: string
  badge: string
  bar: string
}

export function getCpaProviderTheme(provider?: string): CpaProviderTheme {
  const normalized = (provider ?? '').trim().toLowerCase()
  if (normalized === 'xai' || normalized === 'grok') {
    return {
      label: 'Grok',
      iconKey: 'Grok',
      avatar: 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400',
      badge:
        'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
      bar: '[&_[data-slot=progress-indicator]]:bg-emerald-500',
    }
  }
  if (normalized === 'claude' || normalized === 'anthropic') {
    return {
      label: 'Claude',
      iconKey: 'Claude.Color',
      avatar: 'bg-orange-500/15 text-orange-600 dark:text-orange-400',
      badge:
        'border-orange-500/30 bg-orange-500/10 text-orange-700 dark:text-orange-300',
      bar: '[&_[data-slot=progress-indicator]]:bg-orange-500',
    }
  }
  return {
    label: 'Codex',
    iconKey: 'Codex.Color',
    avatar: 'bg-violet-500/15 text-violet-600 dark:text-violet-400',
    badge:
      'border-violet-500/30 bg-violet-500/10 text-violet-700 dark:text-violet-300',
    bar: '[&_[data-slot=progress-indicator]]:bg-violet-500',
  }
}
