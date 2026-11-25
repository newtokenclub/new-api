package controller

import (
	"net/http"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

type smartGroupSettingsRequest struct {
	Value string `json:"value" binding:"required"`
}

func GetSmartGroupSettings(c *gin.Context) {
	raw, err := common.Marshal(model.GetSmartGroups())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	groupRatios := ratio_setting.GetGroupRatioCopy()
	groupNames := make([]string, 0, len(groupRatios))
	for name := range groupRatios {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"smart_groups":  string(raw),
			"pricing_groups": groupNames,
		},
	})
}

func UpdateSmartGroupSettings(c *gin.Context) {
	var req smartGroupSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid smart group settings")
		return
	}
	if err := model.SaveSmartGroups(req.Value); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
