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
import type { TimeGranularity } from '@/lib/time'

// ============================================================================
// Quota & Usage Data Types
// ============================================================================

export interface QuotaDataItem {
  id?: number
  user_id?: number
  username?: string
  model_name?: string
  created_at: number
  token_used?: number
  count?: number
  quota?: number
}

// ============================================================================
// Uptime Monitoring Types
// ============================================================================

export interface UptimeMonitor {
  name: string
  uptime: number
  status: number
  group?: string
}

export interface UptimeGroupResult {
  categoryName: string
  monitors: UptimeMonitor[]
}

// ============================================================================
// Dashboard Filter Types
// ============================================================================

export interface DashboardFilters {
  start_timestamp?: Date
  end_timestamp?: Date
  time_granularity?: TimeGranularity
  username?: string
}

export type ConsumptionDistributionChartType = 'bar' | 'area'

export type ModelAnalyticsChartTab = 'trend' | 'proportion' | 'top'

export interface DashboardChartPreferences {
  consumptionDistributionChart: ConsumptionDistributionChartType
  modelAnalyticsChart: ModelAnalyticsChartTab
  defaultTimeRangeDays: number
  defaultTimeGranularity: TimeGranularity
}

// ============================================================================
// API Info Types
// ============================================================================

export interface ApiInfoItem {
  url: string
  route: string
  description: string
  color: string
}

export interface PingStatus {
  latency: number | null
  testing: boolean
  error: boolean
}

export type PingStatusMap = Record<string, PingStatus>

// ============================================================================
// Chart Types
// ============================================================================

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type VChartSpec = Record<string, any>

export interface ProcessedChartData {
  spec_pie: VChartSpec
  spec_line: VChartSpec
  spec_area: VChartSpec
  spec_model_line: VChartSpec
  spec_rank_bar: VChartSpec
  totalQuotaDisplay: string
  totalCountDisplay: string
}

export interface ProcessedUserChartData {
  spec_user_rank: VChartSpec
  spec_user_trend: VChartSpec
}

// ============================================================================
// Announcement Types
// ============================================================================

export interface AnnouncementItem {
  id?: number
  content: string
  publishDate?: string
  type?: 'default' | 'ongoing' | 'success' | 'warning' | 'error'
  extra?: string
}

// ============================================================================
// FAQ Types
// ============================================================================

export interface FAQItem {
  id?: number
  question: string
  answer: string
}

export interface DashboardModelDistributionItem {
  model_name: string
  request_count: number
  token_used: number
  percentage: number
}

export type DashboardModelDistributionPeriod = 'today' | 'week' | 'all'

export interface DashboardHealthModelItem {
  model_name: string
  success_rate: number
  avg_latency_ms: number
  avg_tps: number
  request_count: number
}

export interface DashboardCacheHitSnapshot {
  hit_rate: number
  cached_tokens: number
  total_tokens: number
  request_count: number
}

export interface DashboardClientCacheHitSnapshot {
  configured: boolean
  hit_rate: number
  cached_tokens: number
  input_tokens: number
  request_count: number
}

export interface DashboardCacheHitByClientSnapshot {
  codex: DashboardClientCacheHitSnapshot
  claude_code: DashboardClientCacheHitSnapshot
}

export interface DashboardSiteOverview {
  total_tokens: number
  total_requests: number
  recent_tokens: number
  recent_requests: number
  cache_hit_24h: DashboardCacheHitSnapshot
  cache_hit_24h_by_client: DashboardCacheHitByClientSnapshot
  site_uptime_seconds: number
  avg_rpm: number
  avg_tpm: number
  window_hours: number
  health: {
    success_rate: number
    avg_latency_ms: number
    avg_tps: number
    top_models: DashboardHealthModelItem[]
  }
  model_distribution: DashboardModelDistributionItem[]
}

export interface DashboardUserRankingItem {
  rank: number
  username: string
  display_name: string
  total_tokens: number
  request_count: number
}

export interface DashboardCPAChannelItem {
  id: number
  name: string
  type: number
  status: number
}

export interface DashboardCPAQuotaWindow {
  used_percent: number
  remaining_percent: number
  reset_at?: number
  reset_after_seconds?: number
  limit_window_seconds?: number
}

export interface DashboardCPAQuotaAccount {
  name: string
  email?: string
  account?: string
  auth_index?: string
  status?: string
  status_message?: string
  plan_type?: string
  last_refresh_at?: number
  next_retry_after?: number
  account_expires_at?: number
  account_remaining_seconds?: number
  five_hour_window?: DashboardCPAQuotaWindow
  weekly_window?: DashboardCPAQuotaWindow
  error?: string
}

export interface DashboardCPAQuotaData {
  configured: boolean
  channels_configured: boolean
  message?: string
  channels: DashboardCPAChannelItem[]
  summary: {
    total_accounts: number
    available_accounts: number
    exhausted_accounts: number
    error_accounts: number
  }
  accounts: DashboardCPAQuotaAccount[]
}
