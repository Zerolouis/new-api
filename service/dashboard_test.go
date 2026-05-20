package service

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/model"
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
