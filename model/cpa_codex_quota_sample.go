package model

import (
	"context"

	"gorm.io/gorm/clause"
)

type CPACodexQuotaSample struct {
	ID                       int64    `json:"id" gorm:"primaryKey"`
	AccountKey               string   `json:"account_key" gorm:"type:varchar(64);not null;uniqueIndex:idx_cpa_codex_quota_sample_account_time,priority:1"`
	FiveHourRemainingPercent *float64 `json:"five_hour_remaining_percent,omitempty"`
	FiveHourResetAt          int64    `json:"five_hour_reset_at" gorm:"bigint"`
	WeeklyRemainingPercent   *float64 `json:"weekly_remaining_percent,omitempty"`
	WeeklyResetAt            int64    `json:"weekly_reset_at" gorm:"bigint"`
	BindingRemainingPercent  float64  `json:"binding_remaining_percent" gorm:"not null"`
	BindingResetAt           int64    `json:"binding_reset_at" gorm:"bigint"`
	SampledAt                int64    `json:"sampled_at" gorm:"bigint;not null;uniqueIndex:idx_cpa_codex_quota_sample_account_time,priority:2;index"`
}

func CreateCPACodexQuotaSamples(ctx context.Context, samples []CPACodexQuotaSample) error {
	if len(samples) == 0 {
		return nil
	}
	return DB.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&samples).Error
}

func GetCPACodexQuotaSamples(ctx context.Context, accountKeys []string, sampledAfter int64) ([]CPACodexQuotaSample, error) {
	if len(accountKeys) == 0 {
		return []CPACodexQuotaSample{}, nil
	}
	samples := make([]CPACodexQuotaSample, 0)
	err := DB.WithContext(ctx).
		Where("account_key IN ? AND sampled_at >= ?", accountKeys, sampledAfter).
		Order("account_key ASC").
		Order("sampled_at ASC").
		Find(&samples).Error
	return samples, err
}

func DeleteCPACodexQuotaSamplesBefore(ctx context.Context, sampledBefore int64) (int64, error) {
	result := DB.WithContext(ctx).
		Where("sampled_at < ?", sampledBefore).
		Delete(&CPACodexQuotaSample{})
	return result.RowsAffected, result.Error
}
