package authz

const (
	ResourceSmartGroup = "smart_group"
	ActionManage       = "manage"
)

var SmartGroupManage = Permission{Resource: ResourceSmartGroup, Action: ActionManage}

func init() {
	RegisterResource(ResourceDefinition{
		Resource: ResourceSmartGroup,
		LabelKey: "Smart Group Management",
		Actions: []ActionDefinition{
			{
				Action:         ActionManage,
				LabelKey:       "Manage smart groups",
				DescriptionKey: "Create and edit smart group routing and assigned users without access to pricing group settings.",
			},
		},
	})
}
