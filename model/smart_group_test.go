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

func TestSmartGroupJSONPathHasValue(t *testing.T) {
	tests := []struct {
		name string
		body string
		path string
		want bool
	}{
		{name: "array object has a video value", body: `{"content":[{"type":"video_url","video_url":{"url":"asset://video"}}]}`, path: "content[].video_url", want: true},
		{name: "null is empty", body: `{"video_url":null}`, path: "video_url", want: false},
		{name: "blank string is empty", body: `{"video_url":" \t "}`, path: "video_url", want: false},
		{name: "empty array is empty", body: `{"video_url":[]}`, path: "video_url", want: false},
		{name: "array with only empty values is empty", body: `{"content":[{"video_url":null},{"video_url":""}]}`, path: "content[].video_url", want: false},
		{name: "zero is a value", body: `{"seed":0}`, path: "seed", want: true},
		{name: "false is a value", body: `{"enabled":false}`, path: "enabled", want: true},
		{name: "missing path", body: `{"content":[]}`, path: "content[].video_url", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SmartGroupJSONPathHasValue([]byte(tt.body), tt.path)
			if err != nil {
				t.Fatalf("SmartGroupJSONPathHasValue() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("SmartGroupJSONPathHasValue() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestValidateSmartGroupParameterPath(t *testing.T) {
	for _, path := range []string{"video_url", "metadata.video_url", "content[].video_url", "content[].metadata.reference_audio"} {
		if err := ValidateSmartGroupParameterPath(path); err != nil {
			t.Errorf("expected path %q to be valid: %v", path, err)
		}
	}
	for _, path := range []string{"", ".video_url", "content..video_url", "content[].", "content[*].video_url", "content[0].video_url"} {
		if err := ValidateSmartGroupParameterPath(path); err == nil {
			t.Errorf("expected path %q to be invalid", path)
		}
	}
}

func TestResolveSmartGroupRouteSupportsOptionalSecondLevel(t *testing.T) {
	group := SmartGroup{
		DefaultGroup: "legacy-default",
		Routes: []SmartGroupRoute{{
			Values: []string{"480p"},
			Presence: &SmartGroupPresenceRoute{
				Parameter: "content[].video_url",
				PresentGroup: "with-video",
				AbsentGroup: "without-video",
			},
		}},
	}
	got, matched, err := ResolveSmartGroupRoute(group, "480p", func(parameter string) (bool, error) {
		if parameter != "content[].video_url" {
			t.Fatalf("unexpected parameter %q", parameter)
		}
		return true, nil
	})
	if err != nil || !matched || got != "with-video" {
		t.Fatalf("ResolveSmartGroupRoute() = (%q, %t, %v), want (with-video, true, nil)", got, matched, err)
	}

	got, matched, err = ResolveSmartGroupRoute(group, "720p", nil)
	if err != nil || matched || got != "legacy-default" {
		t.Fatalf("legacy default = (%q, %t, %v), want (legacy-default, false, nil)", got, matched, err)
	}
}

func boolPointer(value bool) *bool {
	return &value
}
