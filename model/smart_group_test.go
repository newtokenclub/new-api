package model

import "testing"

func TestSmartGroupCanBeSelectedBy(t *testing.T) {
	tests := []struct {
		name     string
		group    SmartGroup
		userID   int
		want     bool
	}{
		{
			name:   "legacy groups remain selectable by everyone",
			group:  SmartGroup{},
			userID: 42,
			want:   true,
		},
		{
			name:   "public groups are selectable by everyone",
			group:  SmartGroup{UserSelectable: boolPointer(true)},
			userID: 42,
			want:   true,
		},
		{
			name:   "assigned users can select private groups",
			group:  SmartGroup{UserSelectable: boolPointer(false), UserIDs: []int{42}},
			userID: 42,
			want:   true,
		},
		{
			name:   "unassigned users cannot select private groups",
			group:  SmartGroup{UserSelectable: boolPointer(false), UserIDs: []int{42}},
			userID: 43,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.group.CanBeSelectedBy(tt.userID); got != tt.want {
				t.Fatalf("CanBeSelectedBy(%d) = %t, want %t", tt.userID, got, tt.want)
			}
		})
	}
}

func TestSmartGroupPricingGroupsByUser(t *testing.T) {
	groups := SmartGroupMap{
		"private": {
			UserSelectable: boolPointer(false),
			UserIDs:        []int{7},
			DefaultGroup:   "default-target",
			Routes:         []SmartGroupRoute{{Group: "route-target"}},
		},
		"public-protected": {
			UserSelectable: boolPointer(true),
			UserIDs:        []int{7, 8},
			DefaultGroup:   "protected-default",
			Routes:         []SmartGroupRoute{{Group: "protected-route"}},
		},
		"public": {
			UserSelectable: boolPointer(true),
			UserIDs:        []int{7},
			DefaultGroup:   "public-target",
		},
	}
	got := smartGroupPricingGroupsByUser(groups, map[string]string{
		"public-target": "public",
	}, map[string]bool{
		"protected-default": true,
		"protected-route":  true,
	})
	if len(got[7]) != 4 {
		t.Fatalf("user 7 got %v, want four auto-assigned groups", got[7])
	}
	want := []string{"default-target", "protected-default", "protected-route", "route-target"}
	for i, group := range want {
		if got[7][i] != group {
			t.Fatalf("user 7 groups = %v, want %v", got[7], want)
		}
	}
	if len(got[8]) != 2 {
		t.Fatalf("user 8 got %v, want only protected groups", got[8])
	}
}

func boolPointer(value bool) *bool {
	return &value
}
