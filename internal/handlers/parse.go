package handlers

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/vaishak/tf-migrate/internal/interfaces"
)

type ParseHandler struct {
	interfaces.BaseHandler
}

func NewParseHandler() interfaces.TransformationHandler {
	return &ParseHandler{}
}

func (h *ParseHandler) Handle(ctx *interfaces.TransformContext) (*interfaces.TransformContext, error) {
	file, diags := hclwrite.ParseConfig(ctx.Content, ctx.Filename, hcl.Pos{Line: 1, Column: 1})

	if diags.HasErrors() {
		ctx.Diagnostics = append(ctx.Diagnostics, diags...)
		return ctx, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}

	ctx.AST = file

	if diags != nil && !diags.HasErrors() {
		ctx.Diagnostics = append(ctx.Diagnostics, diags...)
	}

	return h.CallNext(ctx)
}