package structural

import (
	"testing"
	"github.com/tidwall/gjson"
)

func TestResourceTypeUpdater(t *testing.T) {
	updater := ResourceTypeUpdater{
		OldType: "cloudflare_teams_list",
		NewType: "cloudflare_zero_trust_list",
	}
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Update matching type",
			input: `{
				"type": "cloudflare_teams_list",
				"name": "test"
			}`,
			expected: "cloudflare_zero_trust_list",
		},
		{
			name: "Skip non-matching type",
			input: `{
				"type": "cloudflare_other",
				"name": "test"
			}`,
			expected: "cloudflare_other",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := updater.UpdateResourceType(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			resultType := gjson.Get(result, "type").String()
			if resultType != tt.expected {
				t.Errorf("Expected type %s, got %s", tt.expected, resultType)
			}
		})
	}
}