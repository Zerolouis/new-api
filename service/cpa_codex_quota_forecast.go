package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

const (
	cpaCodexQuotaForecastWindow     = 24 * time.Hour
	cpaCodexQuotaSampleRetention    = 30 * 24 * time.Hour
	cpaCodexQuotaMinimumSampleGap   = 10 * time.Minute
	cpaCodexQuotaResetIncrease      = 0.5
	cpaCodexQuotaMinimumUsagePerSec = 1e-9
)

const (
	DashboardCPACodexForecastUnavailable = "unavailable"
	DashboardCPACodexForecastCollecting  = "collecting"
	DashboardCPACodexForecastStable      = "stable"
	DashboardCPACodexForecastEstimated   = "estimated"
)

type DashboardCPACodexForecast struct {
	Status                           string  `json:"status"`
	MeasuredAccounts                 int     `json:"measured_accounts"`
	ForecastReadyAccounts            int     `json:"forecast_ready_accounts"`
	TotalRemainingPercent            float64 `json:"total_remaining_percent"`
	TotalRemainingAccountEquivalents float64 `json:"total_remaining_account_equivalents"`
	EstimatedExhaustedAt             int64   `json:"estimated_exhausted_at,omitempty"`
	NextResetAt                      int64   `json:"next_reset_at,omitempty"`
	CanLastUntilReset                *bool   `json:"can_last_until_reset,omitempty"`
	SampledAt                        int64   `json:"sampled_at,omitempty"`
}

type DashboardCPACodexSampleResult struct {
	DiscoveredAccounts int   `json:"discovered_accounts"`
	SampledAccounts    int   `json:"sampled_accounts"`
	ErrorAccounts      int   `json:"error_accounts"`
	DeletedSamples     int64 `json:"deleted_samples"`
	SampledAt          int64 `json:"sampled_at"`
}

func unavailableDashboardCPACodexForecast() DashboardCPACodexForecast {
	return DashboardCPACodexForecast{Status: DashboardCPACodexForecastUnavailable}
}

func IsDashboardCPACodexSamplingConfigured() bool {
	settings := getCPASettings()
	return strings.TrimSpace(settings.BaseURL) != "" && strings.TrimSpace(settings.ManagementKey) != ""
}

func SampleDashboardCPACodexQuotas(ctx context.Context) (DashboardCPACodexSampleResult, error) {
	settings := getCPASettings()
	if strings.TrimSpace(settings.BaseURL) == "" || strings.TrimSpace(settings.ManagementKey) == "" {
		return DashboardCPACodexSampleResult{}, errors.New("CPA is not configured")
	}

	files, err := fetchCPAAuthFiles(ctx, settings)
	if err != nil {
		return DashboardCPACodexSampleResult{}, err
	}
	codexFiles := make([]cpaAuthFileEntry, 0, len(files))
	for _, file := range files {
		if isCPACodexProvider(cpaAuthFileProvider(file)) {
			codexFiles = append(codexFiles, file)
		}
	}

	sampledAt := time.Now().Unix()
	accounts := fetchCPAQuotaAccounts(ctx, settings, codexFiles)
	sampledAccounts, errorAccounts, deletedSamples, err := persistDashboardCPACodexQuotaSamples(ctx, accounts, sampledAt)
	result := DashboardCPACodexSampleResult{
		DiscoveredAccounts: len(codexFiles),
		SampledAccounts:    sampledAccounts,
		ErrorAccounts:      errorAccounts,
		DeletedSamples:     deletedSamples,
		SampledAt:          sampledAt,
	}
	return result, err
}

func persistDashboardCPACodexQuotaSamples(ctx context.Context, accounts []DashboardCPAQuotaAccount, sampledAt int64) (int, int, int64, error) {
	samples := make([]model.CPACodexQuotaSample, 0, len(accounts))
	errorAccounts := 0
	for _, account := range accounts {
		if !isCPACodexProvider(account.Provider) {
			continue
		}
		if account.Error != "" {
			errorAccounts++
			continue
		}
		bindingRemaining, bindingResetAt, ok := resolveDashboardCPACodexBindingWindow(account)
		if !ok || strings.TrimSpace(account.Name) == "" {
			errorAccounts++
			continue
		}

		sample := model.CPACodexQuotaSample{
			AccountKey:              dashboardCPACodexAccountKey(account.Name),
			BindingRemainingPercent: bindingRemaining,
			BindingResetAt:          bindingResetAt,
			SampledAt:               sampledAt,
		}
		if account.FiveHourWindow != nil {
			remaining := account.FiveHourWindow.RemainingPercent
			sample.FiveHourRemainingPercent = &remaining
			sample.FiveHourResetAt = account.FiveHourWindow.ResetAt
		}
		if account.WeeklyWindow != nil {
			remaining := account.WeeklyWindow.RemainingPercent
			sample.WeeklyRemainingPercent = &remaining
			sample.WeeklyResetAt = account.WeeklyWindow.ResetAt
		}
		samples = append(samples, sample)
	}

	if err := model.CreateCPACodexQuotaSamples(ctx, samples); err != nil {
		logger.LogWarn(ctx, "CPA Codex quota sample persistence failed: "+err.Error())
		return 0, errorAccounts, 0, err
	}
	deleted, err := model.DeleteCPACodexQuotaSamplesBefore(ctx, sampledAt-int64(cpaCodexQuotaSampleRetention.Seconds()))
	if err != nil {
		logger.LogWarn(ctx, "CPA Codex quota sample cleanup failed: "+err.Error())
		return len(samples), errorAccounts, 0, err
	}
	return len(samples), errorAccounts, deleted, nil
}

func buildDashboardCPACodexForecast(ctx context.Context, accounts []DashboardCPAQuotaAccount, now time.Time) DashboardCPACodexForecast {
	forecast := unavailableDashboardCPACodexForecast()
	accountKeys := make([]string, 0, len(accounts))
	for _, account := range accounts {
		if !isCPACodexProvider(account.Provider) || account.Error != "" || strings.TrimSpace(account.Name) == "" {
			continue
		}
		remaining, resetAt, ok := resolveDashboardCPACodexBindingWindow(account)
		if !ok {
			continue
		}
		accountKey := dashboardCPACodexAccountKey(account.Name)
		accountKeys = append(accountKeys, accountKey)
		forecast.TotalRemainingPercent += remaining
		if resetAt > now.Unix() && (forecast.NextResetAt == 0 || resetAt < forecast.NextResetAt) {
			forecast.NextResetAt = resetAt
		}
	}

	forecast.MeasuredAccounts = len(accountKeys)
	forecast.TotalRemainingAccountEquivalents = forecast.TotalRemainingPercent / 100
	if len(accountKeys) == 0 {
		return forecast
	}

	samples, err := model.GetCPACodexQuotaSamples(ctx, accountKeys, now.Add(-cpaCodexQuotaForecastWindow).Unix())
	if err != nil {
		logger.LogWarn(ctx, "CPA Codex quota forecast query failed: "+err.Error())
		return forecast
	}
	samplesByAccount := make(map[string][]model.CPACodexQuotaSample, len(accountKeys))
	for _, sample := range samples {
		samplesByAccount[sample.AccountKey] = append(samplesByAccount[sample.AccountKey], sample)
		if sample.SampledAt > forecast.SampledAt {
			forecast.SampledAt = sample.SampledAt
		}
	}

	var totalUsagePerSecond float64
	for _, accountKey := range accountKeys {
		usagePerSecond, ready := estimateDashboardCPACodexUsageRate(samplesByAccount[accountKey])
		if !ready {
			continue
		}
		forecast.ForecastReadyAccounts++
		totalUsagePerSecond += usagePerSecond
	}
	if forecast.ForecastReadyAccounts < forecast.MeasuredAccounts {
		forecast.Status = DashboardCPACodexForecastCollecting
		return forecast
	}

	if forecast.TotalRemainingPercent <= 0 {
		forecast.Status = DashboardCPACodexForecastEstimated
		forecast.EstimatedExhaustedAt = now.Unix()
		if forecast.NextResetAt > 0 {
			canLast := false
			forecast.CanLastUntilReset = &canLast
		}
		return forecast
	}
	if totalUsagePerSecond <= cpaCodexQuotaMinimumUsagePerSec {
		forecast.Status = DashboardCPACodexForecastStable
		if forecast.NextResetAt > 0 {
			canLast := true
			forecast.CanLastUntilReset = &canLast
		}
		return forecast
	}

	secondsUntilExhausted := math.Ceil(forecast.TotalRemainingPercent / totalUsagePerSecond)
	if math.IsInf(secondsUntilExhausted, 0) || math.IsNaN(secondsUntilExhausted) || secondsUntilExhausted > float64(math.MaxInt64-now.Unix()) {
		forecast.Status = DashboardCPACodexForecastStable
		if forecast.NextResetAt > 0 {
			canLast := true
			forecast.CanLastUntilReset = &canLast
		}
		return forecast
	}

	forecast.Status = DashboardCPACodexForecastEstimated
	forecast.EstimatedExhaustedAt = now.Unix() + int64(secondsUntilExhausted)
	if forecast.NextResetAt > 0 {
		canLast := forecast.EstimatedExhaustedAt >= forecast.NextResetAt
		forecast.CanLastUntilReset = &canLast
	}
	return forecast
}

func resolveDashboardCPACodexBindingWindow(account DashboardCPAQuotaAccount) (float64, int64, bool) {
	windows := []*DashboardCPAQuotaWindow{account.FiveHourWindow, account.WeeklyWindow, account.MonthlyWindow}
	var remaining float64
	var resetAt int64
	found := false
	for _, window := range windows {
		if window == nil {
			continue
		}
		windowRemaining := clampPercent(window.RemainingPercent)
		if !found || windowRemaining < remaining || (windowRemaining == remaining && earlierDashboardCPAReset(window.ResetAt, resetAt)) {
			remaining = windowRemaining
			resetAt = window.ResetAt
			found = true
		}
	}
	return remaining, resetAt, found
}

func earlierDashboardCPAReset(candidate int64, current int64) bool {
	if candidate <= 0 {
		return false
	}
	return current <= 0 || candidate < current
}

func dashboardCPACodexAccountKey(name string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(name)))
	return fmt.Sprintf("%x", sum)
}

func estimateDashboardCPACodexUsageRate(samples []model.CPACodexQuotaSample) (float64, bool) {
	if len(samples) < 2 {
		return 0, false
	}
	segmentStart := 0
	for idx := 1; idx < len(samples); idx++ {
		increase := samples[idx].BindingRemainingPercent - samples[idx-1].BindingRemainingPercent
		if increase >= cpaCodexQuotaResetIncrease {
			segmentStart = idx
		}
	}
	segment := samples[segmentStart:]
	if len(segment) < 2 {
		return 0, false
	}

	slopes := make([]float64, 0, len(segment)*(len(segment)-1)/2)
	minimumGapSeconds := int64(cpaCodexQuotaMinimumSampleGap.Seconds())
	for left := 0; left < len(segment)-1; left++ {
		for right := left + 1; right < len(segment); right++ {
			deltaSeconds := segment[right].SampledAt - segment[left].SampledAt
			if deltaSeconds < minimumGapSeconds {
				continue
			}
			slopes = append(slopes, (segment[right].BindingRemainingPercent-segment[left].BindingRemainingPercent)/float64(deltaSeconds))
		}
	}
	if len(slopes) == 0 {
		return 0, false
	}
	sort.Float64s(slopes)
	median := slopes[len(slopes)/2]
	if len(slopes)%2 == 0 {
		median = (slopes[len(slopes)/2-1] + slopes[len(slopes)/2]) / 2
	}
	return math.Max(0, -median), true
}
