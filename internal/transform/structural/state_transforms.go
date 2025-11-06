package structural

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ResourceTypeUpdater updates resource types in state
type ResourceTypeUpdater struct {
	OldType string
	NewType string
}

// UpdateResourceType changes the resource type in state
func (u *ResourceTypeUpdater) UpdateResourceType(stateJSON string) (string, error) {
	resourceType := gjson.Get(stateJSON, "type").String()
	if resourceType == u.OldType {
		return sjson.Set(stateJSON, "type", u.NewType)
	}
	return stateJSON, nil
}
