package relay

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendTaskGroupRatio(t *testing.T) {
	task := &model.Task{PrivateData: model.TaskPrivateData{
		BillingContext: &model.TaskBillingContext{GroupRatio: 1.75},
	}}

	body, err := appendTaskGroupRatio([]byte(`{"id":"task_1"}`), task)
	require.NoError(t, err)
	var response map[string]any
	require.NoError(t, json.Unmarshal(body, &response))
	assert.Equal(t, "task_1", response["id"])
	assert.Equal(t, 1.75, response["group_ratio"])
}

func TestAppendTaskGroupRatioOmitsUnavailableSnapshot(t *testing.T) {
	original := []byte(`{"id":"task_1"}`)
	body, err := appendTaskGroupRatio(original, &model.Task{})
	require.NoError(t, err)
	assert.Equal(t, original, body)
}
