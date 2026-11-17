package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyTaskQueryFiltersRunningIncludesAllNonFinalStatuses(t *testing.T) {
	tx := DB.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()

	tasks := []Task{
		{TaskID: "filter-running-not-start", Status: TaskStatusNotStart},
		{TaskID: "filter-running-submitted", Status: TaskStatusSubmitted},
		{TaskID: "filter-running-queued", Status: TaskStatusQueued},
		{TaskID: "filter-running-progress", Status: TaskStatusInProgress},
		{TaskID: "filter-running-unknown", Status: TaskStatusUnknown},
		{TaskID: "filter-running-success", Status: TaskStatusSuccess},
		{TaskID: "filter-running-failure", Status: TaskStatusFailure},
	}
	require.NoError(t, tx.Create(&tasks).Error)

	var taskIDs []string
	err := applyTaskQueryFilters(
		tx.Model(&Task{}),
		SyncTaskQueryParams{Status: TaskStatusFilterRunning},
		false,
	).Order("task_id").Pluck("task_id", &taskIDs).Error
	require.NoError(t, err)
	require.Equal(t, []string{
		"filter-running-not-start",
		"filter-running-progress",
		"filter-running-queued",
		"filter-running-submitted",
		"filter-running-unknown",
	}, taskIDs)
}

func TestApplyTaskQueryFiltersRunningCombinesWithOtherFilters(t *testing.T) {
	tx := DB.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()

	tasks := []Task{
		{TaskID: "filter-running-user-match", UserId: 902342, Status: TaskStatusQueued},
		{TaskID: "filter-running-user-mismatch", UserId: 902343, Status: TaskStatusQueued},
		{TaskID: "filter-running-final-mismatch", UserId: 902343, Status: TaskStatusSuccess},
	}
	require.NoError(t, tx.Create(&tasks).Error)

	var taskIDs []string
	err := applyTaskQueryFilters(
		tx.Model(&Task{}).Where("user_id = ?", 902342),
		SyncTaskQueryParams{Status: TaskStatusFilterRunning},
		false,
	).Pluck("task_id", &taskIDs).Error
	require.NoError(t, err)
	require.Equal(t, []string{"filter-running-user-match"}, taskIDs)
}

func TestTaskSumQuotaRunningFilterReturnsOnlyNonFinalQuota(t *testing.T) {
	tx := DB.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()

	tasks := []Task{
		{TaskID: "filter-running-stat-success", UserId: 902341, Status: TaskStatusSuccess, Quota: 100},
		{TaskID: "filter-running-stat-failure", UserId: 902341, Status: TaskStatusFailure, Quota: 200},
		{TaskID: "filter-running-stat-queued", UserId: 902341, Status: TaskStatusQueued, Quota: 300},
		{TaskID: "filter-running-stat-progress", UserId: 902341, Status: TaskStatusInProgress, Quota: 400},
	}
	require.NoError(t, tx.Create(&tasks).Error)

	stat, err := taskSumQuotaQuery(applyTaskQueryFilters(
		tx.Model(&Task{}).Where("user_id = ?", 902341),
		SyncTaskQueryParams{Status: TaskStatusFilterRunning},
		false,
	))
	require.NoError(t, err)
	require.Equal(t, int64(0), stat.SuccessQuota)
	require.Equal(t, int64(0), stat.FailureQuota)
	require.Equal(t, int64(700), stat.RunningQuota)
}
