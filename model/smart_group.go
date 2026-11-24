package model

import (
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
	Values []string `json:"values"`
	Group  string   `json:"group"`
}

type SmartGroup struct {
	Description    string            `json:"description"`
	Parameter      string            `json:"parameter"`
	Routes         []SmartGroupRoute `json:"routes"`
	DefaultGroup   string            `json:"default_group"`
	UserSelectable *bool             `json:"user_selectable,omitempty"`
	UserIDs        []int             `json:"user_ids,omitempty"`
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
		if group.DefaultGroup != strings.TrimSpace(group.DefaultGroup) ||
			!hasPricingGroup(pricingGroups, group.DefaultGroup) {
			return fmt.Errorf("smart group %q has an invalid default pricing group", name)
		}
		seen := map[string]struct{}{}
		for _, route := range group.Routes {
			if route.Group != strings.TrimSpace(route.Group) ||
				!hasPricingGroup(pricingGroups, route.Group) || len(route.Values) == 0 {
				return fmt.Errorf("smart group %q has an invalid route", name)
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

func hasPricingGroup(groups map[string]float64, name string) bool {
	_, ok := groups[strings.TrimSpace(name)]
	return ok
}

func ResolveSmartGroup(name, parameterValue string) (string, bool, error) {
	group, ok := GetSmartGroups()[name]
	if !ok {
		return "", false, errors.New("smart group does not exist")
	}
	parameterValue = strings.TrimSpace(parameterValue)
	for _, route := range group.Routes {
		for _, value := range route.Values {
			if strings.TrimSpace(value) == parameterValue && parameterValue != "" {
				return route.Group, true, nil
			}
		}
	}
	return group.DefaultGroup, false, nil
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
		targets := map[string]struct{}{smartGroup.DefaultGroup: {}}
		for _, route := range smartGroup.Routes {
			targets[route.Group] = struct{}{}
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
