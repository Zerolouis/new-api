package service

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const (
	dashboardPerfWindowHours = 24
	dashboardRankingLimit    = 20
	cpaCodexWhamBaseURL      = "https://chatgpt.com"

	dashboardCodexAffinityRule  = "codex cli trace"
	dashboardClaudeAffinityRule = "claude cli trace"
)

type DashboardSiteOverview struct {
	TotalTokens         int64                             `json:"total_tokens"`
	TotalRequests       int64                             `json:"total_requests"`
	RecentTokens        int64                             `json:"recent_tokens"`
	RecentRequests      int64                             `json:"recent_requests"`
	CacheHit24h         DashboardCacheHitSnapshot         `json:"cache_hit_24h"`
	CacheHit24hByClient DashboardCacheHitByClientSnapshot `json:"cache_hit_24h_by_client"`
	SiteUptimeSeconds   int64                             `json:"site_uptime_seconds"`
	AvgRPM              float64                           `json:"avg_rpm"`
	AvgTPM              float64                           `json:"avg_tpm"`
	WindowHours         int                               `json:"window_hours"`
	Health              DashboardHealthSnapshot           `json:"health"`
	ModelDistribution   []DashboardModelDistributionItem  `json:"model_distribution"`
}

type DashboardHealthSnapshot struct {
	SuccessRate  float64                    `json:"success_rate"`
	AvgLatencyMs int64                      `json:"avg_latency_ms"`
	AvgTps       float64                    `json:"avg_tps"`
	TopModels    []DashboardHealthModelItem `json:"top_models"`
}

type DashboardCacheHitSnapshot struct {
	HitRate      float64 `json:"hit_rate"`
	CachedTokens int64   `json:"cached_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
	RequestCount int64   `json:"request_count"`
}

type DashboardCacheHitByClientSnapshot struct {
	Codex      DashboardClientCacheHitSnapshot `json:"codex"`
	ClaudeCode DashboardClientCacheHitSnapshot `json:"claude_code"`
}

type DashboardClientCacheHitSnapshot struct {
	Configured   bool    `json:"configured"`
	HitRate      float64 `json:"hit_rate"`
	CachedTokens int64   `json:"cached_tokens"`
	InputTokens  int64   `json:"input_tokens"`
	RequestCount int64   `json:"request_count"`
}

type DashboardHealthModelItem struct {
	ModelName    string  `json:"model_name"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMs int64   `json:"avg_latency_ms"`
	AvgTps       float64 `json:"avg_tps"`
	RequestCount int64   `json:"request_count"`
}

type DashboardModelDistributionItem struct {
	ModelName    string  `json:"model_name"`
	RequestCount int64   `json:"request_count"`
	TokenUsed    int64   `json:"token_used"`
	Percentage   float64 `json:"percentage"`
}

type DashboardUserRankingItem struct {
	Rank         int    `json:"rank"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	TotalTokens  int64  `json:"total_tokens"`
	RequestCount int64  `json:"request_count"`
}

type DashboardCPAQuotaData struct {
	Configured         bool                       `json:"configured"`
	ChannelsConfigured bool                       `json:"channels_configured"`
	Message            string                     `json:"message,omitempty"`
	Channels           []DashboardCPAChannelItem  `json:"channels"`
	Summary            DashboardCPAQuotaSummary   `json:"summary"`
	Accounts           []DashboardCPAQuotaAccount `json:"accounts"`
}

type DashboardCPAChannelItem struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Type   int    `json:"type"`
	Status int    `json:"status"`
}

type DashboardCPAQuotaSummary struct {
	TotalAccounts     int `json:"total_accounts"`
	AvailableAccounts int `json:"available_accounts"`
	ExhaustedAccounts int `json:"exhausted_accounts"`
	ErrorAccounts     int `json:"error_accounts"`
}

type DashboardCPAQuotaAccount struct {
	Name                    string                   `json:"name"`
	Email                   string                   `json:"email,omitempty"`
	Account                 string                   `json:"account,omitempty"`
	AuthIndex               string                   `json:"auth_index,omitempty"`
	Status                  string                   `json:"status,omitempty"`
	StatusMessage           string                   `json:"status_message,omitempty"`
	PlanType                string                   `json:"plan_type,omitempty"`
	LastRefreshAt           int64                    `json:"last_refresh_at,omitempty"`
	NextRetryAfter          int64                    `json:"next_retry_after,omitempty"`
	AccountExpiresAt        int64                    `json:"account_expires_at,omitempty"`
	AccountRemainingSeconds int64                    `json:"account_remaining_seconds,omitempty"`
	FiveHourWindow          *DashboardCPAQuotaWindow `json:"five_hour_window,omitempty"`
	WeeklyWindow            *DashboardCPAQuotaWindow `json:"weekly_window,omitempty"`
	Error                   string                   `json:"error,omitempty"`
}

type DashboardCPAQuotaWindow struct {
	UsedPercent        float64 `json:"used_percent"`
	RemainingPercent   float64 `json:"remaining_percent"`
	ResetAt            int64   `json:"reset_at,omitempty"`
	ResetAfterSeconds  int64   `json:"reset_after_seconds,omitempty"`
	LimitWindowSeconds int64   `json:"limit_window_seconds,omitempty"`
}

type dashboardPeriod string

const (
	dashboardPeriodAll   dashboardPeriod = "all"
	dashboardPeriodToday dashboardPeriod = "today"
	dashboardPeriodWeek  dashboardPeriod = "week"
)

type cpaSettings struct {
	BaseURL           string
	ManagementKey     string
	ChannelIDs        []int
	ManagementBaseURL string
}

type cpaAuthFilesResponse struct {
	Files []cpaAuthFileEntry `json:"files"`
}

type cpaAuthFileEntry struct {
	Name           string          `json:"name"`
	Provider       string          `json:"provider"`
	Email          string          `json:"email"`
	Account        string          `json:"account"`
	AuthIndex      any             `json:"auth_index"`
	Status         any             `json:"status"`
	StatusMessage  string          `json:"status_message"`
	LastRefresh    any             `json:"last_refresh"`
	NextRetryAfter any             `json:"next_retry_after"`
	IDToken        cpaIDTokenClaim `json:"id_token"`
}

type cpaIDTokenClaim struct {
	ChatGPTAccountID               string `json:"chatgpt_account_id"`
	PlanType                       string `json:"plan_type"`
	ChatGPTSubscriptionActiveUntil any    `json:"chatgpt_subscription_active_until"`
}

type cpaCodexAuthFile struct {
	Type         string `json:"type"`
	Email        string `json:"email"`
	AccountID    string `json:"account_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ProxyURL     string `json:"proxy_url"`
}

type codexRateLimitWindow struct {
	UsedPercent        float64 `json:"used_percent"`
	ResetAt            int64   `json:"reset_at"`
	ResetAfterSeconds  int64   `json:"reset_after_seconds"`
	LimitWindowSeconds int64   `json:"limit_window_seconds"`
}

type codexRateLimit struct {
	PlanType        string                `json:"plan_type"`
	PrimaryWindow   *codexRateLimitWindow `json:"primary_window"`
	SecondaryWindow *codexRateLimitWindow `json:"secondary_window"`
}

type codexUsagePayload struct {
	PlanType  string          `json:"plan_type"`
	Email     string          `json:"email"`
	AccountID string          `json:"account_id"`
	RateLimit *codexRateLimit `json:"rate_limit"`
}

func GetDashboardSiteOverview(modelDistributionPeriodRaw string) (*DashboardSiteOverview, error) {
	totalTotals, err := model.GetDashboardSiteTotals(0, 0)
	if err != nil {
		return nil, err
	}

	endTs := time.Now().Unix()
	startTs := endTs - int64(dashboardPerfWindowHours)*3600
	recentTotals, err := model.GetDashboardSiteTotals(startTs, endTs)
	if err != nil {
		return nil, err
	}

	perfSummary, err := perfmetrics.QuerySummaryAll(dashboardPerfWindowHours)
	if err != nil {
		return nil, err
	}

	cacheHit24h, cacheHit24hByClient, err := getDashboardCacheHitSnapshots(startTs, endTs)
	if err != nil {
		return nil, err
	}

	distributionStartTs, distributionEndTs := resolveDashboardPeriodRange(parseDashboardPeriod(modelDistributionPeriodRaw))
	distributionRows, err := model.GetDashboardModelDistribution(distributionStartTs, distributionEndTs, 0)
	if err != nil {
		return nil, err
	}

	distributionTotalRequests := totalTotals.TotalRequests
	if distributionStartTs > 0 || distributionEndTs > 0 {
		distributionTotals, err := model.GetDashboardSiteTotals(distributionStartTs, distributionEndTs)
		if err != nil {
			return nil, err
		}
		distributionTotalRequests = distributionTotals.TotalRequests
	}

	modelDistribution := make([]DashboardModelDistributionItem, 0, len(distributionRows))
	for _, row := range distributionRows {
		modelDistribution = append(modelDistribution, DashboardModelDistributionItem{
			ModelName:    row.ModelName,
			RequestCount: row.RequestCount,
			TokenUsed:    row.TokenUsed,
			Percentage:   safePercent(float64(row.RequestCount), float64(distributionTotalRequests)),
		})
	}

	health := DashboardHealthSnapshot{}
	if len(perfSummary.Models) > 0 {
		var latencyTotal float64
		var latencyCount float64
		var tpsTotal float64
		var tpsCount float64
		var successTotal float64
		var successCount float64

		for _, row := range perfSummary.Models {
			if row.AvgLatencyMs > 0 {
				latencyTotal += float64(row.AvgLatencyMs)
				latencyCount++
			}
			if row.AvgTps > 0 {
				tpsTotal += row.AvgTps
				tpsCount++
			}
			if !math.IsNaN(row.SuccessRate) {
				successTotal += row.SuccessRate
				successCount++
			}
			health.TopModels = append(health.TopModels, DashboardHealthModelItem{
				ModelName:    row.ModelName,
				SuccessRate:  row.SuccessRate,
				AvgLatencyMs: row.AvgLatencyMs,
				AvgTps:       row.AvgTps,
				RequestCount: row.RequestCount,
			})
		}

		if latencyCount > 0 {
			health.AvgLatencyMs = int64(math.Round(latencyTotal / latencyCount))
		}
		if tpsCount > 0 {
			health.AvgTps = tpsTotal / tpsCount
		}
		if successCount > 0 {
			health.SuccessRate = successTotal / successCount
		}
		if len(health.TopModels) > 5 {
			health.TopModels = health.TopModels[:5]
		}
	}

	windowMinutes := float64(dashboardPerfWindowHours * 60)
	return &DashboardSiteOverview{
		TotalTokens:         totalTotals.TotalTokens,
		TotalRequests:       totalTotals.TotalRequests,
		RecentTokens:        recentTotals.TotalTokens,
		RecentRequests:      recentTotals.TotalRequests,
		CacheHit24h:         cacheHit24h,
		CacheHit24hByClient: cacheHit24hByClient,
		SiteUptimeSeconds:   maxInt64(0, time.Now().Unix()-common.StartTime),
		AvgRPM:              float64(recentTotals.TotalRequests) / windowMinutes,
		AvgTPM:              float64(recentTotals.TotalTokens) / windowMinutes,
		WindowHours:         dashboardPerfWindowHours,
		Health:              health,
		ModelDistribution:   modelDistribution,
	}, nil
}

func getDashboardCacheHitSnapshot(startTs int64, endTs int64) (DashboardCacheHitSnapshot, error) {
	legacy, _, err := getDashboardCacheHitSnapshots(startTs, endTs)
	return legacy, err
}

func getDashboardCacheHitSnapshots(startTs int64, endTs int64) (DashboardCacheHitSnapshot, DashboardCacheHitByClientSnapshot, error) {
	rows, err := model.GetDashboardCacheHitLogRows(startTs, endTs)
	if err != nil {
		return DashboardCacheHitSnapshot{}, DashboardCacheHitByClientSnapshot{}, err
	}
	return buildDashboardCacheHitSnapshot(rows), buildDashboardCacheHitByClientSnapshot(rows, dashboardConfiguredAffinityRules()), nil
}

func buildDashboardCacheHitSnapshot(rows []model.DashboardCacheHitLogRow) DashboardCacheHitSnapshot {
	snapshot := DashboardCacheHitSnapshot{}
	for _, row := range rows {
		totalTokens := int64(row.PromptTokens + row.CompletionTokens)
		if totalTokens > 0 {
			snapshot.TotalTokens += totalTokens
		}
		snapshot.RequestCount++
		cacheTokens := dashboardOtherInt64(row.Other, "cache_tokens")
		if cacheTokens > 0 {
			snapshot.CachedTokens += cacheTokens
		}
	}
	snapshot.HitRate = safePercent(float64(snapshot.CachedTokens), float64(snapshot.TotalTokens))
	return snapshot
}

func buildDashboardCacheHitByClientSnapshot(rows []model.DashboardCacheHitLogRow, configuredRules map[string]bool) DashboardCacheHitByClientSnapshot {
	snapshot := DashboardCacheHitByClientSnapshot{
		Codex: DashboardClientCacheHitSnapshot{
			Configured: configuredRules[dashboardCodexAffinityRule],
		},
		ClaudeCode: DashboardClientCacheHitSnapshot{
			Configured: configuredRules[dashboardClaudeAffinityRule],
		},
	}

	for _, row := range rows {
		other, ok := dashboardOtherMap(row.Other)
		if !ok {
			continue
		}
		ruleName := dashboardAffinityRuleName(other)
		switch ruleName {
		case dashboardCodexAffinityRule:
			if !snapshot.Codex.Configured {
				continue
			}
			snapshot.Codex.RequestCount++
			snapshot.Codex.CachedTokens += maxInt64(0, dashboardAnyInt64(other["cache_tokens"]))
			snapshot.Codex.InputTokens += maxInt64(0, int64(row.PromptTokens))
		case dashboardClaudeAffinityRule:
			if !snapshot.ClaudeCode.Configured {
				continue
			}
			cacheTokens := maxInt64(0, dashboardAnyInt64(other["cache_tokens"]))
			cacheCreationTokens := dashboardCacheCreationTokens(other)
			snapshot.ClaudeCode.RequestCount++
			snapshot.ClaudeCode.CachedTokens += cacheTokens
			snapshot.ClaudeCode.InputTokens += maxInt64(0, int64(row.PromptTokens)) + cacheTokens + cacheCreationTokens
		}
	}

	snapshot.Codex.HitRate = safePercent(float64(snapshot.Codex.CachedTokens), float64(snapshot.Codex.InputTokens))
	snapshot.ClaudeCode.HitRate = safePercent(float64(snapshot.ClaudeCode.CachedTokens), float64(snapshot.ClaudeCode.InputTokens))
	return snapshot
}

func dashboardConfiguredAffinityRules() map[string]bool {
	configured := map[string]bool{
		dashboardCodexAffinityRule:  false,
		dashboardClaudeAffinityRule: false,
	}
	setting := operation_setting.GetChannelAffinitySetting()
	if setting == nil || !setting.Enabled {
		return configured
	}
	for _, rule := range setting.Rules {
		name := strings.TrimSpace(rule.Name)
		if _, ok := configured[name]; ok {
			configured[name] = true
		}
	}
	return configured
}

func dashboardOtherMap(other string) (map[string]any, bool) {
	if strings.TrimSpace(other) == "" {
		return nil, false
	}
	var values map[string]any
	if err := common.UnmarshalJsonStr(other, &values); err != nil {
		return nil, false
	}
	return values, true
}

func dashboardOtherInt64(other string, key string) int64 {
	values, ok := dashboardOtherMap(other)
	if !ok {
		return 0
	}
	return dashboardAnyInt64(values[key])
}

func dashboardAffinityRuleName(other map[string]any) string {
	adminInfo, ok := dashboardAnyMap(other["admin_info"])
	if !ok {
		return ""
	}
	affinity, ok := dashboardAnyMap(adminInfo["channel_affinity"])
	if !ok {
		return ""
	}
	name, _ := affinity["rule_name"].(string)
	return strings.TrimSpace(name)
}

func dashboardCacheCreationTokens(other map[string]any) int64 {
	cacheCreationTokens := maxInt64(0, dashboardAnyInt64(other["cache_creation_tokens"]))
	if cacheCreationTokens > 0 {
		return cacheCreationTokens
	}
	return maxInt64(0, dashboardAnyInt64(other["cache_creation_tokens_5m"])) +
		maxInt64(0, dashboardAnyInt64(other["cache_creation_tokens_1h"]))
}

func dashboardAnyMap(value any) (map[string]any, bool) {
	values, ok := value.(map[string]any)
	if ok {
		return values, true
	}
	return nil, false
}

func dashboardAnyInt64(value any) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		if v <= 0 {
			return 0
		}
		return int64(v)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil || parsed < 0 {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func GetDashboardUserTokenRankings(periodRaw string) ([]DashboardUserRankingItem, error) {
	period := parseDashboardPeriod(periodRaw)
	startTs, endTs := resolveDashboardPeriodRange(period)
	rows, err := model.GetDashboardUserTokenRankings(startTs, endTs, dashboardRankingLimit)
	if err != nil {
		return nil, err
	}

	items := make([]DashboardUserRankingItem, 0, len(rows))
	for idx, row := range rows {
		items = append(items, DashboardUserRankingItem{
			Rank:         idx + 1,
			Username:     row.Username,
			DisplayName:  maskDashboardUsername(row.Username),
			TotalTokens:  row.TotalTokens,
			RequestCount: row.RequestCount,
		})
	}
	return items, nil
}

func GetDashboardCPAQuotaData(ctx context.Context) DashboardCPAQuotaData {
	settings := getCPASettings()
	result := DashboardCPAQuotaData{
		Configured:         false,
		ChannelsConfigured: len(settings.ChannelIDs) > 0,
		Channels:           getCPAChannels(settings.ChannelIDs),
		Accounts:           make([]DashboardCPAQuotaAccount, 0),
	}

	if strings.TrimSpace(settings.BaseURL) == "" || strings.TrimSpace(settings.ManagementKey) == "" {
		result.Message = "CPA is not configured"
		return result
	}

	result.Configured = true
	files, err := fetchCPAAuthFiles(ctx, settings)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	if len(files) == 0 {
		result.Message = "No Codex accounts found in CPA"
		return result
	}

	accounts := fetchCPAQuotaAccounts(ctx, settings, files)
	result.Accounts = accounts
	result.Summary = summarizeCPAQuotaAccounts(accounts)
	if len(result.Channels) == 0 && len(settings.ChannelIDs) > 0 {
		result.Message = "Configured CPA channels were not found"
	}
	return result
}

func parseDashboardPeriod(periodRaw string) dashboardPeriod {
	switch strings.ToLower(strings.TrimSpace(periodRaw)) {
	case string(dashboardPeriodToday):
		return dashboardPeriodToday
	case string(dashboardPeriodWeek):
		return dashboardPeriodWeek
	default:
		return dashboardPeriodAll
	}
}

func resolveDashboardPeriodRange(period dashboardPeriod) (int64, int64) {
	endTs := time.Now().Unix()
	switch period {
	case dashboardPeriodToday:
		return endTs - 24*3600, endTs
	case dashboardPeriodWeek:
		return endTs - 7*24*3600, endTs
	default:
		return 0, 0
	}
}

func maskDashboardUsername(username string) string {
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return "****"
	}
	runes := []rune(trimmed)
	switch len(runes) {
	case 1:
		return "*"
	case 2:
		return string(runes[0]) + "*"
	default:
		return string(runes[0]) + strings.Repeat("*", len(runes)-2) + string(runes[len(runes)-1])
	}
}

func getCPASettings() cpaSettings {
	settings := cpaSettings{}
	common.OptionMapRWMutex.RLock()
	settings.BaseURL = strings.TrimSpace(common.OptionMap["console_setting.cpa_base_url"])
	settings.ManagementKey = strings.TrimSpace(common.OptionMap["console_setting.cpa_management_key"])
	rawChannelIDs := strings.TrimSpace(common.OptionMap["console_setting.cpa_channel_ids"])
	common.OptionMapRWMutex.RUnlock()
	settings.ChannelIDs = parseCPAChannelIDs(rawChannelIDs)
	settings.ManagementBaseURL = normalizeCPAManagementBaseURL(settings.BaseURL)
	return settings
}

func normalizeCPAManagementBaseURL(rawBaseURL string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(rawBaseURL), "/")
	if baseURL == "" {
		return ""
	}
	lowerBaseURL := strings.ToLower(baseURL)
	for strings.HasSuffix(lowerBaseURL, "/management.html") {
		baseURL = strings.TrimRight(baseURL[:len(baseURL)-len("/management.html")], "/")
		lowerBaseURL = strings.ToLower(baseURL)
	}
	if strings.HasSuffix(lowerBaseURL, "/v0/management") {
		return baseURL
	}
	return strings.TrimRight(baseURL, "/") + "/v0/management"
}

func parseCPAChannelIDs(raw string) []int {
	if raw == "" {
		return nil
	}
	var values []any
	if err := common.UnmarshalJsonStr(raw, &values); err != nil {
		return nil
	}
	channelIDs := make([]int, 0, len(values))
	seen := make(map[int]struct{}, len(values))
	for _, value := range values {
		id, ok := parseCPAChannelIDValue(value)
		if !ok {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		channelIDs = append(channelIDs, id)
	}
	return channelIDs
}

func parseCPAChannelIDValue(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		id := int(v)
		return id, float64(id) == v && id > 0
	case string:
		id, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || id <= 0 {
			return 0, false
		}
		return id, true
	default:
		return 0, false
	}
}

func stringifyCPAValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		if math.Trunc(v) == v {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func getCPAChannels(channelIDs []int) []DashboardCPAChannelItem {
	if len(channelIDs) == 0 {
		return []DashboardCPAChannelItem{}
	}
	channels, err := model.GetChannelsByIds(channelIDs)
	if err != nil || len(channels) == 0 {
		return []DashboardCPAChannelItem{}
	}
	ordered := make([]DashboardCPAChannelItem, 0, len(channelIDs))
	channelMap := make(map[int]*model.Channel, len(channels))
	for _, channel := range channels {
		if channel != nil {
			channelMap[channel.Id] = channel
		}
	}
	for _, channelID := range channelIDs {
		channel := channelMap[channelID]
		if channel == nil {
			continue
		}
		ordered = append(ordered, DashboardCPAChannelItem{
			ID:     channel.Id,
			Name:   channel.Name,
			Type:   channel.Type,
			Status: channel.Status,
		})
	}
	return ordered
}

func fetchCPAAuthFiles(ctx context.Context, settings cpaSettings) ([]cpaAuthFileEntry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(settings.ManagementBaseURL, "/")+"/auth-files", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+settings.ManagementKey)
	req.Header.Set("X-Management-Key", settings.ManagementKey)
	req.Header.Set("Accept", "application/json")

	resp, err := GetHttpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to fetch CPA auth files: status=%d", resp.StatusCode)
	}

	var payload cpaAuthFilesResponse
	if err := common.DecodeJson(resp.Body, &payload); err != nil {
		return nil, err
	}

	files := make([]cpaAuthFileEntry, 0, len(payload.Files))
	for _, entry := range payload.Files {
		if !strings.EqualFold(strings.TrimSpace(entry.Provider), "codex") {
			continue
		}
		files = append(files, entry)
	}
	return files, nil
}

func fetchCPAQuotaAccounts(ctx context.Context, settings cpaSettings, files []cpaAuthFileEntry) []DashboardCPAQuotaAccount {
	accounts := make([]DashboardCPAQuotaAccount, len(files))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 4)
	for idx, file := range files {
		wg.Add(1)
		go func(index int, entry cpaAuthFileEntry) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			accounts[index] = fetchSingleCPAQuotaAccount(ctx, settings, entry)
		}(idx, file)
	}
	wg.Wait()
	return accounts
}

func fetchSingleCPAQuotaAccount(ctx context.Context, settings cpaSettings, entry cpaAuthFileEntry) DashboardCPAQuotaAccount {
	account := DashboardCPAQuotaAccount{
		Name:             entry.Name,
		Email:            entry.Email,
		Account:          entry.Account,
		AuthIndex:        stringifyCPAValue(entry.AuthIndex),
		Status:           stringifyCPAValue(entry.Status),
		StatusMessage:    entry.StatusMessage,
		PlanType:         strings.TrimSpace(entry.IDToken.PlanType),
		LastRefreshAt:    parseAnyUnixTime(entry.LastRefresh),
		NextRetryAfter:   parseAnyUnixTime(entry.NextRetryAfter),
		AccountExpiresAt: parseAnyUnixTime(entry.IDToken.ChatGPTSubscriptionActiveUntil),
	}
	if account.AccountExpiresAt > 0 {
		account.AccountRemainingSeconds = maxInt64(0, account.AccountExpiresAt-time.Now().Unix())
	}

	authFile, err := downloadCPACodexAuthFile(ctx, settings, entry.Name)
	if err != nil {
		account.Error = err.Error()
		return account
	}
	if strings.TrimSpace(account.Email) == "" {
		account.Email = strings.TrimSpace(authFile.Email)
	}

	accessToken := strings.TrimSpace(authFile.AccessToken)
	refreshToken := strings.TrimSpace(authFile.RefreshToken)
	accountID := strings.TrimSpace(authFile.AccountID)
	if accountID == "" {
		accountID = strings.TrimSpace(entry.IDToken.ChatGPTAccountID)
	}
	if accessToken == "" || accountID == "" {
		account.Error = "missing access token or account id"
		return account
	}

	proxyClient, err := GetHttpClientWithProxy(strings.TrimSpace(authFile.ProxyURL))
	if err != nil {
		account.Error = err.Error()
		return account
	}

	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	statusCode, body, err := FetchCodexWhamUsage(requestCtx, proxyClient, cpaCodexWhamBaseURL, accessToken, accountID)
	if err != nil {
		account.Error = err.Error()
		return account
	}

	if (statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden) && refreshToken != "" {
		refreshCtx, refreshCancel := context.WithTimeout(ctx, 10*time.Second)
		defer refreshCancel()
		refreshed, refreshErr := RefreshCodexOAuthTokenWithProxy(refreshCtx, refreshToken, strings.TrimSpace(authFile.ProxyURL))
		if refreshErr == nil {
			retryCtx, retryCancel := context.WithTimeout(ctx, 15*time.Second)
			defer retryCancel()
			statusCode, body, err = FetchCodexWhamUsage(retryCtx, proxyClient, cpaCodexWhamBaseURL, refreshed.AccessToken, accountID)
			if err == nil {
				account.LastRefreshAt = time.Now().Unix()
			}
		}
	}

	if err != nil {
		account.Error = err.Error()
		return account
	}
	if statusCode < 200 || statusCode >= 300 {
		account.Error = fmt.Sprintf("upstream status: %d", statusCode)
		return account
	}

	var payload codexUsagePayload
	if err := common.Unmarshal(body, &payload); err != nil {
		account.Error = "invalid usage payload"
		return account
	}
	if strings.TrimSpace(account.PlanType) == "" {
		account.PlanType = strings.TrimSpace(payload.PlanType)
	}
	if strings.TrimSpace(account.Email) == "" {
		account.Email = strings.TrimSpace(payload.Email)
	}
	if strings.TrimSpace(account.Account) == "" {
		account.Account = strings.TrimSpace(payload.AccountID)
	}
	fiveHour, weekly := resolveCodexQuotaWindows(payload.PlanType, payload.RateLimit)
	account.FiveHourWindow = fiveHour
	account.WeeklyWindow = weekly
	return account
}

func downloadCPACodexAuthFile(ctx context.Context, settings cpaSettings, name string) (*cpaCodexAuthFile, error) {
	downloadURL := strings.TrimRight(settings.ManagementBaseURL, "/") + "/auth-files/download?name=" + url.QueryEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+settings.ManagementKey)
	req.Header.Set("X-Management-Key", settings.ManagementKey)
	req.Header.Set("Accept", "application/json")

	resp, err := GetHttpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to download auth file: status=%d", resp.StatusCode)
	}
	var payload cpaCodexAuthFile
	if err := common.DecodeJson(resp.Body, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func summarizeCPAQuotaAccounts(accounts []DashboardCPAQuotaAccount) DashboardCPAQuotaSummary {
	summary := DashboardCPAQuotaSummary{TotalAccounts: len(accounts)}
	for _, account := range accounts {
		if account.Error != "" {
			summary.ErrorAccounts++
			continue
		}
		if isCPAAccountExhausted(account) {
			summary.ExhaustedAccounts++
			continue
		}
		summary.AvailableAccounts++
	}
	return summary
}

func isCPAAccountExhausted(account DashboardCPAQuotaAccount) bool {
	if account.FiveHourWindow != nil && account.FiveHourWindow.RemainingPercent <= 0 {
		return true
	}
	if account.WeeklyWindow != nil && account.WeeklyWindow.RemainingPercent <= 0 {
		return true
	}
	return false
}

func resolveCodexQuotaWindows(planType string, rateLimit *codexRateLimit) (*DashboardCPAQuotaWindow, *DashboardCPAQuotaWindow) {
	if rateLimit == nil {
		return nil, nil
	}

	primary := rateLimit.PrimaryWindow
	secondary := rateLimit.SecondaryWindow
	normalizedPlan := strings.ToLower(strings.TrimSpace(planType))

	var fiveHour *codexRateLimitWindow
	var weekly *codexRateLimitWindow
	for _, window := range []*codexRateLimitWindow{primary, secondary} {
		if window == nil {
			continue
		}
		if window.LimitWindowSeconds >= 24*3600 {
			if weekly == nil {
				weekly = window
			}
			continue
		}
		if fiveHour == nil {
			fiveHour = window
		}
	}

	if normalizedPlan == "free" {
		if weekly == nil {
			weekly = firstNonNilWindow(primary, secondary)
		}
		return nil, normalizeCodexQuotaWindow(weekly)
	}

	if fiveHour == nil && weekly == nil {
		fiveHour = primary
		weekly = secondary
	}
	if fiveHour == nil {
		fiveHour = firstNonNilWindow(primary, secondary, weekly)
	}
	if weekly == nil {
		weekly = firstNonNilWindow(primary, secondary, fiveHour)
	}
	return normalizeCodexQuotaWindow(fiveHour), normalizeCodexQuotaWindow(weekly)
}

func normalizeCodexQuotaWindow(window *codexRateLimitWindow) *DashboardCPAQuotaWindow {
	if window == nil {
		return nil
	}
	used := clampPercent(window.UsedPercent)
	return &DashboardCPAQuotaWindow{
		UsedPercent:        used,
		RemainingPercent:   clampPercent(100 - used),
		ResetAt:            window.ResetAt,
		ResetAfterSeconds:  window.ResetAfterSeconds,
		LimitWindowSeconds: window.LimitWindowSeconds,
	}
}

func firstNonNilWindow(candidates ...*codexRateLimitWindow) *codexRateLimitWindow {
	for _, candidate := range candidates {
		if candidate != nil {
			return candidate
		}
	}
	return nil
}

func parseAnyUnixTime(value any) int64 {
	switch v := value.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0
		}
		if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
			return parsed.Unix()
		}
		if unixValue, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return unixValue
		}
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}

func safePercent(value float64, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return (value / total) * 100
}

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
