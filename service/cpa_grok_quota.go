package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	cpaGrokBillingWeeklyURL    = "https://cli-chat-proxy.grok.com/v1/billing?format=credits"
	cpaGrokBillingMonthlyURL   = "https://cli-chat-proxy.grok.com/v1/billing"
	cpaGrokClientVersion       = "0.2.120"
	cpaGrokTokenAuthValue      = "xai-grok-cli"
	cpaGrokPaidPlanType        = "Paid plan"
	cpaGrokSuperGrokPlan       = "SuperGrok"
	cpaGrokSuperGrokHeavy      = "SuperGrok Heavy"
	cpaGrokSuperGrokCents      = 15000
	cpaGrokSuperGrokHeavyCents = 150000
)

type cpaXAIAuthFile struct {
	Type          string `json:"type"`
	Email         string `json:"email"`
	AccessToken   string `json:"access_token"`
	IDToken       string `json:"id_token"`
	Token         string `json:"token"`
	Sub           string `json:"sub"`
	Subject       string `json:"subject"`
	UserID        string `json:"user_id"`
	UsingAPI      any    `json:"using_api"`
	UsingAPICamel any    `json:"usingApi"`
	Prefix        string `json:"prefix"`
}

type cpaAPICallRequest struct {
	AuthIndex string            `json:"auth_index"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Header    map[string]string `json:"header"`
}

type cpaAPICallResponse struct {
	StatusCode int    `json:"status_code"`
	Body       string `json:"body"`
}

type cpaGrokBillingPayload struct {
	Config cpaGrokBillingConfig `json:"config"`
}

type cpaGrokBillingConfig struct {
	CreditUsagePercent  any                   `json:"credit_usage_percent"`
	CreditUsageCamel    any                   `json:"creditUsagePercent"`
	CurrentPeriod       *cpaGrokBillingPeriod `json:"current_period"`
	CurrentPeriodCamel  *cpaGrokBillingPeriod `json:"currentPeriod"`
	MonthlyLimit        any                   `json:"monthly_limit"`
	MonthlyLimitCamel   any                   `json:"monthlyLimit"`
	Used                any                   `json:"used"`
	OnDemandCap         any                   `json:"on_demand_cap"`
	OnDemandCapCamel    any                   `json:"onDemandCap"`
	OnDemandUsed        any                   `json:"on_demand_used"`
	OnDemandUsedCamel   any                   `json:"onDemandUsed"`
	BillingPeriodStart  any                   `json:"billing_period_start"`
	BillingPeriodStartC any                   `json:"billingPeriodStart"`
	BillingPeriodEnd    any                   `json:"billing_period_end"`
	BillingPeriodEndC   any                   `json:"billingPeriodEnd"`
}

type cpaGrokBillingPeriod struct {
	Type  string `json:"type"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type cpaGrokBillingSummary struct {
	WeeklyUsedPercent  *float64
	WeeklyResetAt      int64
	MonthlyUsedPercent *float64
	MonthlyResetAt     int64
	MonthlyLimitCents  *float64
	UsedCents          *float64
}

func applyCPAGrokQuota(
	ctx context.Context,
	settings cpaSettings,
	entry cpaAuthFileEntry,
	account *DashboardCPAQuotaAccount,
) {
	if account == nil {
		return
	}
	if strings.TrimSpace(account.AuthIndex) == "" {
		account.Error = "missing auth index"
		return
	}

	authFile, err := downloadCPAXAIAuthFile(ctx, settings, entry.Name)
	if err == nil {
		if strings.TrimSpace(account.Email) == "" {
			account.Email = strings.TrimSpace(authFile.Email)
		}
		if isPaidXAIAuthFile(authFile) {
			account.PlanType = cpaGrokPaidPlanType
			return
		}
	}

	var weekly *cpaGrokBillingSummary
	var monthly *cpaGrokBillingSummary
	var weeklyErr error
	var monthlyErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		weekly, weeklyErr = fetchCPAGrokBilling(ctx, settings, account.AuthIndex, authFile, cpaGrokBillingWeeklyURL)
	}()
	go func() {
		defer wg.Done()
		monthly, monthlyErr = fetchCPAGrokBilling(ctx, settings, account.AuthIndex, authFile, cpaGrokBillingMonthlyURL)
	}()
	wg.Wait()

	summary := mergeCPAGrokBilling(weekly, monthly)
	if summary == nil {
		if weeklyErr != nil {
			account.Error = weeklyErr.Error()
			return
		}
		if monthlyErr != nil {
			account.Error = monthlyErr.Error()
			return
		}
		account.Error = "no grok quota data"
		return
	}

	if summary.WeeklyUsedPercent != nil {
		account.WeeklyWindow = percentWindow(*summary.WeeklyUsedPercent, summary.WeeklyResetAt)
	}
	if summary.MonthlyUsedPercent != nil {
		account.MonthlyWindow = percentWindow(*summary.MonthlyUsedPercent, summary.MonthlyResetAt)
	}
	if strings.TrimSpace(account.PlanType) == "" {
		account.PlanType = grokPlanTypeFromLimit(summary.MonthlyLimitCents)
	}
}

func downloadCPAXAIAuthFile(ctx context.Context, settings cpaSettings, name string) (*cpaXAIAuthFile, error) {
	data, err := downloadCPAAuthFile(ctx, settings, name)
	if err != nil {
		return nil, err
	}
	var payload cpaXAIAuthFile
	if err := common.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func fetchCPAGrokBilling(
	ctx context.Context,
	settings cpaSettings,
	authIndex string,
	authFile *cpaXAIAuthFile,
	billingURL string,
) (*cpaGrokBillingSummary, error) {
	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	statusCode, body, err := callCPAManagementAPI(requestCtx, settings, cpaAPICallRequest{
		AuthIndex: authIndex,
		Method:    http.MethodGet,
		URL:       billingURL,
		Header:    grokBillingHeaders(authFile),
	})
	if err != nil {
		return nil, err
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("upstream status: %d", statusCode)
	}
	return parseCPAGrokBilling(body)
}

func callCPAManagementAPI(ctx context.Context, settings cpaSettings, payload cpaAPICallRequest) (int, []byte, error) {
	requestBody, err := common.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(settings.ManagementBaseURL, "/")+"/api-call",
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+settings.ManagementKey)
	req.Header.Set("X-Management-Key", settings.ManagementKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := GetHttpClient().Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, nil, fmt.Errorf("failed to call CPA api-call: status=%d", resp.StatusCode)
	}

	var result cpaAPICallResponse
	if err := common.DecodeJson(resp.Body, &result); err != nil {
		return 0, nil, err
	}
	return result.StatusCode, []byte(result.Body), nil
}

func grokBillingHeaders(authFile *cpaXAIAuthFile) map[string]string {
	headers := map[string]string{
		"Authorization":            "Bearer $TOKEN$",
		"x-xai-token-auth":         cpaGrokTokenAuthValue,
		"x-grok-client-version":    cpaGrokClientVersion,
		"x-grok-client-identifier": "grok-shell",
		"User-Agent":               "xai-grok-workspace/" + cpaGrokClientVersion,
		"Accept":                   "application/json",
	}
	if userID := grokUserID(authFile); userID != "" {
		headers["x-userid"] = userID
	}
	return headers
}

func grokUserID(authFile *cpaXAIAuthFile) string {
	if authFile == nil {
		return ""
	}
	for _, value := range []string{authFile.Sub, authFile.Subject, authFile.UserID} {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func parseCPAGrokBilling(body []byte) (*cpaGrokBillingSummary, error) {
	var payload cpaGrokBillingPayload
	if err := common.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid grok billing payload")
	}
	return buildCPAGrokBillingSummary(payload.Config), nil
}

func buildCPAGrokBillingSummary(config cpaGrokBillingConfig) *cpaGrokBillingSummary {
	period := config.CurrentPeriodCamel
	if period == nil {
		period = config.CurrentPeriod
	}
	periodType := ""
	periodEnd := ""
	if period != nil {
		periodType = strings.ToLower(strings.TrimSpace(period.Type))
		periodEnd = strings.TrimSpace(period.End)
	}
	if periodEnd == "" {
		periodEnd = stringifyCPAValue(firstNonNil(config.BillingPeriodEndC, config.BillingPeriodEnd))
	}

	creditUsage := parseOptionalFloat(firstNonNil(config.CreditUsageCamel, config.CreditUsagePercent))
	monthlyLimit := parseXAICentValue(firstNonNil(config.MonthlyLimitCamel, config.MonthlyLimit))
	usedCents := parseXAICentValue(config.Used)

	summary := &cpaGrokBillingSummary{
		MonthlyLimitCents: monthlyLimit,
		UsedCents:         usedCents,
	}

	isWeekly := creditUsage != nil || strings.Contains(periodType, "weekly")
	if isWeekly && creditUsage != nil {
		summary.WeeklyUsedPercent = creditUsage
		summary.WeeklyResetAt = parseAnyUnixTime(periodEnd)
	}

	if monthlyLimit != nil && *monthlyLimit > 0 && usedCents != nil {
		included := *usedCents
		if included > *monthlyLimit {
			included = *monthlyLimit
		}
		usedPercent := clampPercent((included / *monthlyLimit) * 100)
		summary.MonthlyUsedPercent = &usedPercent
		monthlyEnd := stringifyCPAValue(firstNonNil(config.BillingPeriodEndC, config.BillingPeriodEnd))
		if monthlyEnd == "" {
			monthlyEnd = periodEnd
		}
		summary.MonthlyResetAt = parseAnyUnixTime(monthlyEnd)
	}

	if summary.WeeklyUsedPercent == nil && summary.MonthlyUsedPercent == nil {
		return nil
	}
	return summary
}

func mergeCPAGrokBilling(primary *cpaGrokBillingSummary, fallback *cpaGrokBillingSummary) *cpaGrokBillingSummary {
	if primary == nil {
		return fallback
	}
	if fallback == nil {
		return primary
	}
	merged := *primary
	if merged.WeeklyUsedPercent == nil {
		merged.WeeklyUsedPercent = fallback.WeeklyUsedPercent
		if merged.WeeklyResetAt == 0 {
			merged.WeeklyResetAt = fallback.WeeklyResetAt
		}
	}
	if merged.MonthlyUsedPercent == nil {
		merged.MonthlyUsedPercent = fallback.MonthlyUsedPercent
		merged.MonthlyResetAt = fallback.MonthlyResetAt
	}
	if merged.MonthlyLimitCents == nil {
		merged.MonthlyLimitCents = fallback.MonthlyLimitCents
	}
	if merged.UsedCents == nil {
		merged.UsedCents = fallback.UsedCents
	}
	if merged.WeeklyUsedPercent == nil && merged.MonthlyUsedPercent == nil {
		return nil
	}
	return &merged
}

func percentWindow(usedPercent float64, resetAt int64) *DashboardCPAQuotaWindow {
	used := clampPercent(usedPercent)
	window := &DashboardCPAQuotaWindow{
		UsedPercent:      used,
		RemainingPercent: clampPercent(100 - used),
		ResetAt:          resetAt,
	}
	if resetAt > 0 {
		window.ResetAfterSeconds = maxInt64(0, resetAt-time.Now().Unix())
	}
	return window
}

func grokPlanTypeFromLimit(limitCents *float64) string {
	if limitCents == nil {
		return ""
	}
	switch int64(*limitCents) {
	case cpaGrokSuperGrokHeavyCents:
		return cpaGrokSuperGrokHeavy
	case cpaGrokSuperGrokCents:
		return cpaGrokSuperGrokPlan
	default:
		return ""
	}
}

func isPaidXAIAuthFile(authFile *cpaXAIAuthFile) bool {
	if authFile == nil {
		return false
	}
	usesOfficialAPI := isTruthyCPAValue(authFile.UsingAPI) || isTruthyCPAValue(authFile.UsingAPICamel)
	if usesOfficialAPI && strings.EqualFold(strings.TrimSpace(authFile.Prefix), "paid") {
		return true
	}
	for _, token := range []string{authFile.AccessToken, authFile.IDToken, authFile.Token} {
		if jwtTier(token) >= 1 {
			return true
		}
	}
	return false
}

func jwtTier(token string) float64 {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) < 2 {
		return 0
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		padded := parts[1]
		if rem := len(padded) % 4; rem != 0 {
			padded += strings.Repeat("=", 4-rem)
		}
		payload, err = base64.URLEncoding.DecodeString(padded)
		if err != nil {
			return 0
		}
	}
	var claims map[string]any
	if err := common.Unmarshal(payload, &claims); err != nil {
		return 0
	}
	for key, value := range claims {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized != "tier" && !strings.HasSuffix(normalized, "/tier") && !strings.HasSuffix(normalized, ":tier") {
			continue
		}
		if parsed := parseOptionalFloat(value); parsed != nil {
			return *parsed
		}
	}
	return 0
}

func isTruthyCPAValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case float64:
		return typed == 1
	case int:
		return typed == 1
	case string:
		normalized := strings.ToLower(strings.TrimSpace(typed))
		return normalized == "true" || normalized == "1" || normalized == "yes" || normalized == "on"
	default:
		return false
	}
}

func parseXAICentValue(value any) *float64 {
	if value == nil {
		return nil
	}
	if record, ok := value.(map[string]any); ok {
		return parseOptionalFloat(record["val"])
	}
	return parseOptionalFloat(value)
}

func parseOptionalFloat(value any) *float64 {
	switch typed := value.(type) {
	case nil:
		return nil
	case float64:
		return &typed
	case float32:
		parsed := float64(typed)
		return &parsed
	case int:
		parsed := float64(typed)
		return &parsed
	case int64:
		parsed := float64(typed)
		return &parsed
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil
		}
		parsed, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return nil
		}
		return &parsed
	default:
		return nil
	}
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
