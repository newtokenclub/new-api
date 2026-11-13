package controller

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type forceFailTaskRequest struct {
	Reason string `json:"reason"`
}

// ForceFailTask lets root administrators terminate an unfinished async task.
// The status transition is guarded so a concurrent poller or duplicate request
// cannot trigger a second refund.
func ForceFailTask(c *gin.Context) {
	taskID := strings.TrimSpace(c.Param("task_id"))
	if taskID == "" {
		common.ApiErrorMsg(c, "task id is required")
		return
	}
	var req forceFailTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		common.ApiErrorMsg(c, "invalid request")
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "Task manually marked as failed by administrator"
	}
	if len(reason) > 1000 {
		reason = reason[:1000]
	}

	task, exists, err := model.GetByOnlyTaskId(taskID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !exists || task == nil {
		common.ApiErrorMsg(c, "task not found")
		return
	}
	if task.Status == model.TaskStatusSuccess || task.Status == model.TaskStatusFailure {
		common.ApiErrorMsg(c, "task is already in a final state")
		return
	}

	quota := task.Quota
	previousStatus := task.Status
	now := time.Now().Unix()
	task.Status = model.TaskStatusFailure
	task.Progress = "100%"
	task.FinishTime = now
	task.FailReason = reason
	won, err := task.UpdateWithStatus(previousStatus)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !won {
		common.ApiErrorMsg(c, "task status changed, please refresh and try again")
		return
	}

	if quota > 0 {
		if err := model.IncreaseUserQuota(task.UserId, quota, false); err != nil {
			common.SysLog(fmt.Sprintf("failed to refund force-failed task %s: %v", taskID, err))
			common.ApiErrorMsg(c, "task marked as failed, but refund failed; please check the task logs")
			return
		}
		model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
			UserId:        task.UserId,
			LogType:       model.LogTypeRefund,
			Content:       reason,
			ChannelId:     task.ChannelId,
			ModelName:     task.ModelName,
			Quota:         quota,
			CancelRequest: true,
			Other: map[string]interface{}{
				"task_id": task.TaskID,
				"reason":  reason,
				"source":  "root_force_fail",
			},
		})
	}

	common.ApiSuccess(c, gin.H{
		"task_id": task.TaskID,
		"status":  task.Status,
		"refunded_quota": quota,
	})
}

func GetAllTask(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	// 解析其他查询参数
	queryParams := model.SyncTaskQueryParams{
		Platform:       constant.TaskPlatform(c.Query("platform")),
		TaskID:         c.Query("task_id"),
		ModelName:      c.Query("model_name"),
		Status:         c.Query("status"),
		Action:         c.Query("action"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		ChannelID:      c.Query("channel_id"),
		UserID:         c.Query("user_id"),
	}
	applyTaskUsernameFilter(c, &queryParams)

	items := model.TaskGetAllTasks(pageInfo.GetStartIdx(), pageInfo.GetPageSize(), queryParams)
	total := model.TaskCountAllTasks(queryParams)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tasksToDto(items, true))
	common.ApiSuccess(c, pageInfo)
}

func GetUserTask(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)

	userId := c.GetInt("id")

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)

	queryParams := model.SyncTaskQueryParams{
		Platform:       constant.TaskPlatform(c.Query("platform")),
		TaskID:         c.Query("task_id"),
		ModelName:      c.Query("model_name"),
		Status:         c.Query("status"),
		Action:         c.Query("action"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
	}

	items := model.TaskGetAllUserTask(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), queryParams)
	total := model.TaskCountAllUserTask(userId, queryParams)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tasksToDto(items, false))
	common.ApiSuccess(c, pageInfo)
}

// applyTaskUsernameFilter 将 username 查询解析为 user id 列表，复用已有的 UserIDs 过滤（走 user_id 索引）。
// 找不到匹配用户时写入一个不存在的 id，保证结果为空而非返回全部。
func applyTaskUsernameFilter(c *gin.Context, queryParams *model.SyncTaskQueryParams) {
	username := c.Query("username")
	if username == "" {
		return
	}
	ids := model.GetUserIdsByUsername(username)
	if len(ids) == 0 {
		queryParams.UserIDs = []int{-1}
		return
	}
	queryParams.UserIDs = ids
}

func GetTaskStat(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	queryParams := model.SyncTaskQueryParams{
		Platform:       constant.TaskPlatform(c.Query("platform")),
		TaskID:         c.Query("task_id"),
		ModelName:      c.Query("model_name"),
		Status:         c.Query("status"),
		Action:         c.Query("action"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		ChannelID:      c.Query("channel_id"),
		UserID:         c.Query("user_id"),
	}
	applyTaskUsernameFilter(c, &queryParams)

	stat, err := model.TaskSumQuota(0, queryParams)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, stat)
}

func GetUserTaskStat(c *gin.Context) {
	userId := c.GetInt("id")
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	queryParams := model.SyncTaskQueryParams{
		Platform:       constant.TaskPlatform(c.Query("platform")),
		TaskID:         c.Query("task_id"),
		ModelName:      c.Query("model_name"),
		Status:         c.Query("status"),
		Action:         c.Query("action"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
	}

	stat, err := model.TaskSumQuota(userId, queryParams)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, stat)
}

func tasksToDto(tasks []*model.Task, fillUser bool) []*dto.TaskDto {
	var userIdMap map[int]*model.UserBase
	var displayNameMap map[int]string
	if fillUser {
		userIdMap = make(map[int]*model.UserBase)
		userIds := types.NewSet[int]()
		for _, task := range tasks {
			userIds.Add(task.UserId)
		}
		for _, userId := range userIds.Items() {
			cacheUser, err := model.GetUserCache(userId)
			if err == nil {
				userIdMap[userId] = cacheUser
			}
		}
		var err error
		displayNameMap, err = model.GetUserDisplayNamesByIds(userIds.Items())
		if err != nil {
			common.SysLog("failed to load task user display names: " + err.Error())
			displayNameMap = map[int]string{}
		}
	}
	result := make([]*dto.TaskDto, len(tasks))
	for i, task := range tasks {
		if fillUser {
			if user, ok := userIdMap[task.UserId]; ok {
				task.Username = user.Username
			}
		}
		result[i] = relay.TaskModel2Dto(task)
		if fillUser {
			result[i].DisplayName = displayNameMap[task.UserId]
		}
	}
	return result
}
