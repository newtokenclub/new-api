package model

import (
	"encoding/json"
	"testing"
)

func TestRenameGroupOption(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		want     string
		wantErr  bool
	}{
		{
			name:  "simple group maps",
			key:   "TopupGroupRatio",
			value: `{"old":1.5}`,
			want:  `{"new":1.5}`,
		},
		{
			name:  "nested group ratio keys",
			key:   "GroupGroupRatio",
			value: `{"old":{"other":0.8},"other":{"old":0.9}}`,
			want:  `{"new":{"other":0.8},"other":{"new":0.9}}`,
		},
		{
			name:  "special usable rules",
			key:   "group_ratio_setting.group_special_usable_group",
			value: `{"old":{"-:old":"remove","+:old":"description"}}`,
			want:  `{"new":{"-:new":"remove","+:new":"description"}}`,
		},
		{
			name:    "target nested key conflict",
			key:     "GroupGroupRatio",
			value:   `{"old":{"other":0.8},"new":{"other":0.9}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renameGroupOption(tt.key, tt.value, "old", "new")
			if (err != nil) != tt.wantErr {
				t.Fatalf("renameGroupOption() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			var gotJSON, wantJSON any
			if err := json.Unmarshal([]byte(got), &gotJSON); err != nil {
				t.Fatalf("invalid result JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.want), &wantJSON); err != nil {
				t.Fatalf("invalid expected JSON: %v", err)
			}
			gotBytes, _ := json.Marshal(gotJSON)
			wantBytes, _ := json.Marshal(wantJSON)
			if string(gotBytes) != string(wantBytes) {
				t.Fatalf("renameGroupOption() = %s, want %s", gotBytes, wantBytes)
			}
		})
	}
}

func TestRemoveGroupOption(t *testing.T) {
	got, err := removeGroupOption(
		"group_ratio_setting.group_special_usable_group",
		`{"vip":{"-:retired":"remove","+:kept":"keep"},"retired":{"+:vip":"desc"}}`,
		"retired",
	)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]map[string]string
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded["retired"]; ok {
		t.Fatal("deleted group rules should be removed")
	}
	if _, ok := decoded["vip"]["-:retired"]; ok {
		t.Fatal("references to the deleted group should be removed")
	}
	if decoded["vip"]["+:kept"] != "keep" {
		t.Fatal("unrelated rules should remain intact")
	}
}
