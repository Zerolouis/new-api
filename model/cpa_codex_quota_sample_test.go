package model

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCPACodexQuotaSamplePersistenceAndRetention(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&CPACodexQuotaSample{}))
	require.NoError(t, DB.Exec("DELETE FROM cpa_codex_quota_samples").Error)
	t.Cleanup(func() {
		DB.Exec("DELETE FROM cpa_codex_quota_samples")
	})

	ctx := context.Background()
	samples := []CPACodexQuotaSample{
		{AccountKey: "account-a", BindingRemainingPercent: 80, SampledAt: 1000},
		{AccountKey: "account-a", BindingRemainingPercent: 70, SampledAt: 2000},
		{AccountKey: "account-b", BindingRemainingPercent: 60, SampledAt: 2000},
	}
	require.NoError(t, CreateCPACodexQuotaSamples(ctx, samples))
	require.NoError(t, CreateCPACodexQuotaSamples(ctx, samples))

	accountSamples, err := GetCPACodexQuotaSamples(ctx, []string{"account-a"}, 1500)
	require.NoError(t, err)
	require.Len(t, accountSamples, 1)
	assert.Equal(t, float64(70), accountSamples[0].BindingRemainingPercent)

	deleted, err := DeleteCPACodexQuotaSamplesBefore(ctx, 1500)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	var count int64
	require.NoError(t, DB.Model(&CPACodexQuotaSample{}).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}
