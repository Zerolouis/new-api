package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/gin-gonic/gin"
)

func GetChannelConstraints(c *gin.Context) *dto.ChannelConstraints {
	if c == nil {
		return &dto.ChannelConstraints{}
	}
	if existing, ok := common.GetContextKeyType[*dto.ChannelConstraints](c, constant.ContextKeyChannelConstraints); ok && existing != nil {
		return existing
	}
	constraints := &dto.ChannelConstraints{}
	common.SetContextKey(c, constant.ContextKeyChannelConstraints, constraints)
	return constraints
}

func AppendTaskPluginIdentityFilter(c *gin.Context, pluginKey string) {
	if c == nil {
		return
	}
	GetChannelConstraints(c).AddFilter(dto.ChannelFilter{
		Kind:                   dto.FilterTaskPluginIdentity,
		TaskPluginKey:          pluginKey,
		TaskPluginChannelTypes: pinnedTaskPluginChannelTypes(c, pluginKey),
	})
}

type RetryParam struct {
	Ctx          *gin.Context
	TokenGroup   string
	ModelName    string
	RequestPath  string
	Retry        *int
	resetNextTry bool
}

func (p *RetryParam) GetRetry() int {
	if p.Retry == nil {
		return 0
	}
	return *p.Retry
}

func (p *RetryParam) SetRetry(retry int) {
	p.Retry = &retry
}

func (p *RetryParam) IncreaseRetry() {
	if p.resetNextTry {
		p.resetNextTry = false
		return
	}
	if p.Retry == nil {
		p.Retry = new(int)
	}
	*p.Retry++
}

func (p *RetryParam) ResetRetryNextTry() {
	p.resetNextTry = true
}

func (p *RetryParam) advanceAutoGroup(groupIndex int) {
	common.SetContextKey(p.Ctx, constant.ContextKeyAutoGroupIndex, groupIndex+1)
	common.SetContextKey(p.Ctx, constant.ContextKeyAutoGroupRetryIndex, 0)
	p.SetRetry(0)
}

type ChannelSelection struct {
	Channel *model.Channel
	Group   string
	Lease   *ChannelAdmissionLease
}

type ChannelCapacityError struct {
	RetryAfter         time.Duration
	ConcurrencyRejects int
	RPMRejects         int
}

func (e *ChannelCapacityError) Error() string {
	return "all matching channels are at their configured capacity"
}

func (e *ChannelCapacityError) RetryAfterSeconds() int64 {
	if e == nil || e.RetryAfter <= 0 {
		return 1
	}
	seconds := int64((e.RetryAfter + time.Second - 1) / time.Second)
	if seconds < 1 {
		return 1
	}
	return seconds
}

func (e *ChannelCapacityError) addDecision(decision ChannelAdmissionDecision) {
	if decision.Reason == ChannelAdmissionReasonConcurrency {
		e.ConcurrencyRejects++
	} else if decision.Reason == ChannelAdmissionReasonRPM {
		e.RPMRejects++
	}
	if decision.RetryAfter > 0 && (e.RetryAfter <= 0 || decision.RetryAfter < e.RetryAfter) {
		e.RetryAfter = decision.RetryAfter
	}
}

func (e *ChannelCapacityError) merge(other *ChannelCapacityError) {
	if other == nil {
		return
	}
	e.ConcurrencyRejects += other.ConcurrencyRejects
	e.RPMRejects += other.RPMRejects
	if other.RetryAfter > 0 && (e.RetryAfter <= 0 || other.RetryAfter < e.RetryAfter) {
		e.RetryAfter = other.RetryAfter
	}
}

func SelectChannelWithAdmission(param *RetryParam) (*ChannelSelection, error) {
	if param == nil || param.Ctx == nil {
		return nil, errors.New("channel selection requires a request context")
	}
	requestContext := context.Context(param.Ctx)
	if param.Ctx.Request != nil {
		requestContext = param.Ctx.Request.Context()
	}
	filters := GetChannelConstraints(param.Ctx).Filters
	if param.TokenGroup != "auto" {
		tiers, err := model.GetSatisfiedChannelTiers(param.TokenGroup, param.ModelName, filters)
		if err != nil {
			return nil, err
		}
		return selectAdmittedChannel(requestContext, param.TokenGroup, tiers, param.GetRetry())
	}

	userGroup := common.GetContextKeyString(param.Ctx, constant.ContextKeyUserGroup)
	autoGroups := GetRequestAutoGroups(param.Ctx, userGroup)
	if len(autoGroups) == 0 {
		return nil, errors.New("auto groups is not enabled")
	}
	startGroupIndex := 0
	if lastGroupIndex, exists := common.GetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex); exists {
		if index, ok := lastGroupIndex.(int); ok && index >= 0 {
			startGroupIndex = index
		}
	}
	if startGroupIndex >= len(autoGroups) {
		return nil, nil
	}
	crossGroupRetry := common.GetContextKeyBool(param.Ctx, constant.ContextKeyTokenCrossGroupRetry)
	capacityErr := &ChannelCapacityError{}
	for groupIndex := startGroupIndex; groupIndex < len(autoGroups); groupIndex++ {
		selectGroup := autoGroups[groupIndex]
		priorityRetry := param.GetRetry()
		if groupIndex > startGroupIndex {
			priorityRetry = 0
		}
		tiers, err := model.GetSatisfiedChannelTiers(selectGroup, param.ModelName, filters)
		if err != nil {
			return nil, err
		}
		if len(tiers) == 0 {
			param.advanceAutoGroup(groupIndex)
			continue
		}
		selection, err := selectAdmittedChannel(requestContext, selectGroup, tiers, priorityRetry)
		if err != nil {
			var groupCapacityErr *ChannelCapacityError
			if !errors.As(err, &groupCapacityErr) {
				return nil, err
			}
			capacityErr.merge(groupCapacityErr)
			param.advanceAutoGroup(groupIndex)
			continue
		}
		if selection == nil {
			param.advanceAutoGroup(groupIndex)
			continue
		}
		common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, selectGroup)
		if crossGroupRetry && priorityRetry >= common.RetryTimes {
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, groupIndex+1)
			param.SetRetry(0)
			param.ResetRetryNextTry()
		} else {
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, groupIndex)
		}
		return selection, nil
	}
	if capacityErr.ConcurrencyRejects > 0 || capacityErr.RPMRejects > 0 {
		return nil, capacityErr
	}
	return nil, nil
}

func CacheGetRandomSatisfiedChannel(param *RetryParam) (*model.Channel, string, error) {
	group := ""
	if param != nil {
		group = param.TokenGroup
	}
	selection, err := SelectChannelWithAdmission(param)
	if err != nil {
		return nil, group, err
	}
	if selection == nil || selection.Channel == nil {
		return nil, group, nil
	}
	if err := selection.Lease.Release(); err != nil {
		return nil, selection.Group, err
	}
	return selection.Channel, selection.Group, nil
}

func selectAdmittedChannel(ctx context.Context, group string, tiers []model.ChannelCandidateTier, startTier int) (*ChannelSelection, error) {
	if len(tiers) == 0 {
		return nil, nil
	}
	if startTier < 0 {
		startTier = 0
	}
	if startTier >= len(tiers) {
		startTier = len(tiers) - 1
	}
	capacityErr := &ChannelCapacityError{}
	var lastAcquireErr error
	for tierIndex := startTier; tierIndex < len(tiers); tierIndex++ {
		candidates := append([]model.ChannelCandidate(nil), tiers[tierIndex].Candidates...)
		for len(candidates) > 0 {
			candidate, candidateIndex := model.PickWeightedChannelCandidate(candidates)
			if candidateIndex < 0 {
				break
			}
			candidates = append(candidates[:candidateIndex], candidates[candidateIndex+1:]...)
			if candidate.Channel == nil {
				continue
			}
			lease, decision, err := AcquireChannelAdmission(ctx, candidate.Channel)
			if err != nil {
				lastAcquireErr = fmt.Errorf("acquire channel #%d admission: %w", candidate.Channel.Id, err)
				logger.LogWarn(ctx, lastAcquireErr.Error())
				continue
			}
			if decision.Allowed {
				return &ChannelSelection{Channel: candidate.Channel, Group: group, Lease: lease}, nil
			}
			capacityErr.addDecision(decision)
		}
	}
	if capacityErr.ConcurrencyRejects > 0 || capacityErr.RPMRejects > 0 {
		return nil, capacityErr
	}
	if lastAcquireErr != nil {
		return nil, lastAcquireErr
	}
	return nil, nil
}

func pinnedTaskPluginChannelTypes(c *gin.Context, expected string) []int {
	if c == nil || expected == "" {
		return nil
	}
	if value, exists := c.Get(jsplugin.ContextKeyPinnedEndpoint); exists {
		pinned, ok := value.(jsplugin.PinnedEndpoint)
		if ok && pinned.Generation != nil && len(pinned.Candidates) > 1 {
			expectedFound := false
			channelTypes := make([]int, 0, len(pinned.Candidates))
			seen := make(map[int]struct{}, len(pinned.Candidates))
			for _, candidate := range pinned.Candidates {
				if candidate.Plugin == nil {
					continue
				}
				if candidate.Plugin.Meta.Key == expected {
					expectedFound = true
				}
				for _, channelType := range candidate.Plugin.Meta.ChannelTypes {
					if channelType == 0 || channelType == constant.ChannelTypeTaskPlugin {
						continue
					}
					if _, duplicate := seen[channelType]; duplicate {
						continue
					}
					if plugin, indexed := pinned.Generation.GetByChannelType(channelType); indexed && plugin == candidate.Plugin {
						seen[channelType] = struct{}{}
						channelTypes = append(channelTypes, channelType)
					}
				}
			}
			if expectedFound {
				return channelTypes
			}
		}
	}
	value, exists := c.Get(jsplugin.ContextKeyPinnedPlugin)
	pinned, ok := value.(jsplugin.PinnedPlugin)
	if !exists || !ok || pinned.Generation == nil || pinned.Plugin == nil || pinned.Plugin.Meta.Key != expected {
		return nil
	}
	channelTypes := make([]int, 0, len(pinned.Plugin.Meta.ChannelTypes))
	for _, channelType := range pinned.Plugin.Meta.ChannelTypes {
		if channelType == 0 || channelType == constant.ChannelTypeTaskPlugin {
			continue
		}
		channelTypes = append(channelTypes, channelType)
	}
	if len(channelTypes) == 0 {
		return nil
	}
	return channelTypes
}
