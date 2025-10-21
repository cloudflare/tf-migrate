package processing

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	
	"github.com/cloudflare/tf-migrate/internal/core"
)

// ProcessState transforms Terraform state files
func ProcessState(content []byte, filename string, registry *core.Registry) ([]byte, error) {
	// Step 1: Validate JSON
	stateJSON := string(content)
	if !gjson.Valid(stateJSON) {
		return nil, core.NewError(core.ParseError).
			WithFile(filename).
			WithOperation("parsing state JSON").
			WithCause(fmt.Errorf("invalid JSON syntax")).
			Build()
	}
	
	// Step 2: Transform resources
	transformed, err := transformState(stateJSON, registry, filename)
	if err != nil {
		// Error already wrapped by transformState
		return nil, err
	}
	
	// Step 3: Format JSON
	formatted, err := formatJSON([]byte(transformed))
	if err != nil {
		return nil, core.NewError(core.StateError).
			WithFile(filename).
			WithOperation("formatting state JSON").
			WithCause(err).
			Build()
	}
	return formatted, nil
}

// transformState applies transformations to resources in the state
func transformState(stateJSON string, registry *core.Registry, filename string) (string, error) {
	result := gjson.Parse(stateJSON)
	resources := result.Get("resources")
	
	if !resources.Exists() {
		return stateJSON, nil // No resources to transform
	}
	
	modifiedState := stateJSON
	errorList := core.NewErrorList(20) // Allow more errors for state files
	
	resources.ForEach(func(key, resource gjson.Result) bool {
		resourceType := resource.Get("type").String()
		resourceName := resource.Get("name").String()
		
		if resourceType == "" {
			return true // Skip resources without type
		}
		
		transformer := registry.Find(resourceType)
		if transformer == nil {
			return true // No transformer for this type - this is ok
		}
		
		// Check if the transformer changes the resource type
		if targetType := transformer.GetTargetResourceType(); targetType != "" {
			// Update the resource type in the state
			typePath := fmt.Sprintf("resources.%d.type", key.Int())
			newState, err := sjson.Set(modifiedState, typePath, targetType)
			if err != nil {
				errorList.Add(core.NewError(core.StateError).
					WithResource(resourceType).
					WithFile(filename).
					WithOperation("updating resource type").
					WithContext("resource_name", resourceName).
					WithContext("target_type", targetType).
					WithCause(err).
					Build())
			} else {
				modifiedState = newState
			}
		}
		
		instances := resource.Get("instances")
		if !instances.Exists() {
			return true // No instances to transform
		}
		
		instances.ForEach(func(instKey, instance gjson.Result) bool {
			resourcePath := fmt.Sprintf("resources.%d.instances.%d", key.Int(), instKey.Int())
			
			transformedJSON, err := transformer.State(instance, resourcePath)
			if err != nil {
				errorList.Add(core.NewError(core.TransformError).
					WithResource(resourceType).
					WithFile(filename).
					WithOperation("transforming state instance").
					WithContext("resource_name", resourceName).
					WithContext("instance_index", instKey.Int()).
					WithCause(err).
					Build())
				return true // Continue with other instances
			}
			
			if transformedJSON != "" && transformedJSON != instance.String() {
				newState, err := sjson.SetRaw(modifiedState, resourcePath, transformedJSON)
				if err != nil {
					errorList.Add(core.NewError(core.StateError).
						WithResource(resourceType).
						WithFile(filename).
						WithOperation("updating state instance").
						WithContext("resource_name", resourceName).
						WithContext("instance_index", instKey.Int()).
						WithCause(err).
						Build())
				} else {
					modifiedState = newState
				}
			}
			
			return true
		})
		
		return true // Continue with other resources
	})
	
	// Return any errors that occurred
	if errorList.HasErrors() {
		return "", errorList
	}
	
	return modifiedState, nil
}

// formatJSON pretty-prints JSON
func formatJSON(content []byte) ([]byte, error) {
	var formatted interface{}
	if err := json.Unmarshal(content, &formatted); err != nil {
		return content, nil // Return as-is if we can't format
	}
	
	prettyJSON, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		return content, nil // Return as-is if we can't format
	}
	
	return prettyJSON, nil
}