package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type channelPauseRequest struct {
	Enabled bool   `json:"enabled"`
	Message string `json:"message"`
}

func GetChannelPause(c *gin.Context) {
	enabled, message := model.GetChannelPauseState()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"enabled": enabled, "message": message}})
}

func UpdateChannelPause(c *gin.Context) {
	if c.GetInt("role") != common.RoleRootUser {
		common.ApiErrorMsg(c, "root permission required")
		return
	}
	var req channelPauseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid channel pause settings")
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Enabled && req.Message == "" {
		common.ApiErrorMsg(c, "maintenance message is required")
		return
	}
	if err := model.UpdateChannelPause(req.Enabled, req.Message); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"enabled": req.Enabled, "message": req.Message}})
}
