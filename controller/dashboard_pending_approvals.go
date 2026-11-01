package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetDashboardPendingApprovals returns only the two counts needed by the
// dashboard polling strip. RootAuth is applied at the route.
func GetDashboardPendingApprovals(c *gin.Context) {
	var quotaApplications int64
	if err := model.DB.Model(&model.QuotaApplication{}).
		Where("status = ?", model.QuotaApplicationStatusPending).
		Count(&quotaApplications).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
			return
	}

	var settlementApplications int64
	if err := model.DB.Model(&model.AffiliateSettlementApplication{}).
		Where("status = ?", model.AffiliateSettlementPending).
		Count(&settlementApplications).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
			return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"quota_applications":     quotaApplications,
			"settlement_applications": settlementApplications,
		},
	})
}
