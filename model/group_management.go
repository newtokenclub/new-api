package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var groupOptionKeys = []string{
	"GroupRatio", "TopupGroupRatio", "UserUsableGroups", "AuthorizationRequiredGroups",
	"GroupGroupRatio", "AutoGroups", "group_ratio_setting.group_special_usable_group",
	"ModelRequestRateLimitGroup", SmartGroupsOption,
}

func RenameGroup(oldName, newName string) error {
	oldName, newName = strings.TrimSpace(oldName), strings.TrimSpace(newName)
	if oldName == "" || newName == "" || oldName == newName || oldName == "default" || oldName == "auto" || newName == "auto" {
		return errors.New("invalid group rename")
	}
	var affectedUserIDs, affectedPlanIDs, affectedTokenUserIDs []int
	err := DB.Transaction(func(tx *gorm.DB) error {
		options, optionMap, err := getGroupOptions(tx)
		if err != nil {
			return err
		}
		groups := map[string]json.RawMessage{}
		if err = json.Unmarshal([]byte(optionMap["GroupRatio"]), &groups); err != nil {
			return fmt.Errorf("invalid group ratio settings: %w", err)
		}
		if _, ok := groups[oldName]; !ok {
			return errors.New("source group does not exist")
		}
		if _, ok := groups[newName]; ok {
			return errors.New("target group already exists")
		}
		if err = ensureGroupUnused(tx, newName); err != nil {
			return err
		}
		if err = ensureNoAbilityGroup(tx, newName); err != nil {
			return err
		}
		if err = ensureGroupNameNotHistorical(tx, newName); err != nil {
			return err
		}
		if err = ensureGroupOptionNameAvailable(optionMap, newName); err != nil {
			return err
		}
		if err = tx.Model(&SubscriptionPlan{}).Where("upgrade_group = ? OR downgrade_group = ?", oldName, oldName).Pluck("id", &affectedPlanIDs).Error; err != nil {
			return err
		}
		for _, option := range options {
			updated, updateErr := renameGroupOption(option.Key, option.Value, oldName, newName)
			if updateErr != nil {
				return fmt.Errorf("cannot rename group in %s: %w", option.Key, updateErr)
			}
			if updated != option.Value {
				if updateErr = tx.Model(&Option{}).Where(commonKeyCol+" = ?", option.Key).Update("value", updated).Error; updateErr != nil {
					return updateErr
				}
			}
		}

		var userRows []struct {
			Id      int
			Group   string `gorm:"column:group_value"`
			Setting string
		}
		if err = tx.Table("users").Select("id, "+commonGroupCol+" AS group_value, setting").Scan(&userRows).Error; err != nil {
			return err
		}
		for _, row := range userRows {
			user := User{Id: row.Id, Group: row.Group, Setting: row.Setting}
			group, groupChanged := renameGroupList(user.Group, oldName, newName)
			userSetting := user.GetSetting()
			originalRestricted := append([]string(nil), userSetting.RestrictedPublicGroups...)
			originalAuthorized := append([]string(nil), userSetting.AuthorizedGroups...)
			userSetting.RestrictedPublicGroups, _ = renameGroupSlice(userSetting.RestrictedPublicGroups, oldName, newName)
			userSetting.AuthorizedGroups, _ = renameGroupSlice(userSetting.AuthorizedGroups, oldName, newName)
			settingChanged := !equalStringSlices(originalRestricted, userSetting.RestrictedPublicGroups) ||
				!equalStringSlices(originalAuthorized, userSetting.AuthorizedGroups)
			if groupChanged || settingChanged {
				updates := map[string]interface{}{}
				if groupChanged {
					updates["group"] = group
				}
				if settingChanged {
					raw, marshalErr := json.Marshal(userSetting)
					if marshalErr != nil {
						return marshalErr
					}
					updates["setting"] = string(raw)
				}
				if err = tx.Model(&User{}).Where("id = ?", user.Id).Updates(updates).Error; err != nil {
					return err
				}
				affectedUserIDs = append(affectedUserIDs, user.Id)
			}
		}

		var tokens []Token
		if err = tx.Select("user_id").Where(commonGroupCol+" = ?", oldName).Find(&tokens).Error; err != nil {
			return err
		}
		for _, token := range tokens {
			affectedTokenUserIDs = append(affectedTokenUserIDs, token.UserId)
		}
		if err = tx.Model(&Token{}).Where(commonGroupCol+" = ?", oldName).Update("group", newName).Error; err != nil {
			return err
		}

		var channelRows []struct {
			Id       int
			Group    string  `gorm:"column:group_value"`
			Models   string
			Status   int
			Priority *int64
			Weight   *uint
			Tag      *string
		}
		if err = tx.Table("channels").Select("id, "+commonGroupCol+" AS group_value, models, status, priority, weight, tag").Scan(&channelRows).Error; err != nil {
			return err
		}
		for _, row := range channelRows {
			group, changed := renameGroupList(row.Group, oldName, newName)
			if !changed {
				continue
			}
			if err = tx.Model(&Channel{}).Where("id = ?", row.Id).Update("group", group).Error; err != nil {
				return err
			}
			channel := Channel{Id: row.Id, Group: group, Models: row.Models, Status: row.Status, Priority: row.Priority, Weight: row.Weight, Tag: row.Tag}
			if err = channel.UpdateAbilities(tx); err != nil {
				return err
			}
		}

		for _, table := range []struct{ name, column string }{
			{"subscription_plans", "upgrade_group"}, {"subscription_plans", "downgrade_group"},
			{"user_subscriptions", "upgrade_group"}, {"user_subscriptions", "downgrade_group"}, {"user_subscriptions", "prev_user_group"},
			{"affiliate_commission_rules", "group_name"}, {"affiliate_commission_plan_rates", "group_name"},
			{"quota_data", "use_group"},
		} {
			if err = tx.Table(table.name).Where(table.column+" = ?", oldName).Update(table.column, newName).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, userID := range affectedUserIDs {
		_ = InvalidateUserCache(userID)
	}
	for _, userID := range affectedTokenUserIDs {
		_ = InvalidateUserTokensCache(userID)
	}
	for _, planID := range affectedPlanIDs {
		InvalidateSubscriptionPlanCache(planID)
	}
	InitChannelCache()
	loadOptionsFromDatabase()
	return nil
}

func DeleteGroupSettings(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || name == "default" || name == "auto" {
		return errors.New("this group cannot be deleted")
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureGroupUnused(tx, name); err != nil {
			return err
		}
		if err := tx.Where(commonGroupCol+" = ?", name).Delete(&Ability{}).Error; err != nil {
			return err
		}
		options, optionMap, err := getGroupOptions(tx)
		if err != nil {
			return err
		}
		var groups map[string]json.RawMessage
		if err = json.Unmarshal([]byte(optionMap["GroupRatio"]), &groups); err != nil {
			return fmt.Errorf("invalid group ratio settings: %w", err)
		}
		if _, exists := groups[name]; !exists {
			return errors.New("group does not exist")
		}
		for _, option := range options {
			updated, updateErr := removeGroupOption(option.Key, option.Value, name)
			if updateErr != nil {
				return fmt.Errorf("cannot delete group from %s: %w", option.Key, updateErr)
			}
			if updated != option.Value {
				if updateErr = tx.Model(&Option{}).Where(commonKeyCol+" = ?", option.Key).Update("value", updated).Error; updateErr != nil {
					return updateErr
				}
			}
		}
		return nil
	})
	if err == nil {
		loadOptionsFromDatabase()
		InitChannelCache()
	}
	return err
}

func getGroupOptions(tx *gorm.DB) ([]Option, map[string]string, error) {
	var options []Option
	if err := tx.Where(commonKeyCol+" IN ?", groupOptionKeys).Find(&options).Error; err != nil {
		return nil, nil, err
	}
	optionMap := make(map[string]string, len(options))
	for _, option := range options {
		optionMap[option.Key] = option.Value
	}
	return options, optionMap, nil
}

func ensureGroupUnused(db *gorm.DB, name string) error {
	for smartName, smartGroup := range GetSmartGroups() {
		if smartGroup.DefaultGroup == name {
			return fmt.Errorf("group %q is the default target of smart group %q", name, smartName)
		}
		for _, route := range smartGroup.Routes {
			if route.Group == name {
				return fmt.Errorf("group %q is a target of smart group %q", name, smartName)
			}
		}
	}
	checks := []struct{ table, column string }{
		{"tokens", commonGroupCol},
		{"subscription_plans", "upgrade_group"}, {"subscription_plans", "downgrade_group"},
		{"user_subscriptions", "upgrade_group"}, {"user_subscriptions", "downgrade_group"}, {"user_subscriptions", "prev_user_group"},
		{"affiliate_commission_rules", "group_name"}, {"affiliate_commission_plan_rates", "group_name"},
	}
	for _, check := range checks {
		var count int64
		if err := db.Table(check.table).Where(check.column+" = ?", name).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("group %q is still used by %s", name, check.table)
		}
	}
	for _, table := range []struct{ name, column string }{{"users", commonGroupCol}, {"channels", commonGroupCol}} {
		var rows []struct {
			Group string `gorm:"column:group_value"`
		}
		if err := db.Table(table.name).Select(table.column + " AS group_value").Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			for _, part := range strings.Split(row.Group, ",") {
				if strings.TrimSpace(part) == name {
					return fmt.Errorf("group %q is still used by %s", name, table.name)
				}
			}
		}
	}
	var users []User
	if err := db.Select("id", "setting").Find(&users).Error; err != nil {
		return err
	}
	for _, user := range users {
		userSetting := user.GetSetting()
		for _, group := range append(userSetting.RestrictedPublicGroups, userSetting.AuthorizedGroups...) {
			if group == name {
				return errors.New("group is still used in user access settings")
			}
		}
	}
	return nil
}

func ensureNoAbilityGroup(db *gorm.DB, name string) error {
	var count int64
	if err := db.Model(&Ability{}).Where(commonGroupCol+" = ?", name).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("group %q is still used by abilities", name)
	}
	return nil
}

func ensureGroupNameNotHistorical(db *gorm.DB, name string) error {
	for _, table := range []struct{ name, column string }{
		{"quota_data", "use_group"}, {"tasks", commonGroupCol}, {"perf_metrics", commonGroupCol},
		{"affiliate_settlement_items", "group_name"},
	} {
		var count int64
		if err := db.Table(table.name).Where(table.column+" = ?", name).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("target group name %q exists in historical records and cannot be reused", name)
		}
	}
	logDB := LOG_DB
	if logDB == nil {
		logDB = db
	}
	var logCount int64
	if err := logDB.Table("logs").Where("logs."+logGroupCol+" = ?", name).Count(&logCount).Error; err != nil {
		return err
	}
	if logCount > 0 {
		return fmt.Errorf("target group name %q exists in historical logs and cannot be reused", name)
	}
	return nil
}

func ensureGroupOptionNameAvailable(options map[string]string, newName string) error {
	for _, key := range groupOptionKeys {
		value, ok := options[key]
		if !ok {
			continue
		}
		if key == "AutoGroups" {
			groups, err := parseAutoGroups(value)
			if err != nil {
				return fmt.Errorf("invalid %s settings: %w", key, err)
			}
			for _, group := range groups {
				if group == newName {
					return fmt.Errorf("target group %q already has settings in %s", newName, key)
				}
			}
			continue
		}
		if key == "GroupGroupRatio" || key == "group_ratio_setting.group_special_usable_group" {
			var nested map[string]map[string]json.RawMessage
			if err := json.Unmarshal([]byte(value), &nested); err != nil {
				return fmt.Errorf("invalid %s settings: %w", key, err)
			}
			if _, exists := nested[newName]; exists {
				return fmt.Errorf("target group %q already has settings in %s", newName, key)
			}
			for _, row := range nested {
				if key == "GroupGroupRatio" {
					if _, exists := row[newName]; exists {
						return fmt.Errorf("target group %q already has settings in %s", newName, key)
					}
					continue
				}
				for action := range row {
					target := action
					if strings.HasPrefix(action, "+:") || strings.HasPrefix(action, "-:") {
						target = action[2:]
					}
					if target == newName {
						return fmt.Errorf("target group %q already has settings in %s", newName, key)
					}
				}
			}
			continue
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(value), &object); err != nil {
			return fmt.Errorf("invalid %s settings: %w", key, err)
		}
		if _, exists := object[newName]; exists {
			return fmt.Errorf("target group %q already has settings in %s", newName, key)
		}
	}
	return nil
}

func renameGroupList(value, oldName, newName string) (string, bool) {
	parts := strings.Split(value, ",")
	changed := false
	seen := make(map[string]bool, len(parts))
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == oldName {
			trimmed = newName
			changed = true
		}
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			result = append(result, trimmed)
		} else if trimmed == "" {
			result = append(result, part)
		}
	}
	return strings.Join(result, ","), changed
}

func renameGroupSlice(values []string, oldName, newName string) ([]string, bool) {
	changed := false
	for i, value := range values {
		if value == oldName {
			values[i] = newName
			changed = true
		}
	}
	return values, changed
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func renameGroupOption(key, value, oldName, newName string) (string, error) {
	if key == SmartGroupsOption {
		var groups SmartGroupMap
		if err := common.UnmarshalJsonStr(value, &groups); err != nil {
			return value, err
		}
		for name, group := range groups {
			if group.DefaultGroup == oldName {
				group.DefaultGroup = newName
			}
			for index := range group.Routes {
				if group.Routes[index].Group == oldName {
					group.Routes[index].Group = newName
				}
			}
			groups[name] = group
		}
		raw, err := common.Marshal(groups)
		return string(raw), err
	}
	if key == "AutoGroups" {
		groups, err := parseAutoGroups(value)
		if err != nil {
			return value, err
		}
		seen := map[string]bool{}
		result := make([]string, 0, len(groups))
		for _, group := range groups {
			if group == oldName {
				group = newName
			}
			if !seen[group] {
				seen[group] = true
				result = append(result, group)
			}
		}
		raw, err := json.Marshal(result)
		return string(raw), err
	}
	if key == "GroupGroupRatio" || key == "group_ratio_setting.group_special_usable_group" {
		var nested map[string]map[string]json.RawMessage
		if err := json.Unmarshal([]byte(value), &nested); err != nil {
			return value, err
		}
		if row, ok := nested[oldName]; ok {
			if _, exists := nested[newName]; exists {
				return value, errors.New("target key already exists")
			}
			delete(nested, oldName)
			nested[newName] = row
		}
		for group, row := range nested {
			if key == "group_ratio_setting.group_special_usable_group" {
				for action, raw := range row {
					op := ""
					groupName := action
					if strings.HasPrefix(action, "+:") || strings.HasPrefix(action, "-:") {
						op, groupName = action[:2], action[2:]
					}
					if groupName == oldName {
						newAction := op + newName
						if _, exists := row[newAction]; exists && newAction != action {
							return value, errors.New("target key already exists")
						}
						delete(row, action)
						row[newAction] = raw
					}
				}
				nested[group] = row
				continue
			}
			if raw, ok := row[oldName]; ok {
				if _, exists := row[newName]; exists {
					return value, errors.New("target key already exists")
				}
				delete(row, oldName)
				row[newName] = raw
			}
			nested[group] = row
		}
		raw, err := json.Marshal(nested)
		return string(raw), err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return value, err
	}
	if raw, ok := object[oldName]; ok {
		if _, exists := object[newName]; exists {
			return value, errors.New("target key already exists")
		}
		delete(object, oldName)
		object[newName] = raw
	}
	raw, err := json.Marshal(object)
	return string(raw), err
}

func removeGroupOption(key, value, name string) (string, error) {
	if key == SmartGroupsOption {
		return value, nil
	}
	if key == "AutoGroups" {
		groups, err := parseAutoGroups(value)
		if err != nil {
			return value, err
		}
		filtered := make([]string, 0, len(groups))
		for _, group := range groups {
			if group != name {
				filtered = append(filtered, group)
			}
		}
		raw, err := json.Marshal(filtered)
		return string(raw), err
	}
	if key == "GroupGroupRatio" || key == "group_ratio_setting.group_special_usable_group" {
		var nested map[string]map[string]json.RawMessage
		if err := json.Unmarshal([]byte(value), &nested); err != nil {
			return value, err
		}
		delete(nested, name)
		for group, row := range nested {
			if key == "group_ratio_setting.group_special_usable_group" {
				for action := range row {
					target := action
					if strings.HasPrefix(action, "+:") || strings.HasPrefix(action, "-:") {
						target = action[2:]
					}
					if target == name {
						delete(row, action)
					}
				}
				nested[group] = row
				continue
			}
			delete(row, name)
			nested[group] = row
		}
		raw, err := json.Marshal(nested)
		return string(raw), err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return value, err
	}
	delete(object, name)
	raw, err := json.Marshal(object)
	return string(raw), err
}

func parseAutoGroups(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return []string{}, nil
	}
	var groups []string
	if err := json.Unmarshal([]byte(value), &groups); err != nil {
		return nil, err
	}
	if groups == nil {
		groups = []string{}
	}
	return groups, nil
}
