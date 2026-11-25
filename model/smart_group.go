package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"gorm.io/gorm"
)

const SmartGroupsOption = "SmartGroups"

type SmartGroupRoute struct {
	Values   []string                 `json:"values"`
	Group    string                   `json:"group,omitempty"`
	Presence *SmartGroupPresenceRoute `json:"presence,omitempty"`
}

type SmartGroupPresenceRoute struct {
	Parameter    string `json:"parameter"`
	PresentGroup string `json:"present_group"`
	AbsentGroup  string `json:"absent_group"`
}

type SmartGroup struct {
	Description     string                   `json:"description"`
	Parameter       string                   `json:"parameter"`
	Routes          []SmartGroupRoute        `json:"routes"`
	DefaultGroup    string                   `json:"default_group"`
	DefaultPresence *SmartGroupPresenceRoute `json:"default_presence,omitempty"`
	UserSelectable  *bool                    `json:"user_selectable,omitempty"`
	UserIDs         []int                    `json:"user_ids,omitempty"`
}

type SmartGroupMap map[string]SmartGroup

func GetSmartGroups() SmartGroupMap {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[SmartGroupsOption]
	common.OptionMapRWMutex.RUnlock()
	groups := SmartGroupMap{}
	if raw == "" || common.UnmarshalJsonStr(raw, &groups) != nil || groups == nil {
		return groups
	}
	return groups
}

func IsSmartGroup(name string) bool {
	_, ok := GetSmartGroups()[name]
	return ok
}

func (group SmartGroup) IsUserSelectable() bool {
	return group.UserSelectable == nil || *group.UserSelectable
}

func (group SmartGroup) CanBeSelectedBy(userID int) bool {
	if group.IsUserSelectable() {
		return true
	}
	if userID <= 0 {
		return false
	}
	for _, allowedUserID := range group.UserIDs {
		if allowedUserID == userID {
			return true
		}
	}
	return false
}

func UserCanSelectSmartGroup(userID int, name string) bool {
	group, ok := GetSmartGroups()[name]
	return ok && group.CanBeSelectedBy(userID)
}

func ValidateSmartGroups(raw string) error {
	groups := SmartGroupMap{}
	if err := common.UnmarshalJsonStr(raw, &groups); err != nil {
		return fmt.Errorf("invalid smart group settings: %w", err)
	}
	if groups == nil {
		return errors.New("smart group settings must be a JSON object")
	}
	pricingGroups := ratio_setting.GetGroupRatioCopy()
	for name, group := range groups {
		trimmedName := strings.TrimSpace(name)
		_, pricingGroupExists := pricingGroups[name]
		if trimmedName == "" || trimmedName != name || name == "auto" || name == "default" || pricingGroupExists {
			return fmt.Errorf("invalid or conflicting smart group name %q", name)
		}
		if strings.TrimSpace(group.Parameter) == "" {
			return fmt.Errorf("smart group %q requires a request parameter", name)
		}
		seenUsers := map[int]struct{}{}
		for _, userID := range group.UserIDs {
			if userID <= 0 {
				return fmt.Errorf("smart group %q has an invalid user id", name)
			}
			if _, exists := seenUsers[userID]; exists {
				return fmt.Errorf("smart group %q has a duplicate user id %d", name, userID)
			}
			seenUsers[userID] = struct{}{}
		}
		if group.DefaultPresence == nil {
			if group.DefaultGroup != strings.TrimSpace(group.DefaultGroup) ||
				!hasPricingGroup(pricingGroups, group.DefaultGroup) {
				return fmt.Errorf("smart group %q has an invalid default pricing group", name)
			}
		} else if err := validatePresenceRoute(name, *group.DefaultPresence, pricingGroups); err != nil {
			return err
		}
		seen := map[string]struct{}{}
		for _, route := range group.Routes {
			if len(route.Values) == 0 {
				return fmt.Errorf("smart group %q has an invalid route", name)
			}
			if route.Presence == nil {
				if route.Group != strings.TrimSpace(route.Group) || !hasPricingGroup(pricingGroups, route.Group) {
					return fmt.Errorf("smart group %q has an invalid route target", name)
				}
			} else if err := validatePresenceRoute(name, *route.Presence, pricingGroups); err != nil {
				return err
			}
			for _, value := range route.Values {
				value = strings.TrimSpace(value)
				if value == "" {
					return fmt.Errorf("smart group %q has an empty parameter value", name)
				}
				if _, exists := seen[value]; exists {
					return fmt.Errorf("smart group %q has duplicate parameter value %q", name, value)
				}
				seen[value] = struct{}{}
			}
		}
	}
	return nil
}

func validatePresenceRoute(groupName string, route SmartGroupPresenceRoute, pricingGroups map[string]float64) error {
	if route.Parameter != strings.TrimSpace(route.Parameter) {
		return fmt.Errorf("smart group %q has an invalid second-level parameter: surrounding whitespace is not allowed", groupName)
	}
	if err := ValidateSmartGroupParameterPath(route.Parameter); err != nil {
		return fmt.Errorf("smart group %q has an invalid second-level parameter: %w", groupName, err)
	}
	if !hasPricingGroup(pricingGroups, route.PresentGroup) || !hasPricingGroup(pricingGroups, route.AbsentGroup) {
		return fmt.Errorf("smart group %q has an invalid second-level pricing group", groupName)
	}
	return nil
}

// ValidateSmartGroupParameterPath accepts dotted JSON paths with optional []
// array traversal, such as content[].video_url.
func ValidateSmartGroupParameterPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("parameter path is required")
	}
	for _, part := range strings.Split(path, ".") {
		part = strings.TrimSuffix(part, "[]")
		if part == "" {
			return errors.New("parameter path contains an empty segment")
		}
		for _, char := range part {
			if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' || char == '-') {
				return fmt.Errorf("unsupported character %q", char)
			}
		}
	}
	return nil
}

// SmartGroupJSONPathHasValue reports whether a JSON path resolves to a value
// that is not null, blank, or an empty collection. Zero and false are values.
func SmartGroupJSONPathHasValue(body []byte, path string) (bool, error) {
	if err := ValidateSmartGroupParameterPath(path); err != nil {
		return false, err
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return false, err
	}
	return smartGroupPathHasValue(value, strings.Split(strings.TrimSpace(path), ".")), nil
}

func smartGroupPathHasValue(value any, parts []string) bool {
	if len(parts) == 0 {
		return smartGroupValueIsMeaningful(value)
	}
	part := parts[0]
	traverseArray := strings.HasSuffix(part, "[]")
	key := strings.TrimSuffix(part, "[]")
	object, ok := value.(map[string]any)
	if !ok {
		return false
	}
	child, ok := object[key]
	if !ok {
		return false
	}
	if traverseArray {
		items, ok := child.([]any)
		if !ok {
			return false
		}
		for _, item := range items {
			if smartGroupPathHasValue(item, parts[1:]) {
				return true
			}
		}
		return false
	}
	return smartGroupPathHasValue(child, parts[1:])
}

func smartGroupValueIsMeaningful(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		for _, item := range typed {
			if smartGroupValueIsMeaningful(item) {
				return true
			}
		}
		return false
	case map[string]any:
		for _, item := range typed {
			if smartGroupValueIsMeaningful(item) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func hasPricingGroup(groups map[string]float64, name string) bool {
	_, ok := groups[strings.TrimSpace(name)]
	return ok
}

func ResolveSmartGroup(name, parameterValue string) (string, bool, error) {
	group, ok := GetSmartGroups()[name]
	if !ok {
		return "", false, errors.New("smart group does not exist")
	}
	return ResolveSmartGroupRoute(group, parameterValue, nil)
}

func ResolveSmartGroupRoute(group SmartGroup, parameterValue string, hasValue func(string) (bool, error)) (string, bool, error) {
	parameterValue = strings.TrimSpace(parameterValue)
	for _, route := range group.Routes {
		for _, value := range route.Values {
			if strings.TrimSpace(value) == parameterValue && parameterValue != "" {
				resolved, err := resolveSmartGroupTarget(route.Group, route.Presence, hasValue)
				return resolved, true, err
			}
		}
	}
	resolved, err := resolveSmartGroupTarget(group.DefaultGroup, group.DefaultPresence, hasValue)
	return resolved, false, err
}

func resolveSmartGroupTarget(group string, presence *SmartGroupPresenceRoute, hasValue func(string) (bool, error)) (string, error) {
	if presence == nil {
		return group, nil
	}
	if hasValue == nil {
		return "", errors.New("smart group second-level parameter resolver is unavailable")
	}
	exists, err := hasValue(presence.Parameter)
	if err != nil {
		return "", err
	}
	if exists {
		return presence.PresentGroup, nil
	}
	return presence.AbsentGroup, nil
}

func SaveSmartGroups(raw string) error {
	if err := ValidateSmartGroups(raw); err != nil {
		return err
	}
	newGroups := SmartGroupMap{}
	if err := common.UnmarshalJsonStr(raw, &newGroups); err != nil {
		return err
	}
	oldGroups := GetSmartGroups()
	removed, added := groupNameDiff(oldGroups, newGroups)
	if len(removed) > 0 && len(added) > 0 && (len(removed) != 1 || len(added) != 1) {
		return errors.New("rename one smart group at a time")
	}
	var affectedUsers []int
	if len(removed) == 1 && len(added) == 1 {
		if err := DB.Model(&Token{}).Where(commonGroupCol+" = ?", removed[0]).Distinct("user_id").Pluck("user_id", &affectedUsers).Error; err != nil {
			return err
		}
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if len(removed) == 1 && len(added) == 1 {
			oldName := removed[0]
			newName := added[0]
			if hasPricingGroup(ratio_setting.GetGroupRatioCopy(), newName) {
				return fmt.Errorf("smart group %q already exists", newName)
			}
			if err := tx.Model(&Token{}).Where(commonGroupCol+" = ?", oldName).Update("group", newName).Error; err != nil {
				return err
			}
		} else {
			for _, oldName := range removed {
				var count int64
				if err := tx.Model(&Token{}).Where(commonGroupCol+" = ?", oldName).Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					return fmt.Errorf("smart group %q is still assigned to API keys", oldName)
				}
			}
		}
		if err := syncSmartGroupUserPermissions(tx, newGroups); err != nil {
			return err
		}
		option := Option{Key: SmartGroupsOption}
		if err := tx.FirstOrCreate(&option, Option{Key: SmartGroupsOption}).Error; err != nil {
			return err
		}
		return tx.Model(&Option{}).Where(commonKeyCol+" = ?", SmartGroupsOption).Update("value", raw).Error
	})
	if err == nil {
		loadOptionsFromDatabase()
		var userIDs []int
		if err := DB.Model(&User{}).Distinct("id").Pluck("id", &userIDs).Error; err == nil {
			for _, userID := range userIDs {
				_ = InvalidateUserCache(userID)
			}
		}
		if len(removed) == 1 && len(added) == 1 {
			for _, userID := range affectedUsers {
				_ = InvalidateUserTokensCache(userID)
			}
		}
	}
	return err
}

func ResyncSmartGroupUserPermissions() error {
	err := DB.Transaction(func(tx *gorm.DB) error {
		return syncSmartGroupUserPermissions(tx, GetSmartGroups())
	})
	if err != nil {
		return err
	}
	var userIDs []int
	if err := DB.Model(&User{}).Distinct("id").Pluck("id", &userIDs).Error; err != nil {
		return err
	}
	for _, userID := range userIDs {
		_ = InvalidateUserCache(userID)
	}
	return nil
}

// Smart-group-derived access is stored separately from manually granted
// authorized_groups so removing a smart-group assignment cannot revoke a
// permission granted independently by an administrator.
func syncSmartGroupUserPermissions(tx *gorm.DB, groups SmartGroupMap) error {
	users := smartGroupPricingGroupsByUser(groups, setting.GetUserUsableGroupsCopy(), setting.GetAuthorizationRequiredGroupsCopy())
	var rows []User
	if err := tx.Select("id", "setting").Find(&rows).Error; err != nil {
		return err
	}
	for _, user := range rows {
		want := users[user.Id]
		userSetting := user.GetSetting()
		if equalStringSlices(userSetting.SmartGroupPricingGroups, want) { continue }
		userSetting.SmartGroupPricingGroups = want
		raw, err := common.Marshal(userSetting)
		if err != nil { return err }
		if err = tx.Model(&User{}).Where("id = ?", user.Id).Update("setting", string(raw)).Error; err != nil {
			return err
		}
	}
	return nil
}

func smartGroupPricingGroupsByUser(groups SmartGroupMap, userSelectableGroups map[string]string, authorizationRequired map[string]bool) map[int][]string {
	users := map[int]map[string]struct{}{}
	for _, smartGroup := range groups {
		targets := map[string]struct{}{}
		if smartGroup.DefaultPresence == nil {
			targets[smartGroup.DefaultGroup] = struct{}{}
		} else {
			targets[smartGroup.DefaultPresence.PresentGroup] = struct{}{}
			targets[smartGroup.DefaultPresence.AbsentGroup] = struct{}{}
		}
		for _, route := range smartGroup.Routes {
			if route.Presence == nil {
				targets[route.Group] = struct{}{}
			} else {
				targets[route.Presence.PresentGroup] = struct{}{}
				targets[route.Presence.AbsentGroup] = struct{}{}
			}
		}
		for _, userID := range smartGroup.UserIDs {
			for group := range targets {
				_, userSelectable := userSelectableGroups[group]
				if userSelectable && !authorizationRequired[group] {
					continue
				}
				if users[userID] == nil {
					users[userID] = map[string]struct{}{}
				}
				users[userID][group] = struct{}{}
			}
		}
	}
	result := make(map[int][]string, len(users))
	for userID, groupSet := range users {
		groups := make([]string, 0, len(groupSet))
		for group := range groupSet {
			groups = append(groups, group)
		}
		sort.Strings(groups)
		result[userID] = groups
	}
	return result
}

func groupNameDiff(oldGroups, newGroups SmartGroupMap) ([]string, []string) {
	removed, added := []string{}, []string{}
	for name := range oldGroups {
		if _, ok := newGroups[name]; !ok {
			removed = append(removed, name)
		}
	}
	for name := range newGroups {
		if _, ok := oldGroups[name]; !ok {
			added = append(added, name)
		}
	}
	sort.Strings(removed)
	sort.Strings(added)
	return removed, added
}
