package controller

import (
	"context"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetDashboardSiteOverview(c *gin.Context) {
	data, err := service.GetDashboardSiteOverview(c.Query("model_distribution_period"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func GetDashboardUserRankings(c *gin.Context) {
	data, err := service.GetDashboardUserTokenRankings(c.Query("period"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func GetDashboardCPAQuotas(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	common.ApiSuccess(c, service.GetDashboardCPAQuotaData(ctx))
}

func RefreshDashboardCPAQuotaStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	common.ApiSuccess(c, service.GetDashboardCPAQuotaData(ctx))
}
