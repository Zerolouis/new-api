package service

import "testing"

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

func TestNormalizeCPAURLs(t *testing.T) {
	testCases := []struct {
		input          string
		publicBase     string
		managementBase string
	}{
		{
			input:          "http://example.com:18789/management.html",
			publicBase:     "http://example.com:18789",
			managementBase: "http://example.com:18789/v0/management",
		},
		{
			input:          "http://example.com:18789/v0/management",
			publicBase:     "http://example.com:18789",
			managementBase: "http://example.com:18789/v0/management",
		},
		{
			input:          " http://example.com:18789/ ",
			publicBase:     "http://example.com:18789",
			managementBase: "http://example.com:18789/v0/management",
		},
	}

	for _, testCase := range testCases {
		if actual := normalizeCPAPublicBaseURL(testCase.input); actual != testCase.publicBase {
			t.Fatalf("normalizeCPAPublicBaseURL(%q) = %q, want %q", testCase.input, actual, testCase.publicBase)
		}
		if actual := normalizeCPAManagementBaseURL(testCase.input); actual != testCase.managementBase {
			t.Fatalf("normalizeCPAManagementBaseURL(%q) = %q, want %q", testCase.input, actual, testCase.managementBase)
		}
	}
}

func TestMapCPAPublicStatusQuotaAccount(t *testing.T) {
	account := mapCPAPublicStatusQuotaAccount(0, cpaPublicStatusQuotaItem{
		ID:                 "quota-1",
		DisplayName:        "te**st@example.com",
		Status:             "active",
		PlanType:           "plus",
		SubscriptionEndsAt: "2026-06-01T00:00:00Z",
		FiveHour: &cpaPublicStatusQuotaWindow{
			UsedPercent:        40,
			RemainingPercent:   60,
			ResetAt:            "2026-05-19T12:00:00Z",
			ResetAfterSeconds:  3600,
			LimitWindowSeconds: 18000,
		},
	})

	if account.Name != "quota-1" || account.Email != "te**st@example.com" || account.PlanType != "plus" {
		t.Fatalf("unexpected account identity: %+v", account)
	}
	if account.FiveHourWindow == nil || account.FiveHourWindow.RemainingPercent != 60 || account.FiveHourWindow.ResetAt == 0 {
		t.Fatalf("unexpected quota window: %+v", account.FiveHourWindow)
	}
	if account.Error != "" {
		t.Fatalf("expected no account error, got %q", account.Error)
	}
}
