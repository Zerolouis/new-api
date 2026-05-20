package model

import "gorm.io/gorm"

type DashboardSiteTotals struct {
	TotalTokens   int64 `json:"total_tokens"`
	TotalRequests int64 `json:"total_requests"`
}

type DashboardModelDistributionRow struct {
	ModelName    string `json:"model_name"`
	RequestCount int64  `json:"request_count"`
	TokenUsed    int64  `json:"token_used"`
}

type DashboardUserRankingRow struct {
	UserID       int    `json:"user_id"`
	Username     string `json:"username"`
	TotalTokens  int64  `json:"total_tokens"`
	RequestCount int64  `json:"request_count"`
}

type DashboardCacheHitLogRow struct {
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	Other            string `json:"other"`
}

func GetDashboardSiteTotals(startTime int64, endTime int64) (DashboardSiteTotals, error) {
	var totals DashboardSiteTotals
	query := applyDashboardTimeRange(DB.Table("quota_data"), startTime, endTime)
	err := query.Select("COALESCE(SUM(token_used), 0) as total_tokens, COALESCE(SUM(count), 0) as total_requests").Scan(&totals).Error
	return totals, err
}

func GetDashboardModelDistribution(startTime int64, endTime int64, limit int) ([]DashboardModelDistributionRow, error) {
	rows := make([]DashboardModelDistributionRow, 0)
	query := applyDashboardTimeRange(DB.Table("quota_data"), startTime, endTime).
		Select("model_name, COALESCE(SUM(count), 0) as request_count, COALESCE(SUM(token_used), 0) as token_used").
		Where("model_name <> ''").
		Group("model_name").
		Having("SUM(count) > 0").
		Order("request_count DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&rows).Error
	return rows, err
}

func GetDashboardUserTokenRankings(startTime int64, endTime int64, limit int) ([]DashboardUserRankingRow, error) {
	rows := make([]DashboardUserRankingRow, 0)
	query := applyDashboardTimeRange(DB.Table("quota_data"), startTime, endTime).
		Select("user_id, username, COALESCE(SUM(token_used), 0) as total_tokens, COALESCE(SUM(count), 0) as request_count").
		Where("username <> ''").
		Group("user_id, username").
		Having("SUM(token_used) > 0").
		Order("total_tokens DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&rows).Error
	return rows, err
}

func GetDashboardCacheHitLogRows(startTime int64, endTime int64) ([]DashboardCacheHitLogRow, error) {
	rows := make([]DashboardCacheHitLogRow, 0)
	query := applyDashboardTimeRange(LOG_DB.Model(&Log{}), startTime, endTime).
		Select("prompt_tokens, completion_tokens, other").
		Where("type = ?", LogTypeConsume)
	err := query.Find(&rows).Error
	return rows, err
}

func applyDashboardTimeRange(query *gorm.DB, startTime int64, endTime int64) *gorm.DB {
	if startTime > 0 {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime > 0 {
		query = query.Where("created_at <= ?", endTime)
	}
	return query
}
