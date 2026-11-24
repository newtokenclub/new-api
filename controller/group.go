package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

type RenameGroupRequest struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

type DeleteGroupRequest struct {
	Name string `json:"name"`
}

func RenameGroup(c *gin.Context) {
	if c.GetInt("role") != common.RoleRootUser {
		common.ApiErrorMsg(c, "root permission required")
		return
	}
	var req RenameGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.OldName) == "" || strings.TrimSpace(req.NewName) == "" {
		common.ApiErrorMsg(c, "invalid group rename")
		return
	}
	if err := model.RenameGroup(req.OldName, req.NewName); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func DeleteGroup(c *gin.Context) {
	if c.GetInt("role") != common.RoleRootUser {
		common.ApiErrorMsg(c, "root permission required")
		return
	}
	var req DeleteGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		common.ApiErrorMsg(c, "invalid group name")
		return
	}
	if err := model.DeleteGroupSettings(req.Name); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func GetGroups(c *gin.Context) {
	groupNames := make([]string, 0)
	for groupName := range ratio_setting.GetGroupRatioCopy() {
		groupNames = append(groupNames, groupName)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    groupNames,
	})
}

func GetUserGroups(c *gin.Context) {
	usableGroups := make(map[string]map[string]interface{})
	userGroup := ""
	userId := c.GetInt("id")
	userGroup, _ = model.GetUserGroup(userId, false)
	var restricted []string
	var additionalAuthorizedGroups []string
	if user, err := model.GetUserCache(userId); err == nil {
		userSettings := user.GetSetting()
		restricted = userSettings.RestrictedPublicGroups
		additionalAuthorizedGroups = userSettings.AuthorizedGroups
	}
	var smartGroupPricingGroups []string
	if user, err := model.GetUserCache(userId); err == nil {
		smartGroupPricingGroups = user.GetSetting().SmartGroupPricingGroups
	}
	userUsableGroups := service.GetUserUsableGroupsWithSmartGroups(userGroup, restricted, smartGroupPricingGroups)
	authorized := service.GetUserAuthorizedGroups(userGroup, append(additionalAuthorizedGroups, smartGroupPricingGroups...))
	authorizationRequiredGroups := setting.GetAuthorizationRequiredGroupsCopy()
	for groupName, _ := range ratio_setting.GetGroupRatioCopy() {
		if desc, ok := userUsableGroups[groupName]; ok {
			_, hasAuthorization := authorized[groupName]
			usableGroups[groupName] = map[string]interface{}{
				"ratio":                  service.GetUserGroupRatio(userGroup, groupName),
				"desc":                   desc,
				"authorization_required": authorizationRequiredGroups[groupName],
				"authorized":             !authorizationRequiredGroups[groupName] || hasAuthorization,
			}
		}
	}
	if c.Query("include_smart") == "true" {
		for groupName, group := range model.GetSmartGroups() {
			if !group.CanBeSelectedBy(userId) {
				continue
			}
			usableGroups[groupName] = map[string]interface{}{
				"type":          "smart",
				"desc":          group.Description,
				"parameter":     group.Parameter,
				"routes":        group.Routes,
				"default_group": group.DefaultGroup,
				"user_selectable": group.IsUserSelectable(),
			}
		}
	}
	if _, ok := userUsableGroups["auto"]; ok {
		usableGroups["auto"] = map[string]interface{}{
			"ratio": "自动",
			"desc":  setting.GetUsableGroupDescription("auto"),
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    usableGroups,
	})
}

func SearchSmartGroupUsers(c *gin.Context) {
	if c.GetInt("role") != common.RoleRootUser {
		common.ApiErrorMsg(c, "root permission required")
		return
	}
	type userOption struct {
		Id          int    `json:"id"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}
	query := model.DB.Model(&model.User{}).Select("id, username, display_name").Where("deleted_at IS NULL")
	if idsValue := strings.TrimSpace(c.Query("ids")); idsValue != "" {
		ids := make([]int, 0)
		for _, rawID := range strings.Split(idsValue, ",") {
			id, err := strconv.Atoi(strings.TrimSpace(rawID))
			if err != nil || id <= 0 {
				common.ApiErrorMsg(c, "invalid user ids")
				return
			}
			ids = append(ids, id)
		}
		if len(ids) > 200 {
			common.ApiErrorMsg(c, "too many user ids")
			return
		}
		query = query.Where("id IN ?", ids)
	} else {
		keyword := strings.TrimSpace(c.Query("keyword"))
		if keyword == "" {
			c.JSON(http.StatusOK, gin.H{"success": true, "data": []userOption{}})
			return
		}
		like := "%" + keyword + "%"
		if id, err := strconv.Atoi(keyword); err == nil {
			query = query.Where("id = ? OR username LIKE ? OR display_name LIKE ?", id, like, like)
		} else {
			query = query.Where("username LIKE ? OR display_name LIKE ?", like, like)
		}
	}
	var users []userOption
	if err := query.Order("id DESC").Limit(200).Find(&users).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": users})
}
