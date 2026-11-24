package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"gorm.io/gorm"
)

const SmartGroupsOption = "SmartGroups"

type SmartGroupRoute struct {
	Values []string `json:"values"`
	Group  string   `json:"group"`
}

type SmartGroup struct {
	Description  string            `json:"description"`
	Parameter    string            `json:"parameter"`
	Routes       []SmartGroupRoute `json:"routes"`
	DefaultGroup string            `json:"default_group"`
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
		option := Option{Key: SmartGroupsOption}
		if err := tx.FirstOrCreate(&option, Option{Key: SmartGroupsOption}).Error; err != nil {
			return err
		}
		return tx.Model(&Option{}).Where(commonKeyCol+" = ?", SmartGroupsOption).Update("value", raw).Error
	})
	if err == nil {
		loadOptionsFromDatabase()
		if len(removed) == 1 && len(added) == 1 {
			for _, userID := range affectedUsers {
				_ = InvalidateUserTokensCache(userID)
			}
		}
	}
	return err
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
