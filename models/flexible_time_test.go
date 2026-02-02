package models

import (
	"encoding/json"
	"testing"
)

func TestFlexibleTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"format ISO", `"2025-12-31T20:00:00"`, false},
		{"format court", `"2025-12-31T20:00"`, false},
		{"null", `null`, false},
		{"vide", `""`, false},
		{"invalide", `"invalid"`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ft FlexibleTime
			err := json.Unmarshal([]byte(tt.input), &ft)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() erreur = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFlexibleTime_MarshalJSON(t *testing.T) {
	var ft FlexibleTime
	_ = json.Unmarshal([]byte(`"2025-12-31T20:00:00"`), &ft)
	data, err := json.Marshal(ft)
	if err != nil {
		t.Fatalf("MarshalJSON() erreur = %v", err)
	}
	if len(data) == 0 {
		t.Error("MarshalJSON() ne doit pas retourner vide")
	}
}
