package handlers

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/cloudflare/tf-migrate/internal/transform"
)

type FormatterHandler struct {
	transform.BaseHandler
	log hclog.Logger
}

func NewFormatterHandler(log hclog.Logger) transform.TransformationHandler {
	return &FormatterHandler{
		log: log,
	}
}

func (h *FormatterHandler) Handle(ctx *transform.Context) (*transform.Context, error) {
	if ctx.CFGFile == nil {
		return ctx, fmt.Errorf("CFGFile is nil - cannot format without CFGFile")
	}

	bytes := ctx.CFGFile.Bytes()
	formatted := hclwrite.Format(bytes)

	ctx.Content = formatted

	return h.Next(ctx)
}
