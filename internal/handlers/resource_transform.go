package handlers

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/vaishak/tf-migrate/internal/interfaces"
	"github.com/vaishak/tf-migrate/internal/logger"
	"github.com/vaishak/tf-migrate/internal/registry"
)

type ResourceTransformHandler struct {
	interfaces.BaseHandler
	registry *registry.StrategyRegistry
}

func NewResourceTransformHandler(reg *registry.StrategyRegistry) interfaces.TransformationHandler {
	return &ResourceTransformHandler{
		registry: reg,
	}
}

func (h *ResourceTransformHandler) Handle(ctx *interfaces.TransformContext) (*interfaces.TransformContext, error) {
	if ctx.AST == nil {
		return ctx, fmt.Errorf("AST is nil - ParseHandler must run before ResourceTransformHandler")
	}

	body := ctx.AST.Body()
	blocks := body.Blocks()

	var blocksToRemove []*hclwrite.Block
	var blocksToAdd []*hclwrite.Block

	for _, block := range blocks {
		if block.Type() != "resource" {
			continue
		}

		labels := block.Labels()
		if len(labels) < 1 {
			continue
		}

		resourceType := labels[0]
		strategy := h.registry.Find(resourceType)

		if strategy == nil {
			logger.Debug("No strategy found for resource type", "type", resourceType)
			continue
		}

		result, err := strategy.TransformConfig(block)
		if err != nil {
			logger.Error("Error transforming resource", "type", resourceType, "error", err)
			ctx.Diagnostics = append(ctx.Diagnostics, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Failed to transform %s resource", resourceType),
				Detail:   err.Error(),
			})
			continue
		}

		if result.RemoveOriginal {
			blocksToRemove = append(blocksToRemove, block)
			if len(result.Blocks) > 0 {
				blocksToAdd = append(blocksToAdd, result.Blocks...)
			}
		}
		key := fmt.Sprintf("transformed_%s", resourceType)
		if count, ok := ctx.Metadata[key]; ok {
			ctx.Metadata[key] = count.(int) + 1
		} else {
			ctx.Metadata[key] = 1
		}
	}

	for _, block := range blocksToRemove {
		body.RemoveBlock(block)
	}

	for _, block := range blocksToAdd {
		body.AppendBlock(block)
	}

	return h.CallNext(ctx)
}
