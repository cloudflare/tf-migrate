package handlers

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

type ParseHandler struct {
	transform.BaseHandler
	log hclog.Logger
}

func NewParseHandler(log hclog.Logger) transform.TransformationHandler {
	return &ParseHandler{
		log: log,
	}
}

func (h *ParseHandler) Handle(ctx *transform.Context) (*transform.Context, error) {
	file, diags := hclwrite.ParseConfig(ctx.Content, ctx.Filename, hcl.Pos{Line: 1, Column: 1})

	if diags.HasErrors() {
		ctx.Diagnostics = append(ctx.Diagnostics, diags...)
		return ctx, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}

	ctx.AST = file

	if diags != nil && !diags.HasErrors() {
		ctx.Diagnostics = append(ctx.Diagnostics, diags...)
	}

	return h.Next(ctx)
}
