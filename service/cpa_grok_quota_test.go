package service

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCPAGrokBillingWeeklySnakeCase(t *testing.T) {
	summary, err := parseCPAGrokBilling([]byte(`{
		"config": {
			"credit_usage_percent": 40,
			"current_period": {
				"type": "USAGE_PERIOD_TYPE_WEEKLY",
				"end": "2026-08-17T00:00:00Z"
			}
		}
	}`))
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.NotNil(t, summary.WeeklyUsedPercent)
	assert.InDelta(t, 40, *summary.WeeklyUsedPercent, 0.0001)
	assert.Equal(t, int64(1786924800), summary.WeeklyResetAt)
}

func TestParseCPAGrokBillingWeekly(t *testing.T) {
	summary, err := parseCPAGrokBilling([]byte(`{
		"config": {
			"creditUsagePercent": 12.5,
			"currentPeriod": {
				"type": "USAGE_PERIOD_TYPE_WEEKLY",
				"start": "2026-08-10T00:00:00Z",
				"end": "2026-08-17T00:00:00Z"
			}
		}
	}`))
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.NotNil(t, summary.WeeklyUsedPercent)
	assert.InDelta(t, 12.5, *summary.WeeklyUsedPercent, 0.0001)
	assert.Equal(t, int64(1786924800), summary.WeeklyResetAt)
	assert.Nil(t, summary.MonthlyUsedPercent)
}

func TestParseCPAGrokBillingMonthly(t *testing.T) {
	summary, err := parseCPAGrokBilling([]byte(`{
		"config": {
			"monthlyLimit": {"val": 15000},
			"used": {"val": 1800},
			"billingPeriodEnd": "2026-09-01T00:00:00Z"
		}
	}`))
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.NotNil(t, summary.MonthlyUsedPercent)
	assert.InDelta(t, 12, *summary.MonthlyUsedPercent, 0.0001)
	assert.Equal(t, int64(1788220800), summary.MonthlyResetAt)
	require.NotNil(t, summary.MonthlyLimitCents)
	assert.Equal(t, 15000.0, *summary.MonthlyLimitCents)
}

func TestMergeCPAGrokBilling(t *testing.T) {
	weeklyUsed := 20.0
	monthlyUsed := 8.0
	limit := 15000.0
	merged := mergeCPAGrokBilling(
		&cpaGrokBillingSummary{WeeklyUsedPercent: &weeklyUsed, WeeklyResetAt: 100},
		&cpaGrokBillingSummary{MonthlyUsedPercent: &monthlyUsed, MonthlyResetAt: 200, MonthlyLimitCents: &limit},
	)
	require.NotNil(t, merged)
	require.NotNil(t, merged.WeeklyUsedPercent)
	require.NotNil(t, merged.MonthlyUsedPercent)
	assert.Equal(t, 20.0, *merged.WeeklyUsedPercent)
	assert.Equal(t, 8.0, *merged.MonthlyUsedPercent)
	assert.Equal(t, int64(100), merged.WeeklyResetAt)
	assert.Equal(t, int64(200), merged.MonthlyResetAt)
	assert.Equal(t, "SuperGrok", grokPlanTypeFromLimit(merged.MonthlyLimitCents))
}

func TestIsPaidXAIAuthFile(t *testing.T) {
	assert.False(t, isPaidXAIAuthFile(&cpaXAIAuthFile{}))
	assert.False(t, isPaidXAIAuthFile(&cpaXAIAuthFile{UsingAPI: true}))
	assert.True(t, isPaidXAIAuthFile(&cpaXAIAuthFile{UsingAPI: true, Prefix: "paid"}))
	assert.True(t, isPaidXAIAuthFile(&cpaXAIAuthFile{UsingAPICamel: true, Prefix: "paid"}))

	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"tier":1}`))
	token := "header." + payload + ".sig"
	assert.True(t, isPaidXAIAuthFile(&cpaXAIAuthFile{AccessToken: token}))
}

func TestParseCPAGrokBillingEmptyConfig(t *testing.T) {
	summary, err := parseCPAGrokBilling([]byte(`{"config":{}}`))
	require.NoError(t, err)
	assert.Nil(t, summary)
}

func TestGrokPlanTypeFromLimit(t *testing.T) {
	heavy := float64(cpaGrokSuperGrokHeavyCents)
	super := float64(cpaGrokSuperGrokCents)
	other := 500.0
	assert.Equal(t, "SuperGrok Heavy", grokPlanTypeFromLimit(&heavy))
	assert.Equal(t, "SuperGrok", grokPlanTypeFromLimit(&super))
	assert.Equal(t, "", grokPlanTypeFromLimit(&other))
	assert.Equal(t, "", grokPlanTypeFromLimit(nil))
}

func TestIsCPAGrokProvider(t *testing.T) {
	assert.True(t, isCPAGrokProvider("xai"))
	assert.True(t, isCPAGrokProvider("Grok"))
	assert.False(t, isCPAGrokProvider("codex"))
}
