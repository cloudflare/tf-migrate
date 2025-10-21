package handlers

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal/interfaces"
)

type FormatterHandler struct {
	interfaces.BaseHandler
}

func NewFormatterHandler() interfaces.TransformationHandler {
	return &FormatterHandler{}
}

func (h *FormatterHandler) Handle(ctx *interfaces.TransformContext) (*interfaces.TransformContext, error) {
	if ctx.AST == nil {
		return ctx, fmt.Errorf("AST is nil - cannot format without AST")
	}

	bytes := ctx.AST.Bytes()
	formatted := hclwrite.Format(bytes)

	ctx.Content = formatted

	return h.CallNext(ctx)
}
