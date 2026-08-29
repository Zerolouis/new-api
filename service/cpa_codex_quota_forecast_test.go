package service

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveDashboardCPACodexBindingWindowUsesLowestRemainingQuota(t *testing.T) {
	remaining, resetAt, ok := resolveDashboardCPACodexBindingWindow(DashboardCPAQuotaAccount{
		FiveHourWindow: &DashboardCPAQuotaWindow{RemainingPercent: 65, ResetAt: 3000},
		WeeklyWindow:   &DashboardCPAQuotaWindow{RemainingPercent: 25, ResetAt: 9000},
	})

	require.True(t, ok)
	assert.Equal(t, float64(25), remaining)
	assert.Equal(t, int64(9000), resetAt)
}

func TestEstimateDashboardCPACodexUsageRateUsesLatestPostResetSegment(t *testing.T) {
	samples := []model.CPACodexQuotaSample{
		{BindingRemainingPercent: 80, SampledAt: 1000},
		{BindingRemainingPercent: 70, SampledAt: 1600},
		{BindingRemainingPercent: 100, SampledAt: 2200},
		{BindingRemainingPercent: 90, SampledAt: 2800},
	}

	rate, ready := estimateDashboardCPACodexUsageRate(samples)
	require.True(t, ready)
	assert.InDelta(t, 10.0/600.0, rate, 0.0000001)
}

func TestBuildDashboardCPACodexForecastCollectsUntilEveryMeasuredAccountIsReady(t *testing.T) {
	resetCPACodexForecastSamples(t)
	now := time.Unix(10_000, 0)
	accounts := []DashboardCPAQuotaAccount{
		codexForecastTestAccount("first.json", 40, now.Unix()+3600),
		codexForecastTestAccount("second.json", 60, now.Unix()+3600),
	}
	require.NoError(t, model.CreateCPACodexQuotaSamples(context.Background(), []model.CPACodexQuotaSample{
		{AccountKey: dashboardCPACodexAccountKey("first.json"), BindingRemainingPercent: 50, SampledAt: now.Unix() - 1200},
		{AccountKey: dashboardCPACodexAccountKey("first.json"), BindingRemainingPercent: 45, SampledAt: now.Unix() - 600},
	}))

	forecast := buildDashboardCPACodexForecast(context.Background(), accounts, now)

	assert.Equal(t, DashboardCPACodexForecastCollecting, forecast.Status)
	assert.Equal(t, 2, forecast.MeasuredAccounts)
	assert.Equal(t, 1, forecast.ForecastReadyAccounts)
	assert.Equal(t, float64(100), forecast.TotalRemainingPercent)
	assert.Equal(t, float64(1), forecast.TotalRemainingAccountEquivalents)
}

func TestBuildDashboardCPACodexForecastMarksInsufficientQuotaBeforeReset(t *testing.T) {
	resetCPACodexForecastSamples(t)
	now := time.Unix(20_000, 0)
	accountName := "insufficient.json"
	require.NoError(t, model.CreateCPACodexQuotaSamples(context.Background(), []model.CPACodexQuotaSample{
		{AccountKey: dashboardCPACodexAccountKey(accountName), BindingRemainingPercent: 20, SampledAt: now.Unix() - 1200},
		{AccountKey: dashboardCPACodexAccountKey(accountName), BindingRemainingPercent: 15, SampledAt: now.Unix() - 600},
	}))

	forecast := buildDashboardCPACodexForecast(context.Background(), []DashboardCPAQuotaAccount{
		codexForecastTestAccount(accountName, 10, now.Unix()+1800),
	}, now)

	require.NotNil(t, forecast.CanLastUntilReset)
	assert.Equal(t, DashboardCPACodexForecastEstimated, forecast.Status)
	assert.Equal(t, now.Unix()+1200, forecast.EstimatedExhaustedAt)
	assert.False(t, *forecast.CanLastUntilReset)
}

func TestBuildDashboardCPACodexForecastCanReachAnEarlierReset(t *testing.T) {
	resetCPACodexForecastSamples(t)
	now := time.Unix(25_000, 0)
	accountName := "sufficient.json"
	require.NoError(t, model.CreateCPACodexQuotaSamples(context.Background(), []model.CPACodexQuotaSample{
		{AccountKey: dashboardCPACodexAccountKey(accountName), BindingRemainingPercent: 20, SampledAt: now.Unix() - 1200},
		{AccountKey: dashboardCPACodexAccountKey(accountName), BindingRemainingPercent: 15, SampledAt: now.Unix() - 600},
	}))

	forecast := buildDashboardCPACodexForecast(context.Background(), []DashboardCPAQuotaAccount{
		codexForecastTestAccount(accountName, 10, now.Unix()+600),
	}, now)

	require.NotNil(t, forecast.CanLastUntilReset)
	assert.Equal(t, DashboardCPACodexForecastEstimated, forecast.Status)
	assert.Equal(t, now.Unix()+1200, forecast.EstimatedExhaustedAt)
	assert.True(t, *forecast.CanLastUntilReset)
}

func TestBuildDashboardCPACodexForecastTreatsNoUsageAsStable(t *testing.T) {
	resetCPACodexForecastSamples(t)
	now := time.Unix(30_000, 0)
	accountName := "stable.json"
	require.NoError(t, model.CreateCPACodexQuotaSamples(context.Background(), []model.CPACodexQuotaSample{
		{AccountKey: dashboardCPACodexAccountKey(accountName), BindingRemainingPercent: 80, SampledAt: now.Unix() - 1200},
		{AccountKey: dashboardCPACodexAccountKey(accountName), BindingRemainingPercent: 80, SampledAt: now.Unix() - 600},
	}))

	forecast := buildDashboardCPACodexForecast(context.Background(), []DashboardCPAQuotaAccount{
		codexForecastTestAccount(accountName, 80, now.Unix()+1800),
	}, now)

	require.NotNil(t, forecast.CanLastUntilReset)
	assert.Equal(t, DashboardCPACodexForecastStable, forecast.Status)
	assert.True(t, *forecast.CanLastUntilReset)
	assert.Zero(t, forecast.EstimatedExhaustedAt)
}

func TestPersistDashboardCPACodexQuotaSamplesExcludesNonCodexAndErrors(t *testing.T) {
	resetCPACodexForecastSamples(t)
	accounts := []DashboardCPAQuotaAccount{
		codexForecastTestAccount("stored.json", 50, 9000),
		{Name: "grok.json", Provider: "grok", WeeklyWindow: &DashboardCPAQuotaWindow{RemainingPercent: 50}},
		{Name: "broken.json", Provider: "codex", Error: "upstream failed"},
	}

	sampled, errors, _, err := persistDashboardCPACodexQuotaSamples(context.Background(), accounts, 5000)
	require.NoError(t, err)
	assert.Equal(t, 1, sampled)
	assert.Equal(t, 1, errors)

	var stored []model.CPACodexQuotaSample
	require.NoError(t, model.DB.Find(&stored).Error)
	require.Len(t, stored, 1)
	assert.Equal(t, dashboardCPACodexAccountKey("stored.json"), stored[0].AccountKey)
	assert.Len(t, stored[0].AccountKey, sha256HexLength)
	assert.NotEqual(t, "stored.json", stored[0].AccountKey)
}

const sha256HexLength = 64

func codexForecastTestAccount(name string, remaining float64, resetAt int64) DashboardCPAQuotaAccount {
	return DashboardCPAQuotaAccount{
		Name:     name,
		Provider: "codex",
		FiveHourWindow: &DashboardCPAQuotaWindow{
			RemainingPercent: remaining,
			ResetAt:          resetAt,
		},
	}
}

func resetCPACodexForecastSamples(t *testing.T) {
	t.Helper()
	require.NoError(t, model.DB.Exec("DELETE FROM cpa_codex_quota_samples").Error)
	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM cpa_codex_quota_samples")
	})
}

func TestDashboardCPACodexAccountKeyIsDeterministicAndAnonymous(t *testing.T) {
	first := dashboardCPACodexAccountKey("account@example.com.json")
	second := dashboardCPACodexAccountKey("account@example.com.json")

	assert.Equal(t, first, second)
	assert.Len(t, first, sha256HexLength)
	assert.NotContains(t, first, "account@example.com")
}
