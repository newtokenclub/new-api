package controller

import (
	"errors"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type accountMonitorAddRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

func accountMonitorUser(userId int) (*model.User, error) {
	return model.GetUserById(userId, true)
}

func accountMonitorView(user *model.User) gin.H {
	return gin.H{"id": user.Id, "username": user.Username, "display_name": user.DisplayName, "quota": user.Quota}
}

func AddAccountMonitor(c *gin.Context) {
	var req accountMonitorAddRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid account credentials"})
		return
	}
	currentId := c.GetInt("id")
	secondary := model.User{Username: req.Username, Password: req.Password}
	if err := secondary.ValidateAndFill(); err != nil || secondary.Id == currentId {
		if errors.Is(err, model.ErrDatabase) {
			common.ApiError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Account verification failed"})
		return
	}
	if model.IsTwoFAEnabled(secondary.Id) {
		twoFA, err := model.GetTwoFAByUserId(secondary.Id)
		valid := false
		if err == nil && twoFA != nil && req.Code != "" {
			valid, _ = twoFA.ValidateTOTPAndUpdateUsage(req.Code)
			if !valid {
				valid, _ = twoFA.ValidateBackupCodeAndUpdateUsage(req.Code)
			}
		}
		if !valid {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "A valid two-factor code is required", "data": gin.H{"require_2fa": true}})
			return
		}
	}
	if _, err := model.GetAccountMonitorPairForUser(currentId); err == nil {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "An account is already linked"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		common.ApiError(c, err)
		return
	}
	if _, err := model.GetAccountMonitorPairForUser(secondary.Id); err == nil {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "This account is already linked"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		common.ApiError(c, err)
		return
	}
	_, token, err := model.CreateAccountMonitorPair(currentId, secondary.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"token": token, "account": accountMonitorView(&secondary)}})
}

func GetAccountMonitorBalance(c *gin.Context) {
	userId := c.GetInt("id")
	pair, err := model.GetAccountMonitorPairForUser(userId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": nil})
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	otherId := pair.UserAId
	if otherId == userId {
		otherId = pair.UserBId
	}
	user, err := accountMonitorUser(otherId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": accountMonitorView(user)})
}

func SwitchAccountMonitor(c *gin.Context) {
	userId := c.GetInt("id")
	var req struct{ Token string `json:"token"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid account monitor token"})
		return
	}
	pair, err := model.GetAccountMonitorPairByToken(req.Token)
	if err != nil || (pair.UserAId != userId && pair.UserBId != userId) {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Account monitor authorization is invalid"})
		return
	}
	otherId := pair.UserAId
	if otherId == userId {
		otherId = pair.UserBId
	}
	user, err := accountMonitorUser(otherId)
	if err != nil || user.Status != common.UserStatusEnabled {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Target account is unavailable"})
		return
	}
	session := sessions.Default(c)
	session.Clear()
	session.Set("id", user.Id)
	session.Set("username", user.Username)
	session.Set("role", user.Role)
	session.Set("status", user.Status)
	session.Set("group", user.Group)
	if err := session.Save(); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": accountMonitorView(user)})
}

func RemoveAccountMonitor(c *gin.Context) {
	if err := model.DeleteAccountMonitorPairForUser(c.GetInt("id")); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": nil})
}
