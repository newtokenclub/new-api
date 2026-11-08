package authz

const (
	ResourceTaskLog = "task_log"
	ActionViewAll   = "view_all"
)

var TaskLogViewAll = Permission{Resource: ResourceTaskLog, Action: ActionViewAll}

func init() {
	RegisterResource(ResourceDefinition{
		Resource: ResourceTaskLog,
		LabelKey: "Task Log Support",
		Actions: []ActionDefinition{
			{
				Action:         ActionViewAll,
				LabelKey:       "Technical Support",
				DescriptionKey: "View and manage task logs for all users without access to aggregate cost statistics.",
			},
		},
	})
}
