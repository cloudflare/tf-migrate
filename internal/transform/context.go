package transform

import "fmt"

func SetStateTypeRename(ctx *Context, resourceName, originalType, targetType string) {
	stateTypeRenameKey := fmt.Sprintf("%s.%s", originalType, resourceName)
	if ctx.StateTypeRenames == nil {
		ctx.StateTypeRenames = make(map[string]interface{})
	}
	ctx.StateTypeRenames[stateTypeRenameKey] = targetType
}
