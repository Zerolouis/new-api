package service

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskDashboardUsername(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: "****"},
		{name: "single", input: "a", expected: "*"},
		{name: "double", input: "ab", expected: "a*"},
		{name: "long", input: "albert", expected: "a****t"},
		{name: "unicode", input: "测试用户", expected: "测**户"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := maskDashboardUsername(testCase.input)
			if actual != testCase.expected {
				t.Fatalf("maskDashboardUsername(%q) = %q, want %q", testCase.input, actual, testCase.expected)
			}
		})
	}
}

func TestParseDashboardPeriod(t *testing.T) {
	testCases := []struct {
		input    string
		expected dashboardPeriod
	}{
		{input: "", expected: dashboardPeriodAll},
		{input: "all", expected: dashboardPeriodAll},
		{input: "today", expected: dashboardPeriodToday},
		{input: "week", expected: dashboardPeriodWeek},
		{input: "unknown", expected: dashboardPeriodAll},
	}

	for _, testCase := range testCases {
		actual := parseDashboardPeriod(testCase.input)
		if actual != testCase.expected {
			t.Fatalf("parseDashboardPeriod(%q) = %q, want %q", testCase.input, actual, testCase.expected)
		}
	}
}

func TestNormalizeCPAManagementBaseURL(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{input: "http://example.com:18789/management.html", expected: "http://example.com:18789/v0/management"},
		{input: "http://example.com:18789/v0/management", expected: "http://example.com:18789/v0/management"},
		{input: " http://example.com:18789/ ", expected: "http://example.com:18789/v0/management"},
	}

	for _, testCase := range testCases {
		if actual := normalizeCPAManagementBaseURL(testCase.input); actual != testCase.expected {
			t.Fatalf("normalizeCPAManagementBaseURL(%q) = %q, want %q", testCase.input, actual, testCase.expected)
		}
	}
}

func TestStringifyCPAValue(t *testing.T) {
	if actual := stringifyCPAValue("f02086ab66bf69d9"); actual != "f02086ab66bf69d9" {
		t.Fatalf("string auth_index = %q", actual)
	}
	if actual := stringifyCPAValue(float64(12)); actual != "12" {
		t.Fatalf("numeric auth_index = %q", actual)
	}
}

func TestBuildDashboardCacheHitSnapshot(t *testing.T) {
	rows := []model.DashboardCacheHitLogRow{
		{PromptTokens: 100, CompletionTokens: 50, Other: `{"cache_tokens":30}`},
		{PromptTokens: 40, CompletionTokens: 10, Other: `{"cache_tokens":"20"}`},
		{PromptTokens: 25, CompletionTokens: 25, Other: `{"cache_tokens":0}`},
		{PromptTokens: 10, CompletionTokens: 0, Other: `{bad json`},
	}

	actual := buildDashboardCacheHitSnapshot(rows)
	if actual.CachedTokens != 50 {
		t.Fatalf("CachedTokens = %d, want 50", actual.CachedTokens)
	}
	if actual.TotalTokens != 260 {
		t.Fatalf("TotalTokens = %d, want 260", actual.TotalTokens)
	}
	if actual.RequestCount != 4 {
		t.Fatalf("RequestCount = %d, want 4", actual.RequestCount)
	}
	wantRate := 50.0 / 260.0 * 100
	if math.Abs(actual.HitRate-wantRate) > 0.000001 {
		t.Fatalf("HitRate = %f, want %f", actual.HitRate, wantRate)
	}
}

func TestBuildDashboardCacheHitSnapshotZeroTokens(t *testing.T) {
	actual := buildDashboardCacheHitSnapshot([]model.DashboardCacheHitLogRow{
		{Other: `{"cache_tokens":20}`},
	})
	if actual.HitRate != 0 {
		t.Fatalf("HitRate = %f, want 0", actual.HitRate)
	}
	if actual.CachedTokens != 20 {
		t.Fatalf("CachedTokens = %d, want 20", actual.CachedTokens)
	}
}

func TestBuildDashboardCacheHitByClientSnapshot(t *testing.T) {
	rows := []model.DashboardCacheHitLogRow{
		{
			PromptTokens: 174295,
			Other:        `{"admin_info":{"channel_affinity":{"rule_name":"codex cli trace"}},"cache_tokens":172928}`,
		},
		{
			PromptTokens:     1,
			CompletionTokens: 758,
			Other:            `{"admin_info":{"channel_affinity":{"rule_name":"claude cli trace"}},"cache_tokens":83789,"cache_creation_tokens":856}`,
		},
		{
			PromptTokens: 500,
			Other:        `{"admin_info":{"channel_affinity":{"rule_name":"other rule"}},"cache_tokens":400}`,
		},
		{
			PromptTokens: 100,
			Other:        `{bad json`,
		},
		{
			PromptTokens: 100,
			Other:        `{"cache_tokens":90}`,
		},
	}

	actual := buildDashboardCacheHitByClientSnapshot(rows, map[string]bool{
		dashboardCodexAffinityRule:  true,
		dashboardClaudeAffinityRule: true,
	})

	if !actual.Codex.Configured {
		t.Fatal("Codex.Configured = false, want true")
	}
	if actual.Codex.CachedTokens != 172928 {
		t.Fatalf("Codex.CachedTokens = %d, want 172928", actual.Codex.CachedTokens)
	}
	if actual.Codex.InputTokens != 174295 {
		t.Fatalf("Codex.InputTokens = %d, want 174295", actual.Codex.InputTokens)
	}
	if actual.Codex.RequestCount != 1 {
		t.Fatalf("Codex.RequestCount = %d, want 1", actual.Codex.RequestCount)
	}
	wantCodexRate := 172928.0 / 174295.0 * 100
	if math.Abs(actual.Codex.HitRate-wantCodexRate) > 0.000001 {
		t.Fatalf("Codex.HitRate = %f, want %f", actual.Codex.HitRate, wantCodexRate)
	}

	if !actual.ClaudeCode.Configured {
		t.Fatal("ClaudeCode.Configured = false, want true")
	}
	if actual.ClaudeCode.CachedTokens != 83789 {
		t.Fatalf("ClaudeCode.CachedTokens = %d, want 83789", actual.ClaudeCode.CachedTokens)
	}
	if actual.ClaudeCode.InputTokens != 84646 {
		t.Fatalf("ClaudeCode.InputTokens = %d, want 84646", actual.ClaudeCode.InputTokens)
	}
	if actual.ClaudeCode.RequestCount != 1 {
		t.Fatalf("ClaudeCode.RequestCount = %d, want 1", actual.ClaudeCode.RequestCount)
	}
	wantClaudeRate := 83789.0 / 84646.0 * 100
	if math.Abs(actual.ClaudeCode.HitRate-wantClaudeRate) > 0.000001 {
		t.Fatalf("ClaudeCode.HitRate = %f, want %f", actual.ClaudeCode.HitRate, wantClaudeRate)
	}
}

func TestBuildDashboardCacheHitByClientSnapshotUnconfigured(t *testing.T) {
	rows := []model.DashboardCacheHitLogRow{
		{
			PromptTokens: 100,
			Other:        `{"admin_info":{"channel_affinity":{"rule_name":"codex cli trace"}},"cache_tokens":90}`,
		},
		{
			PromptTokens: 100,
			Other:        `{"admin_info":{"channel_affinity":{"rule_name":"claude cli trace"}},"cache_tokens":90}`,
		},
	}

	actual := buildDashboardCacheHitByClientSnapshot(rows, map[string]bool{
		dashboardCodexAffinityRule: true,
	})

	if !actual.Codex.Configured {
		t.Fatal("Codex.Configured = false, want true")
	}
	if actual.Codex.RequestCount != 1 {
		t.Fatalf("Codex.RequestCount = %d, want 1", actual.Codex.RequestCount)
	}
	if actual.ClaudeCode.Configured {
		t.Fatal("ClaudeCode.Configured = true, want false")
	}
	if actual.ClaudeCode.RequestCount != 0 || actual.ClaudeCode.CachedTokens != 0 || actual.ClaudeCode.InputTokens != 0 || actual.ClaudeCode.HitRate != 0 {
		t.Fatalf("ClaudeCode snapshot counted unconfigured logs: %+v", actual.ClaudeCode)
	}
}

func TestIsCPADashboardAuthFile(t *testing.T) {
	testCases := []struct {
		name     string
		entry    cpaAuthFileEntry
		included bool
		provider string
	}{
		{
			name:     "codex provider",
			entry:    cpaAuthFileEntry{Provider: "codex"},
			included: true,
			provider: "codex",
		},
		{
			name:     "xai provider",
			entry:    cpaAuthFileEntry{Provider: "xai"},
			included: true,
			provider: "xai",
		},
		{
			name:     "grok provider",
			entry:    cpaAuthFileEntry{Provider: "Grok"},
			included: true,
			provider: "grok",
		},
		{
			name:     "type fallback when provider empty",
			entry:    cpaAuthFileEntry{Type: "xai"},
			included: true,
			provider: "xai",
		},
		{
			name:     "provider wins over type",
			entry:    cpaAuthFileEntry{Provider: "codex", Type: "claude"},
			included: true,
			provider: "codex",
		},
		{
			name:     "claude excluded",
			entry:    cpaAuthFileEntry{Provider: "claude"},
			included: false,
			provider: "claude",
		},
		{
			name:     "empty excluded",
			entry:    cpaAuthFileEntry{},
			included: false,
			provider: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.provider, cpaAuthFileProvider(testCase.entry))
			assert.Equal(t, testCase.included, isCPADashboardAuthFile(testCase.entry))
		})
	}
}

func TestIsCPAAccountExhausted(t *testing.T) {
	assert.False(t, isCPAAccountExhausted(DashboardCPAQuotaAccount{}))
	assert.False(t, isCPAAccountExhausted(DashboardCPAQuotaAccount{
		WeeklyWindow: &DashboardCPAQuotaWindow{RemainingPercent: 12.5},
	}))
	assert.True(t, isCPAAccountExhausted(DashboardCPAQuotaAccount{
		MonthlyWindow: &DashboardCPAQuotaWindow{RemainingPercent: 0},
	}))
}
