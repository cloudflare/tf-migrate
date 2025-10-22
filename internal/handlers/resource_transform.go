package handlers

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

type ResourceTransformHandler struct {
	transform.BaseHandler
	log      hclog.Logger
	provider transform.Provider
}

func NewResourceTransformHandler(log hclog.Logger, provider transform.Provider) transform.TransformationHandler {
	return &ResourceTransformHandler{
		log:      log,
		provider: provider,
	}
}

func (h *ResourceTransformHandler) Handle(ctx *transform.Context) (*transform.Context, error) {
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
		migrator := h.provider.GetMigrator(resourceType)
		if migrator == nil {
			h.log.Debug("No migrator found for resource type", "type", resourceType)
			continue
		}

		result, err := migrator.TransformConfig(ctx, block)
		if err != nil {
			h.log.Error("Error transforming resource", "type", resourceType, "error", err)
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

	return h.Next(ctx)
}
